package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/phoenix-vnext/analytics/internal/api"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	// Get configuration from environment
	port := getEnv("ANALYTICS_PORT", "8080")
	promAddr := getEnv("PROMETHEUS_ADDR", "http://prometheus:9090")
	
	// Create handler
	handler, err := api.NewHandler(promAddr, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to create handler")
	}

	// Setup routes
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	
	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/trends/analyze", handler.AnalyzeTrend)
		r.Post("/correlations/analyze", handler.AnalyzeCorrelations)
		r.Post("/visualizations/generate", handler.GenerateVisualization)
		
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		})
	})

	// Static files for web UI (if needed)
	workDir, _ := os.Getwd()
	filesDir := http.Dir(workDir + "/static")
	FileServer(r, "/", filesDir)

	// Create server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		logger.WithField("port", port).Info("starting analytics service")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("server forced to shutdown")
	}

	logger.Info("server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := fmt.Sprintf("/%s", rctx.RoutePattern())
		if pathPrefix != "/" && pathPrefix[len(pathPrefix)-1] != '/' {
			pathPrefix += "/"
		}
		r.URL.Path = fmt.Sprintf("/%s", rctx.URLParam("*"))
		http.FileServer(root).ServeHTTP(w, r)
	})
}