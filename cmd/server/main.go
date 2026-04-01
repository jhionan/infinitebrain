// Package main is the server entry point for Infinite Brain.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/api/gen/ping/v1/pingv1connect"
	"github.com/rian/infinite_brain/internal/health"
	"github.com/rian/infinite_brain/internal/ping"
	"github.com/rian/infinite_brain/internal/security"
	"github.com/rian/infinite_brain/pkg/metrics"
	"github.com/rian/infinite_brain/pkg/middleware"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      buildMux(logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("infinite-brain starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		cancel()
		log.Printf("forced shutdown: %v", err)
		os.Exit(1)
	}
	cancel()

	log.Println("server stopped")
}

func buildMux(logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()

	// connect-go RPC handlers
	mux.Handle(pingv1connect.NewPingServiceHandler(ping.NewHandler()))

	// Plain HTTP health endpoints (k8s liveness + readiness probes)
	checker := health.NewChecker()
	h := health.NewHandler(checker)
	mux.HandleFunc("/health/live", h.Live)
	mux.HandleFunc("/health/ready", h.Ready)

	// Legacy health route kept for backwards compatibility
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"infinite-brain"}`) //nolint:errcheck
	})

	// Prometheus metrics
	mux.Handle("/metrics", metrics.Handler())

	// Honeypot traps — scanners and bots hitting these paths are recorded and can be blocked
	honeypot := security.NewHandler(security.NoopRepository{}, logger)
	for _, path := range []string{"/.env", "/.git/config", "/wp-admin", "/admin", "/phpMyAdmin"} {
		mux.Handle(path, honeypot)
	}

	// Middleware chain (outermost → innermost)
	allowedOrigins := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	var handler http.Handler = mux
	handler = middleware.CORS(allowedOrigins)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.IPBlocker(security.NoopRepository{}, logger)(handler)
	handler = http.MaxBytesHandler(handler, maxBodyBytes)

	return handler
}

// parseAllowedOrigins splits a comma-separated ALLOWED_ORIGINS env var.
// Returns an empty slice if the value is empty.
func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
