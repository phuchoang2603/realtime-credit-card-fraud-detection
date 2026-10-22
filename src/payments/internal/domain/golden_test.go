package domain_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/payments/gen/payments/v1"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
)

const testdataGlob = "../../../../contracts/payments/v1/testdata/*.json"

type goldenHistory struct {
	Name        string            `json:"name"`
	Valid       bool              `json:"valid"`
	InvalidRule string            `json:"invalid_rule"`
	Events      []json.RawMessage `json:"events"`
}

func loadHistories(t *testing.T) map[string]goldenHistory {
	t.Helper()
	files, err := filepath.Glob(testdataGlob)
	if err != nil || len(files) == 0 {
		t.Fatalf("golden histories not found: %v", err)
	}
	histories := make(map[string]goldenHistory, len(files))
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var history goldenHistory
		if err := json.Unmarshal(raw, &history); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		histories[filepath.Base(file)] = history
	}
	return histories
}

func parseEvents(t *testing.T, history goldenHistory) []*paymentsv1.PaymentEvent {
	t.Helper()
	events := make([]*paymentsv1.PaymentEvent, len(history.Events))
	for i, raw := range history.Events {
		event := &paymentsv1.PaymentEvent{}
		if err := protojson.Unmarshal(raw, event); err != nil {
			t.Fatalf("%s event %d: %v", history.Name, i+1, err)
		}
		events[i] = event
	}
	return events
}

func commandFor(t *testing.T, event *paymentsv1.PaymentEvent) domain.Command {
	t.Helper()
	envelope := domain.Envelope{EventID: event.GetEventId(), Now: event.GetOccurredAt().AsTime(), ExpectedVersion: event.GetAggregateVersion() - 1}
	switch payload := event.GetPayload().(type) {
	case *paymentsv1.PaymentEvent_PaymentCreated:
		return domain.CreatePayment{
			Envelope: envelope, PaymentID: event.GetPaymentId(), IntegrationID: event.GetIntegrationId(),
			CorrelationID: event.GetCorrelationId(), SessionID: "session",
			Merchant: payload.PaymentCreated.GetMerchant(), Customer: payload.PaymentCreated.GetCustomer(), Snapshot: payload.PaymentCreated.GetSnapshot(),
		}
	case *paymentsv1.PaymentEvent_AttemptStarted:
		return domain.StartAttempt{Envelope: envelope, AttemptID: payload.AttemptStarted.GetAttemptId(), Signals: payload.AttemptStarted.GetSignals()}
	case *paymentsv1.PaymentEvent_RiskDecisionRecorded:
		return domain.RecordRiskDecision{Envelope: envelope, AttemptID: payload.RiskDecisionRecorded.GetAttemptId(), Assessment: payload.RiskDecisionRecorded.GetAssessment()}
	case *paymentsv1.PaymentEvent_ProcessorOperationRequested:
		return domain.RequestProcessorOperation{Envelope: envelope, AttemptID: payload.ProcessorOperationRequested.GetAttemptId(), OperationKey: payload.ProcessorOperationRequested.GetOperationKey()}
	case *paymentsv1.PaymentEvent_ProcessorOutcomeUnknown:
		return domain.RecordOutcomeUnknown{Envelope: envelope, AttemptID: payload.ProcessorOutcomeUnknown.GetAttemptId()}
	case *paymentsv1.PaymentEvent_AttemptDeclined:
		return domain.DeclineAttempt{Envelope: envelope, AttemptID: payload.AttemptDeclined.GetAttemptId(), Source: payload.AttemptDeclined.GetSource(), Reason: payload.AttemptDeclined.GetReason()}
	case *paymentsv1.PaymentEvent_PaymentSucceeded:
		return domain.SucceedPayment{Envelope: envelope, AttemptID: payload.PaymentSucceeded.GetAttemptId(), ProcessorReference: payload.PaymentSucceeded.GetProcessorReference()}
	case *paymentsv1.PaymentEvent_PaymentExpired:
		return domain.ExpirePayment{Envelope: envelope}
	}
	t.Fatalf("event %s has no payload", event.GetEventId())
	return nil
}

func eventKind(event *paymentsv1.PaymentEvent) string {
	switch event.GetPayload().(type) {
	case *paymentsv1.PaymentEvent_PaymentCreated:
		return "created"
	case *paymentsv1.PaymentEvent_AttemptStarted:
		return "started"
	case *paymentsv1.PaymentEvent_RiskDecisionRecorded:
		if event.GetRiskDecisionRecorded().GetAssessment().GetOutcome() == paymentsv1.RiskOutcome_RISK_OUTCOME_DECLINE {
			return "decision_decline"
		}
		return "decision_approve"
	case *paymentsv1.PaymentEvent_ProcessorOperationRequested:
		return "requested"
	case *paymentsv1.PaymentEvent_ProcessorOutcomeUnknown:
		return "unknown"
	case *paymentsv1.PaymentEvent_AttemptDeclined:
		if event.GetAttemptDeclined().GetSource() == paymentsv1.DeclineSource_DECLINE_SOURCE_RISK {
			return "declined_risk"
		}
		return "declined_processor"
	case *paymentsv1.PaymentEvent_PaymentSucceeded:
		return "succeeded"
	case *paymentsv1.PaymentEvent_PaymentExpired:
		return "expired"
	}
	return ""
}

