package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyMigrations executes all SQL migration files in order.
// Used in integration tests and local setup when Atlas CLI is not available.
// Production environments use `atlas schema apply --env local`.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		"001_core.sql",
		"002_nodes.sql",
		"003_security.sql",
		"004_sessions.sql",
		"005_multi_tenancy.sql",
	}

	for _, name := range migrations {
		if err := ApplyMigration(ctx, pool, name); err != nil {
			return err
		}
	}

	return nil
}

// ApplyMigration executes a single named SQL migration file.
// name must be the bare filename (e.g. "001_core.sql"); the directory is
// resolved relative to this source file so it works regardless of working dir.
func ApplyMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	path := filepath.Join(migrationsDir(), name)
	sql, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	return nil
}

// migrationsDir resolves the path to db/migrations/ relative to this file.
// This works regardless of where tests are run from.
func migrationsDir() string {
	// runtime.Caller returns (pc, file, line, ok); we only need file.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	// pkg/database/migrate.go → ../../db/migrations
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}
