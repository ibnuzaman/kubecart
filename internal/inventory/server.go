package inventory

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
)

// Server implements inventoryv1.InventoryServiceServer.
type Server struct {
	inventoryv1.UnimplementedInventoryServiceServer
	store Store
}

func NewServer(store Store) *Server {
	return &Server{store: store}
}

func (s *Server) CheckStock(ctx context.Context, req *inventoryv1.CheckStockRequest) (*inventoryv1.CheckStockResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}
	n, err := s.store.Available(ctx, req.GetSku())
	if err != nil {
		return nil, toStatus(err)
	}
	return &inventoryv1.CheckStockResponse{Sku: req.GetSku(), Available: n}, nil
}

func (s *Server) ReserveStock(ctx context.Context, req *inventoryv1.ReserveStockRequest) (*inventoryv1.ReserveStockResponse, error) {
	if req.GetSku() == "" {
		return nil, status.Error(codes.InvalidArgument, "sku is required")
	}
	if req.GetQuantity() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be positive")
	}
	remaining, err := s.store.Reserve(ctx, req.GetSku(), req.GetQuantity())
	if err != nil {
		return nil, toStatus(err)
	}
	return &inventoryv1.ReserveStockResponse{Sku: req.GetSku(), Remaining: remaining}, nil
}

// toStatus maps domain errors to gRPC status codes. Unknown errors become
// Internal without their message, so internals don't leak to callers.
func toStatus(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrInsufficientStock):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
