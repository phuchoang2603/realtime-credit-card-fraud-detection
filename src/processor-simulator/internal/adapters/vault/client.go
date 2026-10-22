package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/gen/vault/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/config"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/processor-simulator/internal/domain"
)

type bearer string

func (b bearer) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + string(b)}, nil
}

func (bearer) RequireTransportSecurity() bool { return true }

type Client struct {
	conn    *grpc.ClientConn
	client  vaultv1.VaultServiceClient
	timeout time.Duration
}

func NewClient(cfg config.Vault) (*Client, error) {
	pem, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read VAULT_CA_FILE: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("VAULT_CA_FILE contains no certificates")
	}
	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: cfg.ServerName,
			MinVersion: tls.VersionTLS13,
		})),
		grpc.WithPerRPCCredentials(bearer(cfg.Credential)),
	)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, client: vaultv1.NewVaultServiceClient(conn), timeout: cfg.Timeout}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Resolve(ctx context.Context, integrationID, token string) (domain.Behavior, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	fixture, err := c.client.ResolveForProcessing(ctx, &vaultv1.ResolveRequest{IntegrationId: integrationID, Token: token})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound, codes.InvalidArgument:
			return "", application.ErrUnknownToken
		default:
			return "", fmt.Errorf("%w: %s", application.ErrVaultUnavailable, status.Code(err))
		}
	}
	switch fixture.GetBehavior() {
	case vaultv1.Behavior_BEHAVIOR_APPROVE:
		return domain.BehaviorApprove, nil
	case vaultv1.Behavior_BEHAVIOR_INSUFFICIENT_FUNDS:
		return domain.BehaviorInsufficientFunds, nil
	case vaultv1.Behavior_BEHAVIOR_ISSUER_DECLINE:
		return domain.BehaviorIssuerDecline, nil
	case vaultv1.Behavior_BEHAVIOR_INVALID:
		return domain.BehaviorInvalid, nil
	default:
		return "", fmt.Errorf("%w: unsupported behavior", application.ErrVaultUnavailable)
	}
}
