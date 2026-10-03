package inventory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
)

func TestReserveStock(t *testing.T) {
	tests := []struct {
		name          string
		sku           string
		quantity      int32
		wantCode      codes.Code
		wantRemaining int32
	}{
		{name: "reserves and returns remaining", sku: "apple", quantity: 3, wantCode: codes.OK, wantRemaining: 7},
		{name: "can take the last unit", sku: "apple", quantity: 10, wantCode: codes.OK, wantRemaining: 0},
		{name: "empty sku", sku: "", quantity: 1, wantCode: codes.InvalidArgument},
		{name: "zero quantity", sku: "apple", quantity: 0, wantCode: codes.InvalidArgument},
		{name: "negative quantity", sku: "apple", quantity: -1, wantCode: codes.InvalidArgument},
		{name: "unknown sku", sku: "durian", quantity: 1, wantCode: codes.NotFound},
		{name: "not enough stock", sku: "apple", quantity: 11, wantCode: codes.FailedPrecondition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(NewMemStore(map[string]int32{"apple": 10}))

			resp, err := srv.ReserveStock(context.Background(), &inventoryv1.ReserveStockRequest{
				Sku:      tt.sku,
				Quantity: tt.quantity,
			})

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", got, tt.wantCode, err)
			}
			if tt.wantCode == codes.OK && resp.GetRemaining() != tt.wantRemaining {
				t.Errorf("remaining = %d, want %d", resp.GetRemaining(), tt.wantRemaining)
			}
		})
	}
}

// TestReserveStockConcurrent checks that concurrent reservations never
// oversell: 20 goroutines race for 5 units, so exactly 5 must succeed.
func TestReserveStockConcurrent(t *testing.T) {
	srv := NewServer(NewMemStore(map[string]int32{"apple": 5}))

	var ok atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := srv.ReserveStock(context.Background(), &inventoryv1.ReserveStockRequest{Sku: "apple", Quantity: 1})
			if err == nil {
				ok.Add(1)
			}
		})
	}
	wg.Wait()

	if ok.Load() != 5 {
		t.Errorf("successful reservations = %d, want 5", ok.Load())
	}
}
