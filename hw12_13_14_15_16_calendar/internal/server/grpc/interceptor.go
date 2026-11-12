package internalgrpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func loggingInterceptor(logg Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		ip := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			if host, _, splitErr := net.SplitHostPort(p.Addr.String()); splitErr == nil { // (2)
				ip = host
			}
		}

		logg.Info(
			"grpc request",
			"ip", ip,
			"method", info.FullMethod,
			"status", status.Code(err).String(),
			"start", start.Format(time.RFC3339),
			"duration", time.Since(start),
		)

		return resp, err
	}
}
