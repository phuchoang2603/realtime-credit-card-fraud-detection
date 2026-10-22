package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/db"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/adapters/postgres/sqlcgen"
	"github.com/phuchoang2603/payment-gateway-fraud-detection/vault-simulator/internal/application"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	latest  int64
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	latest, err := db.LatestVersion()
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, queries: sqlcgen.New(pool), latest: latest}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ready(ctx context.Context) error {
	var version int64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&version)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if version != s.latest {
		return fmt.Errorf("migration version %d, want %d", version, s.latest)
	}
	return nil
}

func (s *Store) InsertToken(ctx context.Context, token application.NewToken) error {
	return s.queries.InsertToken(ctx, sqlcgen.InsertTokenParams{
		TokenDigest:       token.Digest,
		IntegrationID:     token.IntegrationID,
		MethodType:        token.MethodType,
		Fingerprint:       token.Fingerprint,
		FixtureCiphertext: token.FixtureCiphertext,
	})
}

func (s *Store) ActiveToken(ctx context.Context, digest []byte, integrationID uuid.UUID) (application.StoredToken, error) {
	row, err := s.queries.GetActiveToken(ctx, sqlcgen.GetActiveTokenParams{TokenDigest: digest, IntegrationID: integrationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return application.StoredToken{}, application.ErrTokenNotFound
	}
	if err != nil {
		return application.StoredToken{}, err
	}
	return application.StoredToken{
		MethodType:        row.MethodType,
		Fingerprint:       row.Fingerprint,
		FixtureCiphertext: row.FixtureCiphertext,
	}, nil
}

func Migrate(ctx context.Context, databaseURL string) error {
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	conn := stdlib.OpenDB(*config)
	defer closeQuietly(conn)
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, db.Migrations())
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func closeQuietly(conn *sql.DB) {
	_ = conn.Close()
}
