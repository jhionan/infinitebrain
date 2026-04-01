package org_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/rian/infinite_brain/internal/org"
	"github.com/rian/infinite_brain/pkg/database"
	apperrors "github.com/rian/infinite_brain/pkg/errors"
)

// mustTestDB spins up a PostgreSQL 18 testcontainer, applies all migrations,
// and returns a pgxpool.Pool with pgvector types registered.
func mustTestDB(t *testing.T) *pgxpool.Pool {
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

	// Phase 1: apply all migrations with a plain pool so the vector extension
	// OID is present before pgvector.RegisterTypes resolves it.
	bootstrapPool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("create bootstrap pool: %v", err)
	}
	if err := database.ApplyMigrations(ctx, bootstrapPool); err != nil {
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

// seedOrg inserts an org directly and returns its ID.
func seedOrg(t *testing.T, pool *pgxpool.Pool, name, slug, plan string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO orgs (name, slug, plan) VALUES ($1, $2, $3) RETURNING id`,
		name, slug, plan,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedOrg %q: %v", slug, err)
	}
	return id
}

// seedUser inserts a user in the given org and returns the user ID.
func seedUser(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (org_id, email, display_name, role) VALUES ($1, $2, $3, 'member') RETURNING id`,
		orgID, email, email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedUser %q: %v", email, err)
	}
	return id
}

