// Package order implements OrderService.
package order

import (
	"context"
	"sync"
)

// Order is the domain model, kept separate from the protobuf type so storage
// doesn't depend on the wire format.
type Order struct {
	ID       string
	SKU      string
	Quantity int32
}

// Store persists orders.
type Store interface {
	Save(ctx context.Context, o Order) error
}

// MemStore is an in-memory Store. Its data is lost on restart.
type MemStore struct {
	mu     sync.Mutex
	orders map[string]Order
}

func NewMemStore() *MemStore {
	return &MemStore{orders: make(map[string]Order)}
}

func (s *MemStore) Save(_ context.Context, o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[o.ID] = o
	return nil
}

// Len returns the number of stored orders.
func (s *MemStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.orders)
}
