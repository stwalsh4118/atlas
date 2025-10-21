package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/atlas/api/internal/logger"
)

// Recovery creates a middleware that recovers from panics and logs them.
// It returns a 500 Internal Server Error response instead of crashing.
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Get request ID if available
				requestID := GetRequestID(c)

				// Get logger from context or use provided logger
				requestLogger := GetLogger(c)
				if requestLogger == nil {
					requestLogger = log
				}

				// Log the panic with full details
				requestLogger.Error(
					"Panic recovered",
					fmt.Errorf("panic: %v", err),
					map[string]interface{}{
						"request_id": requestID,
						"method":     c.Request.Method,
						"path":       c.Request.URL.Path,
						"stack":      string(stack),
					},
				)

				// Return 500 error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       "INTERNAL_SERVER_ERROR",
						"message":    "An unexpected error occurred",
						"request_id": requestID,
					},
				})

				// Abort further processing
				c.Abort()
			}
		}()

		c.Next()
	}
}
