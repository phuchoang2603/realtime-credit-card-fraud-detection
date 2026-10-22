package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func lookupFrom(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func validValues(t *testing.T) map[string]string {
	callers := writeFile(t, `[{"name":"edge","credential_sha256":"`+strings.Repeat("ab", 32)+`","permissions":["tokenize","describe"]}]`)
	return map[string]string{
		"VAULT_DATABASE_URL":    "postgres://x",
		"VAULT_ENCRYPTION_KEY":  "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
		"VAULT_FINGERPRINT_KEY": "synthetic-fingerprint-v1",
		"VAULT_TLS_CERT_FILE":   "/tls/tls.crt",
		"VAULT_TLS_KEY_FILE":    "/tls/tls.key",
		"VAULT_TLS_CA_FILE":     "/tls/ca.crt",
		"VAULT_CALLERS_FILE":    callers,
	}
}

func TestFromLookup(t *testing.T) {
	values := validValues(t)
	cfg, err := FromLookup(lookupFrom(values))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GRPCAddr != ":8000" || cfg.MetricsAddr != ":8010" || cfg.MigrateDatabaseURL != "postgres://x" || cfg.TLSCAFile != "/tls/ca.crt" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.Callers) != 1 || !cfg.Callers[0].Permissions["tokenize"] || cfg.Callers[0].Permissions["resolve"] {
		t.Fatalf("unexpected callers: %+v", cfg.Callers)
	}

	values["VAULT_ENCRYPTION_KEY_FILE"] = writeFile(t, "c2hvcnQ=\n")
	if _, err := FromLookup(lookupFrom(values)); err == nil {
		t.Fatal("short encryption key accepted")
	}
}

func TestFromLookupRejectsInvalid(t *testing.T) {
	tests := map[string]func(map[string]string){
		"missing database":    func(v map[string]string) { delete(v, "VAULT_DATABASE_URL") },
		"missing tls":         func(v map[string]string) { delete(v, "VAULT_TLS_KEY_FILE") },
		"missing fingerprint": func(v map[string]string) { delete(v, "VAULT_FINGERPRINT_KEY") },
		"bad address":         func(v map[string]string) { v["VAULT_GRPC_ADDR"] = "nope" },
		"bad timeout":         func(v map[string]string) { v["VAULT_SHUTDOWN_TIMEOUT"] = "x" },
		"unknown permission": func(v map[string]string) {
			v["VAULT_CALLERS_FILE"] = writeFile(t, `[{"name":"a","credential_sha256":"`+strings.Repeat("ab", 32)+`","permissions":["admin"]}]`)
		},
		"bad digest": func(v map[string]string) {
			v["VAULT_CALLERS_FILE"] = writeFile(t, `[{"name":"a","credential_sha256":"zz","permissions":[]}]`)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			values := validValues(t)
			mutate(values)
			if _, err := FromLookup(lookupFrom(values)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
