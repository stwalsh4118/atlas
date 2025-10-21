# Error Handling Package

This package provides standardized error response structures and helper functions for consistent error handling across the Atlas API.

## Usage

Import the package in your handlers:

```go
import "github.com/stwalsh4118/atlas/api/internal/errors"
```

### Example: NotFound Error

```go
func (h *Handler) GetParcel(c *gin.Context) {
    id := c.Param("id")
    
    parcel, err := h.db.FindParcel(id)
    if err == sql.ErrNoRows {
        errors.NotFound(c, "Parcel not found")
        return
    }
    
    c.JSON(http.StatusOK, parcel)
}
```

Response:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Parcel not found",
    "request_id": "abc-123-def"
  }
}
```

### Example: BadRequest with Details

```go
func (h *Handler) CreateParcel(c *gin.Context) {
    var input CreateParcelInput
    if err := c.ShouldBindJSON(&input); err != nil {
        errors.BadRequest(c, "Invalid request body", map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    
    // ... process request
}
```

Response:
```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "Invalid request body",
    "details": {
      "error": "invalid JSON syntax"
    },
    "request_id": "abc-123-def"
  }
}
```

### Example: ValidationError

```go
func (h *Handler) CreateUser(c *gin.Context) {
    var input CreateUserInput
    if err := c.ShouldBindJSON(&input); err != nil {
        // Check if it's a validation error
        if validationErrors, ok := err.(validator.ValidationErrors); ok {
            errors.ValidationError(c, validationErrors)
            return
        }
        errors.BadRequest(c, "Invalid request body", nil)
        return
    }
    
    // ... process request
}
```

Response:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed for one or more fields",
    "details": {
      "Email": "Must be a valid email address",
      "Age": "Must be greater than or equal to 18"
    },
    "request_id": "abc-123-def"
  }
}
```

### Example: InternalServerError

```go
func (h *Handler) GetParcels(c *gin.Context) {
    parcels, err := h.db.FindAllParcels()
    if err != nil {
        // Logs full error internally, returns generic message to client
        errors.InternalServerError(c, "Failed to retrieve parcels", err)
        return
    }
    
    c.JSON(http.StatusOK, parcels)
}
```

Response:
```json
{
  "error": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "Failed to retrieve parcels",
    "request_id": "abc-123-def"
  }
}
```

Note: The actual error details are logged but NOT exposed to the client for security reasons.

## Error Codes

The following error code constants are available:

- `errors.ErrNotFound` - "NOT_FOUND"
- `errors.ErrBadRequest` - "BAD_REQUEST"
- `errors.ErrInternalServer` - "INTERNAL_SERVER_ERROR"
- `errors.ErrValidation` - "VALIDATION_ERROR"
- `errors.ErrDatabaseConnection` - "DATABASE_CONNECTION_ERROR"

## Logging

All error helpers automatically:
1. Retrieve the logger from the Gin context (set by Logger middleware)
2. Retrieve the request ID from the Gin context (set by RequestID middleware)
3. Log with appropriate level (Warn for 4xx, Error for 5xx)
4. Include request ID in both logs and response

## Response Format

All error responses follow this structure:

```go
type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code      string                 `json:"code"`
    Message   string                 `json:"message"`
    Details   map[string]interface{} `json:"details,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
}
```

## Supported Validation Tags

The `ValidationError` helper supports the following validation tags:

- `required` - "This field is required"
- `email` - "Must be a valid email address"
- `min` - "Value is too short or small (minimum: X)"
- `max` - "Value is too long or large (maximum: X)"
- `len` - "Must have length of X"
- `gt` - "Must be greater than X"
- `gte` - "Must be greater than or equal to X"
- `lt` - "Must be less than X"
- `lte` - "Must be less than or equal to X"
- `oneof` - "Must be one of: X"
- `url` - "Must be a valid URL"
- `uuid` - "Must be a valid UUID"

For unsupported tags, a generic message is returned: "Validation failed for tag: X"

