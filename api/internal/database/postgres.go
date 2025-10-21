package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sean/atlas/api/internal/config"
)

// Database wraps the pgx connection pool and provides database operations.
type Database struct {
	Pool *pgxpool.Pool
}

// NewPostgresPool creates a new PostgreSQL connection pool using pgx.
// It configures the pool based on the provided database configuration,
// tests the connection, and returns a Database instance.
func NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	// Build connection string (DSN)
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	// Parse connection string and create pool config
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool settings
	poolConfig.MinConns = int32(cfg.PoolMin)
	poolConfig.MaxConns = int32(cfg.PoolMax)

	// Set connection timeouts
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	poolConfig.MaxConnIdleTime = 30 * time.Second
	poolConfig.MaxConnLifetime = 1 * time.Hour

	// Health check period (how often to check idle connections)
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection immediately
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Ping checks if the database connection is alive.
// It returns an error if the connection is not available.
func (db *Database) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close gracefully closes the database connection pool.
// It waits for all connections to be returned to the pool before closing.
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Stats returns statistics about the connection pool.
// This is useful for monitoring and debugging.
func (db *Database) Stats() *pgxpool.Stat {
	if db.Pool == nil {
		return nil
	}
	return db.Pool.Stat()
}
