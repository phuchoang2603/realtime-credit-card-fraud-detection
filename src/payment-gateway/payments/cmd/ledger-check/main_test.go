package main

import (
	"encoding/base64"
	"fmt"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
)

const (
	created   = `{"paymentCreated":{"snapshot":{}}}`
	started   = `{"attemptStarted":{"attemptId":"attempt"}}`
	approved  = `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":"RISK_OUTCOME_APPROVE","policyVersion":"policy"}}}`
	requested = `{"processorOperationRequested":{"attemptId":"attempt","operationKey":"operation"}}`
	unknown   = `{"processorOutcomeUnknown":{"attemptId":"attempt"}}`
	succeeded = `{"paymentSucceeded":{"attemptId":"attempt","processorReference":"reference"}}`
	expired   = `{"paymentExpired":{}}`
)

func ledgerRow(t *testing.T, payload string, version uint64) row {
	t.Helper()
	event := &paymentsv1.PaymentEvent{}
	if err := protojson.Unmarshal([]byte(payload), event); err != nil {
		t.Fatal(err)
	}
	event.EventId, event.PaymentId, event.AggregateVersion = fmt.Sprintf("event-%d", version), "payment", version
	raw, err := proto.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return row{EventID: event.EventId, PaymentID: event.PaymentId, AggregateVersion: version, EventType: kind(event), Payload: base64.StdEncoding.EncodeToString(raw)}
}

func TestLedgerHistoryPartitions(t *testing.T) {
	for _, test := range []struct {
		name    string
		history []string
		valid   bool
	}{
		{"open", []string{created}, true},
		{"expired", []string{created, expired}, true},
		{"expiry before processor intent", []string{created, started, approved, expired}, true},
		{"success", []string{created, started, approved, requested, succeeded}, true},
		{"explicit approval reason", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":"RISK_OUTCOME_APPROVE","policyVersion":"policy","reasonCodes":["APPROVED"]}}}`, requested, succeeded}, true},
		{"risk decline", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":"RISK_OUTCOME_DECLINE","policyVersion":"policy"}}}`, `{"attemptDeclined":{"attemptId":"attempt","source":"DECLINE_SOURCE_RISK","reason":"declined"}}`}, true},
		{"unknown resolved", []string{created, started, approved, requested, unknown, succeeded}, true},
		{"missing creation", []string{expired}, false},
		{"attempt before creation", []string{started}, false},
		{"missing assessment", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt"}}`}, false},
		{"unspecified assessment", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"policyVersion":"policy"}}}`}, false},
		{"unknown assessment", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":99,"policyVersion":"policy"}}}`}, false},
		{"missing policy", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":"RISK_OUTCOME_APPROVE"}}}`}, false},
		{"empty reason", []string{created, started, `{"riskDecisionRecorded":{"attemptId":"attempt","assessment":{"outcome":"RISK_OUTCOME_APPROVE","policyVersion":"policy","reasonCodes":[""]}}}`}, false},
		{"expiry during request", []string{created, started, approved, requested, expired}, false},
		{"expiry during unknown outcome", []string{created, started, approved, requested, unknown, expired}, false},
		{"after terminal outcome", []string{created, expired, started}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &lifecycle{attempts: map[string]string{}}
			var err error
			for i, payload := range test.history {
				r := ledgerRow(t, payload, uint64(i+1))
				event, decodeErr := decode(r)
				if decodeErr != nil {
					t.Fatal(decodeErr)
				}
				if err = apply(state, r, event); err != nil {
					break
				}
			}
			if (err == nil) != test.valid {
				t.Fatalf("history validity = %t, want %t: %v", err == nil, test.valid, err)
			}
		})
	}
}

func TestLedgerRejectsVersionGapOrRepeat(t *testing.T) {
	for _, version := range []uint64{0, 1, 3} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			r := ledgerRow(t, expired, version)
			event, err := decode(r)
			if err != nil {
				t.Fatal(err)
			}
			state := &lifecycle{version: 1, attempts: map[string]string{}}
			if err := apply(state, r, event); err == nil {
				t.Fatal("accepted noncontiguous version")
			}
		})
	}
}

func TestLedgerRejectsCorruptEnvelope(t *testing.T) {
	for name, mutate := range map[string]func(*row){
		"base64":     func(r *row) { r.Payload = "!" },
		"protobuf":   func(r *row) { r.Payload = base64.StdEncoding.EncodeToString([]byte{0xff}) },
		"event id":   func(r *row) { r.EventID = "other" },
		"payment id": func(r *row) { r.PaymentID = "other" },
		"version":    func(r *row) { r.AggregateVersion++ },
		"event type": func(r *row) { r.EventType = "payment_expired" },
	} {
		t.Run(name, func(t *testing.T) {
			r := ledgerRow(t, created, 1)
			mutate(&r)
			if _, err := decode(r); err == nil {
				t.Fatal("corrupt row accepted")
			}
		})
	}
}
