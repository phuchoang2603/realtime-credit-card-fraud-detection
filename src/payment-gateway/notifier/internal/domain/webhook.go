package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

const (
	StatusDeclined  = "DECLINED"
	StatusSucceeded = "SUCCEEDED"
	StatusExpired   = "EXPIRED"
)

type Outcome struct {
	Type      string
	Status    string
	AttemptID string
	Reason    string
}

type webhookBody struct {
	EventID        string `json:"event_id"`
	Type           string `json:"type"`
	PaymentID      string `json:"payment_id"`
	OrderReference string `json:"order_reference"`
	AttemptID      string `json:"attempt_id,omitempty"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	OccurredAt     string `json:"occurred_at"`
}

type Event struct {
	ID         string
	PaymentID  string
	OccurredAt time.Time
}

func BuildBody(event Event, outcome Outcome, orderReference string) ([]byte, error) {
	return json.Marshal(webhookBody{
		EventID:        event.ID,
		Type:           outcome.Type,
		PaymentID:      event.PaymentID,
		OrderReference: orderReference,
		AttemptID:      outcome.AttemptID,
		Status:         outcome.Status,
		Reason:         outcome.Reason,
		OccurredAt:     event.OccurredAt.UTC().Format(time.RFC3339Nano),
	})
}

func Sign(secret string, timestamp time.Time, body []byte) string {
	unix := strconv.FormatInt(timestamp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unix))
	mac.Write([]byte("."))
	mac.Write(body)
	return "t=" + unix + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
