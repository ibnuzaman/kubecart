// Package db holds the Postgres plumbing shared by order-service and
// inventory-service: connecting with retry and running migrations.
package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Connect opens a connection pool and waits until Postgres answers. It retries
// with backoff until ctx is done, so a service that starts before the database
// is ready waits instead of crashing. Pool size comes from pool_max_conns in
// url.
func Connect(ctx context.Context, url string, logger *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	delay := 500 * time.Millisecond
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			logger.Info("connected to postgres", "max_conns", pool.Config().MaxConns)
			return pool, nil
		}
		logger.Warn("postgres not ready, retrying", "error", err, "retry_in", delay)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("postgres not ready: %w", err)
		case <-time.After(delay):
		}
		delay = min(delay*2, 5*time.Second)
	}
}

// Migrate applies all pending up migrations from the "migrations" directory
// of fsys. golang-migrate takes a Postgres advisory lock, so two migrators
// running at once queue up instead of racing.
func Migrate(url string, fsys fs.FS, logger *slog.Logger) error {
	// pgxpool.ParseConfig strips the pool_* parameters, which plain Postgres
	// would reject as unknown runtime parameters.
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	sqlDB := stdlib.OpenDB(*cfg.ConnConfig)
	defer sqlDB.Close()

	driver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	src, err := iofs.New(fsys, "migrations")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	version, dirty, err := m.Version()
	if err != nil {
		return err
	}
	logger.Info("migrations up to date", "version", version, "dirty", dirty)
	return nil
}
