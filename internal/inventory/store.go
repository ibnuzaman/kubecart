// Package inventory implements InventoryService.
package inventory

import (
	"context"
	"errors"
)

// Domain errors. The store returns these; the gRPC layer maps them to status
// codes, so MemStore (tests) and PgStore share the same server code.
var (
	ErrNotFound          = errors.New("sku not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// Store keeps stock levels.
type Store interface {
	Available(ctx context.Context, sku string) (int32, error)
	// Reserve checks and decrements stock as one atomic step and returns the
	// units left.
	Reserve(ctx context.Context, sku string, quantity int32) (int32, error)
}
