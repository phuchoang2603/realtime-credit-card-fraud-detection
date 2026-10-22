package domain

import (
	"bytes"
	"testing"
	"time"
)

func TestDecideMapsBehaviorToOutcome(t *testing.T) {
	tests := []struct {
		behavior Behavior
		outcome  Outcome
		reason   string
		ok       bool
	}{
		{BehaviorApprove, OutcomeApproved, "", true},
		{BehaviorInsufficientFunds, OutcomeDeclined, "INSUFFICIENT_FUNDS", true},
		{BehaviorIssuerDecline, OutcomeDeclined, "ISSUER_DECLINED", true},
		{BehaviorInvalid, OutcomeDeclined, "INVALID_CARD", true},
		{Behavior(""), "", "", false},
	}
	for _, test := range tests {
		outcome, reason, ok := Decide(test.behavior)
		if outcome != test.outcome || reason != test.reason || ok != test.ok {
			t.Fatalf("%q: got %s %s %v", test.behavior, outcome, reason, ok)
		}
	}
}

func TestRequestHashDistinguishesEveryField(t *testing.T) {
	base := Sale{OperationKey: "k", IntegrationID: "i", Token: "vt_a", AmountMinorUnit: 100, Currency: "USD"}
	variants := []Sale{
		{OperationKey: "k", IntegrationID: "j", Token: "vt_a", AmountMinorUnit: 100, Currency: "USD"},
		{OperationKey: "k", IntegrationID: "i", Token: "vt_b", AmountMinorUnit: 100, Currency: "USD"},
		{OperationKey: "k", IntegrationID: "i", Token: "vt_a", AmountMinorUnit: 101, Currency: "USD"},
		{OperationKey: "k", IntegrationID: "i", Token: "vt_a", AmountMinorUnit: 100, Currency: "EUR"},
		{OperationKey: "k", IntegrationID: "ivt_", Token: "a", AmountMinorUnit: 100, Currency: "USD"},
	}
	for _, variant := range variants {
		if string(variant.Hash()) == string(base.Hash()) {
			t.Fatalf("hash collision for %+v", variant)
		}
	}
	repeat := base
	if !bytes.Equal(base.Hash(), repeat.Hash()) {
		t.Fatal("hash not deterministic")
	}
}

func TestFaultPlanIsSeededAndBounded(t *testing.T) {
	faults := Faults{Seed: 42, LatencyMin: 10 * time.Millisecond, LatencyMax: 50 * time.Millisecond, TimeoutRate: 0.25}
	timeouts := 0
	const samples = 4000
	for i := range samples {
		key := "op-" + string(rune('a'+i%26)) + time.Duration(i).String()
		plan := faults.Plan(key)
		if plan != faults.Plan(key) {
			t.Fatalf("plan for %s is not deterministic", key)
		}
		if plan.Latency < faults.LatencyMin || plan.Latency >= faults.LatencyMax {
			t.Fatalf("latency %s out of range", plan.Latency)
		}
		if plan.Timeout {
			timeouts++
		}
	}
	if rate := float64(timeouts) / samples; rate < 0.2 || rate > 0.3 {
		t.Fatalf("timeout rate %f far from 0.25", rate)
	}
	other := Faults{Seed: 43, LatencyMin: faults.LatencyMin, LatencyMax: faults.LatencyMax, TimeoutRate: 0.25}
	differs := false
	for i := range 50 {
		key := time.Duration(i).String()
		if faults.Plan(key) != other.Plan(key) {
			differs = true
		}
	}
	if !differs {
		t.Fatal("different seeds produce identical plans")
	}
	if (Faults{Seed: 1}).Plan("x").Timeout {
		t.Fatal("zero timeout rate must never time out")
	}
}
