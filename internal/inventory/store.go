// Package inventory implements InventoryService.
package inventory

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
)

// Domain errors. The store returns these; the gRPC layer maps them to status
// codes, so a Postgres store (Phase 2) can reuse the same server code.
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

// MemStore is an in-memory Store. Its data is lost on restart.
type MemStore struct {
	mu    sync.Mutex
	stock map[string]int32
}

// NewMemStore returns a MemStore seeded with a copy of stock.
func NewMemStore(stock map[string]int32) *MemStore {
	s := &MemStore{stock: make(map[string]int32, len(stock))}
	maps.Copy(s.stock, stock)
	return s
}

func (s *MemStore) Available(_ context.Context, sku string) (int32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.stock[sku]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, sku)
	}
	return n, nil
}

func (s *MemStore) Reserve(_ context.Context, sku string, quantity int32) (int32, error) {
	// One lock around check + decrement, so two concurrent reservations can't
	// both see enough stock and oversell.
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.stock[sku]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, sku)
	}
	if n < quantity {
		return 0, fmt.Errorf("%w: %s has %d, want %d", ErrInsufficientStock, sku, n, quantity)
	}
	s.stock[sku] = n - quantity
	return n - quantity, nil
}
