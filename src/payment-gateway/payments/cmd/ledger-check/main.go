package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/proto/payments/v1"
	"google.golang.org/protobuf/proto"
)

type row struct {
	GlobalPosition   int64  `json:"global_position"`
	EventID          string `json:"event_id"`
	PaymentID        string `json:"payment_id"`
	AggregateVersion uint64 `json:"aggregate_version"`
	EventType        string `json:"event_type"`
	Payload          string `json:"payload_b64"`
}

type lifecycle struct {
	attempts map[string]string
	active   string
	version  uint64
	terminal bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: ledger-check <ledger.json>")
		return 2
	}
	raw, err := os.ReadFile(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	var rows []row
	if err := json.Unmarshal(raw, &rows); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PaymentID != rows[j].PaymentID {
			return rows[i].PaymentID < rows[j].PaymentID
		}
		return rows[i].AggregateVersion < rows[j].AggregateVersion
	})
	states := map[string]*lifecycle{}
	failures := 0
	for _, r := range rows {
		event, err := decode(r)
		if err != nil {
			failures++
			report(stderr, r, err)
			continue
		}
		state := states[r.PaymentID]
		if state == nil {
			state = &lifecycle{attempts: map[string]string{}}
			states[r.PaymentID] = state
		}
		if err := apply(state, r, event); err != nil {
			failures++
			report(stderr, r, err)
		}
	}
	if failures != 0 {
		_, _ = fmt.Fprintf(stderr, "ledger-check: %d violation(s) across %d payments and %d events\n", failures, len(states), len(rows))
		return 1
	}
	if _, err := fmt.Fprintf(stdout, "ledger-check: %d events across %d payments satisfy the lifecycle\n", len(rows), len(states)); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func decode(r row) (*paymentsv1.PaymentEvent, error) {
	payload, err := base64.StdEncoding.DecodeString(r.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	event := &paymentsv1.PaymentEvent{}
	if err := proto.Unmarshal(payload, event); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	if event.GetEventId() != r.EventID || event.GetPaymentId() != r.PaymentID || event.GetAggregateVersion() != r.AggregateVersion {
		return nil, fmt.Errorf("payload envelope does not match ledger row")
	}
	if kind(event) != r.EventType {
		return nil, fmt.Errorf("payload kind %s does not match event_type %s", kind(event), r.EventType)
	}
	return event, nil
}

func kind(event *paymentsv1.PaymentEvent) string {
	switch {
	case event.GetPaymentCreated() != nil:
		return "payment_created"
	case event.GetAttemptStarted() != nil:
		return "attempt_started"
	case event.GetRiskDecisionRecorded() != nil:
		return "risk_decision_recorded"
	case event.GetProcessorOperationRequested() != nil:
		return "processor_operation_requested"
	case event.GetProcessorOutcomeUnknown() != nil:
		return "processor_outcome_unknown"
	case event.GetAttemptDeclined() != nil:
		return "attempt_declined"
	case event.GetPaymentSucceeded() != nil:
		return "payment_succeeded"
	case event.GetPaymentExpired() != nil:
		return "payment_expired"
	default:
		return "unknown"
	}
}

func apply(state *lifecycle, r row, event *paymentsv1.PaymentEvent) error {
	if state.terminal {
		return fmt.Errorf("event after terminal outcome")
	}
	if r.AggregateVersion != state.version+1 {
		return fmt.Errorf("aggregate version %d follows %d", r.AggregateVersion, state.version)
	}
	if state.version == 0 && event.GetPaymentCreated() == nil {
		return fmt.Errorf("first event must be payment_created")
	}
	phase := ""
	if state.active != "" {
		phase = state.attempts[state.active]
	}
	switch kind(event) {
	case "payment_created":
		if r.AggregateVersion != 1 {
			return fmt.Errorf("payment_created must be version 1, got %d", r.AggregateVersion)
		}
		if event.GetPaymentCreated().GetSnapshot() == nil {
			return fmt.Errorf("payment_created without snapshot")
		}
	case "attempt_started":
		if state.active != "" {
			return fmt.Errorf("attempt started while %s is active", state.active)
		}
		id := event.GetAttemptStarted().GetAttemptId()
		if id == "" {
			return fmt.Errorf("attempt_started without attempt id")
		}
		if _, exists := state.attempts[id]; exists {
			return fmt.Errorf("duplicate attempt %s", id)
		}
		state.attempts[id] = "started"
		state.active = id
	case "risk_decision_recorded":
		id := event.GetRiskDecisionRecorded().GetAttemptId()
		if err := expectActive(state, id, "started"); err != nil {
			return err
		}
		assessment := event.GetRiskDecisionRecorded().GetAssessment()
		if assessment == nil || assessment.GetPolicyVersion() == "" {
			return fmt.Errorf("risk decision requires an assessment and policy version")
		}
		for _, reason := range assessment.GetReasonCodes() {
			if reason == "" {
				return fmt.Errorf("risk decision has an empty reason code")
			}
		}
		switch assessment.GetOutcome() {
		case paymentsv1.RiskOutcome_RISK_OUTCOME_APPROVE:
			state.attempts[id] = "risk_approved"
		case paymentsv1.RiskOutcome_RISK_OUTCOME_DECLINE:
			state.attempts[id] = "risk_declined"
		default:
			return fmt.Errorf("risk outcome must be approve or decline")
		}
	case "processor_operation_requested":
		id := event.GetProcessorOperationRequested().GetAttemptId()
		if err := expectActive(state, id, "risk_approved"); err != nil {
			return err
		}
		state.attempts[id] = "requested"
	case "processor_outcome_unknown":
		id := event.GetProcessorOutcomeUnknown().GetAttemptId()
		if err := expectActive(state, id, "requested"); err != nil {
			return err
		}
		state.attempts[id] = "unknown"
	case "attempt_declined":
		id := event.GetAttemptDeclined().GetAttemptId()
		if state.active != id {
			return fmt.Errorf("decline for inactive attempt %s", id)
		}
		if phase != "risk_declined" && phase != "requested" && phase != "unknown" {
			return fmt.Errorf("decline from phase %s", phase)
		}
		state.attempts[id] = "declined"
		state.active = ""
	case "payment_succeeded":
		id := event.GetPaymentSucceeded().GetAttemptId()
		if state.active != id || (phase != "requested" && phase != "unknown") {
			return fmt.Errorf("success from phase %s", phase)
		}
		state.attempts[id] = "succeeded"
		state.active = ""
		state.terminal = true
	case "payment_expired":
		if phase == "requested" || phase == "unknown" {
			return fmt.Errorf("expiry while attempt %s is %s", state.active, phase)
		}
		state.terminal = true
	default:
		return fmt.Errorf("unknown event kind")
	}
	state.version = r.AggregateVersion
	return nil
}

func expectActive(state *lifecycle, id, want string) error {
	if id == "" || state.active != id {
		return fmt.Errorf("event for inactive attempt %q", id)
	}
	if got := state.attempts[id]; got != want {
		return fmt.Errorf("attempt %s is %s, want %s", id, got, want)
	}
	return nil
}

func report(stderr io.Writer, r row, err error) {
	_, _ = fmt.Fprintf(stderr, "%s version %d (%s): %v\n", r.PaymentID, r.AggregateVersion, r.EventType, err)
}