func TestGoldenHistoriesAreAcceptedOrRejectedForTheirRecordedRule(t *testing.T) {
	histories := loadHistories(t)
	for file, history := range histories {
		t.Run(file, func(t *testing.T) {
			events := parseEvents(t, history)
			state := domain.State{}
			for i, event := range events {
				next, produced, err := domain.Transition(state, commandFor(t, event))
				last := i == len(events)-1
				if history.Valid {
					if err != nil {
						t.Fatalf("valid history rejected at event %d: %v", i+1, err)
					}
					if produced.GetAggregateVersion() != event.GetAggregateVersion() {
						t.Fatalf("event %d version = %d, want %d", i+1, produced.GetAggregateVersion(), event.GetAggregateVersion())
					}
					if !proto.Equal(withoutEnvelope(produced), withoutEnvelope(event)) {
						t.Fatalf("event %d payload differs from the golden event", i+1)
					}
					state = next
					continue
				}
				if err == nil {
					if last {
						t.Fatalf("invalid history (%s) was accepted", history.InvalidRule)
					}
					state = next
					continue
				}
				if !last {
					t.Fatalf("invalid history rejected before its final event %d: %v", i+1, err)
				}
				if rule := domain.RuleOf(err); rule != history.InvalidRule {
					t.Fatalf("rejected for rule %q, recorded rule %q: %v", rule, history.InvalidRule, err)
				}
			}
			if history.Valid && state.Status != domain.StatusSucceeded && state.Status != domain.StatusExpired {
				t.Fatalf("valid history ends in non-terminal status %q", state.Status)
			}
		})
	}
}

func TestGoldenHistoriesCoverEveryLegalAndProhibitedTransition(t *testing.T) {
	legal := map[[2]string]bool{
		{"", "created"}:                       false,
		{"created", "started"}:                false,
		{"created", "expired"}:                false,
		{"started", "decision_decline"}:       false,
		{"started", "decision_approve"}:       false,
		{"decision_decline", "declined_risk"}: false,
		{"decision_approve", "requested"}:     false,
		{"requested", "succeeded"}:            false,
		{"requested", "declined_processor"}:   false,
		{"requested", "unknown"}:              false,
		{"unknown", "succeeded"}:              false,
		{"unknown", "declined_processor"}:     false,
		{"declined_risk", "started"}:          false,
		{"declined_processor", "started"}:     false,
		{"declined_risk", "expired"}:          false,
		{"declined_processor", "expired"}:     false,
		{"started", "expired"}:                false,
		{"decision_approve", "expired"}:       false,
	}
	rules := map[string]bool{}
	for _, history := range loadHistories(t) {
		events := parseEvents(t, history)
		if !history.Valid {
			rules[history.InvalidRule] = true
			continue
		}
		previous := ""
		for _, event := range events {
			kind := eventKind(event)
			if _, ok := legal[[2]string{previous, kind}]; ok {
				legal[[2]string{previous, kind}] = true
			}
			previous = kind
		}
	}
	for transition, covered := range legal {
		if !covered {
			t.Errorf("no valid golden history covers %q -> %q", transition[0], transition[1])
		}
	}
	for _, rule := range []string{
		domain.RuleAttemptWhileActive, domain.RuleEventAfterTerminal, domain.RuleProcessorRequiresApprove,
		domain.RuleSuccessRequiresRequest, domain.RuleExpiryWhileUnresolved, domain.RuleVersionGap,
		domain.RuleFirstEventNotCreated, domain.RuleAttemptMismatch, domain.RuleNoActiveAttempt,
		domain.RuleUnknownRequiresRequest, domain.RulePaymentExists, domain.RuleDuplicateAttempt,
		domain.RuleRiskDeclineRequiresRisk, domain.RuleProcessorDeclineNoReq, domain.RuleRiskRequiresStart,
		domain.RuleInvalidCommand,
	} {
		if !rules[rule] {
			t.Errorf("no invalid golden history records rule %q", rule)
		}
	}
}

func TestStaleExpectedVersionIsReportedAsStale(t *testing.T) {
	histories := loadHistories(t)
	events := parseEvents(t, histories["valid_processor_success.json"])
	state, _, err := domain.Transition(domain.State{}, commandFor(t, events[0]))
	if err != nil {
		t.Fatal(err)
	}
	start := commandFor(t, events[1]).(domain.StartAttempt)
	start.ExpectedVersion = 0
	if _, _, err := domain.Transition(state, start); !isStale(err) {
		t.Fatalf("err = %v, want stale version", err)
	}
}

func isStale(err error) bool {
	return errors.Is(err, domain.ErrStaleVersion)
}

func withoutEnvelope(event *paymentsv1.PaymentEvent) *paymentsv1.PaymentEvent {
	stripped := proto.Clone(event).(*paymentsv1.PaymentEvent)
	stripped.EventId = ""
	stripped.RecordedAt = nil
	return stripped
}
