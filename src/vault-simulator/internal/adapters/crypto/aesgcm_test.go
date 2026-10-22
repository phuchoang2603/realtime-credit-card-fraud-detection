package crypto

import (
	"bytes"
	"testing"
)

func TestSealerRoundTripAndAuthentication(t *testing.T) {
	sealer, err := NewSealer(bytes.Repeat([]byte{7}, KeySize))
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte(`{"card_reference":"card-0001"}`)
	sealed, err := sealer.Seal(plaintext, []byte("row-a"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("card-0001")) {
		t.Fatal("ciphertext contains plaintext")
	}
	opened, err := sealer.Open(sealed, []byte("row-a"))
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("open: %v %q", err, opened)
	}
	if _, err := sealer.Open(sealed, []byte("row-b")); err == nil {
		t.Fatal("ciphertext opened under different associated data")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := sealer.Open(sealed, []byte("row-a")); err == nil {
		t.Fatal("tampered ciphertext opened")
	}
	if _, err := NewSealer([]byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
}
