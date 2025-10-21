package config

import (
	"os"
	"testing"
)

func TestLoad_WithDefaults(t *testing.T) {
	// Clear all environment variables
	clearConfigEnvVars()
	
	// Set only required env var (password has no default)
	os.Setenv("DB_PASSWORD", "testpass")
	defer os.Unsetenv("DB_PASSWORD")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if cfg.Server.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Server.Env != "development" {
		t.Errorf("Expected env development, got %s", cfg.Server.Env)
	}
	if cfg.Database.Host != "host.docker.internal" {
		t.Errorf("Expected host host.docker.internal, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("Expected port 5432, got %s", cfg.Database.Port)
	}
	if cfg.Database.Name != "atlas" {
		t.Errorf("Expected db name atlas, got %s", cfg.Database.Name)
	}
	if cfg.Database.User != "postgres" {
		t.Errorf("Expected user postgres, got %s", cfg.Database.User)
	}
	if cfg.Database.PoolMin != 2 {
		t.Errorf("Expected pool min 2, got %d", cfg.Database.PoolMin)
	}
	if cfg.Database.PoolMax != 10 {
		t.Errorf("Expected pool max 10, got %d", cfg.Database.PoolMax)
	}
	if len(cfg.CORS.Origins) != 2 {
		t.Errorf("Expected 2 CORS origins, got %d", len(cfg.CORS.Origins))
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set all environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("ENV", "production")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_POOL_MIN", "5")
	os.Setenv("DB_POOL_MAX", "20")
	os.Setenv("CORS_ORIGINS", "http://example.com,https://app.example.com")
	defer clearConfigEnvVars()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify all values from environment
	if cfg.Server.Port != "9090" {
		t.Errorf("Expected port 9090, got %s", cfg.Server.Port)
	}
	if cfg.Server.Env != "production" {
		t.Errorf("Expected env production, got %s", cfg.Server.Env)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != "5433" {
		t.Errorf("Expected port 5433, got %s", cfg.Database.Port)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Expected db name testdb, got %s", cfg.Database.Name)
	}
	if cfg.Database.User != "testuser" {
		t.Errorf("Expected user testuser, got %s", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("Expected password testpass, got %s", cfg.Database.Password)
	}
	if cfg.Database.PoolMin != 5 {
		t.Errorf("Expected pool min 5, got %d", cfg.Database.PoolMin)
	}
	if cfg.Database.PoolMax != 20 {
		t.Errorf("Expected pool max 20, got %d", cfg.Database.PoolMax)
	}
	if len(cfg.CORS.Origins) != 2 {
		t.Errorf("Expected 2 CORS origins, got %d", len(cfg.CORS.Origins))
	}
	if cfg.CORS.Origins[0] != "http://example.com" {
		t.Errorf("Expected first origin http://example.com, got %s", cfg.CORS.Origins[0])
	}
}

func TestLoad_MissingPassword(t *testing.T) {
	// Clear all environment variables (password has no default)
	clearConfigEnvVars()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when DB_PASSWORD is missing")
	}
}

func TestValidate_InvalidPoolSizes(t *testing.T) {
	tests := []struct {
		name    string
		poolMin int
		poolMax int
		wantErr bool
	}{
		{
			name:    "negative pool min",
			poolMin: -1,
			poolMax: 10,
			wantErr: true,
		},
		{
			name:    "zero pool max",
			poolMin: 0,
			poolMax: 0,
			wantErr: true,
		},
		{
			name:    "pool min greater than max",
			poolMin: 15,
			poolMax: 10,
			wantErr: true,
		},
		{
			name:    "valid pool sizes",
			poolMin: 2,
			poolMax: 10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: "8080",
					Env:  "development",
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "5432",
					Name:     "atlas",
					User:     "postgres",
					Password: "postgres",
					PoolMin:  tt.poolMin,
					PoolMax:  tt.poolMax,
				},
				CORS: CORSConfig{
					Origins: []string{"http://localhost:3000"},
				},
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "missing port",
			config: &Config{
				Server: ServerConfig{Port: "", Env: "development"},
				Database: DatabaseConfig{
					Host: "localhost", Port: "5432", Name: "atlas",
					User: "postgres", Password: "postgres", PoolMin: 2, PoolMax: 10,
				},
				CORS: CORSConfig{Origins: []string{"http://localhost:3000"}},
			},
		},
		{
			name: "missing db host",
			config: &Config{
				Server: ServerConfig{Port: "8080", Env: "development"},
				Database: DatabaseConfig{
					Host: "", Port: "5432", Name: "atlas",
					User: "postgres", Password: "postgres", PoolMin: 2, PoolMax: 10,
				},
				CORS: CORSConfig{Origins: []string{"http://localhost:3000"}},
			},
		},
		{
			name: "missing db password",
			config: &Config{
				Server: ServerConfig{Port: "8080", Env: "development"},
				Database: DatabaseConfig{
					Host: "localhost", Port: "5432", Name: "atlas",
					User: "postgres", Password: "", PoolMin: 2, PoolMax: 10,
				},
				CORS: CORSConfig{Origins: []string{"http://localhost:3000"}},
			},
		},
		{
			name: "missing CORS origins",
			config: &Config{
				Server: ServerConfig{Port: "8080", Env: "development"},
				Database: DatabaseConfig{
					Host: "localhost", Port: "5432", Name: "atlas",
					User: "postgres", Password: "postgres", PoolMin: 2, PoolMax: 10,
				},
				CORS: CORSConfig{Origins: []string{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Error("Expected validation error but got none")
			}
		})
	}
}

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "single origin",
			input:  "http://localhost:3000",
			expect: []string{"http://localhost:3000"},
		},
		{
			name:   "multiple origins",
			input:  "http://localhost:3000,http://localhost:3001",
			expect: []string{"http://localhost:3000", "http://localhost:3001"},
		},
		{
			name:   "origins with spaces",
			input:  " http://localhost:3000 , http://localhost:3001 ",
			expect: []string{"http://localhost:3000", "http://localhost:3001"},
		},
		{
			name:   "empty string",
			input:  "",
			expect: []string{},
		},
		{
			name:   "only commas",
			input:  ",,,",
			expect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOrigins(tt.input)
			if len(result) != len(tt.expect) {
				t.Errorf("Expected %d origins, got %d", len(tt.expect), len(result))
				return
			}
			for i, origin := range result {
				if origin != tt.expect[i] {
					t.Errorf("Expected origin %s at index %d, got %s", tt.expect[i], i, origin)
				}
			}
		})
	}
}

// Helper function to clear all config-related environment variables
func clearConfigEnvVars() {
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_POOL_MIN")
	os.Unsetenv("DB_POOL_MAX")
	os.Unsetenv("CORS_ORIGINS")
}

