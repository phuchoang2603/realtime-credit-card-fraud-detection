package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"hash/fnv"
	"math/rand/v2"
	"time"
)

type Behavior string

const (
	BehaviorApprove           Behavior = "approve"
	BehaviorInsufficientFunds Behavior = "insufficient_funds"
	BehaviorIssuerDecline     Behavior = "issuer_decline"
	BehaviorInvalid           Behavior = "invalid"
)

type Outcome string

const (
	OutcomeApproved Outcome = "APPROVED"
	OutcomeDeclined Outcome = "DECLINED"
)

const (
	ReasonInsufficientFunds = "INSUFFICIENT_FUNDS"
	ReasonIssuerDeclined    = "ISSUER_DECLINED"
	ReasonInvalidCard       = "INVALID_CARD"
)

type Sale struct {
	OperationKey    string
	IntegrationID   string
	Token           string
	AmountMinorUnit int64
	Currency        string
}

type Operation struct {
	Key                string
	Outcome            Outcome
	DeclineReason      string
	ProcessorReference string
	RecordedAt         time.Time
}

func (s Sale) Hash() []byte {
	digest := sha256.New()
	for _, field := range []string{s.IntegrationID, s.Token, s.Currency} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		digest.Write(length[:])
		digest.Write([]byte(field))
	}
	var amount [8]byte
	binary.BigEndian.PutUint64(amount[:], uint64(s.AmountMinorUnit))
	digest.Write(amount[:])
	return digest.Sum(nil)
}

func Decide(behavior Behavior) (Outcome, string, bool) {
	switch behavior {
	case BehaviorApprove:
		return OutcomeApproved, "", true
	case BehaviorInsufficientFunds:
		return OutcomeDeclined, ReasonInsufficientFunds, true
	case BehaviorIssuerDecline:
		return OutcomeDeclined, ReasonIssuerDeclined, true
	case BehaviorInvalid:
		return OutcomeDeclined, ReasonInvalidCard, true
	default:
		return "", "", false
	}
}

type Faults struct {
	Seed        uint64
	LatencyMin  time.Duration
	LatencyMax  time.Duration
	TimeoutRate float64
}

type FaultPlan struct {
	Latency time.Duration
	Timeout bool
}

func (f Faults) Plan(operationKey string) FaultPlan {
	keyHash := fnv.New64a()
	keyHash.Write([]byte(operationKey))
	source := rand.New(rand.NewPCG(f.Seed, keyHash.Sum64()))
	latency := f.LatencyMin
	if f.LatencyMax > f.LatencyMin {
		latency += time.Duration(source.Int64N(int64(f.LatencyMax - f.LatencyMin)))
	}
	return FaultPlan{Latency: latency, Timeout: source.Float64() < f.TimeoutRate}
}
