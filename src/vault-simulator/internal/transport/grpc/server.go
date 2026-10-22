package grpctransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/gen/vault/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/domain"
)

const (
	LivenessService  = "liveness"
	ReadinessService = "readiness"
	readinessPeriod  = 2 * time.Second
)

type Vault interface {
	Tokenize(ctx context.Context, integrationID uuid.UUID, cardNumber string) (string, application.Method, error)
	Describe(ctx context.Context, integrationID uuid.UUID, token string) (application.Method, error)
	Resolve(ctx context.Context, integrationID uuid.UUID, token string) (application.Fixture, error)
}

type Readiness interface {
	Ready(ctx context.Context) error
}

type Server struct {
	vaultv1.UnimplementedVaultServiceServer
	vault           Vault
	readiness       Readiness
	authenticator   *Authenticator
	tlsConfig       *tls.Config
	shutdownTimeout time.Duration
	health          *health.Server
	registry        *prometheus.Registry
	requests        *prometheus.CounterVec
	denials         *prometheus.CounterVec
}

func NewServer(cfg config.Config, vault Vault, readiness Readiness) (*Server, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load vault TLS key pair: %w", err)
	}
	server := &Server{
		vault:         vault,
		readiness:     readiness,
		authenticator: NewAuthenticator(cfg.Callers),
		tlsConfig: &tls.Config{
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS13,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		health:          health.NewServer(),
		registry:        prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vault_requests_total", Help: "Vault gRPC requests by method and status code.",
		}, []string{"method", "code"}),
		denials: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "vault_authorization_failures_total", Help: "Vault calls rejected by authentication or permission checks.",
		}, []string{"method", "reason"}),
	}
	server.registry.MustRegister(server.requests, server.denials)
	server.health.SetServingStatus(LivenessService, healthpb.HealthCheckResponse_SERVING)
	server.health.SetServingStatus(ReadinessService, healthpb.HealthCheckResponse_NOT_SERVING)
	return server, nil
}

func (s *Server) Run(ctx context.Context, grpcListener, metricsListener net.Listener) error {
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(s.tlsConfig)),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			s.observe,
			s.authenticator.Unary(func(method, reason string) { s.denials.WithLabelValues(method, reason).Inc() }),
		),
	)
	vaultv1.RegisterVaultServiceServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, s.health)

	metrics := http.NewServeMux()
	metrics.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}))
	metricsServer := &http.Server{Handler: metrics, ReadHeaderTimeout: 5 * time.Second}

	errs := make(chan error, 2)
	go func() { errs <- grpcServer.Serve(grpcListener) }()
	go func() {
		if err := metricsServer.Serve(metricsListener); !errors.Is(err, http.ErrServerClosed) {
			errs <- err
			return
		}
		errs <- nil
	}()
	healthCtx, stopHealth := context.WithCancel(ctx)
	defer stopHealth()
	go s.pollReadiness(healthCtx)

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errs:
	}
	s.health.Shutdown()
	stopHealth()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		_ = metricsServer.Close()
	}
	return runErr
}

func (s *Server) pollReadiness(ctx context.Context) {
	check := func() {
		checkCtx, cancel := context.WithTimeout(ctx, readinessPeriod)
		defer cancel()
		state := healthpb.HealthCheckResponse_SERVING
		if err := s.readiness.Ready(checkCtx); err != nil {
			state = healthpb.HealthCheckResponse_NOT_SERVING
		}
		if ctx.Err() == nil {
			s.health.SetServingStatus(ReadinessService, state)
		}
	}
	check()
	ticker := time.NewTicker(readinessPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

func (s *Server) observe(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	s.requests.WithLabelValues(info.FullMethod, code.String()).Inc()
	slog.InfoContext(ctx, "vault request",
		"method", info.FullMethod,
		"code", code.String(),
		"caller", CallerName(ctx),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return resp, err
}

func (s *Server) TokenizeCard(ctx context.Context, req *vaultv1.TokenizeCardRequest) (*vaultv1.TokenizeCardResponse, error) {
	integrationID, err := parseIntegration(req.GetIntegrationId())
	if err != nil {
		return nil, err
	}
	token, method, err := s.vault.Tokenize(ctx, integrationID, req.GetCardNumber())
	if err != nil {
		return nil, mapError(err)
	}
	return &vaultv1.TokenizeCardResponse{Token: token, Metadata: methodMetadata(method)}, nil
}

func (s *Server) DescribeMethod(ctx context.Context, req *vaultv1.DescribeMethodRequest) (*vaultv1.MethodMetadata, error) {
	integrationID, err := parseIntegration(req.GetIntegrationId())
	if err != nil {
		return nil, err
	}
	method, err := s.vault.Describe(ctx, integrationID, req.GetToken())
	if err != nil {
		return nil, mapError(err)
	}
	return methodMetadata(method), nil
}

func (s *Server) ResolveForProcessing(ctx context.Context, req *vaultv1.ResolveRequest) (*vaultv1.ProcessingFixture, error) {
	integrationID, err := parseIntegration(req.GetIntegrationId())
	if err != nil {
		return nil, err
	}
	fixture, err := s.vault.Resolve(ctx, integrationID, req.GetToken())
	if err != nil {
		return nil, mapError(err)
	}
	return &vaultv1.ProcessingFixture{CardReference: fixture.CardReference, Behavior: protoBehavior(fixture.Behavior)}, nil
}

func parseIntegration(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "integration_id must be a UUID")
	}
	return id, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, application.ErrNotAllowListed):
		return status.Error(codes.InvalidArgument, application.ErrNotAllowListed.Error())
	case errors.Is(err, application.ErrTokenNotFound):
		return status.Error(codes.NotFound, "token not found")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "canceled")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func methodMetadata(method application.Method) *vaultv1.MethodMetadata {
	return &vaultv1.MethodMetadata{Type: method.Type, Fingerprint: method.Fingerprint}
}

func protoBehavior(behavior domain.Behavior) vaultv1.Behavior {
	switch behavior {
	case domain.BehaviorOK:
		return vaultv1.Behavior_BEHAVIOR_APPROVE
	case domain.BehaviorNSF:
		return vaultv1.Behavior_BEHAVIOR_INSUFFICIENT_FUNDS
	case domain.BehaviorIssuer:
		return vaultv1.Behavior_BEHAVIOR_ISSUER_DECLINE
	case domain.BehaviorInvalid:
		return vaultv1.Behavior_BEHAVIOR_INVALID
	default:
		return vaultv1.Behavior_BEHAVIOR_UNSPECIFIED
	}
}
