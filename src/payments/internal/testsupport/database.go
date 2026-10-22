package testsupport

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/internal/adapters/postgres"
)

const (
	OwnerRole    = "payments"
	AppRole      = "payments_app"
	CDCRole      = "payments_cdc"
	rolePassword = "payments-test"
	roleLockID   = 0x7061796d
)

var databaseCounter atomic.Int64

type Database struct {
	Name     string
	AdminURL string
	OwnerURL string
	AppURL   string
	Admin    string
	Owner    *pgxpool.Pool
	App      *pgxpool.Pool
}

func RedisAddr(t testing.TB) string {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}
	return addr
}

func withCredentials(t testing.TB, adminURL, user, password, database string) string {
	t.Helper()
	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse TEST_POSTGRES_URL: %v", err)
	}
	parsed.User = url.UserPassword(user, password)
	parsed.Path = "/" + database
	return parsed.String()
}

func EnsureRole(ctx context.Context, admin *pgx.Conn, name, attributes string) error {
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock($1)`, roleLockID); err != nil {
		return err
	}
	defer func() { _, _ = admin.Exec(ctx, `SELECT pg_advisory_unlock($1)`, roleLockID) }()
	var exists bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT FROM pg_roles WHERE rolname = $1)`, name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := admin.Exec(ctx, fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s' %s`, pgx.Identifier{name}.Sanitize(), rolePassword, attributes))
	return err
}

func NewDatabase(t testing.TB) *Database {
	t.Helper()
	adminURL := os.Getenv("TEST_POSTGRES_URL")
	if adminURL == "" {
		t.Skip("TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to admin database: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) })
	for role, attributes := range map[string]string{OwnerRole: "", AppRole: "", CDCRole: "REPLICATION"} {
		if err := EnsureRole(ctx, admin, role, attributes); err != nil {
			t.Fatalf("ensure role %s: %v", role, err)
		}
	}
	name := fmt.Sprintf("payments_test_%s_%d", strings.ReplaceAll(uuid.NewString()[:8], "-", ""), databaseCounter.Add(1))
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, name, OwnerRole)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		cleanup, err := pgx.Connect(ctx, adminURL)
		if err != nil {
			return
		}
		defer func() { _ = cleanup.Close(ctx) }()
		_, _ = cleanup.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name))
	})

	database := &Database{
		Name:     name,
		OwnerURL: withCredentials(t, adminURL, OwnerRole, rolePassword, name),
		AppURL:   withCredentials(t, adminURL, AppRole, rolePassword, name),
	}
	database.AdminURL = replaceDatabase(t, adminURL, name)
	if err := postgres.Migrate(ctx, database.OwnerURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.Admin = replaceDatabase(t, adminURL, name)
	database.Owner = newPool(t, database.OwnerURL)
	database.App = newPool(t, database.AppURL)
	return database
}

func replaceDatabase(t testing.TB, adminURL, database string) string {
	t.Helper()
	parsed, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + database
	return parsed.String()
}

func newPool(t testing.TB, databaseURL string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 16
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func WaitFor(t testing.TB, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func (d *Database) CDCURL(t testing.TB) string {
	t.Helper()
	parsed, err := url.Parse(d.AppURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(CDCRole, rolePassword)
	return parsed.String()
}
