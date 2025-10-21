package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/atlas/api/internal/logger"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
)

func init() {
	// Set Gin to test mode to suppress logs during tests
	gin.SetMode(gin.TestMode)
}

// setupTestContext creates a test Gin context with logger and request ID in context.
func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Add logger to context (using development logger for tests)
	log := logger.New("development")
	c.Set("logger", log)

	// Add request ID to context
	c.Set(middleware.RequestIDKey, "test-request-id")

	return c, w
}

// parseErrorResponse parses the JSON response into an ErrorResponse struct.
func parseErrorResponse(t *testing.T, body *bytes.Buffer) ErrorResponse {
	var response ErrorResponse
	err := json.Unmarshal(body.Bytes(), &response)
	require.NoError(t, err, "Failed to parse error response JSON")
	return response
}

func TestNotFound(t *testing.T) {
	c, w := setupTestContext()

	NotFound(c, "Resource not found")

	// Check status code
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected status 404 Not Found")

	// Parse response
	response := parseErrorResponse(t, w.Body)

	// Verify error structure
	assert.Equal(t, ErrNotFound, response.Error.Code, "Expected NOT_FOUND error code")
	assert.Equal(t, "Resource not found", response.Error.Message, "Expected correct error message")
	assert.Equal(t, "test-request-id", response.Error.RequestID, "Expected request ID in response")
	assert.Nil(t, response.Error.Details, "Expected no details for NotFound")
}

func TestBadRequest(t *testing.T) {
	t.Run("without details", func(t *testing.T) {
		c, w := setupTestContext()

		BadRequest(c, "Invalid input", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected status 400 Bad Request")

		response := parseErrorResponse(t, w.Body)
		assert.Equal(t, ErrBadRequest, response.Error.Code, "Expected BAD_REQUEST error code")
		assert.Equal(t, "Invalid input", response.Error.Message, "Expected correct error message")
		assert.Equal(t, "test-request-id", response.Error.RequestID, "Expected request ID in response")
		assert.Nil(t, response.Error.Details, "Expected no details when nil is passed")
	})

	t.Run("with details", func(t *testing.T) {
		c, w := setupTestContext()

		details := map[string]interface{}{
			"field": "email",
			"value": "invalid",
		}
		BadRequest(c, "Invalid input", details)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected status 400 Bad Request")

		response := parseErrorResponse(t, w.Body)
		assert.Equal(t, ErrBadRequest, response.Error.Code, "Expected BAD_REQUEST error code")
		assert.Equal(t, "Invalid input", response.Error.Message, "Expected correct error message")
		assert.Equal(t, "test-request-id", response.Error.RequestID, "Expected request ID in response")
		assert.NotNil(t, response.Error.Details, "Expected details to be present")
		assert.Equal(t, "email", response.Error.Details["field"], "Expected field in details")
		assert.Equal(t, "invalid", response.Error.Details["value"], "Expected value in details")
	})
}

func TestInternalServerError(t *testing.T) {
	c, w := setupTestContext()

	testErr := errors.New("database connection failed")
	InternalServerError(c, "An unexpected error occurred", testErr)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "Expected status 500 Internal Server Error")

	response := parseErrorResponse(t, w.Body)
	assert.Equal(t, ErrInternalServer, response.Error.Code, "Expected INTERNAL_SERVER_ERROR code")
	assert.Equal(t, "An unexpected error occurred", response.Error.Message, "Expected correct error message")
	assert.Equal(t, "test-request-id", response.Error.RequestID, "Expected request ID in response")
	assert.Nil(t, response.Error.Details, "Expected no details for InternalServerError")
}

