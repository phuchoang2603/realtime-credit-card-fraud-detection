package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/app"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/healthcheck"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "payments"))
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "healthcheck" {
		os.Exit(healthcheckCommand())
	}
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("invalid payments configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch command {
	case "migrate":
		if err := cfg.ValidateMigrate(); err != nil {
			slog.Error("invalid payments configuration", "error", err)
			os.Exit(1)
		}
		if err := postgres.Migrate(ctx, cfg.MigrationURL()); err != nil {
			slog.Error("migrate payments database", "error", err)
			os.Exit(1)
		}
		slog.Info("payments migrations applied")
	case "serve":
		if err := serve(ctx, cfg); err != nil {
			slog.Error("payments stopped with error", "error", err)
			os.Exit(1)
		}
	default:
		slog.Error("unknown command", "command", command)
		os.Exit(2)
	}
}

func serve(ctx context.Context, cfg config.Config) error {
	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		application.Close()
		return err
	}
	metricsListener, err := net.Listen("tcp", cfg.MetricsAddr)
	if err != nil {
		_ = grpcListener.Close()
		application.Close()
		return err
	}
	slog.Info("starting payments service", "grpc", cfg.GRPCAddr, "metrics", cfg.MetricsAddr)
	return application.Serve(ctx, grpcListener, metricsListener)
}

func healthcheckCommand() int {
	if len(os.Args) != 3 || (os.Args[2] != "liveness" && os.Args[2] != "readiness") {
		slog.Error("usage: payments healthcheck <liveness|readiness>")
		return 2
	}
	cfg, err := config.HealthcheckFromLookup(os.Getenv)
	if err != nil {
		slog.Error("invalid healthcheck configuration", "error", err)
		return 1
	}
	if err := healthcheck.Check(context.Background(), cfg, os.Args[2]); err != nil {
		slog.Error("healthcheck failed", "service", os.Args[2], "error", err)
		return 1
	}
	return 0
}
