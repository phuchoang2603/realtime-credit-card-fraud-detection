package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	fraudv1 "github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/gen/fraud/v1"
	"github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	connection *grpc.ClientConn
	fraud      fraudv1.FraudServiceClient
	config     config.Config
	server     *http.Server
	ready      atomic.Bool
}

const maxRequestIDLength = 128

func New(cfg config.Config) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	connection, err := grpc.NewClient(cfg.FraudTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create fraud client: %w", err)
	}
	s := &Server{config: cfg, connection: connection, fraud: fraudv1.NewFraudServiceClient(connection)}
	s.server = &http.Server{
		Addr:              cfg.Address,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.readiness)
	mux.HandleFunc("POST /predict", s.predict)
	return s.withCorrelation(mux)
}

func (s *Server) Close() error { return s.connection.Close() }

func (s *Server) Run(ctx context.Context) error {
	// The client connection is process-scoped; a close error after serving is not actionable.
	defer func() { _ = s.Close() }()
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.config.Address, err)
	}
	return s.Serve(ctx, listener)
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	errCh := make(chan error, 1)
	s.ready.Store(true)
	go func() { errCh <- s.server.Serve(listener) }()

	select {
	case err := <-errCh:
		s.ready.Store(false)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		s.ready.Store(false)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		closeErr := s.server.Close()
		return errors.Join(fmt.Errorf("shutdown edge service: %w", err), closeErr)
	}
	err := <-errCh
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readiness(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) withCorrelation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.Map(func(character rune) rune {
			if character >= 32 && character < 127 {
				return character
			}
			return -1
		}, r.Header.Get("X-Request-ID"))
		requestID = strings.TrimSpace(requestID)
		if runes := []rune(requestID); len(runes) > maxRequestIDLength {
			requestID = string(runes[:maxRequestIDLength])
		}
		if requestID == "" {
			requestID = rand.Text()
		}
		w.Header().Set("X-Request-ID", requestID)
		// Kubernetes probes poll continuously; keep them out of the request log.
		if r.URL.Path != "/health" && r.URL.Path != "/ready" {
			slog.Info("edge request", "service", "edge", "request_id", requestID, "path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
