package application

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/domain"
	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBuildBodyCarriesOnlyOutcomeFields(t *testing.T) {
	event := &paymentsv1.PaymentEvent{
		EventId: "e1", PaymentId: "p1", IntegrationId: "i1",
		OccurredAt: timestamppb.New(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)),
		Payload: &paymentsv1.PaymentEvent_PaymentSucceeded{PaymentSucceeded: &paymentsv1.PaymentSucceeded{
			AttemptId: "a1", ProcessorReference: "secret-processor-ref",
		}},
	}
	raw, err := proto.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded paymentsv1.PaymentEvent
	if err := proto.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	outcome, ok := outcomeOf(&decoded)
	if !ok {
		t.Fatal("expected outcome")
	}
	body, err := domain.BuildBody(domain.Event{ID: decoded.GetEventId(), PaymentID: decoded.GetPaymentId(), OccurredAt: decoded.GetOccurredAt().AsTime()}, outcome, "order-9")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"event_id":"e1","type":"payment.succeeded","payment_id":"p1","order_reference":"order-9","attempt_id":"a1","status":"SUCCEEDED","occurred_at":"2026-10-01T12:00:00Z"}`
	if string(body) != want {
		t.Fatalf("body = %s\nwant   %s", body, want)
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"secret-processor-ref", "token", "fingerprint", "integration_id"} {
		if strings.Contains(string(body), forbidden) {
			t.Errorf("body leaks %q", forbidden)
		}
	}
}

func TestOutcomeMapping(t *testing.T) {
	declined := &paymentsv1.PaymentEvent{Payload: &paymentsv1.PaymentEvent_AttemptDeclined{AttemptDeclined: &paymentsv1.AttemptDeclined{AttemptId: "a", Reason: "INSUFFICIENT_FUNDS"}}}
	expired := &paymentsv1.PaymentEvent{Payload: &paymentsv1.PaymentEvent_PaymentExpired{PaymentExpired: &paymentsv1.PaymentExpired{Present: true}}}
	started := &paymentsv1.PaymentEvent{Payload: &paymentsv1.PaymentEvent_AttemptStarted{AttemptStarted: &paymentsv1.AttemptStarted{AttemptId: "a"}}}
	tests := []struct {
		name  string
		event *paymentsv1.PaymentEvent
		want  domain.Outcome
		ok    bool
	}{
		{"declined", declined, domain.Outcome{Type: "attempt.declined", Status: "DECLINED", AttemptID: "a", Reason: "INSUFFICIENT_FUNDS"}, true},
		{"expired", expired, domain.Outcome{Type: "payment.expired", Status: "EXPIRED"}, true},
		{"internal event", started, domain.Outcome{}, false},
	}
	for _, tt := range tests {
		got, ok := outcomeOf(tt.event)
		if ok != tt.ok || got != tt.want {
			t.Errorf("%s: got %+v %v, want %+v %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
