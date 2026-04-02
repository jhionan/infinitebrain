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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/api/gen/ping/v1/pingv1connect"
	"github.com/rian/infinite_brain/internal/audit"
	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/capture"
	"github.com/rian/infinite_brain/internal/health"
	"github.com/rian/infinite_brain/internal/org"
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
		// Logger not yet available — write directly and exit.
		fmt.Fprintf(os.Stderr, `{"level":"fatal","error":"%v","message":"failed to load config"}`+"\n", err)
		os.Exit(1)
	}

	level, _ := zerolog.ParseLevel(cfg.App.LogLevel)
	logger := zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()

	pool, err := database.New(ctx, database.DefaultConfig(cfg.Database.URL))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      buildMux(cfg, pool, logger, health.WithProbe("database", pool)),
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
		pool.Close() // release connections before hard exit
		logger.Error().Err(err).Msg("forced shutdown")
		os.Exit(1)
	}
	cancel()
	pool.Close()
	logger.Info().Msg("server stopped")
}

// apiDeps holds the DB-backed dependencies returned from registerAPIRoutes.
type apiDeps struct {
	orgRepo       org.Repository
	auditRecorder audit.Recorder
}

func buildMux(cfg *config.Config, pool *pgxpool.Pool, logger zerolog.Logger, checkerOpts ...health.Option) http.Handler {
	mux := http.NewServeMux()

	deps := registerAPIRoutes(mux, cfg, pool, logger)

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

	return applyMiddleware(mux, deps, logger)
}

// registerAPIRoutes wires all DB-backed API routes into mux.
// Returns nil orgRepo / NoopRecorder when pool is nil (unit-test mode).
func registerAPIRoutes(mux *http.ServeMux, cfg *config.Config, pool *pgxpool.Pool, logger zerolog.Logger) apiDeps {
	if pool == nil {
		return apiDeps{auditRecorder: audit.NoopRecorder{}}
	}

	signer := auth.NewSigner(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenDuration)
	authed := auth.Auth(signer)

	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, signer, cfg.Auth.ArgonPepper, cfg.Auth.RefreshTokenDuration)
	authHandler := auth.NewHandler(authSvc, logger)

	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/v1/auth/me", authed(http.HandlerFunc(authHandler.Me)).ServeHTTP)
	mux.HandleFunc("GET /api/v1/me/orgs", authed(http.HandlerFunc(authHandler.MyOrgs)).ServeHTTP)
	mux.HandleFunc("GET /api/v1/me/permissions", authed(http.HandlerFunc(authHandler.MyPermissions)).ServeHTTP)

	orgRepo := org.NewRepository(pool)
	orgSvc := org.NewService(orgRepo)
	orgHandler := org.NewHandler(orgSvc, logger)

	mux.HandleFunc("GET /api/v1/orgs/{slug}", orgHandler.GetOrg)
	mux.HandleFunc("PUT /api/v1/orgs/{slug}", authed(http.HandlerFunc(orgHandler.UpdateOrg)).ServeHTTP)
	mux.HandleFunc("GET /api/v1/orgs/{slug}/members", authed(http.HandlerFunc(orgHandler.ListMembers)).ServeHTTP)
	mux.HandleFunc("POST /api/v1/orgs/{slug}/members", authed(http.HandlerFunc(orgHandler.AddMember)).ServeHTTP)
	mux.HandleFunc("PUT /api/v1/orgs/{slug}/members/{userID}", authed(http.HandlerFunc(orgHandler.UpdateMemberRole)).ServeHTTP)
	mux.HandleFunc("DELETE /api/v1/orgs/{slug}/members/{userID}", authed(http.HandlerFunc(orgHandler.RemoveMember)).ServeHTTP)

	inviteRepo := org.NewInviteRepository(pool)
	inviteSvc := org.NewInviteService(inviteRepo, orgRepo)
	inviteHandler := org.NewInviteHandler(inviteSvc, logger)
	requireManage := auth.Require(auth.PermManageMembers)
	mux.HandleFunc("POST /api/v1/orgs/{slug}/invites",
		authed(requireManage(http.HandlerFunc(inviteHandler.CreateInvite))).ServeHTTP)
	mux.HandleFunc("POST /api/v1/invites/{token}/accept",
		authed(http.HandlerFunc(inviteHandler.AcceptInvite)).ServeHTTP)

	captureRepo := capture.NewRepository(pool)
	captureSvc := capture.NewService(captureRepo)
	captureHandler := capture.NewHandler(captureSvc, logger)
	capture.RegisterRoutes(mux, captureHandler, authed)

	auditRepo := audit.NewRepository(pool)
	return apiDeps{
		orgRepo:       orgRepo,
		auditRecorder: audit.NewRecorder(auditRepo),
	}
}

// applyMiddleware wraps handler with the full middleware stack.
func applyMiddleware(mux *http.ServeMux, deps apiDeps, logger zerolog.Logger) http.Handler {
	allowedOrigins := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))
	var handler http.Handler = mux
	if deps.orgRepo != nil {
		handler = org.OrgResolver(deps.orgRepo)(handler)
	}
	handler = audit.Middleware(deps.auditRecorder)(handler)
	handler = middleware.CORS(allowedOrigins)(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.IPBlocker(security.NoopRepository{}, logger)(handler)
	return http.MaxBytesHandler(handler, maxBodyBytes)
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
