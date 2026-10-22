package testvectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type Fingerprint struct {
	Kind       string `json:"kind"`
	Input      string `json:"input"`
	Normalized string `json:"normalized"`
	Expected   string `json:"expected"`
}

type TestCard struct {
	Number   string `json:"number"`
	CardRef  string `json:"card_ref"`
	Behavior string `json:"behavior"`
	Expected string `json:"expected"`
}

type Vectors struct {
	IntegrationID string        `json:"integration_id"`
	SyntheticKey  string        `json:"synthetic_key"`
	Fingerprints  []Fingerprint `json:"fingerprints"`
	TestCards     []TestCard    `json:"test_cards"`
}

func Load(t testing.TB) Vectors {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "contracts", "synthetic", "identity-vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read identity vectors: %v", err)
	}
	var vectors Vectors
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("parse identity vectors: %v", err)
	}
	return vectors
}

func (v Vectors) Fingerprint(t testing.TB, kind string) Fingerprint {
	t.Helper()
	for _, item := range v.Fingerprints {
		if item.Kind == kind {
			return item
		}
	}
	t.Fatalf("identity vectors have no %s case", kind)
	return Fingerprint{}
}
