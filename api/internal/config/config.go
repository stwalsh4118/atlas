package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	CORS     CORSConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
	Env  string
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	PoolMin  int
	PoolMax  int
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	Origins []string
}

// Load reads configuration from environment variables.
// It uses viper to read values and provides sensible defaults for development.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults for development
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENV", "development")
	v.SetDefault("DB_HOST", "host.docker.internal")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_NAME", "atlas")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_POOL_MIN", 2)
	v.SetDefault("DB_POOL_MAX", 10)
	v.SetDefault("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")

	// Bind environment variables
	v.AutomaticEnv()

	// Build configuration
	cfg := &Config{
		Server: ServerConfig{
			Port: v.GetString("PORT"),
			Env:  v.GetString("ENV"),
		},
		Database: DatabaseConfig{
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetString("DB_PORT"),
			Name:     v.GetString("DB_NAME"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
			PoolMin:  v.GetInt("DB_POOL_MIN"),
			PoolMax:  v.GetInt("DB_POOL_MAX"),
		},
		CORS: CORSConfig{
			Origins: parseOrigins(v.GetString("CORS_ORIGINS")),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that required configuration is present and valid.
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.Port == "" {
		return fmt.Errorf("DB_PORT is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.PoolMin < 0 {
		return fmt.Errorf("DB_POOL_MIN must be non-negative")
	}
	if c.Database.PoolMax < 1 {
		return fmt.Errorf("DB_POOL_MAX must be at least 1")
	}
	if c.Database.PoolMin > c.Database.PoolMax {
		return fmt.Errorf("DB_POOL_MIN must be less than or equal to DB_POOL_MAX")
	}

	// Validate CORS config
	if len(c.CORS.Origins) == 0 {
		return fmt.Errorf("CORS_ORIGINS is required")
	}

	return nil
}

// parseOrigins splits a comma-separated string of origins into a slice.
func parseOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}

	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
