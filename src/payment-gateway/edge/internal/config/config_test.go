package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/edge/internal/config"
)

func baseEnv(t *testing.T) map[string]string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	return map[string]string{
		"EDGE_INTEGRATIONS_FILE":         write("integrations.json", "[]"),
		"EDGE_PAYMENTS_ADDR":             "payments:8000",
		"EDGE_VAULT_ADDRESS":             "vault:8000",
		"EDGE_PUBLIC_BASE_URL":           "https://pay.example.com/",
		"EDGE_FINGERPRINT_KEY_FILE":      write("fingerprint", "real-key\n"),
		"EDGE_CHECKOUT_COOKIE_KEY":       "cookie-key",
		"EDGE_SYNTHETIC_FINGERPRINT_KEY": "",
	}
}

func lookup(env map[string]string) func(string) string {
	return func(name string) string { return env[name] }
}

func TestDefaultsAndSecretFiles(t *testing.T) {
	cfg, err := config.FromLookup(lookup(baseEnv(t)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Address != ":8080" || cfg.TrustedProxyHeader != "X-Forwarded-For" || cfg.TrustedProxyHops != 1 || !cfg.CookieSecure {
		t.Fatalf("defaults = %+v", cfg)
	}
	if cfg.SyntheticFingerprintKey != "synthetic-fingerprint-v1" {
		t.Fatalf("synthetic key = %q", cfg.SyntheticFingerprintKey)
	}
	if cfg.FingerprintKey != "real-key" || cfg.CheckoutCookieKey != "cookie-key" {
		t.Fatalf("secrets were not loaded and trimmed: %+v", cfg)
	}
	if cfg.PaymentsTimeout != 6*time.Second || cfg.VaultTimeout != 2*time.Second {
		t.Fatalf("timeouts = %s, %s", cfg.PaymentsTimeout, cfg.VaultTimeout)
	}
	if cfg.PublicOrigin() != "https://pay.example.com" {
		t.Fatalf("origin = %q", cfg.PublicOrigin())
	}
}

func TestVaultAddressPrefersEdgeVariable(t *testing.T) {
	env := baseEnv(t)
	env["VAULT_ADDRESS"] = "shared:1"
	delete(env, "EDGE_VAULT_ADDRESS")
	cfg, err := config.FromLookup(lookup(env))
	if err != nil || cfg.VaultAddr != "shared:1" {
		t.Fatalf("addr = %q, %v", cfg.VaultAddr, err)
	}
	env["EDGE_VAULT_ADDRESS"] = "edge:2"
	cfg, _ = config.FromLookup(lookup(env))
	if cfg.VaultAddr != "edge:2" {
		t.Fatalf("addr = %q", cfg.VaultAddr)
	}
}

func TestInvalidConfigurationIsRejected(t *testing.T) {
	tests := map[string]func(map[string]string){
		"missing integrations":  func(e map[string]string) { delete(e, "EDGE_INTEGRATIONS_FILE") },
		"missing payments":      func(e map[string]string) { delete(e, "EDGE_PAYMENTS_ADDR") },
		"missing fingerprint":   func(e map[string]string) { delete(e, "EDGE_FINGERPRINT_KEY_FILE") },
		"missing cookie key":    func(e map[string]string) { delete(e, "EDGE_CHECKOUT_COOKIE_KEY") },
		"unreadable secret":     func(e map[string]string) { e["EDGE_FINGERPRINT_KEY_FILE"] = "/nonexistent/key" },
		"base url with path":    func(e map[string]string) { e["EDGE_PUBLIC_BASE_URL"] = "https://pay.example.com/x" },
		"base url wrong scheme": func(e map[string]string) { e["EDGE_PUBLIC_BASE_URL"] = "ftp://pay.example.com" },
		"bad address":           func(e map[string]string) { e["EDGE_HTTP_ADDR"] = "8080" },
		"bad timeout":           func(e map[string]string) { e["EDGE_VAULT_TIMEOUT"] = "soon" },
		"zero timeout":          func(e map[string]string) { e["EDGE_PAYMENTS_TIMEOUT"] = "0s" },
		"bad hops":              func(e map[string]string) { e["EDGE_TRUSTED_PROXY_HOPS"] = "0" },
		"bad cookie flag":       func(e map[string]string) { e["EDGE_COOKIE_SECURE"] = "maybe" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			env := baseEnv(t)
			mutate(env)
			if _, err := config.FromLookup(lookup(env)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestCookieSecureCanBeDisabledForLocalRuns(t *testing.T) {
	env := baseEnv(t)
	env["EDGE_COOKIE_SECURE"] = "false"
	cfg, err := config.FromLookup(lookup(env))
	if err != nil || cfg.CookieSecure {
		t.Fatalf("secure = %v, %v", cfg.CookieSecure, err)
	}
}
