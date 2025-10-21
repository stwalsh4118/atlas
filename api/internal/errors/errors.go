package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
)

// Error code constants for standardized error responses
const (
	ErrNotFound           = "NOT_FOUND"
	ErrBadRequest         = "BAD_REQUEST"
	ErrInternalServer     = "INTERNAL_SERVER_ERROR"
	ErrValidation         = "VALIDATION_ERROR"
	ErrDatabaseConnection = "DATABASE_CONNECTION_ERROR"
)

// ErrorResponse is the top-level error response structure.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error information.
type ErrorDetail struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// NotFound returns a 404 Not Found error response.
// It logs a warning and sends a JSON response with the error details.
func NotFound(c *gin.Context, message string) {
	log := middleware.GetLogger(c)
	requestID := middleware.GetRequestID(c)

	if log != nil {
		log.Warn("Resource not found", map[string]interface{}{
			"message":    message,
			"request_id": requestID,
			"path":       c.Request.URL.Path,
		})
	}

	c.JSON(http.StatusNotFound, ErrorResponse{
		Error: ErrorDetail{
			Code:      ErrNotFound,
			Message:   message,
			RequestID: requestID,
		},
	})
}

// BadRequest returns a 400 Bad Request error response with optional details.
// It logs a warning and sends a JSON response with the error details.
func BadRequest(c *gin.Context, message string, details map[string]interface{}) {
	log := middleware.GetLogger(c)
	requestID := middleware.GetRequestID(c)

	logFields := map[string]interface{}{
		"message":    message,
		"request_id": requestID,
		"path":       c.Request.URL.Path,
	}
	if details != nil {
		logFields["details"] = details
	}

	if log != nil {
		log.Warn("Bad request", logFields)
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorDetail{
			Code:      ErrBadRequest,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
	})
}

// InternalServerError returns a 500 Internal Server Error response.
// It logs the error with full context and sends a generic error message to the client.
// The actual error details are not exposed to the client for security reasons.
func InternalServerError(c *gin.Context, message string, err error) {
	log := middleware.GetLogger(c)
	requestID := middleware.GetRequestID(c)

	logFields := map[string]interface{}{
		"message":    message,
		"request_id": requestID,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
	}

	if log != nil {
		log.Error("Internal server error", err, logFields)
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorDetail{
			Code:      ErrInternalServer,
			Message:   message,
			RequestID: requestID,
		},
	})
}

// ValidationError returns a 400 Bad Request error response with field-specific validation errors.
// It parses the validation errors from the validator library and formats them for the client.
func ValidationError(c *gin.Context, validationErrors validator.ValidationErrors) {
	log := middleware.GetLogger(c)
	requestID := middleware.GetRequestID(c)

	// Convert validation errors to a map of field -> error message
	details := make(map[string]interface{})
	for _, err := range validationErrors {
		field := err.Field()
		details[field] = formatValidationError(err)
	}

	if log != nil {
		log.Warn("Validation error", map[string]interface{}{
			"request_id": requestID,
			"path":       c.Request.URL.Path,
			"fields":     details,
		})
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorDetail{
			Code:      ErrValidation,
			Message:   "Validation failed for one or more fields",
			Details:   details,
			RequestID: requestID,
		},
	})
}

// formatValidationError converts a validator.FieldError to a human-readable message.
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Value is too short or small (minimum: " + err.Param() + ")"
	case "max":
		return "Value is too long or large (maximum: " + err.Param() + ")"
	case "len":
		return "Must have length of " + err.Param()
	case "gt":
		return "Must be greater than " + err.Param()
	case "gte":
		return "Must be greater than or equal to " + err.Param()
	case "lt":
		return "Must be less than " + err.Param()
	case "lte":
		return "Must be less than or equal to " + err.Param()
	case "oneof":
		return "Must be one of: " + err.Param()
	case "url":
		return "Must be a valid URL"
	case "uuid":
		return "Must be a valid UUID"
	default:
		return "Validation failed for tag: " + err.Tag()
	}
}
