package grpc

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestReadinessFollowsOwnDatabaseOnly(t *testing.T) {
	var healthy atomic.Bool
	server := New(func(context.Context) error {
		if healthy.Load() {
			return nil
		}
		return errors.New("database down")
	})
	server.pollInterval = 20 * time.Millisecond
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, listener, time.Second) }()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	client := healthpb.NewHealthClient(conn)
	status := func(service string) healthpb.HealthCheckResponse_ServingStatus {
		resp, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{Service: service})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Status
	}
	wait := func(want healthpb.HealthCheckResponse_ServingStatus) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if status(ReadinessService) == want {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("readiness never became %v", want)
	}

	wait(healthpb.HealthCheckResponse_NOT_SERVING)
	if status(LivenessService) != healthpb.HealthCheckResponse_SERVING {
		t.Fatal("liveness must stay SERVING while the database is down")
	}
	healthy.Store(true)
	wait(healthpb.HealthCheckResponse_SERVING)
	healthy.Store(false)
	wait(healthpb.HealthCheckResponse_NOT_SERVING)

	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
