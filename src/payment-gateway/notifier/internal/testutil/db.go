package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/adapters/postgres"
	testpostgres "github.com/phuchoang2603/payment-gateway-fraud-detection/shared/testutil/postgres"
)

// NewPool creates a migrated database on the test PostgreSQL and closes it on cleanup.
func NewPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	url := testpostgres.SetupPostgres(t)
	ctx := context.Background()
	if err := postgres.Migrate(ctx, url); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, url
}
