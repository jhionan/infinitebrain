// Package database provides the PostgreSQL connection pool.
// All other packages receive a *pgxpool.Pool via constructor injection.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go/pgx"
)

// Config holds connection pool parameters.
// Values are loaded from environment variables via pkg/config.
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig(dsn string) Config {
	return Config{
		DSN:             dsn,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// New creates and validates a pgxpool connection pool.
// pgvector types are registered on every new connection so VECTOR columns
// can be read and written as []float32 values without manual scanning.
func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Register pgvector OID types on each connection so VECTOR(1536) columns
	// map to []float32 without manual type conversion.
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := pgvector.RegisterTypes(ctx, conn); err != nil {
			return fmt.Errorf("register pgvector types: %w", err)
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
