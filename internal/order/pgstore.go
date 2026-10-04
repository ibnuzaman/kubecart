package order

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL migrations for the orders database. They ship
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

func (s *PgStore) Save(ctx context.Context, o Order) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO orders (id, sku, quantity) VALUES ($1, $2, $3)`,
		o.ID, o.SKU, o.Quantity)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}
