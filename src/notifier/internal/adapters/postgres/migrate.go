package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func newProvider(sqlDB *sql.DB) (*goose.Provider, error) {
	migrations, err := fsSub()
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
}

func Migrate(ctx context.Context, url string) error {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("parse migrate database url: %w", err)
	}
	sqlDB := stdlib.OpenDB(*config.ConnConfig)
	defer func() { _ = sqlDB.Close() }()
	provider, err := newProvider(sqlDB)
	if err != nil {
		return err
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

type Readiness struct {
	sqlDB    *sql.DB
	provider *goose.Provider
	latest   int64
}

func NewReadiness(pool *pgxpool.Pool) (*Readiness, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)
	provider, err := newProvider(sqlDB)
	if err != nil {
		return nil, err
	}
	var latest int64
	for _, source := range provider.ListSources() {
		latest = max(latest, source.Version)
	}
	return &Readiness{sqlDB: sqlDB, provider: provider, latest: latest}, nil
}

func (r *Readiness) Check(ctx context.Context) error {
	if err := r.sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	version, err := r.provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if version != r.latest {
		return fmt.Errorf("migration version %d, want %d", version, r.latest)
	}
	return nil
}

func (r *Readiness) Close() error { return r.sqlDB.Close() }
