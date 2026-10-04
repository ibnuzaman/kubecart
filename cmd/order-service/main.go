// Command order-service is a gRPC server that creates orders and calls
// inventory-service to reserve stock.
//
// Usage: order-service [serve|migrate|healthcheck] (default: serve)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
	"github.com/ibnuzaman/kubecart/internal/db"
	"github.com/ibnuzaman/kubecart/internal/grpcx"
	"github.com/ibnuzaman/kubecart/internal/order"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("service", "order")

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "serve":
		err = serve(logger)
	case "migrate":
		err = db.Migrate(mustEnv("DATABASE_URL"), order.Migrations, logger)
	case "healthcheck":
		err = grpcx.CheckHealth(envOr("GRPC_ADDR", ":50051"))
	default:
		err = fmt.Errorf("unknown command %q (want serve, migrate or healthcheck)", cmd)
	}
	if err != nil {
		logger.Error("exiting", "command", cmd, "error", err)
		os.Exit(1)
	}
}

func serve(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pool, err := db.Connect(connectCtx, mustEnv("DATABASE_URL"), logger)
	if err != nil {
		return err
	}
	defer pool.Close()

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
		order.NewPgStore(pool),
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s is required\n", key)
		os.Exit(1)
	}
	return v
}
