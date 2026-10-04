package inventory

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL migrations for the inventory database. They ship
// inside the binary, so the schema version always matches the code.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// PgStore is a Store backed by Postgres.
type PgStore struct {
	pool *pgxpool.Pool
}

func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

func (s *PgStore) Available(ctx context.Context, sku string) (int32, error) {
	var n int32
	err := s.pool.QueryRow(ctx, `SELECT quantity FROM stock WHERE sku = $1`, sku).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, sku)
	}
	if err != nil {
		return 0, fmt.Errorf("query stock: %w", err)
	}
	return n, nil
}

func (s *PgStore) Reserve(ctx context.Context, sku string, quantity int32) (int32, error) {
	// Check and decrement in one statement. Postgres locks the row for the
	// UPDATE, so concurrent reservations queue on it and can't oversell.
	var remaining int32
	err := s.pool.QueryRow(ctx, `
		UPDATE stock SET quantity = quantity - $2
		WHERE sku = $1 AND quantity >= $2
		RETURNING quantity`, sku, quantity).Scan(&remaining)
	if err == nil {
		return remaining, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("reserve stock: %w", err)
	}

	// No row matched: the SKU is unknown or there isn't enough stock. This
	// read only picks the error to return; it doesn't decide anything.
	n, err := s.Available(ctx, sku)
	if err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("%w: %s has %d, want %d", ErrInsufficientStock, sku, n, quantity)
}
