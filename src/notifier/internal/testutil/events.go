package testutil

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/gen/payments/v1"
)

var OccurredAt = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func marshal(t *testing.T, event *paymentsv1.PaymentEvent) []byte {
	t.Helper()
	event.OccurredAt = timestamppb.New(OccurredAt)
	event.SchemaVersion = 1
	data, err := proto.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return data
}

func Created(t *testing.T, eventID, paymentID, integrationID, orderRef string) []byte {
	return marshal(t, &paymentsv1.PaymentEvent{
		EventId: eventID, PaymentId: paymentID, AggregateVersion: 1, IntegrationId: integrationID,
		Payload: &paymentsv1.PaymentEvent_PaymentCreated{PaymentCreated: &paymentsv1.PaymentCreated{
			Snapshot: &paymentsv1.CheckoutSnapshot{ExternalOrderId: orderRef},
		}},
	})
}

func Declined(t *testing.T, eventID, paymentID, integrationID, attemptID, reason string) []byte {
	return marshal(t, &paymentsv1.PaymentEvent{
		EventId: eventID, PaymentId: paymentID, AggregateVersion: 2, IntegrationId: integrationID,
		Payload: &paymentsv1.PaymentEvent_AttemptDeclined{AttemptDeclined: &paymentsv1.AttemptDeclined{
			AttemptId: attemptID, Source: paymentsv1.DeclineSource_DECLINE_SOURCE_PROCESSOR, Reason: reason,
		}},
	})
}

func Succeeded(t *testing.T, eventID, paymentID, integrationID, attemptID string) []byte {
	return marshal(t, &paymentsv1.PaymentEvent{
		EventId: eventID, PaymentId: paymentID, AggregateVersion: 3, IntegrationId: integrationID,
		Payload: &paymentsv1.PaymentEvent_PaymentSucceeded{PaymentSucceeded: &paymentsv1.PaymentSucceeded{
			AttemptId: attemptID, ProcessorReference: "proc-ref-secret",
		}},
	})
}

func Expired(t *testing.T, eventID, paymentID, integrationID string) []byte {
	return marshal(t, &paymentsv1.PaymentEvent{
		EventId: eventID, PaymentId: paymentID, AggregateVersion: 3, IntegrationId: integrationID,
		Payload: &paymentsv1.PaymentEvent_PaymentExpired{PaymentExpired: &paymentsv1.PaymentExpired{Present: true}},
	})
}