func TestValidationError(t *testing.T) {
	c, w := setupTestContext()

	// Create a test struct with validation tags
	type TestStruct struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"required,gte=18"`
	}

	// Create validator and validate a struct that fails validation
	validate := validator.New()
	testData := TestStruct{
		Email: "not-an-email",
		Age:   15,
	}

	err := validate.Struct(testData)
	require.Error(t, err, "Expected validation to fail")

	// Extract validation errors
	validationErrors, ok := err.(validator.ValidationErrors)
	require.True(t, ok, "Expected validator.ValidationErrors")

	// Call ValidationError function
	ValidationError(c, validationErrors)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected status 400 Bad Request")

	response := parseErrorResponse(t, w.Body)
	assert.Equal(t, ErrValidation, response.Error.Code, "Expected VALIDATION_ERROR code")
	assert.Equal(t, "Validation failed for one or more fields", response.Error.Message)
	assert.Equal(t, "test-request-id", response.Error.RequestID, "Expected request ID in response")
	assert.NotNil(t, response.Error.Details, "Expected details to be present")

	// Check that specific fields are in the details
	_, hasEmail := response.Error.Details["Email"]
	_, hasAge := response.Error.Details["Age"]
	assert.True(t, hasEmail || hasAge, "Expected at least one validation error field")
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		param    string
		expected string
	}{
		{
			name:     "required",
			tag:      "required",
			param:    "",
			expected: "This field is required",
		},
		{
			name:     "email",
			tag:      "email",
			param:    "",
			expected: "Must be a valid email address",
		},
		{
			name:     "min",
			tag:      "min",
			param:    "5",
			expected: "Value is too short or small (minimum: 5)",
		},
		{
			name:     "max",
			tag:      "max",
			param:    "100",
			expected: "Value is too long or large (maximum: 100)",
		},
		{
			name:     "len",
			tag:      "len",
			param:    "10",
			expected: "Must have length of 10",
		},
		{
			name:     "gt",
			tag:      "gt",
			param:    "0",
			expected: "Must be greater than 0",
		},
		{
			name:     "gte",
			tag:      "gte",
			param:    "18",
			expected: "Must be greater than or equal to 18",
		},
		{
			name:     "lt",
			tag:      "lt",
			param:    "100",
			expected: "Must be less than 100",
		},
		{
			name:     "lte",
			tag:      "lte",
			param:    "100",
			expected: "Must be less than or equal to 100",
		},
		{
			name:     "oneof",
			tag:      "oneof",
			param:    "red blue green",
			expected: "Must be one of: red blue green",
		},
		{
			name:     "url",
			tag:      "url",
			param:    "",
			expected: "Must be a valid URL",
		},
		{
			name:     "uuid",
			tag:      "uuid",
			param:    "",
			expected: "Must be a valid UUID",
		},
		{
			name:     "unknown",
			tag:      "unknown_tag",
			param:    "",
			expected: "Validation failed for tag: unknown_tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock FieldError
			mockErr := &mockFieldError{
				tag:   tt.tag,
				param: tt.param,
			}

			result := formatValidationError(mockErr)
			assert.Equal(t, tt.expected, result, "Expected correct validation error message")
		})
	}
}

func TestErrorResponseWithoutContext(t *testing.T) {
	// Test that error functions work even without logger/request ID in context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Call NotFound without setting up context (no logger, no request ID)
	NotFound(c, "Resource not found")

	assert.Equal(t, http.StatusNotFound, w.Code, "Expected status 404 even without context")

	response := parseErrorResponse(t, w.Body)
	assert.Equal(t, ErrNotFound, response.Error.Code, "Expected error code")
	assert.Equal(t, "Resource not found", response.Error.Message, "Expected error message")
	// Request ID should be empty string if not in context
	assert.Empty(t, response.Error.RequestID, "Expected empty request ID when not in context")
}

func TestErrorConstants(t *testing.T) {
	// Verify error code constants are defined
	assert.Equal(t, "NOT_FOUND", ErrNotFound)
	assert.Equal(t, "BAD_REQUEST", ErrBadRequest)
	assert.Equal(t, "INTERNAL_SERVER_ERROR", ErrInternalServer)
	assert.Equal(t, "VALIDATION_ERROR", ErrValidation)
	assert.Equal(t, "DATABASE_CONNECTION_ERROR", ErrDatabaseConnection)
}

// mockFieldError is a mock implementation of validator.FieldError for testing.
type mockFieldError struct {
	tag   string
	param string
}

func (m *mockFieldError) Tag() string                    { return m.tag }
func (m *mockFieldError) ActualTag() string              { return m.tag }
func (m *mockFieldError) Namespace() string              { return "" }
func (m *mockFieldError) StructNamespace() string        { return "" }
func (m *mockFieldError) Field() string                  { return "TestField" }
func (m *mockFieldError) StructField() string            { return "TestField" }
func (m *mockFieldError) Value() interface{}             { return nil }
func (m *mockFieldError) Param() string                  { return m.param }
func (m *mockFieldError) Kind() reflect.Kind             { return reflect.String }
func (m *mockFieldError) Type() reflect.Type             { return nil }
func (m *mockFieldError) Translate(ut.Translator) string { return "" }
func (m *mockFieldError) Error() string                  { return "" }
