package order

import (
	"context"
	"sync"
)

// MemStore is an in-memory Store used as a fast fake in unit tests.
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
