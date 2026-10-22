package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

const (
	EventAttemptDeclined  = "attempt.declined"
	EventPaymentSucceeded = "payment.succeeded"
	EventPaymentExpired   = "payment.expired"
)

var knownEventTypes = []string{EventAttemptDeclined, EventPaymentSucceeded, EventPaymentExpired}

type Webhook struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

type Integration struct {
	ID      string   `json:"integration_id"`
	Webhook *Webhook `json:"webhook"`
}

func (i Integration) Subscribes(eventType string) bool {
	return i.Webhook != nil && i.Webhook.URL != "" && slices.Contains(i.Webhook.Events, eventType)
}

type Integrations struct {
	byID map[string]Integration
}

func (s Integrations) Lookup(id string) (Integration, bool) {
	integration, ok := s.byID[id]
	return integration, ok
}

func (s Integrations) Len() int { return len(s.byID) }

func ParseIntegrations(data []byte) (Integrations, error) {
	var list []Integration
	if err := json.Unmarshal(data, &list); err != nil {
		return Integrations{}, fmt.Errorf("parse integrations: %w", err)
	}
	byID := make(map[string]Integration, len(list))
	for _, integration := range list {
		if integration.ID == "" {
			return Integrations{}, errors.New("integration_id must not be empty")
		}
		if _, dup := byID[integration.ID]; dup {
			return Integrations{}, fmt.Errorf("duplicate integration_id %q", integration.ID)
		}
		if integration.Webhook != nil {
			if err := integration.Webhook.validate(); err != nil {
				return Integrations{}, fmt.Errorf("integration %q: %w", integration.ID, err)
			}
		}
		byID[integration.ID] = integration
	}
	return Integrations{byID: byID}, nil
}

func (w Webhook) validate() error {
	if err := ValidateWebhookURL(w.URL); err != nil {
		return err
	}
	if w.Secret == "" {
		return errors.New("webhook secret must not be empty")
	}
	for _, eventType := range w.Events {
		if !slices.Contains(knownEventTypes, eventType) {
			return fmt.Errorf("unknown webhook event type %q", eventType)
		}
	}
	return nil
}

func ValidateWebhookURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("webhook url is not a valid URL")
	}
	if parsed.Hostname() == "" {
		return errors.New("webhook url must include a host")
	}
	if parsed.User != nil {
		return errors.New("webhook url must not contain credentials")
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		host := strings.ToLower(parsed.Hostname())
		if strings.HasSuffix(host, ".svc.cluster.local") || strings.HasSuffix(host, ".svc") {
			return nil
		}
		return errors.New("webhook url must be https unless it is a cluster-local service address")
	default:
		return errors.New("webhook url must use https")
	}
}
