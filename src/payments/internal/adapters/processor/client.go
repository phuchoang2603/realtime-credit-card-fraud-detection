package processor

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	processorv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/processor/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/application"
)

const requestIDHeader = "x-request-id"

type Client struct {
	client processorv1.ProcessorServiceClient
}

var _ application.ProcessorGateway = (*Client)(nil)

func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: processorv1.NewProcessorServiceClient(conn)}
}

func (c *Client) Sale(ctx context.Context, request application.SaleRequest) (application.Operation, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, requestIDHeader, request.CorrelationID)
	operation, err := c.client.Sale(ctx, &processorv1.SaleRequest{
		OperationKey:       request.OperationKey,
		IntegrationId:      request.IntegrationID,
		PaymentMethodToken: request.PaymentMethodToken,
		AmountMinorUnits:   request.AmountMinorUnits,
		Currency:           request.Currency,
	})
	if err != nil {
		return application.Operation{}, err
	}
	return mapOperation(operation)
}

func (c *Client) GetOperation(ctx context.Context, operationKey string) (application.Operation, error) {
	operation, err := c.client.GetOperation(ctx, &processorv1.GetOperationRequest{OperationKey: operationKey})
	if status.Code(err) == codes.NotFound {
		return application.Operation{}, application.ErrOperationNotFound
	}
	if err != nil {
		return application.Operation{}, err
	}
	return mapOperation(operation)
}

func mapOperation(operation *processorv1.Operation) (application.Operation, error) {
	result := application.Operation{
		OperationKey:       operation.GetOperationKey(),
		DeclineReason:      operation.GetDeclineReason(),
		ProcessorReference: operation.GetProcessorReference(),
	}
	switch operation.GetOutcome() {
	case processorv1.OperationOutcome_OPERATION_OUTCOME_APPROVED:
		result.Outcome = application.OutcomeApproved
	case processorv1.OperationOutcome_OPERATION_OUTCOME_DECLINED:
		result.Outcome = application.OutcomeDeclined
	default:
		return application.Operation{}, fmt.Errorf("processor returned unspecified outcome for %q", operation.GetOperationKey())
	}
	return result, nil
}
