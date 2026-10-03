// Package grpcx holds the gRPC server plumbing shared by order-service and
// inventory-service: request logging and graceful shutdown.
package grpcx

import (
	"context"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryLogging logs one line per unary RPC: method, status code, duration and,
// if the caller set one, how much time was left on the deadline when the call
// arrived.
func UnaryLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		attrs := []slog.Attr{slog.String("method", info.FullMethod)}
		if dl, ok := ctx.Deadline(); ok {
			attrs = append(attrs, slog.Duration("deadline_left", time.Until(dl).Round(time.Millisecond)))
		}

		resp, err := handler(ctx, req)

		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelWarn
			attrs = append(attrs, slog.String("error", status.Convert(err).Message()))
		}
		attrs = append(attrs,
			slog.String("code", status.Code(err).String()),
			slog.Duration("duration", time.Since(start)),
		)
		logger.LogAttrs(ctx, level, "rpc", attrs...)
		return resp, err
	}
}

// Serve runs srv on addr until ctx is cancelled (e.g. by SIGTERM), then stops
// it gracefully: no new RPCs are accepted and in-flight ones may finish. If
// they take longer than timeout, the server is stopped forcefully.
func Serve(ctx context.Context, srv *grpc.Server, addr string, timeout time.Duration, logger *slog.Logger) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()
	logger.Info("grpc server listening", "addr", lis.Addr().String())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down, waiting for in-flight RPCs")
	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		logger.Info("stopped gracefully")
	case <-time.After(timeout):
		logger.Warn("graceful stop timed out, forcing stop")
		srv.Stop()
	}
	return nil
}
