package fraud

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	fraudv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/fraud/v1"
	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/payments/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
)

const RequestIDHeader = "x-request-id"

type Client struct {
	client fraudv1.FraudServiceClient
}

var _ application.RiskDecider = (*Client)(nil)

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: fraudv1.NewFraudServiceClient(conn)}
}

func (c *Client) Decide(ctx context.Context, input application.RiskInput) (*paymentsv1.RiskAssessment, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, RequestIDHeader, input.CorrelationID)
	response, err := c.client.Decide(ctx, &fraudv1.DecideRequest{
		IntegrationId: input.IntegrationID,
		PaymentId:     input.PaymentID,
		AttemptId:     input.AttemptID,
		Merchant:      input.Merchant,
		Customer:      input.Customer,
		Snapshot:      input.Snapshot,
		Signals:       input.Signals,
	})
	if err != nil {
		return nil, err
	}
	if response.GetAssessment() == nil {
		return nil, errors.New("fraud response carries no assessment")
	}
	return response.GetAssessment(), nil
}
