// Command order-service is a gRPC server that creates orders and calls
// inventory-service to reserve stock.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
	"github.com/ibnuzaman/kubecart/internal/grpcx"
	"github.com/ibnuzaman/kubecart/internal/order"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("service", "order")
	if err := run(logger); err != nil {
		logger.Error("exiting", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// NewClient does no I/O; it connects lazily on the first RPC. Plaintext is
	// fine locally; TLS between services comes later.
	conn, err := grpc.NewClient(envOr("INVENTORY_ADDR", "localhost:50052"),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	// Closed after Serve returns, so RPCs still draining can reach inventory.
	defer conn.Close()

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcx.UnaryLogging(logger)))
	orderv1.RegisterOrderServiceServer(srv, order.NewServer(
		order.NewMemStore(),
		inventoryv1.NewInventoryServiceClient(conn),
		1*time.Second,
	))

	return grpcx.Serve(ctx, srv, envOr("GRPC_ADDR", ":50051"), 10*time.Second, logger)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
