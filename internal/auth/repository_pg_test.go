package auth_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/rian/infinite_brain/internal/auth"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
	"github.com/rian/infinite_brain/pkg/database"
)

// migrationsDir resolves the path to db/migrations/ relative to this test file.
func migrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	// internal/auth/repository_pg_test.go → ../../db/migrations
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}

func applyAuthMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	dir := migrationsDir()
	for _, name := range []string{
		"001_core.sql",
		"002_nodes.sql",
		"003_security.sql",
		"004_sessions.sql",
		"005_multi_tenancy.sql",
		"006_rbac.sql",
	} {
		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return err
		}
	}
	return nil
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"pgvector/pgvector:pg18",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) }) //nolint:errcheck

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	// Phase 1: apply migrations using a plain pool (vector extension must exist
	// before pgvector.RegisterTypes can look up its OID).
	bootstrapPool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("create bootstrap pool: %v", err)
	}
	if err := applyAuthMigrations(ctx, bootstrapPool); err != nil {
		bootstrapPool.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	bootstrapPool.Close()

	// Phase 2: production pool with pgvector type registration.
	pool, err := database.New(ctx, database.DefaultConfig(url))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPgRepository_Register_CreatesUserAndOrg(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	user, err := repo.Register(context.Background(), "alice@example.com", "Alice", "hash123", 1)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", user.Email)
	}
	if user.OrgID == (uuid.UUID{}) {
		t.Error("OrgID should be set")
	}
	if user.Role != "owner" {
		t.Errorf("Role = %q, want owner", user.Role)
	}
	if user.PasswordHash != "hash123" {
		t.Errorf("PasswordHash = %q, want hash123", user.PasswordHash)
	}
}

func TestPgRepository_Register_DuplicateEmailReturnsConflict(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	_, err := repo.Register(context.Background(), "bob@example.com", "Bob", "hash", 1)
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, err = repo.Register(context.Background(), "bob@example.com", "Bob2", "hash2", 1)
	if !errors.Is(err, apperrors.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestPgRepository_FindUserByEmail_ReturnsUser(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	_, err := repo.Register(context.Background(), "carol@example.com", "Carol", "hash", 1)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := repo.FindUserByEmail(context.Background(), "carol@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if user.DisplayName != "Carol" {
		t.Errorf("DisplayName = %q, want Carol", user.DisplayName)
	}
}

func TestPgRepository_FindUserByEmail_NotFoundReturnsError(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	_, err := repo.FindUserByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPgRepository_Session_CreateAndFind(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	user, _ := repo.Register(context.Background(), "dan@example.com", "Dan", "hash", 1)

	session, err := repo.CreateSession(context.Background(), &auth.Session{
		UserID:    user.ID,
		OrgID:     user.OrgID,
		TokenHash: "sha256hashoftoken",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == (uuid.UUID{}) {
		t.Error("session.ID should be set by DB")
	}

	found, err := repo.FindSessionByTokenHash(context.Background(), "sha256hashoftoken")
	if err != nil {
		t.Fatalf("FindSessionByTokenHash: %v", err)
	}
	if found.ID != session.ID {
		t.Errorf("session ID mismatch: got %v, want %v", found.ID, session.ID)
	}
}

func TestPgRepository_DeleteSession_RemovesSession(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	user, _ := repo.Register(context.Background(), "eve@example.com", "Eve", "hash", 1)
	session, _ := repo.CreateSession(context.Background(), &auth.Session{
		UserID:    user.ID,
		OrgID:     user.OrgID,
		TokenHash: "tokentorevoke",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	if err := repo.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	_, err := repo.FindSessionByTokenHash(context.Background(), "tokentorevoke")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestPgRepository_DeleteSessionsByUserID_RemovesAllUserSessions(t *testing.T) {
	pool := setupTestDB(t)
	repo := auth.NewRepository(pool)

	user, _ := repo.Register(context.Background(), "frank@example.com", "Frank", "hash", 1)

	// Create two sessions for the same user
	_, err := repo.CreateSession(context.Background(), &auth.Session{
		UserID:    user.ID,
		OrgID:     user.OrgID,
		TokenHash: "token-session-one",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession 1: %v", err)
	}
	_, err = repo.CreateSession(context.Background(), &auth.Session{
		UserID:    user.ID,
		OrgID:     user.OrgID,
		TokenHash: "token-session-two",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	// Delete all sessions for this user
	if err := repo.DeleteSessionsByUserID(context.Background(), user.ID); err != nil {
		t.Fatalf("DeleteSessionsByUserID: %v", err)
	}

	// Both sessions should be gone
	_, err = repo.FindSessionByTokenHash(context.Background(), "token-session-one")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("session 1: expected ErrNotFound, got %v", err)
	}
	_, err = repo.FindSessionByTokenHash(context.Background(), "token-session-two")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("session 2: expected ErrNotFound, got %v", err)
	}
}
