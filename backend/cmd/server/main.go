package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	gen "github.com/vector-10/schedula/backend/gen"
	"github.com/vector-10/schedula/backend/internal/appointments"
	"github.com/vector-10/schedula/backend/internal/auth"
	"github.com/vector-10/schedula/backend/internal/db"
	"github.com/vector-10/schedula/backend/internal/logging"
	"github.com/vector-10/schedula/backend/internal/ratelimit"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	databaseURL := requireEnv("DATABASE_URL")
	jwtSecret := requireEnv("JWT_SECRET")
	grpcPort := getEnv("GRPC_PORT", "50051")
	httpPort := getEnv("HTTP_PORT", "8080")
	migrationsPath := getEnv("MIGRATIONS_PATH", "/app/migrations")

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database, migrationsPath); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	authMiddleware := auth.NewMiddleware(jwtSecret)
	rateLimiter := ratelimit.NewLimiter()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			rateLimiter.UnaryInterceptor(),
			logging.UnaryInterceptor(),
			authMiddleware.UnaryInterceptor(),
		),
	)

	gen.RegisterAuthServiceServer(grpcServer, auth.NewService(database, jwtSecret))
	gen.RegisterAppointmentServiceServer(grpcServer, appointments.NewService(database))
	reflection.Register(grpcServer)

	grpcAddr := fmt.Sprintf(":%s", grpcPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	go func() {
		slog.Info("gRPC server listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	ctx := context.Background()
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if key == "Authorization" || key == "authorization" {
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcEndpoint := fmt.Sprintf("localhost:%s", grpcPort)

	if err := gen.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register auth gateway: %v", err)
	}
	if err := gen.RegisterAppointmentServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register appointments gateway: %v", err)
	}

	httpAddr := fmt.Sprintf(":%s", httpPort)
	slog.Info("HTTP server listening", "addr", httpAddr)

	if err := http.ListenAndServe(httpAddr, withCORS(mux)); err != nil {
		log.Fatalf("http serve: %v", err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