// seedOrgUnit inserts a minimal org_unit row and returns its ID.
func seedOrgUnit(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO org_units (org_id, name, unit_type) VALUES ($1, $2, 'unit') RETURNING id`,
		orgID, name,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedOrgUnit %q: %v", name, err)
	}
	return id
}

// ── Tests ──────────────────────────────────────────────────────────────────────

func TestOrgRepository_FindBySlug_ReturnsOrg(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	seedOrg(t, pool, "Acme Corp", "acme-corp", "pro")

	got, err := repo.FindBySlug(ctx, "acme-corp")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Name = %q, want Acme Corp", got.Name)
	}
	if got.Slug != "acme-corp" {
		t.Errorf("Slug = %q, want acme-corp", got.Slug)
	}
}

func TestOrgRepository_FindBySlug_NotFoundReturnsErrNotFound(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)

	_, err := repo.FindBySlug(context.Background(), "does-not-exist")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOrgRepository_FindByID_ReturnsOrg(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	id := seedOrg(t, pool, "Beta Inc", "beta-inc", "teams")

	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
	if got.Plan != "teams" {
		t.Errorf("Plan = %q, want teams", got.Plan)
	}
}

func TestOrgRepository_Update_ChangesNameAndSettings(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	id := seedOrg(t, pool, "Old Name", "old-slug", "pro")

	updated, err := repo.Update(ctx, id, "New Name", org.OrgSettings{
		AIProvider:        "claude",
		DataRetentionDays: 90,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want New Name", updated.Name)
	}
	if updated.Settings.AIProvider != "claude" {
		t.Errorf("Settings.AIProvider = %q, want claude", updated.Settings.AIProvider)
	}
	if updated.Settings.DataRetentionDays != 90 {
		t.Errorf("Settings.DataRetentionDays = %d, want 90", updated.Settings.DataRetentionDays)
	}
}

func TestOrgRepository_AddMember_ListMembersReturnsCorrectRole(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	orgID := seedOrg(t, pool, "Test Org", "test-org", "teams")
	userID := seedUser(t, pool, orgID, "alice@example.com")

	if err := repo.AddMember(ctx, orgID, userID, "editor", nil); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	members, err := repo.ListMembers(ctx, orgID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("ListMembers returned %d members, want 1", len(members))
	}
	if members[0].UserID != userID {
		t.Errorf("UserID = %v, want %v", members[0].UserID, userID)
	}
	if members[0].Role != "editor" {
		t.Errorf("Role = %q, want editor", members[0].Role)
	}
	if members[0].Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", members[0].Email)
	}
}

func TestOrgRepository_UpdateMemberRole_ChangesRole(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	orgID := seedOrg(t, pool, "Role Org", "role-org", "pro")
	userID := seedUser(t, pool, orgID, "bob@example.com")

	if err := repo.AddMember(ctx, orgID, userID, "viewer", nil); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	if err := repo.UpdateMemberRole(ctx, orgID, userID, "admin"); err != nil {
		t.Fatalf("UpdateMemberRole: %v", err)
	}

	member, err := repo.FindMember(ctx, orgID, userID)
	if err != nil {
		t.Fatalf("FindMember: %v", err)
	}
	if member.Role != "admin" {
		t.Errorf("Role = %q, want admin", member.Role)
	}
}

func TestOrgRepository_RemoveMember_MembershipGone(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	orgID := seedOrg(t, pool, "Remove Org", "remove-org", "pro")
	userID := seedUser(t, pool, orgID, "carol@example.com")

	if err := repo.AddMember(ctx, orgID, userID, "member", nil); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if err := repo.RemoveMember(ctx, orgID, userID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	_, err := repo.FindMember(ctx, orgID, userID)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound after removal, got %v", err)
	}
}

func TestOrgRepository_CountMembers_ReturnsCorrectCount(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	orgID := seedOrg(t, pool, "Count Org", "count-org", "teams")
	userA := seedUser(t, pool, orgID, "user-a@example.com")
	userB := seedUser(t, pool, orgID, "user-b@example.com")

	if err := repo.AddMember(ctx, orgID, userA, "editor", nil); err != nil {
		t.Fatalf("AddMember A: %v", err)
	}
	if err := repo.AddMember(ctx, orgID, userB, "viewer", nil); err != nil {
		t.Fatalf("AddMember B: %v", err)
	}

	count, err := repo.CountMembers(ctx, orgID)
	if err != nil {
		t.Fatalf("CountMembers: %v", err)
	}
	if count != 2 {
		t.Errorf("CountMembers = %d, want 2", count)
	}
}

func TestOrgRepository_SoftDelete_OrgDisappears(t *testing.T) {
	pool := mustTestDB(t)
	repo := org.NewRepository(pool)
	ctx := context.Background()

	id := seedOrg(t, pool, "Deleted Org", "deleted-org", "pro")

	if err := repo.SoftDelete(ctx, id); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err := repo.FindByID(ctx, id)
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}
}

// TestOrgRepository_RLS_CrossOrgAccessBlocked verifies that RLS on the nodes
// table prevents org B from reading nodes inserted by org A.
//
// The test superuser role is demoted to a non-superuser app role for this test
// so that FORCE ROW LEVEL SECURITY is respected — PostgreSQL superusers bypass
// RLS regardless of the FORCE flag.
func TestOrgRepository_RLS_CrossOrgAccessBlocked(t *testing.T) {
	pool := mustTestDB(t)
	ctx := context.Background()

	// Create a non-superuser application role and grant it table access.
	// Superusers bypass RLS; app connections must use a non-privileged role.
	_, err := pool.Exec(ctx, `
		DO $$
		BEGIN
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
		    CREATE ROLE app_user LOGIN PASSWORD 'app_password' NOINHERIT;
		  END IF;
		END
		$$;
		GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
		GRANT USAGE ON SCHEMA public TO app_user;
	`)
	if err != nil {
		t.Fatalf("create app_user role: %v", err)
	}

	// Build a separate pool that connects as the non-superuser app role.
	// Derive the DSN from the existing pool config and replace the user.
	cfg := pool.Config().Copy()
	cfg.ConnConfig.User = "app_user"
	cfg.ConnConfig.Password = "app_password"
	// Use a small pool — this is test-only.
	cfg.MaxConns = 5
	cfg.MinConns = 1

	appPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create app pool: %v", err)
	}
	t.Cleanup(appPool.Close)

	// Set up two separate orgs and one user each.
	orgA := seedOrg(t, pool, "Org A", "org-a-rls", "pro")
	orgB := seedOrg(t, pool, "Org B", "org-b-rls", "pro")
	userA := seedUser(t, pool, orgA, "user-a@orga.com")
	unitA := seedOrgUnit(t, pool, orgA, "default-a")

	// Insert a node for org A via the non-privileged pool so RLS applies.
	err = database.WithOrgContext(ctx, appPool, orgA, func(conn *pgxpool.Conn) error {
		_, insertErr := conn.Exec(ctx,
			`INSERT INTO nodes (org_id, user_id, unit_id, type, title)
			 VALUES ($1, $2, $3, 'note', 'Secret Note')`,
			orgA, userA, unitA,
		)
		return insertErr
	})
	if err != nil {
		t.Fatalf("insert node for org A: %v", err)
	}

	// Query nodes as org B — RLS must return zero rows.
	var count int
	err = database.WithOrgContext(ctx, appPool, orgB, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&count)
	})
	if err != nil {
		t.Fatalf("count nodes as org B: %v", err)
	}
	if count != 0 {
		t.Errorf("RLS isolation failed: org B can see %d node(s) belonging to org A", count)
	}

	// Confirm org A can still see its own node.
	err = database.WithOrgContext(ctx, appPool, orgA, func(conn *pgxpool.Conn) error {
		return conn.QueryRow(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&count)
	})
	if err != nil {
		t.Fatalf("count nodes as org A: %v", err)
	}
	if count != 1 {
		t.Errorf("org A should see 1 node, got %d", count)
	}
}
