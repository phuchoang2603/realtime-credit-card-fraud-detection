package config

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultGRPCAddr        = ":8000"
	defaultMetricsAddr     = ":8010"
	defaultShutdownTimeout = 10 * time.Second

	PermissionTokenize = "tokenize"
	PermissionDescribe = "describe"
	PermissionResolve  = "resolve"
)

// Environment variables (each secret also accepts <NAME>_FILE with a mounted file path):
//
//	VAULT_GRPC_ADDR, VAULT_METRICS_ADDR, VAULT_SHUTDOWN_TIMEOUT
//	VAULT_DATABASE_URL, VAULT_MIGRATE_DATABASE_URL (owner role, falls back to VAULT_DATABASE_URL)
//	VAULT_TLS_CERT_FILE, VAULT_TLS_KEY_FILE, VAULT_TLS_CA_FILE (cert-manager secret keys tls.crt, tls.key, ca.crt)
//	VAULT_CALLERS_FILE: JSON array of {"name","credential_sha256","permissions":["tokenize"|"describe"|"resolve"]}
//	VAULT_ENCRYPTION_KEY: base64 of 32 bytes (AES-256-GCM)
//	VAULT_FINGERPRINT_KEY: synthetic HMAC key shared with contracts/synthetic/identity-vectors.json
//
// The healthcheck subcommand reads only VAULT_GRPC_ADDR (port), VAULT_HEALTHCHECK_CA_FILE
// (default VAULT_TLS_CA_FILE, else VAULT_TLS_CERT_FILE), VAULT_HEALTHCHECK_SERVER_NAME
// (default first DNS SAN of VAULT_TLS_CERT_FILE) and VAULT_HEALTHCHECK_TIMEOUT (default 2s).
type Config struct {
	GRPCAddr           string
	MetricsAddr        string
	ShutdownTimeout    time.Duration
	DatabaseURL        string
	MigrateDatabaseURL string
	TLSCertFile        string
	TLSKeyFile         string
	TLSCAFile          string
	Callers            []Caller
	EncryptionKey      []byte
	FingerprintKey     []byte
}

type Caller struct {
	Name             string
	CredentialSHA256 [32]byte
	Permissions      map[string]bool
}

type callerFile struct {
	Name             string   `json:"name"`
	CredentialSHA256 string   `json:"credential_sha256"`
	Permissions      []string `json:"permissions"`
}

func FromEnv() (Config, error) {
	return FromLookup(os.Getenv)
}

func FromLookup(lookup func(string) string) (Config, error) {
	cfg := Config{
		GRPCAddr:        defaultGRPCAddr,
		MetricsAddr:     defaultMetricsAddr,
		ShutdownTimeout: defaultShutdownTimeout,
		TLSCertFile:     lookup("VAULT_TLS_CERT_FILE"),
		TLSKeyFile:      lookup("VAULT_TLS_KEY_FILE"),
		TLSCAFile:       lookup("VAULT_TLS_CA_FILE"),
	}
	if value := lookup("VAULT_GRPC_ADDR"); value != "" {
		cfg.GRPCAddr = value
	}
	if value := lookup("VAULT_METRICS_ADDR"); value != "" {
		cfg.MetricsAddr = value
	}
	if value := lookup("VAULT_SHUTDOWN_TIMEOUT"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse VAULT_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = duration
	}
	var err error
	if cfg.DatabaseURL, err = secret(lookup, "VAULT_DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.MigrateDatabaseURL, err = secret(lookup, "VAULT_MIGRATE_DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.MigrateDatabaseURL == "" {
		cfg.MigrateDatabaseURL = cfg.DatabaseURL
	}
	encoded, err := secret(lookup, "VAULT_ENCRYPTION_KEY")
	if err != nil {
		return Config{}, err
	}
	if encoded != "" {
		if cfg.EncryptionKey, err = base64.StdEncoding.DecodeString(encoded); err != nil {
			return Config{}, fmt.Errorf("VAULT_ENCRYPTION_KEY must be base64: %w", err)
		}
	}
	fingerprintKey, err := secret(lookup, "VAULT_FINGERPRINT_KEY")
	if err != nil {
		return Config{}, err
	}
	cfg.FingerprintKey = []byte(fingerprintKey)
	if path := lookup("VAULT_CALLERS_FILE"); path != "" {
		if cfg.Callers, err = loadCallers(path); err != nil {
			return Config{}, err
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if err := validateAddr("VAULT_GRPC_ADDR", c.GRPCAddr); err != nil {
		return err
	}
	if err := validateAddr("VAULT_METRICS_ADDR", c.MetricsAddr); err != nil {
		return err
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("VAULT_SHUTDOWN_TIMEOUT must be greater than zero")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("VAULT_DATABASE_URL is required")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("VAULT_ENCRYPTION_KEY must be base64 of exactly 32 bytes")
	}
	if len(c.FingerprintKey) == 0 {
		return fmt.Errorf("VAULT_FINGERPRINT_KEY is required")
	}
	if c.TLSCertFile == "" || c.TLSKeyFile == "" {
		return fmt.Errorf("VAULT_TLS_CERT_FILE and VAULT_TLS_KEY_FILE are required")
	}
	if len(c.Callers) == 0 {
		return fmt.Errorf("VAULT_CALLERS_FILE must list at least one caller")
	}
	return nil
}

func MigrateURL(lookup func(string) string) (string, error) {
	url, err := secret(lookup, "VAULT_MIGRATE_DATABASE_URL")
	if err != nil {
		return "", err
	}
	if url == "" {
		if url, err = secret(lookup, "VAULT_DATABASE_URL"); err != nil {
			return "", err
		}
	}
	if url == "" {
		return "", fmt.Errorf("VAULT_MIGRATE_DATABASE_URL or VAULT_DATABASE_URL is required")
	}
	return url, nil
}

func loadCallers(path string) ([]Caller, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read VAULT_CALLERS_FILE: %w", err)
	}
	var entries []callerFile
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse VAULT_CALLERS_FILE: %w", err)
	}
	callers := make([]Caller, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			return nil, fmt.Errorf("caller name must not be empty")
		}
		digest, err := hex.DecodeString(entry.CredentialSHA256)
		if err != nil || len(digest) != 32 {
			return nil, fmt.Errorf("caller %q credential_sha256 must be 64 hex characters", entry.Name)
		}
		caller := Caller{Name: entry.Name, Permissions: map[string]bool{}}
		copy(caller.CredentialSHA256[:], digest)
		for _, permission := range entry.Permissions {
			switch permission {
			case PermissionTokenize, PermissionDescribe, PermissionResolve:
				caller.Permissions[permission] = true
			default:
				return nil, fmt.Errorf("caller %q has unknown permission %q", entry.Name, permission)
			}
		}
		callers = append(callers, caller)
	}
	return callers, nil
}

