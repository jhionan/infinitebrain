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
	}

	dir := migrationsDir()

	for _, name := range migrations {
		path := filepath.Join(dir, name)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
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
