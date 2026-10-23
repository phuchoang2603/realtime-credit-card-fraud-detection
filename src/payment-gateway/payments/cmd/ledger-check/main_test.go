package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	return row{EventID: event.EventId, PaymentID: event.PaymentId, AggregateVersion: version, EventType: fixtureEventType(t, payload), Payload: base64.StdEncoding.EncodeToString(raw)}
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

func fixtureEventType(t *testing.T, payload string) string {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &fields); err != nil {
		t.Fatal(err)
	}
	for field, eventType := range map[string]string{
		"paymentCreated":              "payment_created",
		"attemptStarted":              "attempt_started",
		"riskDecisionRecorded":        "risk_decision_recorded",
		"processorOperationRequested": "processor_operation_requested",
		"processorOutcomeUnknown":     "processor_outcome_unknown",
		"attemptDeclined":             "attempt_declined",
		"paymentSucceeded":            "payment_succeeded",
		"paymentExpired":              "payment_expired",
	} {
		if _, ok := fields[field]; ok {
			return eventType
		}
	}
	t.Fatal("fixture has no supported event")
	return ""
}

func writeLedger(t *testing.T, rows []row) string {
	t.Helper()
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunGroupsAndOrdersPayments(t *testing.T) {
	rows := []row{ledgerRow(t, expired, 2), ledgerRow(t, created, 1)}
	for _, payload := range []string{expired, created} {
		version := uint64(1)
		if payload == expired {
			version = 2
		}
		r := ledgerRow(t, payload, version)
		event, err := decode(r)
		if err != nil {
			t.Fatal(err)
		}
		event.PaymentId = "another-payment"
		event.EventId = "another-" + event.EventId
		raw, err := proto.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		r.PaymentID, r.EventID, r.Payload = event.PaymentId, event.EventId, base64.StdEncoding.EncodeToString(raw)
		rows = append(rows, r)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{writeLedger(t, rows)}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, &stderr)
	}
	if got, want := stdout.String(), "ledger-check: 4 events across 2 payments satisfy the lifecycle\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", &stderr)
	}
}

func TestRunReportsAllViolations(t *testing.T) {
	corrupt := ledgerRow(t, created, 1)
	corrupt.Payload = "!"
	rows := []row{corrupt, ledgerRow(t, started, 2)}
	var stdout, stderr bytes.Buffer
	if code := run([]string{writeLedger(t, rows)}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	for _, want := range []string{
		"payment version 1 (payment_created): decode payload:",
		"payment version 2 (attempt_started): aggregate version 2 follows 0",
		"ledger-check: 2 violation(s) across 1 payments and 2 events\n",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr %q missing %q", stderr.String(), want)
		}
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %s", &stdout)
	}
}

func TestRunInputFailures(t *testing.T) {
	invalid := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalid, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		args    []string
		code    int
		message string
	}{
		{"no path", nil, 2, "usage: ledger-check <ledger.json>"},
		{"extra path", []string{"one", "two"}, 2, "usage: ledger-check <ledger.json>"},
		{"unreadable", []string{filepath.Join(t.TempDir(), "missing.json")}, 1, "missing.json"},
		{"invalid json", []string{invalid}, 1, "unexpected end of JSON input"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(test.args, &stdout, &stderr); code != test.code {
				t.Errorf("exit = %d, want %d", code, test.code)
			}
			if !strings.Contains(stderr.String(), test.message) {
				t.Errorf("stderr = %q, want %q", stderr.String(), test.message)
			}
			if stdout.Len() != 0 {
				t.Fatalf("unexpected stdout: %s", &stdout)
			}
		})
	}
}

func TestRunOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	output, err := os.CreateTemp(t.TempDir(), "closed-output")
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{writeLedger(t, []row{ledgerRow(t, created, 1)})}, output, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "file already closed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
