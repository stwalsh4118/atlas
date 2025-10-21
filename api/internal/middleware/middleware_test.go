package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sean/atlas/api/internal/logger"
)

func init() {
	// Set Gin to test mode to reduce noise in tests
	gin.SetMode(gin.TestMode)
}

// TestRequestID tests the RequestID middleware
func TestRequestID(t *testing.T) {
	t.Run("generates new request ID", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			requestID := GetRequestID(c)
			if requestID == "" {
				t.Error("Expected request ID to be set")
			}
			c.String(200, requestID)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response header
		headerID := w.Header().Get(RequestIDHeader)
		if headerID == "" {
			t.Error("Expected X-Request-ID header to be set")
		}

		// Check response body contains the ID
		if w.Body.String() != headerID {
			t.Errorf("Expected body to contain request ID %s, got %s", headerID, w.Body.String())
		}
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			c.String(200, GetRequestID(c))
		})

		existingID := "existing-request-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RequestIDHeader, existingID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Body.String() != existingID {
			t.Errorf("Expected request ID %s, got %s", existingID, w.Body.String())
		}
	})

	t.Run("GetRequestID returns empty string if not set", func(t *testing.T) {
		c := &gin.Context{}
		requestID := GetRequestID(c)
		if requestID != "" {
			t.Errorf("Expected empty string, got %s", requestID)
		}
	})
}

// TestCORS tests the CORS middleware
func TestCORS(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:3001"}

	t.Run("allows request from allowed origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(allowedOrigins))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CORS headers (gin-contrib/cors sets these on regular requests)
		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Error("Expected Access-Control-Allow-Origin header to be set")
		}
		// Credentials header should be set
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Expected Access-Control-Allow-Credentials header to be set")
		}
	})

	t.Run("does not set CORS headers for disallowed origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(allowedOrigins))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Expected no CORS headers for disallowed origin")
		}
	})

	t.Run("handles OPTIONS preflight for allowed origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(allowedOrigins))
		router.OPTIONS("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 204 {
			t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("rejects OPTIONS preflight for disallowed origin", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS(allowedOrigins))
		router.OPTIONS("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 403 {
			t.Errorf("Expected status 403 for disallowed OPTIONS, got %d", w.Code)
		}
	})
}

// TestLogger tests the Logger middleware
func TestLogger(t *testing.T) {
	t.Run("logs successful request", func(t *testing.T) {
		log := logger.New("test")
		router := gin.New()
		router.Use(RequestID())
		router.Use(Logger(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("logs request with query parameters", func(t *testing.T) {
		log := logger.New("test")
		router := gin.New()
		router.Use(RequestID())
		router.Use(Logger(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test?foo=bar&baz=qux", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetLogger retrieves logger from context", func(t *testing.T) {
		log := logger.New("test")
		router := gin.New()
		router.Use(RequestID())
		router.Use(Logger(log))
		router.GET("/test", func(c *gin.Context) {
			contextLogger := GetLogger(c)
			if contextLogger == nil {
				t.Error("Expected logger to be in context")
			}
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	})

	t.Run("GetLogger returns nil if not set", func(t *testing.T) {
		c := &gin.Context{}
		log := GetLogger(c)
		if log != nil {
			t.Error("Expected nil logger")
		}
	})
}

// TestRecovery tests the Recovery middleware
func TestRecovery(t *testing.T) {
	t.Run("recovers from panic and returns 500", func(t *testing.T) {
		log := logger.New("test")
		router := gin.New()
		router.Use(RequestID())
		router.Use(Recovery(log))
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 500 {
			t.Errorf("Expected status 500 after panic, got %d", w.Code)
		}

		// Check response contains error
		body := w.Body.String()
		if !strings.Contains(body, "INTERNAL_SERVER_ERROR") {
			t.Error("Expected error response to contain INTERNAL_SERVER_ERROR")
		}
		if !strings.Contains(body, "request_id") {
			t.Error("Expected error response to contain request_id")
		}
	})

	t.Run("does not interfere with normal requests", func(t *testing.T) {
		log := logger.New("test")
		router := gin.New()
		router.Use(Recovery(log))
		router.GET("/normal", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req := httptest.NewRequest("GET", "/normal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "OK" {
			t.Errorf("Expected body 'OK', got %s", w.Body.String())
		}
	})
}

// TestMiddlewareStack tests that all middleware work together
func TestMiddlewareStack(t *testing.T) {
	log := logger.New("test")
	allowedOrigins := []string{"http://localhost:3000"}

	router := gin.New()
	router.Use(RequestID())
	router.Use(Logger(log))
	router.Use(Recovery(log))
	router.Use(CORS(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		// Verify all middleware added their data
		requestID := GetRequestID(c)
		if requestID == "" {
			t.Error("Expected request ID from RequestID middleware")
		}

		contextLogger := GetLogger(c)
		if contextLogger == nil {
			t.Error("Expected logger from Logger middleware")
		}

		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check all middleware effects
	if w.Header().Get(RequestIDHeader) == "" {
		t.Error("Expected X-Request-ID header")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Expected CORS headers")
	}
}
