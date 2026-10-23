package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/payments/db"
)

func migrationsFS() (fs.FS, error) {
	return fs.Sub(db.Migrations, "migrations")
}

func Migrate(ctx context.Context, databaseURL string) error {
	fsys, err := migrationsFS()
	if err != nil {
		return err
	}
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer func() { _ = conn.Close() }()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration locker: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, fsys, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

var migrationName = regexp.MustCompile(`^(\d+)_.*\.sql$`)

func LatestMigrationVersion() (int64, error) {
	entries, err := fs.ReadDir(db.Migrations, "migrations")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, entry := range entries {
		match := migrationName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return 0, err
		}
		latest = max(latest, version)
	}
	if latest == 0 {
		return 0, errors.New("no embedded migrations")
	}
	return latest, nil
}

func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	latest, err := LatestMigrationVersion()
	if err != nil {
		return err
	}
	var applied int64
	err = pool.QueryRow(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&applied)
	if err != nil {
		return fmt.Errorf("read applied migration version: %w", err)
	}
	if applied != latest {
		return fmt.Errorf("applied migration version %d, expected %d", applied, latest)
	}
	return nil
}
