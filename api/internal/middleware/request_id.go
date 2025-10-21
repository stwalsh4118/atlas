package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDKey is the context key for the request ID
	RequestIDKey = "request_id"
	// RequestIDHeader is the HTTP header name for the request ID
	RequestIDHeader = "X-Request-ID"
)

// RequestID generates a unique request ID for each request and adds it to the context and response headers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header (from upstream proxy)
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new UUID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in Gin context for access by other middleware and handlers
		c.Set(RequestIDKey, requestID)

		// Add to response headers
		c.Writer.Header().Set(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the Gin context.
// Returns an empty string if not found.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
