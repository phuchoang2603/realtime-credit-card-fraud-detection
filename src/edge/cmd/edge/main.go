package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/internal/config"
	"github.com/phuchoang2603/realtime-credit-card-fraud-detection/edge/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("invalid edge configuration", "error", err)
		os.Exit(1)
	}
	edge, err := server.New(cfg)
	if err != nil {
		slog.Error("create edge service", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("starting edge service", "addr", cfg.Address)
	if err := edge.Run(ctx); err != nil {
		slog.Error("edge service stopped with error", "error", err)
		os.Exit(1)
	}
}
