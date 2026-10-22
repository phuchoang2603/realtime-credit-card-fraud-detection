package vault

import (
	"context"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/adapters/grpcerr"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/application"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/domain"
	vaultv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/vault/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct{ client vaultv1.VaultServiceClient }

func New(conn grpc.ClientConnInterface) *Client {
	return &Client{client: vaultv1.NewVaultServiceClient(conn)}
}

func (c *Client) Tokenize(ctx context.Context, integrationID, cardNumber string) (application.Token, error) {
	response, err := c.client.TokenizeCard(ctx, &vaultv1.TokenizeCardRequest{IntegrationId: integrationID, CardNumber: cardNumber})
	if err != nil {
		return application.Token{}, mapCall(err, "vault tokenize")
	}
	return application.Token{
		Value:  response.GetToken(),
		Method: application.PaymentMethod{Type: response.GetMetadata().GetType(), Fingerprint: response.GetMetadata().GetFingerprint()},
	}, nil
}

func (c *Client) Describe(ctx context.Context, integrationID, token string) (application.PaymentMethod, error) {
	response, err := c.client.DescribeMethod(ctx, &vaultv1.DescribeMethodRequest{IntegrationId: integrationID, Token: token})
	if err != nil {
		return application.PaymentMethod{}, mapCall(err, "vault describe")
	}
	return application.PaymentMethod{Type: response.GetType(), Fingerprint: response.GetFingerprint()}, nil
}

// A vault rejecting Edge's own credential is a gateway fault, not the caller's.
func mapCall(err error, operation string) error {
	if status.Code(err) == codes.Unauthenticated {
		return domain.New(domain.ErrUnavailable, operation+": credential rejected")
	}
	return grpcerr.Map(err, operation)
}
