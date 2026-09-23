package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/internal/config"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(config.Config{Address: ":0", ShutdownTimeout: 200 * time.Millisecond, FraudTarget: "dns:///localhost:8000", PredictionTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCorrelationHeaderIsPreservedOrGenerated(t *testing.T) {
	s := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recording := httptest.NewRecorder()
	s.Handler().ServeHTTP(recording, request)
	if recording.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected generated request ID")
	}
	request = httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "caller-42")
	recording = httptest.NewRecorder()
	s.Handler().ServeHTTP(recording, request)
	if got := recording.Header().Get("X-Request-ID"); got != "caller-42" {
		t.Fatalf("request ID = %q", got)
	}
	request = httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "long-"+strings.Repeat("x", 200))
	recording = httptest.NewRecorder()
	s.Handler().ServeHTTP(recording, request)
	if got := recording.Header().Get("X-Request-ID"); len([]rune(got)) != maxRequestIDLength {
		t.Fatalf("bounded request ID length = %d", len([]rune(got)))
	}
}

func TestActiveRequestDrain(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(fmt.Sprint("deadline=", expire), func(t *testing.T) {
			s := newTestServer(t)
			entered, release := make(chan struct{}), make(chan struct{})
			s.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				select {
				case <-release:
					w.WriteHeader(http.StatusNoContent)
				case <-r.Context().Done():
				}
			})
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- s.Serve(ctx, listener) }()
			responseDone := make(chan struct{})
			go func() {
				defer close(responseDone)
				response, err := http.Get("http://" + listener.Addr().String())
				if err == nil {
					response.Body.Close()
				}
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("request did not start")
			}
			cancel()
			if !expire {
				close(release)
			}
			select {
			case err := <-done:
				if expire != (err != nil) {
					t.Fatalf("expiry=%v error=%v", expire, err)
				}
			case <-time.After(time.Second):
				t.Fatal("unbounded shutdown")
			}
			select {
			case <-responseDone:
			case <-time.After(time.Second):
				t.Fatal("client did not close")
			}
		})
	}
}
