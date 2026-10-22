package domain_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/domain"
)

func verify(secret, header string, body []byte, now time.Time, tolerance time.Duration) error {
	var timestamp, signature string
	for _, part := range strings.Split(header, ",") {
		switch {
		case strings.HasPrefix(part, "t="):
			timestamp = strings.TrimPrefix(part, "t=")
		case strings.HasPrefix(part, "v1="):
			signature = strings.TrimPrefix(part, "v1=")
		}
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("bad timestamp")
	}
	if delta := now.Sub(time.Unix(unix, 0)); delta > tolerance || delta < -tolerance {
		return fmt.Errorf("timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func TestSignatureVector(t *testing.T) {
	body := []byte(`{"event_id":"e1"}`)
	ts := time.Unix(1700000000, 0)
	got := domain.Sign("whsec_test", ts, body)
	want := "t=1700000000,v1=8a9b910184f1d9ed3590e106c991e230a182f3b9356c7c1ed91c013d17ed0775"
	if got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
}

func TestReceiverVerifierToleranceFiveMinutes(t *testing.T) {
	body := []byte(`{"event_id":"e1"}`)
	signedAt := time.Unix(1700000000, 0)
	header := domain.Sign("secret", signedAt, body)
	tests := []struct {
		name    string
		secret  string
		body    []byte
		now     time.Time
		wantErr bool
	}{
		{"fresh", "secret", body, signedAt.Add(time.Second), false},
		{"edge of tolerance", "secret", body, signedAt.Add(5 * time.Minute), false},
		{"too old", "secret", body, signedAt.Add(5*time.Minute + time.Second), true},
		{"from the future", "secret", body, signedAt.Add(-6 * time.Minute), true},
		{"wrong secret", "other", body, signedAt, true},
		{"tampered body", "secret", []byte(`{"event_id":"e2"}`), signedAt, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verify(tt.secret, header, tt.body, tt.now, 5*time.Minute)
			if (err != nil) != tt.wantErr {
				t.Fatalf("verify error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleDelay(t *testing.T) {
	s := domain.Schedule{Initial: time.Minute, Factor: 2, Cap: 10 * time.Minute, MaxAttempts: 5}
	tests := []struct {
		failed int
		jitter float64
		want   time.Duration
	}{
		{1, 1, time.Minute},
		{2, 1, 2 * time.Minute},
		{3, 0.5, 2 * time.Minute},
		{4, 1, 8 * time.Minute},
		{5, 1, 10 * time.Minute},
		{500, 1, 10 * time.Minute},
		{3, 0, 0},
	}
	for _, tt := range tests {
		if got := s.Delay(tt.failed, tt.jitter); got != tt.want {
			t.Errorf("Delay(%d, %v) = %v, want %v", tt.failed, tt.jitter, got, tt.want)
		}
	}
	if s.Exhausted(4) || !s.Exhausted(5) {
		t.Fatal("exhaustion boundary wrong")
	}
}

func TestDefaultScheduleSpansAboutADay(t *testing.T) {
	s := domain.Schedule{Initial: 2 * time.Minute, Factor: 2, Cap: 6 * time.Hour, MaxAttempts: 12}
	var total time.Duration
	for failed := 1; failed < s.MaxAttempts; failed++ {
		total += s.Delay(failed, 1)
	}
	if total < 20*time.Hour || total > 28*time.Hour {
		t.Fatalf("worst-case span = %v, want about 24h", total)
	}
}

func TestWebhookURLValidation(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://shop.example.com/hooks", false},
		{"http://receiver.shop.svc.cluster.local:8080/hook", false},
		{"http://receiver.shop.svc/hook", false},
		{"http://shop.example.com/hook", true},
		{"http://localhost:8080/hook", true},
		{"http://evil.example.com/.svc.cluster.local", true},
		{"ftp://shop.svc/hook", true},
		{"https://user:pw@shop.example.com/", true},
		{"https:///path", true},
		{"", true},
	}
	for _, tt := range tests {
		err := domain.ValidateWebhookURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestParseIntegrations(t *testing.T) {
	valid := `[{"integration_id":"a","name":"x","api_key_sha256":"abc","synthetic":true,
	"webhook":{"url":"https://a.example/h","secret":"s","events":["payment.succeeded"]}},{"integration_id":"b"}]`
	set, err := domain.ParseIntegrations([]byte(valid))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	a, _ := set.Lookup("a")
	b, _ := set.Lookup("b")
	if !a.Subscribes(domain.EventPaymentSucceeded) || a.Subscribes(domain.EventPaymentExpired) || b.Subscribes(domain.EventPaymentSucceeded) {
		t.Fatal("subscription decisions wrong")
	}
	for name, doc := range map[string]string{
		"bad url":      `[{"integration_id":"a","webhook":{"url":"http://x.example/h","secret":"s","events":[]}}]`,
		"no secret":    `[{"integration_id":"a","webhook":{"url":"https://x.example/h","events":[]}}]`,
		"unknown type": `[{"integration_id":"a","webhook":{"url":"https://x.example/h","secret":"s","events":["x"]}}]`,
		"duplicate":    `[{"integration_id":"a"},{"integration_id":"a"}]`,
		"no id":        `[{"name":"x"}]`,
	} {
		if _, err := domain.ParseIntegrations([]byte(doc)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
