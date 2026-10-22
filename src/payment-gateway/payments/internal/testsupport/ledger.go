package testsupport

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
)

func (d *Database) Events(t testing.TB, paymentID string) []*paymentsv1.PaymentEvent {
	t.Helper()
	rows, err := d.App.Query(context.Background(), `SELECT payload FROM payment_events WHERE payment_id = $1 ORDER BY aggregate_version`, paymentID)
	if err != nil {
		t.Fatal(err)
	}
	payloads, err := pgx.CollectRows(rows, pgx.RowTo[[]byte])
	if err != nil {
		t.Fatal(err)
	}
	events := make([]*paymentsv1.PaymentEvent, len(payloads))
	for i, payload := range payloads {
		events[i] = &paymentsv1.PaymentEvent{}
		if err := proto.Unmarshal(payload, events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return events
}

func EventKinds(events []*paymentsv1.PaymentEvent) []string {
	kinds := make([]string, len(events))
	for i, event := range events {
		switch event.GetPayload().(type) {
		case *paymentsv1.PaymentEvent_PaymentCreated:
			kinds[i] = "created"
		case *paymentsv1.PaymentEvent_AttemptStarted:
			kinds[i] = "started"
		case *paymentsv1.PaymentEvent_RiskDecisionRecorded:
			kinds[i] = "risk"
		case *paymentsv1.PaymentEvent_ProcessorOperationRequested:
			kinds[i] = "requested"
		case *paymentsv1.PaymentEvent_ProcessorOutcomeUnknown:
			kinds[i] = "unknown"
		case *paymentsv1.PaymentEvent_AttemptDeclined:
			kinds[i] = "declined"
		case *paymentsv1.PaymentEvent_PaymentSucceeded:
			kinds[i] = "succeeded"
		case *paymentsv1.PaymentEvent_PaymentExpired:
			kinds[i] = "expired"
		}
	}
	return kinds
}

func (d *Database) CountEvents(t testing.TB) int {
	t.Helper()
	var count int
	if err := d.App.QueryRow(context.Background(), `SELECT count(*) FROM payment_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
