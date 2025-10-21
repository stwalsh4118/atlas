package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/atlas/api/internal/database"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
)

const (
	// APIVersion is the current version of the API
	APIVersion = "0.1.0"
	// HealthCheckTimeout is the timeout for database health checks
	HealthCheckTimeout = 2 * time.Second
)

// HealthHandler handles health check and readiness endpoints.
type HealthHandler struct {
	db        *database.Database
	startTime time.Time
	env       string
}

// NewHealthHandler creates a new HealthHandler instance.
func NewHealthHandler(db *database.Database, env string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
		env:       env,
	}
}

// HealthResponse represents the basic health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// InfoResponse represents the API information response.
type InfoResponse struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Uptime      string `json:"uptime"`
}

// Health handles GET /health endpoint.
// This is a basic health check that always returns 200 OK.
// It does not check any dependencies and is used for basic liveness checks.
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "healthy",
	})
}

// Ready handles GET /health/ready endpoint.
// This is a readiness check that verifies the database connection is available.
// Returns 200 OK if the database is connected, 503 Service Unavailable otherwise.
func (h *HealthHandler) Ready(c *gin.Context) {
	// Create context with timeout for database ping
	ctx, cancel := context.WithTimeout(c.Request.Context(), HealthCheckTimeout)
	defer cancel()

	// Check database connectivity
	if err := h.db.Ping(ctx); err != nil {
		// Get logger from context (set by logger middleware)
		if log := middleware.GetLogger(c); log != nil {
			log.Error("Database health check failed", err, map[string]interface{}{
				"timeout": HealthCheckTimeout.String(),
			})
		}

		c.JSON(http.StatusServiceUnavailable, ReadyResponse{
			Status:   "not_ready",
			Database: "disconnected",
		})
		return
	}

	c.JSON(http.StatusOK, ReadyResponse{
		Status:   "ready",
		Database: "connected",
	})
}

// Info handles GET /api/v1/info endpoint.
// Returns API metadata including version, environment, and uptime.
func (h *HealthHandler) Info(c *gin.Context) {
	uptime := time.Since(h.startTime)

	c.JSON(http.StatusOK, InfoResponse{
		Version:     APIVersion,
		Environment: h.env,
		Uptime:      formatUptime(uptime),
	})
}

// formatUptime formats a duration into a human-readable string.
func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}
