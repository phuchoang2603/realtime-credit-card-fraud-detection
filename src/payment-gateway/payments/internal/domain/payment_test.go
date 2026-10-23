package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestTransitionGuards(t *testing.T) {
	envelope := Envelope{EventID: "event", ExpectedVersion: 3, Now: time.Now()}
	open := State{Status: StatusOpen, Version: 3}
	active := func(phase Phase) State {
		state := open
		state.Attempts = []Attempt{{ID: "attempt", Phase: phase}}
		return state
	}
	tests := []struct {
		name    string
		state   State
		command Command
		rule    string
	}{
		{"first event", State{}, StartAttempt{Envelope: envelope}, RuleFirstEventNotCreated},
		{"created twice", open, CreatePayment{Envelope: envelope}, RulePaymentExists},
		{"version gap", open, StartAttempt{Envelope: Envelope{EventID: "event", ExpectedVersion: 4}, AttemptID: "attempt"}, RuleVersionGap},
		{"version repeat", open, StartAttempt{Envelope: Envelope{EventID: "event", ExpectedVersion: 2}, AttemptID: "attempt"}, RuleVersionGap},
		{"attempt while active", active(PhaseStarted), StartAttempt{Envelope: envelope, AttemptID: "other"}, RuleAttemptWhileActive},
		{"attempt while unknown", active(PhaseUnknown), StartAttempt{Envelope: envelope, AttemptID: "other"}, RuleAttemptWhileActive},
		{"duplicate attempt", active(PhaseDeclined), StartAttempt{Envelope: envelope, AttemptID: "attempt"}, RuleDuplicateAttempt},
		{"no active attempt", open, DeclineAttempt{Envelope: envelope, AttemptID: "attempt"}, RuleNoActiveAttempt},
		{"declined attempt inactive", active(PhaseDeclined), DeclineAttempt{Envelope: envelope, AttemptID: "attempt"}, RuleNoActiveAttempt},
		{"attempt mismatch", active(PhaseStarted), RecordRiskDecision{Envelope: envelope, AttemptID: "other"}, RuleAttemptMismatch},
		{"decision twice", active(PhaseRiskApproved), RecordRiskDecision{Envelope: envelope, AttemptID: "attempt"}, RuleRiskRequiresStart},
		{"invalid decision", active(PhaseStarted), RecordRiskDecision{Envelope: envelope, AttemptID: "attempt", Assessment: &RiskAssessment{Outcome: RiskOutcomeApprove}}, RuleInvalidCommand},
		{"empty reason code", active(PhaseStarted), RecordRiskDecision{Envelope: envelope, AttemptID: "attempt", Assessment: &RiskAssessment{Outcome: RiskOutcomeApprove, PolicyVersion: "policy", ReasonCodes: []string{""}}}, RuleInvalidCommand},
		{"processor before decision", active(PhaseStarted), RequestProcessorOperation{Envelope: envelope, AttemptID: "attempt"}, RuleProcessorRequiresApprove},
		{"processor after risk decline", active(PhaseRiskDeclined), RequestProcessorOperation{Envelope: envelope, AttemptID: "attempt"}, RuleProcessorRequiresApprove},
		{"success before request", active(PhaseRiskApproved), SucceedPayment{Envelope: envelope, AttemptID: "attempt"}, RuleSuccessRequiresRequest},
		{"success before decision", active(PhaseStarted), SucceedPayment{Envelope: envelope, AttemptID: "attempt"}, RuleSuccessRequiresRequest},
		{"unknown before request", active(PhaseStarted), RecordOutcomeUnknown{Envelope: envelope, AttemptID: "attempt"}, RuleUnknownRequiresRequest},
		{"risk decline after approve", active(PhaseRiskApproved), DeclineAttempt{Envelope: envelope, AttemptID: "attempt", Source: DeclineSourceRisk, Reason: "declined"}, RuleRiskDeclineRequiresRisk},
		{"processor decline before request", active(PhaseRiskApproved), DeclineAttempt{Envelope: envelope, AttemptID: "attempt", Source: DeclineSourceProcessor, Reason: "declined"}, RuleProcessorDeclineNoReq},
		{"expiry while requested", active(PhaseRequested), ExpirePayment{Envelope: envelope}, RuleExpiryWhileUnresolved},
		{"expiry while unknown", active(PhaseUnknown), ExpirePayment{Envelope: envelope}, RuleExpiryWhileUnresolved},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, event, err := Transition(test.state, test.command)
			if RuleOf(err) != test.rule || event != nil {
				t.Fatalf("event = %v, err = %v, want %s", event, err, test.rule)
			}
			if test.rule == RuleVersionGap && !errors.Is(err, ErrStaleVersion) {
				t.Fatalf("missing stale version error: %v", err)
			}
		})
	}
}

