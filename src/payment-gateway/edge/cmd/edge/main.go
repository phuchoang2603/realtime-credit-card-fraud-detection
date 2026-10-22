package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/grpcclient"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/integrations"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/payments"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/vault"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/config"
	httptransport "github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/transport/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("edge service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	registry, err := integrations.Load(cfg.IntegrationsFile)
	if err != nil {
		return err
	}

	var paymentsTransport credentials.TransportCredentials
	if cfg.PaymentsCAFile != "" {
		if paymentsTransport, err = grpcclient.TLSFromCAFile(cfg.PaymentsCAFile); err != nil {
			return err
		}
	}
	paymentsConn, err := grpcclient.Dial(cfg.PaymentsAddr, cfg.PaymentsTimeout, paymentsTransport)
	if err != nil {
		return err
	}
	defer func() { _ = paymentsConn.Close() }()

	vaultTransport, err := grpcclient.TLSFromCAFile(cfg.VaultCAFile)
	if err != nil {
		return err
	}
	vaultConn, err := grpcclient.Dial(cfg.VaultAddr, cfg.VaultTimeout, vaultTransport,
		grpc.WithPerRPCCredentials(grpcclient.BearerCredentials(cfg.VaultCredential)))
	if err != nil {
		return err
	}
	defer func() { _ = vaultConn.Close() }()

	service := application.New(application.Config{
		Payments:                payments.New(paymentsConn),
		Vault:                   vault.New(vaultConn),
		Integrations:            registry,
		FingerprintKey:          cfg.FingerprintKey,
		SyntheticFingerprintKey: cfg.SyntheticFingerprintKey,
		PublicBaseURL:           cfg.PublicOrigin(),
	})
	edge, err := httptransport.New(httptransport.Options{
		Address:            cfg.Address,
		ShutdownTimeout:    cfg.ShutdownTimeout,
		Service:            service,
		PublicBaseURL:      cfg.PublicOrigin(),
		CookieKey:          []byte(cfg.CheckoutCookieKey),
		CookieSecure:       cfg.CookieSecure,
		TrustedProxyHeader: cfg.TrustedProxyHeader,
		TrustedProxyHops:   cfg.TrustedProxyHops,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("starting edge service", "addr", cfg.Address)
	return edge.Run(ctx)
}
