package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/adapters/crypto"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/adapters/postgres"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/healthcheck"
	grpctransport "github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/transport/grpc"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var err error
	switch command {
	case "serve":
		err = serve(ctx)
	case "migrate":
		err = migrate(ctx)
	case "healthcheck":
		err = healthcheckCommand(ctx)
	default:
		slog.Error("unknown command", "command", command)
		os.Exit(2)
	}
	if err != nil {
		slog.Error("vault simulator failed", "command", command, "error", err)
		os.Exit(1)
	}
}

func healthcheckCommand(ctx context.Context) error {
	if len(os.Args) != 3 || (os.Args[2] != "liveness" && os.Args[2] != "readiness") {
		return errors.New("usage: vault-simulator healthcheck <liveness|readiness>")
	}
	cfg, err := config.HealthcheckFromLookup(os.Getenv)
	if err != nil {
		return err
	}
	return healthcheck.Check(ctx, cfg, os.Args[2])
}

func migrate(ctx context.Context) error {
	url, err := config.MigrateURL(os.Getenv)
	if err != nil {
		return err
	}
	if err := postgres.Migrate(ctx, url); err != nil {
		return err
	}
	slog.Info("migrations applied")
	return nil
}

func serve(ctx context.Context) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	store, err := postgres.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	sealer, err := crypto.NewSealer(cfg.EncryptionKey)
	if err != nil {
		return err
	}
	service := application.NewService(store, sealer, cfg.FingerprintKey)
	server, err := grpctransport.NewServer(cfg, service, store)
	if err != nil {
		return err
	}
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	metricsListener, err := net.Listen("tcp", cfg.MetricsAddr)
	if err != nil {
		_ = grpcListener.Close()
		return err
	}
	slog.Info("starting vault simulator", "grpc_addr", cfg.GRPCAddr, "metrics_addr", cfg.MetricsAddr)
	return server.Run(ctx, grpcListener, metricsListener)
}
