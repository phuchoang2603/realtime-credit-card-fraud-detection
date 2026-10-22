package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/config"
)

func Check(ctx context.Context, cfg config.Healthcheck, service string) error {
	transport := grpc.WithTransportCredentials(insecure.NewCredentials())
	if cfg.CAFile != "" {
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return fmt.Errorf("read CA file: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return errors.New("CA file contains no certificates")
		}
		transport = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: cfg.ServerName,
			MinVersion: tls.VersionTLS13,
		}))
	}
	conn, err := grpc.NewClient(cfg.Address, transport)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	response, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: service})
	if err != nil {
		return err
	}
	if response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("%s is %s", service, response.GetStatus())
	}
	return nil
}
