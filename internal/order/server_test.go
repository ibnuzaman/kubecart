package order

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
)

// fakeInventory is an InventoryServiceClient whose ReserveStock behavior is
// set per test case.
type fakeInventory struct {
	inventoryv1.InventoryServiceClient // nil; calling an unset method panics
	reserve                            func(ctx context.Context) error
	calls                              int
}

func (f *fakeInventory) ReserveStock(ctx context.Context, in *inventoryv1.ReserveStockRequest, _ ...grpc.CallOption) (*inventoryv1.ReserveStockResponse, error) {
	f.calls++
	if err := f.reserve(ctx); err != nil {
		return nil, err
	}
	return &inventoryv1.ReserveStockResponse{Sku: in.GetSku()}, nil
}

func TestCreateOrder(t *testing.T) {
	ok := func(context.Context) error { return nil }
	fail := func(c codes.Code) func(context.Context) error {
		return func(context.Context) error { return status.Error(c, "boom") }
	}
	// slow honors the context like a real gRPC call: it returns when the
	// deadline set by CreateOrder fires.
	slow := func(ctx context.Context) error {
		select {
		case <-time.After(time.Second):
			return nil
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		}
	}

	tests := []struct {
		name          string
		req           *orderv1.CreateOrderRequest
		reserve       func(context.Context) error
		wantCode      codes.Code
		wantCalls     int // calls to inventory
		wantSavedRows int
	}{
		{name: "creates order", req: &orderv1.CreateOrderRequest{Sku: "apple", Quantity: 2}, reserve: ok, wantCode: codes.OK, wantCalls: 1, wantSavedRows: 1},
		{name: "empty sku is rejected before inventory", req: &orderv1.CreateOrderRequest{Quantity: 2}, reserve: ok, wantCode: codes.InvalidArgument},
		{name: "zero quantity is rejected before inventory", req: &orderv1.CreateOrderRequest{Sku: "apple"}, reserve: ok, wantCode: codes.InvalidArgument},
		{name: "unknown sku passes through", req: &orderv1.CreateOrderRequest{Sku: "durian", Quantity: 1}, reserve: fail(codes.NotFound), wantCode: codes.NotFound, wantCalls: 1},
		{name: "out of stock passes through", req: &orderv1.CreateOrderRequest{Sku: "cherry", Quantity: 1}, reserve: fail(codes.FailedPrecondition), wantCode: codes.FailedPrecondition, wantCalls: 1},
		{name: "inventory down passes through", req: &orderv1.CreateOrderRequest{Sku: "apple", Quantity: 1}, reserve: fail(codes.Unavailable), wantCode: codes.Unavailable, wantCalls: 1},
		{name: "inventory internal error is hidden", req: &orderv1.CreateOrderRequest{Sku: "apple", Quantity: 1}, reserve: fail(codes.DataLoss), wantCode: codes.Internal, wantCalls: 1},
		{name: "slow inventory hits deadline", req: &orderv1.CreateOrderRequest{Sku: "apple", Quantity: 1}, reserve: slow, wantCode: codes.DeadlineExceeded, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := &fakeInventory{reserve: tt.reserve}
			store := NewMemStore()
			srv := NewServer(store, inv, 50*time.Millisecond)

			resp, err := srv.CreateOrder(context.Background(), tt.req)

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", got, tt.wantCode, err)
			}
			if inv.calls != tt.wantCalls {
				t.Errorf("inventory calls = %d, want %d", inv.calls, tt.wantCalls)
			}
			if store.Len() != tt.wantSavedRows {
				t.Errorf("saved orders = %d, want %d", store.Len(), tt.wantSavedRows)
			}
			if tt.wantCode == codes.OK && resp.GetOrder().GetId() == "" {
				t.Error("order id is empty")
			}
		})
	}
}
