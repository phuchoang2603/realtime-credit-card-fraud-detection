package secretbox_test

import (
	"bytes"
	"testing"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/secretbox"
)

func TestSealedValuesRoundTripAndAreBoundToTheirContext(t *testing.T) {
	box, err := secretbox.New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal([]byte("vault-token"), []byte("attempt-1"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("vault-token")) {
		t.Fatal("ciphertext contains the plaintext")
	}
	again, _ := box.Seal([]byte("vault-token"), []byte("attempt-1"))
	if bytes.Equal(sealed, again) {
		t.Fatal("sealing is deterministic; nonces must be random")
	}
	plaintext, err := box.Open(sealed, []byte("attempt-1"))
	if err != nil || string(plaintext) != "vault-token" {
		t.Fatalf("open = %q, %v", plaintext, err)
	}
	if _, err := box.Open(sealed, []byte("attempt-2")); err == nil {
		t.Fatal("sealed value opened for another attempt")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := box.Open(sealed, []byte("attempt-1")); err == nil {
		t.Fatal("tampered ciphertext opened")
	}
	if _, err := box.Open([]byte("x"), nil); err == nil {
		t.Fatal("short input opened")
	}
	if _, err := secretbox.New([]byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
}