func secret(lookup func(string) string, name string) (string, error) {
	if path := lookup(name + "_FILE"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", name, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return lookup(name), nil
}

func validateAddr(name, addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s must be host:port: %w", name, err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return fmt.Errorf("%s port must be between 0 and 65535", name)
	}
	return nil
}

type Healthcheck struct {
	Address    string
	CAFile     string
	ServerName string
	Timeout    time.Duration
}

func HealthcheckFromLookup(lookup func(string) string) (Healthcheck, error) {
	addr := lookup("VAULT_GRPC_ADDR")
	if addr == "" {
		addr = defaultGRPCAddr
	}
	if err := validateAddr("VAULT_GRPC_ADDR", addr); err != nil {
		return Healthcheck{}, err
	}
	_, port, _ := net.SplitHostPort(addr)
	check := Healthcheck{
		Address:    net.JoinHostPort("127.0.0.1", port),
		CAFile:     lookup("VAULT_HEALTHCHECK_CA_FILE"),
		ServerName: lookup("VAULT_HEALTHCHECK_SERVER_NAME"),
		Timeout:    2 * time.Second,
	}
	if check.CAFile == "" {
		check.CAFile = lookup("VAULT_TLS_CA_FILE")
	}
	if check.CAFile == "" {
		check.CAFile = lookup("VAULT_TLS_CERT_FILE")
	}
	if check.CAFile == "" {
		return Healthcheck{}, fmt.Errorf("VAULT_HEALTHCHECK_CA_FILE, VAULT_TLS_CA_FILE or VAULT_TLS_CERT_FILE is required")
	}
	if check.ServerName == "" {
		name, err := firstDNSName(lookup("VAULT_TLS_CERT_FILE"))
		if err != nil {
			return Healthcheck{}, fmt.Errorf("derive VAULT_HEALTHCHECK_SERVER_NAME: %w", err)
		}
		check.ServerName = name
	}
	if value := lookup("VAULT_HEALTHCHECK_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return Healthcheck{}, fmt.Errorf("VAULT_HEALTHCHECK_TIMEOUT must be a positive duration")
		}
		check.Timeout = timeout
	}
	return check, nil
}

func firstDNSName(certFile string) (string, error) {
	if certFile == "" {
		return "", fmt.Errorf("VAULT_TLS_CERT_FILE is not set")
	}
	raw, err := os.ReadFile(certFile)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return "", fmt.Errorf("no PEM certificate in %s", certFile)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	if len(cert.DNSNames) == 0 {
		return "", fmt.Errorf("server certificate has no DNS SAN")
	}
	return cert.DNSNames[0], nil
}
