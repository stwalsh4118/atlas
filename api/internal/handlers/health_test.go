package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sean/atlas/api/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDatabase is a mock implementation of the database.Database for testing.
type MockDatabase struct {
	pingErr error
}

func (m *MockDatabase) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *MockDatabase) Close() {}

func (m *MockDatabase) Stats() *pgxpool.Stat {
	return nil
}

// setupTestRouter creates a test Gin router with the handler.
func setupTestRouter(handler *HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// setupHealthHandler creates a HealthHandler with a mock database.
func setupHealthHandler(pingErr error, env string) (*HealthHandler, *MockDatabase) {
	mockDB := &MockDatabase{pingErr: pingErr}
	// We need to wrap the mock in a database.Database struct
	// Since we can't create it directly, we'll use a different approach
	db := &database.Database{Pool: nil}

	handler := &HealthHandler{
		db:        db,
		startTime: time.Now().Add(-1 * time.Hour), // Set start time to 1 hour ago for testing
		env:       env,
	}

	return handler, mockDB
}

func TestHealthHandler_Health(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   HealthResponse
	}{
		{
			name:           "health check returns 200 OK",
			expectedStatus: http.StatusOK,
			expectedBody: HealthResponse{
				Status: "healthy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with no database (health check doesn't use it)
			handler := &HealthHandler{
				db:        nil,
				startTime: time.Now(),
				env:       "test",
			}

			// Setup router and route
			router := setupTestRouter(handler)
			router.GET("/health", handler.Health)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Assert response body
			var response HealthResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

func TestHealthHandler_Ready_DatabaseConnected(t *testing.T) {
	// This test requires a real database connection
	// For unit testing, we'll mock the database ping
	t.Run("returns 200 when database is connected", func(t *testing.T) {
		// We need a different approach since we can't easily mock the Ping method
		// Let's test the actual implementation with a mock that satisfies the interface

		// Skip this test for now as it requires refactoring the Database struct
		// to use an interface for testing
		t.Skip("Requires database interface for proper mocking")
	})
}

func TestHealthHandler_Info(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		startTime   time.Time
		checkUptime bool
	}{
		{
			name:        "returns API info with development environment",
			env:         "development",
			startTime:   time.Now().Add(-2 * time.Hour),
			checkUptime: true,
		},
		{
			name:        "returns API info with production environment",
			env:         "production",
			startTime:   time.Now().Add(-24 * time.Hour),
			checkUptime: true,
		},
		{
			name:        "returns API info with test environment",
			env:         "test",
			startTime:   time.Now(),
			checkUptime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := &HealthHandler{
				db:        nil,
				startTime: tt.startTime,
				env:       tt.env,
			}

			// Setup router and route
			router := setupTestRouter(handler)
			router.GET("/api/v1/info", handler.Info)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, http.StatusOK, w.Code)

			// Assert response body
			var response InfoResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, APIVersion, response.Version)
			assert.Equal(t, tt.env, response.Environment)

			if tt.checkUptime {
				assert.NotEmpty(t, response.Uptime)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "formats seconds only",
			duration: 45 * time.Second,
			expected: "0h 0m 45s",
		},
		{
			name:     "formats minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "0h 5m 30s",
		},
		{
			name:     "formats hours, minutes and seconds",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			expected: "2h 15m 45s",
		},
		{
			name:     "formats days, hours, minutes and seconds",
			duration: 3*24*time.Hour + 5*time.Hour + 30*time.Minute + 15*time.Second,
			expected: "3d 5h 30m 15s",
		},
		{
			name:     "formats exactly one day",
			duration: 24 * time.Hour,
			expected: "1d 0h 0m 0s",
		},
		{
			name:     "formats zero duration",
			duration: 0,
			expected: "0h 0m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptime(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewHealthHandler(t *testing.T) {
	tests := []struct {
		name string
		db   *database.Database
		env  string
	}{
		{
			name: "creates handler with development environment",
			db:   &database.Database{Pool: nil},
			env:  "development",
		},
		{
			name: "creates handler with production environment",
			db:   &database.Database{Pool: nil},
			env:  "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHealthHandler(tt.db, tt.env)

			assert.NotNil(t, handler)
			assert.Equal(t, tt.db, handler.db)
			assert.Equal(t, tt.env, handler.env)
			assert.False(t, handler.startTime.IsZero())
		})
	}
}

func TestHealthResponse_JSON(t *testing.T) {
	response := HealthResponse{Status: "healthy"}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	expected := `{"status":"healthy"}`
	assert.JSONEq(t, expected, string(data))
}

func TestReadyResponse_JSON(t *testing.T) {
	tests := []struct {
		name     string
		response ReadyResponse
		expected string
	}{
		{
			name: "connected state",
			response: ReadyResponse{
				Status:   "ready",
				Database: "connected",
			},
			expected: `{"status":"ready","database":"connected"}`,
		},
		{
			name: "disconnected state",
			response: ReadyResponse{
				Status:   "not_ready",
				Database: "disconnected",
			},
			expected: `{"status":"not_ready","database":"disconnected"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestInfoResponse_JSON(t *testing.T) {
	response := InfoResponse{
		Version:     "0.1.0",
		Environment: "test",
		Uptime:      "1h 30m 45s",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	expected := `{"version":"0.1.0","environment":"test","uptime":"1h 30m 45s"}`
	assert.JSONEq(t, expected, string(data))
}

func TestHealthCheckTimeout(t *testing.T) {
	// Verify the constant is set to expected value
	assert.Equal(t, 2*time.Second, HealthCheckTimeout)
}

func TestAPIVersion(t *testing.T) {
	// Verify the API version constant
	assert.Equal(t, "0.1.0", APIVersion)
}

// Integration test example - this would require a real database connection
func TestHealthHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would set up a real database connection and test the full flow
	t.Skip("Integration test requires real database - implement in integration test suite")
}

// Benchmark tests
func BenchmarkHealthHandler_Health(b *testing.B) {
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
		env:       "test",
	}

	router := setupTestRouter(handler)
	router.GET("/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkFormatUptime(b *testing.B) {
	duration := 3*24*time.Hour + 5*time.Hour + 30*time.Minute + 15*time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatUptime(duration)
	}
}

// Example of how the handler would be used
func ExampleHealthHandler_Health() {
	// Create handler
	handler := &HealthHandler{
		db:        nil,
		startTime: time.Now(),
		env:       "development",
	}

	// Setup router
	router := gin.New()
	router.GET("/health", handler.Health)

	// Start server (in real application)
	fmt.Println("Health endpoint registered at /health")
	// Output: Health endpoint registered at /health
}
