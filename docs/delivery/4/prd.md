# PBI-4: Core Go API Backend

[View in Backlog](../backlog.md#user-content-4)

## Overview

Build the foundational Go API backend using Gin framework, including project structure, database connection management, configuration, middleware, and health check endpoints.

## Problem Statement

Before implementing spatial query endpoints, we need a solid API foundation that:
- Follows Go best practices and clean architecture
- Manages database connections efficiently with pooling
- Handles configuration via environment variables
- Includes proper logging and error handling
- Supports CORS for frontend development
- Provides health check and readiness endpoints
- Is structured for maintainability and testing

## User Stories

- **US-DEV-13**: As a developer, I want a Gin-based HTTP server so that I can serve REST APIs
- **US-DEV-14**: As a developer, I want database connection pooling so that queries are efficient
- **US-DEV-15**: As a developer, I want structured logging so that I can debug issues
- **US-DEV-16**: As a developer, I want CORS configured so that frontends can connect
- **US-DEV-17**: As a frontend developer, I want health check endpoints so that I can verify API availability

## Technical Approach

### Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── database/
│   │   └── postgres.go            # Database connection
│   ├── handlers/
│   │   ├── health.go              # Health check handlers
│   │   └── parcels.go             # Parcel handlers (future)
│   ├── middleware/
│   │   ├── cors.go                # CORS middleware
│   │   ├── logger.go              # Request logging
│   │   └── recovery.go            # Panic recovery
│   ├── models/
│   │   └── parcel.go              # Domain models
│   ├── repository/
│   │   └── parcel_repository.go   # Database operations
│   └── services/
│       └── parcel_service.go      # Business logic
├── go.mod
├── go.sum
└── .golangci.yml
```

### Technology Stack

- **HTTP Framework**: Gin v1.10+
- **Database Driver**: pgx v5 (direct, not GORM for spatial queries)
- **Logging**: zerolog or zap for structured logging
- **Configuration**: viper or env package
- **Validation**: go-playground/validator v10

### Configuration

Environment variables:
```bash
# Server
PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=atlas
DB_USER=postgres
DB_PASSWORD=postgres
DB_POOL_MIN=2
DB_POOL_MAX=10

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:3001
```

### Core Components

**1. Configuration Management** (`config/config.go`)
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    CORS     CORSConfig
}

func Load() (*Config, error) {
    // Load from environment variables
}
```

**2. Database Connection** (`database/postgres.go`)
```go
func NewPostgresPool(cfg DatabaseConfig) (*pgxpool.Pool, error) {
    // Create connection pool with pgx
    // Configure pool size, timeouts
    // Enable PostGIS awareness
}
```

**3. HTTP Server Setup** (`cmd/api/main.go`)
```go
func main() {
    // Load config
    // Initialize database
    // Setup Gin router
    // Register middleware
    // Register routes
    // Start server with graceful shutdown
}
```

### Endpoints (Initial)

**Health Checks**
- `GET /health` - Basic health check
  - Returns: `200 OK` with `{"status": "healthy"}`
  
- `GET /health/ready` - Readiness check
  - Checks database connectivity
  - Returns: `200 OK` if DB connected, `503` otherwise
  - Response: `{"status": "ready", "database": "connected"}`

**API Info**
- `GET /api/v1/info` - API information
  - Returns: Version, build info, uptime

### Middleware Stack

1. **Recovery**: Catch panics and return 500
2. **Logger**: Log all requests with duration
3. **CORS**: Allow frontend origins
4. **Request ID**: Add unique ID to each request

### Error Handling

Standard error responses:
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": {}
  }
}
```

### Logging

Structured logging with fields:
- request_id
- method
- path
- status_code
- duration_ms
- error (if present)

## UX/UI Considerations

N/A - Backend API only

## Acceptance Criteria

1. ✅ Go module initialized with all dependencies
2. ✅ Gin HTTP server starts on configured port (8080)
3. ✅ Configuration loaded from environment variables
4. ✅ Database connection pool established successfully
5. ✅ Connection pool health checked on startup
6. ✅ `GET /health` returns 200 OK
7. ✅ `GET /health/ready` returns 200 when DB connected
8. ✅ `GET /health/ready` returns 503 when DB unavailable
9. ✅ `GET /api/v1/info` returns API metadata
10. ✅ CORS headers set correctly for localhost:3000 and localhost:3001
11. ✅ All requests logged with structured format
12. ✅ Panic recovery middleware prevents crashes
13. ✅ Graceful shutdown implemented (SIGTERM/SIGINT)
14. ✅ golangci-lint passes with no errors
15. ✅ README documents how to run the API locally
16. ✅ Can connect to API from curl/Postman

## Dependencies

- PBI-1: Project Infrastructure Setup (Go environment)
- PBI-2: Database Schema and Spatial Indexing (database must exist)
- Go 1.25+ installed
- PostgreSQL 18.0 running and accessible

## Open Questions

1. Should we use GORM or raw pgx for database access? (Recommendation: pgx for better spatial query control)
2. Do we need API versioning strategy (v1 in path) or is it premature?
3. Should we add rate limiting middleware in MVP?
4. Do we need OpenAPI/Swagger documentation from the start?
5. Should we implement structured error codes or simple messages for MVP?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

