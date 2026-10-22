package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	postgresOnce sync.Once
	postgresURL  string
	postgresErr  error
)

func adminURL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("short mode: skipping tests that need PostgreSQL")
	}
	postgresOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		container, err := pgmodule.Run(ctx, "postgres:17.6-alpine",
			pgmodule.WithDatabase("postgres"),
			pgmodule.WithUsername("postgres"),
			pgmodule.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(2*time.Minute),
			),
		)
		if err != nil {
			postgresErr = err
			return
		}
		postgresURL, postgresErr = container.ConnectionString(ctx, "sslmode=disable")
	})
	if postgresErr != nil {
		t.Fatalf("start postgres container (is Docker available?): %v", postgresErr)
	}
	return postgresURL
}

func NewDatabase(t *testing.T) string {
	t.Helper()
	adminURL := adminURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := "vault_test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", name)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		conn, err := pgx.Connect(cleanupCtx, adminURL)
		if err != nil {
			t.Errorf("connect admin database for cleanup: %v", err)
			return
		}
		defer func() { _ = conn.Close(cleanupCtx) }()
		if _, err := conn.Exec(cleanupCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name)); err != nil {
			t.Errorf("drop database: %v", err)
		}
	})

	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	return parsed.String()
}
