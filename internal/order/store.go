// Package order implements OrderService.
package order

import "context"

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
