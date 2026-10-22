package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/adapters/postgres"
)

var counter atomic.Int64

// NewPool creates a migrated, uniquely named database on TEST_POSTGRES_URL and drops it on cleanup; it skips when unset.
func NewPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	adminURL := os.Getenv("TEST_POSTGRES_URL")
	if adminURL == "" {
		t.Skip("TEST_POSTGRES_URL not set")
	}
	ctx := context.Background()
	name := fmt.Sprintf("notifier_test_%d_%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000, counter.Add(1))
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("create database: %v", err)
	}
	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	parsed.Path = "/" + name
	dbURL := parsed.String()
	if err := postgres.Migrate(ctx, dbURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	return pool, dbURL
}
