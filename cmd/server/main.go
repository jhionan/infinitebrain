// Package main is the server entry point for Infinite Brain.
package main

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/rian/infinite_brain/pkg/config"
	"github.com/rian/infinite_brain/pkg/database"
	"github.com/rian/infinite_brain/pkg/metrics"
	"github.com/rian/infinite_brain/pkg/middleware"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	cfg, err := config.Load()
	if err != nil {
		// Logger not yet available — use zerolog default writer.
		bootstrap := zerolog.New(os.Stderr).With().Timestamp().Logger()
		bootstrap.Fatal().Err(err).Msg("failed to load config")
	}

	level, _ := zerolog.ParseLevel(cfg.App.LogLevel)
	logger := zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()

	pool, err := database.New(ctx, database.DefaultConfig(cfg.Database.URL))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      buildMux(logger, health.WithProbe("database", pool)),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("infinite-brain starting")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()
	stop()
	logger.Info().Msg("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		cancel()
		logger.Error().Err(err).Msg("forced shutdown")
		os.Exit(1)
	}
	cancel()
	logger.Info().Msg("server stopped")
}

func buildMux(logger zerolog.Logger, checkerOpts ...health.Option) http.Handler {
	mux := http.NewServeMux()

	// connect-go RPC handlers
	mux.Handle(pingv1connect.NewPingServiceHandler(ping.NewHandler()))

	// Plain HTTP health endpoints (k8s liveness + readiness probes)
	checker := health.NewChecker(checkerOpts...)
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
