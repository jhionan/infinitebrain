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
	"syscall"
	"time"

	"github.com/rian/infinite_brain/api/gen/ping/v1/pingv1connect"
	"github.com/rian/infinite_brain/internal/health"
	"github.com/rian/infinite_brain/internal/ping"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      buildMux(),
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

func buildMux() *http.ServeMux {
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

	return mux
}
