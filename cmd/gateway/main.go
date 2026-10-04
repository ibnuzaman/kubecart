// Command gateway is the HTTP/REST entry point. It translates REST calls
// into gRPC calls to order-service.
//
// Usage: gateway [serve|healthcheck] (default: serve)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/ibnuzaman/kubecart/gen/go/order/v1"
	"github.com/ibnuzaman/kubecart/internal/gateway"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("service", "gateway")

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "serve":
		err = serve(logger)
	case "healthcheck":
		err = healthcheck(envOr("HTTP_ADDR", ":8080"))
	default:
		err = fmt.Errorf("unknown command %q (want serve or healthcheck)", cmd)
	}
	if err != nil {
		logger.Error("exiting", "command", cmd, "error", err)
		os.Exit(1)
	}
}

func serve(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := grpc.NewClient(envOr("ORDER_ADDR", "localhost:50051"),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	srv := &http.Server{
		Addr: envOr("HTTP_ADDR", ":8080"),
		// 3s for the whole chain: gateway -> order -> inventory.
		Handler:           gateway.NewHandler(orderv1.NewOrderServiceClient(conn), 3*time.Second),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Bind before logging, so "listening" is only printed once it's true.
	lis, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()
	logger.Info("http server listening", "addr", lis.Addr().String())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// Shutdown stops accepting connections and waits for active requests.
	logger.Info("shutting down, waiting for in-flight requests")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("stopped gracefully")
	return nil
}

// healthcheck calls the local /healthz. Distroless images have no curl, so
// the binary probes itself.
func healthcheck(listenAddr string) error {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
