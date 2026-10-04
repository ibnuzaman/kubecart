// Command inventory-service is a gRPC server that owns stock levels.
//
// Usage: inventory-service [serve|migrate|healthcheck] (default: serve)
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

	inventoryv1 "github.com/ibnuzaman/kubecart/gen/go/inventory/v1"
	"github.com/ibnuzaman/kubecart/internal/db"
	"github.com/ibnuzaman/kubecart/internal/grpcx"
	"github.com/ibnuzaman/kubecart/internal/inventory"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("service", "inventory")

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "serve":
		err = serve(logger)
	case "migrate":
		err = db.Migrate(mustEnv("DATABASE_URL"), inventory.Migrations, logger)
	case "healthcheck":
		err = grpcx.CheckHealth(envOr("GRPC_ADDR", ":50052"))
	default:
		err = fmt.Errorf("unknown command %q (want serve, migrate or healthcheck)", cmd)
	}
	if err != nil {
		logger.Error("exiting", "command", cmd, "error", err)
		os.Exit(1)
	}
}

func serve(logger *slog.Logger) error {
	// SIGTERM is what Docker and Kubernetes send on stop; SIGINT is Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	pool, err := db.Connect(connectCtx, mustEnv("DATABASE_URL"), logger)
	if err != nil {
		return err
	}
	// Closed after Serve returns, so draining RPCs can still query.
	defer pool.Close()

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcx.UnaryLogging(logger)))
	inventoryv1.RegisterInventoryServiceServer(srv, inventory.NewServer(inventory.NewPgStore(pool)))

	return grpcx.Serve(ctx, srv, envOr("GRPC_ADDR", ":50052"), 10*time.Second, logger)
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
