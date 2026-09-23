package config

import (
	"testing"
	"time"
)

func TestFromLookupDefaults(t *testing.T) {
	cfg, err := FromLookup(func(string) string { return "" })
	if err != nil {
		t.Fatalf("FromLookup() error = %v", err)
	}
	if cfg.Address != ":8080" || cfg.ShutdownTimeout <= 0 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestFromLookupRejectsInvalidTimeout(t *testing.T) {
	_, err := FromLookup(func(key string) string {
		if key == "EDGE_SHUTDOWN_TIMEOUT" {
			return "soon"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected invalid timeout error")
	}
}

func TestValidateRejectsNonPositiveTimeout(t *testing.T) {
	if err := (Config{Address: ":8080", FraudTarget: "localhost:8000", PredictionTimeout: time.Second}).Validate(); err == nil {
		t.Fatal("expected timeout validation error")
	}
}

func TestValidateRejectsInvalidAddress(t *testing.T) {
	if err := (Config{Address: "not-an-address", ShutdownTimeout: 1}).Validate(); err == nil {
		t.Fatal("expected address validation error")
	}
}

func TestPredictionDeadlineConfiguration(t *testing.T) {
	for _, value := range []string{"0s", "-1s", "soon"} {
		_, err := FromLookup(func(key string) string {
			if key == "PREDICTION_TIMEOUT" {
				return value
			}
			return ""
		})
		if err == nil {
			t.Fatalf("accepted invalid prediction timeout %q", value)
		}
	}
}
