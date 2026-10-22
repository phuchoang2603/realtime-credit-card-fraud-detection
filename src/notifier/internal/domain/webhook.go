package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	paymentsv1 "github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/gen/payments/v1"
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

func OutcomeOf(event *paymentsv1.PaymentEvent) (Outcome, bool) {
	switch payload := event.GetPayload().(type) {
	case *paymentsv1.PaymentEvent_AttemptDeclined:
		return Outcome{
			Type:      EventAttemptDeclined,
			Status:    StatusDeclined,
			AttemptID: payload.AttemptDeclined.GetAttemptId(),
			Reason:    payload.AttemptDeclined.GetReason(),
		}, true
	case *paymentsv1.PaymentEvent_PaymentSucceeded:
		return Outcome{
			Type:      EventPaymentSucceeded,
			Status:    StatusSucceeded,
			AttemptID: payload.PaymentSucceeded.GetAttemptId(),
		}, true
	case *paymentsv1.PaymentEvent_PaymentExpired:
		return Outcome{Type: EventPaymentExpired, Status: StatusExpired}, true
	default:
		return Outcome{}, false
	}
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

func BuildBody(event *paymentsv1.PaymentEvent, outcome Outcome, orderReference string) ([]byte, error) {
	return json.Marshal(webhookBody{
		EventID:        event.GetEventId(),
		Type:           outcome.Type,
		PaymentID:      event.GetPaymentId(),
		OrderReference: orderReference,
		AttemptID:      outcome.AttemptID,
		Status:         outcome.Status,
		Reason:         outcome.Reason,
		OccurredAt:     event.GetOccurredAt().AsTime().UTC().Format(time.RFC3339Nano),
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
