package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	LivenessService  = "liveness"
	ReadinessService = "readiness"
)

func WatchReadiness(ctx context.Context, server *health.Server, interval time.Duration, check func(ctx context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	previous := healthpb.HealthCheckResponse_UNKNOWN
	for {
		probeCtx, cancel := context.WithTimeout(ctx, interval)
		err := check(probeCtx)
		cancel()
		if ctx.Err() != nil {
			return
		}
		state := healthpb.HealthCheckResponse_SERVING
		if err != nil {
			state = healthpb.HealthCheckResponse_NOT_SERVING
		}
		if state != previous {
			slog.Info("readiness changed", "status", state.String(), "error", err)
			previous = state
		}
		server.SetServingStatus(ReadinessService, state)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
