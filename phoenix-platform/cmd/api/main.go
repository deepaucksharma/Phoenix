package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/phoenix/platform/pkg/api"
	pb "github.com/phoenix/platform/pkg/api/v1"
	"github.com/phoenix/platform/pkg/auth"
	"github.com/phoenix/platform/pkg/generator"
	"github.com/phoenix/platform/pkg/metrics"
	"github.com/phoenix/platform/pkg/store"
)

const (
	defaultGRPCPort = 5050
	defaultHTTPPort = 8080
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Initialize metrics
	metrics.InitMetrics()

	// Initialize store
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://phoenix:phoenix@localhost/phoenix?sslmode=disable"
	}

	store, err := store.NewPostgresStore(dbURL)
	if err != nil {
		logger.Fatal("failed to initialize store", zap.Error(err))
	}
	defer store.Close()

	// Initialize services
	authService := auth.NewService(os.Getenv("JWT_SECRET"))
	generatorService := generator.NewService(
		os.Getenv("GIT_REPO_URL"),
		os.Getenv("GIT_TOKEN"),
	)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryInterceptor(authService)),
		grpc.StreamInterceptor(auth.StreamInterceptor(authService)),
	)

	// Register services
	experimentService := api.NewExperimentService(store, generatorService, logger)
	pb.RegisterExperimentServiceServer(grpcServer, experimentService)

	// Enable reflection
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcPort := getEnvInt("GRPC_PORT", defaultGRPCPort)
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		logger.Info("starting gRPC server", zap.Int("port", grpcPort))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("failed to serve gRPC", zap.Error(err))
		}
	}()

	// Create HTTP server
	httpPort := getEnvInt("HTTP_PORT", defaultHTTPPort)
	httpServer := createHTTPServer(httpPort, grpcPort, logger)

	// Start HTTP server
	go func() {
		logger.Info("starting HTTP server", zap.Int("port", httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to serve HTTP", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown HTTP server", zap.Error(err))
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	logger.Info("servers stopped")
}

func createHTTPServer(httpPort, grpcPort int, logger *zap.Logger) *http.Server {
	// Create router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(middleware.Timeout(60 * time.Second))

	// CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Metrics
	router.Handle("/metrics", promhttp.Handler())

	// gRPC-Gateway
	ctx := context.Background()
	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("localhost:%d", grpcPort)

	err := pb.RegisterExperimentServiceHandlerFromEndpoint(ctx, gwmux, endpoint, opts)
	if err != nil {
		logger.Fatal("failed to register gateway", zap.Error(err))
	}

	// Mount API routes
	router.Mount("/api/v1", gwmux)

	// WebSocket handler
	wsHandler := api.NewWebSocketHandler(logger)
	router.HandleFunc("/ws", wsHandler.ServeHTTP)

	// Static files (dashboard)
	if os.Getenv("SERVE_STATIC") == "true" {
		fileServer := http.FileServer(http.Dir("./dist"))
		router.Handle("/*", fileServer)
	}

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}