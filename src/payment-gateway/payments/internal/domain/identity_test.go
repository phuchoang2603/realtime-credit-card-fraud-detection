package domain_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/domain"
)

func TestMerchantAndCustomerIDsMatchTheIdentityVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/synthetic/identity-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		IntegrationID string `json:"integration_id"`
		IDs           []struct {
			Kind       string `json:"kind"`
			ExternalID string `json:"external_id"`
			Expected   string `json:"expected"`
		} `json:"ids"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors.IDs) == 0 {
		t.Fatal("no id vectors")
	}
	for _, vector := range vectors.IDs {
		var got uuid.UUID
		switch vector.Kind {
		case "merchant":
			got = domain.MerchantID(vectors.IntegrationID, vector.ExternalID)
		case "customer":
			got = domain.CustomerID(vectors.IntegrationID, vector.ExternalID)
		default:
			t.Fatalf("unknown kind %q", vector.Kind)
		}
		if got.String() != vector.Expected {
			t.Errorf("%s %s = %s, want %s", vector.Kind, vector.ExternalID, got, vector.Expected)
		}
	}
}

func TestCheckoutTokenRoundTripAndTamperDetection(t *testing.T) {
	tokens := domain.NewCheckoutTokens([]byte("checkout-token-key-for-tests-0123456789"))
	session := uuid.New()
	expiry := time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)

	token := tokens.Issue(session, expiry)
	if again := tokens.Issue(session, expiry); again != token {
		t.Fatal("token is not recomputable")
	}
	gotSession, gotExpiry, err := tokens.Verify(token)
	if err != nil || gotSession != session || !gotExpiry.Equal(expiry) {
		t.Fatalf("verify = %v, %v, %v", gotSession, gotExpiry, err)
	}

	raw, _ := base64.RawURLEncoding.DecodeString(token)
	if len(raw) != 16+8+32 {
		t.Fatalf("token layout has %d bytes", len(raw))
	}
	raw[20] ^= 0xff
	tampered := base64.RawURLEncoding.EncodeToString(raw)
	for name, candidate := range map[string]string{"tampered expiry": tampered, "garbage": "!!!", "empty": "", "truncated": token[:10]} {
		if _, _, err := tokens.Verify(candidate); err == nil {
			t.Errorf("%s token accepted", name)
		}
	}
	other := domain.NewCheckoutTokens([]byte("a-different-key-for-another-service!!"))
	if _, _, err := other.Verify(token); err == nil {
		t.Error("token verified under another key")
	}
}
