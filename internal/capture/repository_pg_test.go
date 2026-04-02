package capture_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/rian/infinite_brain/internal/auth"
	"github.com/rian/infinite_brain/internal/capture"
	"github.com/rian/infinite_brain/pkg/database"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

func migrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
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
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg18",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) }) //nolint:errcheck

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	bootstrap, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("bootstrap pool: %v", err)
	}
	if err := applyMigrations(ctx, bootstrap); err != nil {
		bootstrap.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	bootstrap.Close()

	pool, err := database.New(ctx, database.DefaultConfig(url))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedUser registers a user+org+root_unit and returns their IDs.
func seedUser(t *testing.T, pool *pgxpool.Pool, email string) (orgID, userID uuid.UUID) {
	t.Helper()
	repo := auth.NewRepository(pool)
	u, err := repo.Register(context.Background(), email, "Test User", "hash", 1)
	if err != nil {
		t.Fatalf("seedUser Register: %v", err)
	}
	return u.OrgID, u.ID
}

func TestNoteRepository_Create_Succeeds(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "create@example.com")

	note, err := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Hello, Infinite Brain.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if note.ID == (uuid.UUID{}) {
		t.Error("expected non-zero ID")
	}
	if note.Content != "Hello, Infinite Brain." {
		t.Errorf("Content = %q, want %q", note.Content, "Hello, Infinite Brain.")
	}
	if note.Status != capture.StatusInbox {
		t.Errorf("Status = %q, want inbox", note.Status)
	}
	if note.Source != capture.SourceManual {
		t.Errorf("Source = %q, want manual", note.Source)
	}
}

func TestNoteRepository_Create_EmptyContentFails(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "emptycontent@example.com")

	_, err := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "",
	})
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestNoteRepository_FindByID_ReturnsNote(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "findbyid@example.com")

	created, err := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Find me.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.FindByID(context.Background(), orgID, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID mismatch: got %v, want %v", found.ID, created.ID)
	}
}

func TestNoteRepository_FindByID_WrongOrgReturnsNotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "wrongorg@example.com")

	created, _ := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Secret note.",
		Source:  capture.SourceManual,
	})

	otherOrgID := uuid.New()
	_, err := repo.FindByID(context.Background(), otherOrgID, created.ID)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound for wrong org, got %v", err)
	}
}

func TestNoteRepository_List_ReturnsPaginated(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "list@example.com")

	for i := range 3 {
		_, err := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
			Content: fmt.Sprintf("Note %d", i),
			Source:  capture.SourceManual,
		})
		if err != nil {
			t.Fatalf("Create note %d: %v", i, err)
		}
	}

	result, err := repo.List(context.Background(), orgID, userID, 1, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Notes) != 2 {
		t.Errorf("got %d notes, want 2", len(result.Notes))
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
}

func TestNoteRepository_Update_ChangesFields(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "update@example.com")

	note, _ := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Original.",
		Source:  capture.SourceManual,
	})

	updated, err := repo.Update(context.Background(), orgID, note.ID, capture.UpdateNoteInput{
		Title:   "New Title",
		Content: "Updated content.",
		Tags:    []string{"go", "test"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "New Title" {
		t.Errorf("Title = %q, want New Title", updated.Title)
	}
	if updated.Content != "Updated content." {
		t.Errorf("Content = %q, want Updated content.", updated.Content)
	}
}

func TestNoteRepository_Delete_SoftDeletesNote(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "delete@example.com")

	note, _ := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Delete me.",
		Source:  capture.SourceManual,
	})

	if err := repo.Delete(context.Background(), orgID, note.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID(context.Background(), orgID, note.ID)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestNoteRepository_Archive_SetsArchivedAt(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "archive@example.com")

	note, _ := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Archive me.",
		Source:  capture.SourceManual,
	})

	archived, err := repo.Archive(context.Background(), orgID, note.ID)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set")
	}
	if archived.Status != capture.StatusArchived {
		t.Errorf("Status = %q, want archived", archived.Status)
	}
}

func TestNoteRepository_Archive_ExcludesFromListInbox(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "archiveinbox@example.com")

	note, err := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Should leave inbox on archive.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Confirm it appears in inbox before archiving.
	before, err := repo.ListInbox(context.Background(), orgID, userID, 1, 10)
	if err != nil {
		t.Fatalf("ListInbox before archive: %v", err)
	}
	if before.Total == 0 {
		t.Fatal("expected note in inbox before archive")
	}

	if _, err := repo.Archive(context.Background(), orgID, note.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	// Confirm it is gone from inbox after archiving.
	after, err := repo.ListInbox(context.Background(), orgID, userID, 1, 10)
	if err != nil {
		t.Fatalf("ListInbox after archive: %v", err)
	}
	if after.Total != 0 {
		t.Errorf("expected 0 inbox notes after archive, got %d", after.Total)
	}
}

func TestNoteRepository_List_ExcludesOtherOrgNotes(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)

	orgA, userA := seedUser(t, pool, "orga@example.com")
	orgB, userB := seedUser(t, pool, "orgb@example.com")

	_, err := repo.Create(context.Background(), orgA, userA, capture.CreateNoteInput{
		Content: "Org A note.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create org A note: %v", err)
	}

	_, err = repo.Create(context.Background(), orgB, userB, capture.CreateNoteInput{
		Content: "Org B note.",
		Source:  capture.SourceManual,
	})
	if err != nil {
		t.Fatalf("Create org B note: %v", err)
	}

	resultA, err := repo.List(context.Background(), orgA, userA, 1, 10)
	if err != nil {
		t.Fatalf("List org A: %v", err)
	}
	if resultA.Total != 1 {
		t.Errorf("org A: expected 1 note, got %d", resultA.Total)
	}

	resultB, err := repo.List(context.Background(), orgB, userB, 1, 10)
	if err != nil {
		t.Fatalf("List org B: %v", err)
	}
	if resultB.Total != 1 {
		t.Errorf("org B: expected 1 note, got %d", resultB.Total)
	}
}

func TestNoteRepository_Delete_PreservesRowWithDeletedAt(t *testing.T) {
	pool := setupTestDB(t)
	repo := capture.NewRepository(pool)
	orgID, userID := seedUser(t, pool, "softdelete@example.com")

	note, _ := repo.Create(context.Background(), orgID, userID, capture.CreateNoteInput{
		Content: "Should be soft-deleted.",
		Source:  capture.SourceManual,
	})

	if err := repo.Delete(context.Background(), orgID, note.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Row should still exist with deleted_at set.
	var deletedAt *time.Time
	err := pool.QueryRow(context.Background(),
		`SELECT deleted_at FROM nodes WHERE id = $1`,
		note.ID,
	).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("query deleted node: %v", err)
	}
	if deletedAt == nil {
		t.Error("expected deleted_at to be set after soft-delete")
	}
}
