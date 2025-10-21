package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/atlas/api/internal/logger"
)

// Logger creates a middleware that logs HTTP requests using structured logging.
// It captures request details, duration, status code, and any errors.
func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Get request ID from context
		requestID := GetRequestID(c)

		// Create child logger with request ID
		requestLogger := log.WithRequestID(requestID)

		// Store logger in context for handlers to use
		c.Set("logger", requestLogger)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Build log fields
		fields := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status":      c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
			"ip":          c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
		}

		// Add query parameters if present
		if len(c.Request.URL.RawQuery) > 0 {
			fields["query"] = c.Request.URL.RawQuery
		}

		// Log with appropriate level based on status code
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			// Get error if present
			if len(c.Errors) > 0 {
				fields["errors"] = c.Errors.String()
			}
			requestLogger.Error("Request completed with server error", nil, fields)
		case statusCode >= 400:
			if len(c.Errors) > 0 {
				fields["errors"] = c.Errors.String()
			}
			requestLogger.Warn("Request completed with client error", fields)
		default:
			requestLogger.Info("Request completed", fields)
		}
	}
}

// GetLogger retrieves the logger from the Gin context.
// Returns nil if not found.
func GetLogger(c *gin.Context) *logger.Logger {
	if log, exists := c.Get("logger"); exists {
		if logger, ok := log.(*logger.Logger); ok {
			return logger
		}
	}
	return nil
}
