package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultAddress         = ":8080"
	defaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	FraudTarget       string
	PredictionTimeout time.Duration
	Address           string
	ShutdownTimeout   time.Duration
}

func FromEnv() (Config, error) {
	return FromLookup(os.Getenv)
}

func FromLookup(lookup func(string) string) (Config, error) {
	cfg := Config{Address: defaultAddress, ShutdownTimeout: defaultShutdownTimeout, FraudTarget: "dns:///localhost:8000", PredictionTimeout: 3 * time.Second}
	if value := lookup("FRAUD_GRPC_TARGET"); value != "" {
		cfg.FraudTarget = value
	}
	if value := lookup("PREDICTION_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse PREDICTION_TIMEOUT: %w", err)
		}
		cfg.PredictionTimeout = timeout
	}
	if value := lookup("EDGE_HTTP_ADDR"); value != "" {
		cfg.Address = value
	}
	if value := lookup("EDGE_SHUTDOWN_TIMEOUT"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse EDGE_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = duration
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("EDGE_HTTP_ADDR must not be empty")
	}
	_, port, err := net.SplitHostPort(c.Address)
	if err != nil {
		return fmt.Errorf("EDGE_HTTP_ADDR must be host:port: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 0 || portNumber > 65535 {
		return fmt.Errorf("EDGE_HTTP_ADDR port must be between 0 and 65535")
	}
	if c.FraudTarget == "" {
		return fmt.Errorf("FRAUD_GRPC_TARGET must not be empty")
	}
	if c.PredictionTimeout <= 0 {
		return fmt.Errorf("PREDICTION_TIMEOUT must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be greater than zero")
	}
	return nil
}
