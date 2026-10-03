package order

import (
	"context"
	"crypto/rand"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
)

// Server implements orderv1.OrderServiceServer.
type Server struct {
	orderv1.UnimplementedOrderServiceServer
	store            Store
	inventory        inventoryv1.InventoryServiceClient
	inventoryTimeout time.Duration
}

// NewServer returns a Server. inventoryTimeout caps each call to
// InventoryService; the caller's own deadline still applies if it is shorter.
func NewServer(store Store, inventory inventoryv1.InventoryServiceClient, inventoryTimeout time.Duration) *Server {
	return &Server{store: store, inventory: inventory, inventoryTimeout: inventoryTimeout}
}

func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}
	if req.GetQuantity() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be positive")
	}

	// Reserve directly instead of CheckStock then ReserveStock: a separate
	// check would race with other orders between the two calls.
	invCtx, cancel := context.WithTimeout(ctx, s.inventoryTimeout)
	defer cancel()
	if _, err := s.inventory.ReserveStock(invCtx, &inventoryv1.ReserveStockRequest{
		Sku:      req.GetSku(),
		Quantity: req.GetQuantity(),
	}); err != nil {
		return nil, fromInventory(err)
	}

	o := Order{ID: rand.Text(), SKU: req.GetSku(), Quantity: req.GetQuantity()}
	// If Save fails, the stock stays reserved. In memory it can't fail; with
	// Postgres it can, and handling that is part of Phase 7.
	if err := s.store.Save(ctx, o); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &orderv1.CreateOrderResponse{Order: &orderv1.Order{
		Id:       o.ID,
		Sku:      o.SKU,
		Quantity: o.Quantity,
	}}, nil
}

// fromInventory passes through codes that mean something to our caller and
// hides everything else behind Internal.
func fromInventory(err error) error {
	st := status.Convert(err)
	switch st.Code() {
	case codes.NotFound, codes.FailedPrecondition, codes.InvalidArgument,
		codes.DeadlineExceeded, codes.Unavailable, codes.Canceled:
		return status.Error(st.Code(), "inventory: "+st.Message())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
