// Command inventory-service is a gRPC server that owns stock levels.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	"github.com/ibnuzaman/kubecart/internal/grpcx"
	"github.com/ibnuzaman/kubecart/internal/inventory"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("service", "inventory")
	if err := run(logger); err != nil {
		logger.Error("exiting", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// SIGTERM is what Docker and Kubernetes send on stop; SIGINT is Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Seed data until Postgres arrives in Phase 2. cherry starts empty so
	// FailedPrecondition is easy to try.
	store := inventory.NewMemStore(map[string]int32{
		"apple":  10,
		"banana": 5,
		"cherry": 0,
	})

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcx.UnaryLogging(logger)))
	inventoryv1.RegisterInventoryServiceServer(srv, inventory.NewServer(store))

	return grpcx.Serve(ctx, srv, envOr("GRPC_ADDR", ":50052"), 10*time.Second, logger)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
