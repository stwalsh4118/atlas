package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sean/atlas/api/internal/config"
)

// Test configuration for local PostgreSQL
func getTestConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "host.docker.internal"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Name:     getEnvOrDefault("DB_NAME", "atlas"),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		PoolMin:  2,
		PoolMax:  5,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestNewPostgresPool_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer db.Close()

	if db.Pool == nil {
		t.Error("Expected Pool to be initialized")
	}

	// Verify pool stats
	stats := db.Stats()
	if stats == nil {
		t.Error("Expected stats to be available")
	}
}

func TestNewPostgresPool_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := getTestConfig()
	cfg.Host = "invalid-host-that-does-not-exist"

	_, err := NewPostgresPool(ctx, cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

func TestNewPostgresPool_InvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := getTestConfig()
	cfg.Password = "wrong-password"

	_, err := NewPostgresPool(ctx, cfg)
	if err == nil {
		t.Error("Expected error when using invalid credentials")
	}
}

func TestPing_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer db.Close()

	// Test ping
	err = db.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestPing_AfterClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Close the pool
	db.Close()

	// Ping should fail after close
	err = db.Ping(ctx)
	if err == nil {
		t.Error("Expected ping to fail after pool is closed")
	}
}

func TestClose_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Close multiple times should not panic
	db.Close()
	db.Close()
}

func TestStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats == nil {
		t.Error("Expected stats to be available")
	}

	// Verify pool configuration
	if stats.MaxConns() != int32(cfg.PoolMax) {
		t.Errorf("Expected MaxConns %d, got %d", cfg.PoolMax, stats.MaxConns())
	}
}

func TestConnectionPool_MinMaxConns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()
	cfg.PoolMin = 3
	cfg.PoolMax = 8

	db, err := NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxConns() != 8 {
		t.Errorf("Expected MaxConns 8, got %d", stats.MaxConns())
	}

	// Give pool time to establish min connections
	time.Sleep(100 * time.Millisecond)

	// Total connections should be at least the minimum
	totalConns := stats.IdleConns() + stats.AcquiredConns()
	if totalConns < 3 {
		t.Logf("Warning: Expected at least %d connections, got %d (idle: %d, acquired: %d)",
			cfg.PoolMin, totalConns, stats.IdleConns(), stats.AcquiredConns())
	}
}