func TestExpiryAndTerminalFencing(t *testing.T) {
	envelope := Envelope{EventID: "event", ExpectedVersion: 3, Now: time.Now()}
	for _, phase := range []Phase{"", PhaseStarted, PhaseRiskApproved, PhaseRiskDeclined, PhaseDeclined} {
		t.Run(string(phase), func(t *testing.T) {
			state := State{Status: StatusOpen, Version: 3}
			if phase != "" {
				state.Attempts = []Attempt{{ID: "attempt", Phase: phase}}
			}
			expired, event, err := Transition(state, ExpirePayment{Envelope: envelope})
			if err != nil || expired.Status != StatusExpired || event.Version != 4 {
				t.Fatalf("expiry = %+v, %v, %v", expired, event, err)
			}
		})
	}
	for _, status := range []Status{StatusSucceeded, StatusExpired} {
		for _, command := range []Command{
			StartAttempt{Envelope: envelope, AttemptID: "new"},
			SucceedPayment{Envelope: envelope, AttemptID: "attempt", ProcessorReference: "ref"},
			ExpirePayment{Envelope: envelope},
		} {
			if _, _, err := Transition(State{Status: status, Version: 3}, command); RuleOf(err) != RuleEventAfterTerminal {
				t.Fatalf("%s %T: %v", status, command, err)
			}
		}
	}
}

func TestDeclineThenRetryAndUnknownResolution(t *testing.T) {
	for _, source := range []DeclineSource{DeclineSourceRisk, DeclineSourceProcessor} {
		for _, unknown := range []bool{false, true} {
			t.Run(fmt.Sprintf("source=%d/unknown=%t", source, unknown), func(t *testing.T) {
				state := State{}
				apply := func(command Command) {
					t.Helper()
					next, event, err := Transition(state, command)
					if err != nil {
						t.Fatal(err)
					}
					if next.Version != state.Version+1 || event.Version != next.Version {
						t.Fatal("state and event versions diverged")
					}
					state = next
				}
				envelope := func() Envelope { return Envelope{EventID: "event", ExpectedVersion: state.Version, Now: time.Now()} }
				apply(CreatePayment{Envelope: envelope(), PaymentID: "payment", IntegrationID: "integration", CorrelationID: "correlation", SessionID: "session", Merchant: &MerchantRef{}, Customer: &CustomerRef{}, Snapshot: &CheckoutSnapshot{}})
				apply(StartAttempt{Envelope: envelope(), AttemptID: "first"})
				outcome := RiskOutcomeApprove
				if source == DeclineSourceRisk {
					outcome = RiskOutcomeDecline
				}
				apply(RecordRiskDecision{Envelope: envelope(), AttemptID: "first", Assessment: &RiskAssessment{Outcome: outcome, PolicyVersion: "policy", ReasonCodes: []string{"DECLINED"}}})
				if source == DeclineSourceProcessor {
					apply(RequestProcessorOperation{Envelope: envelope(), AttemptID: "first", OperationKey: "op-first"})
					if unknown {
						apply(RecordOutcomeUnknown{Envelope: envelope(), AttemptID: "first"})
					}
				}
				apply(DeclineAttempt{Envelope: envelope(), AttemptID: "first", Source: source, Reason: "declined"})
				apply(StartAttempt{Envelope: envelope(), AttemptID: "retry"})
				apply(RecordRiskDecision{Envelope: envelope(), AttemptID: "retry", Assessment: &RiskAssessment{Outcome: RiskOutcomeApprove, PolicyVersion: "policy"}})
				apply(RequestProcessorOperation{Envelope: envelope(), AttemptID: "retry", OperationKey: "op-retry"})
				if unknown {
					apply(RecordOutcomeUnknown{Envelope: envelope(), AttemptID: "retry"})
				}
				apply(SucceedPayment{Envelope: envelope(), AttemptID: "retry", ProcessorReference: "ref"})
				if state.Status != StatusSucceeded || len(state.Attempts) != 2 || state.Attempts[0].Phase != PhaseDeclined || state.Attempts[1].Phase != PhaseSucceeded {
					t.Fatalf("final state = %+v", state)
				}
			})
		}
	}
}
