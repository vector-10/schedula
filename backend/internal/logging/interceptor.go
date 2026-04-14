package logging

import (
	"context"
	"log/slog"
	"time"

	"github.com/vector-10/schedula/backend/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		userID := auth.GetUserIDFromContext(ctx)
		code := status.Code(err)

		attrs := []any{
			"method", info.FullMethod,
			"user_id", userID,
			"duration_ms", duration.Milliseconds(),
			"code", code.String(),
		}

		switch {
		case err == nil:
			slog.Info("rpc", attrs...)
		case code == codes.Internal:
			// already logged with underlying cause by internalErr helper
		case code == codes.Unauthenticated || code == codes.PermissionDenied:
			slog.Warn("rpc auth failure", append(attrs, "error", status.Convert(err).Message())...)
		default:
			slog.Info("rpc error", append(attrs, "error", status.Convert(err).Message())...)
		}

		return resp, err
	}
}
