# Atlas API Reference (For AI Agent)

> **Purpose**: Quick reference for existing APIs, data models, and pipeline tools to avoid recreating functionality

Last Updated: 2025-10-21 (Task 5-5)

---

## Middleware Package (`api/internal/middleware`)

### Context Utilities

```go
// Get logger with request ID from context
middleware.GetLogger(c *gin.Context) *logger.Logger  // Returns nil if not found

// Get request ID from context
middleware.GetRequestID(c *gin.Context) string  // Returns "" if not found
```

**Usage**: Always use these in handlers instead of passing logger/request ID separately.

### Middleware Functions

```go
middleware.RequestID() gin.HandlerFunc          // Generates UUID, adds to context & headers
middleware.Logger(log *logger.Logger) gin.HandlerFunc  // Logs requests, stores logger in context
middleware.Recovery(log *logger.Logger) gin.HandlerFunc  // Catches panics, returns 500
middleware.CORS(origins []string) gin.HandlerFunc  // CORS with allowed origins (uses gin-contrib/cors)
```

### Constants

```go
middleware.RequestIDKey = "request_id"
middleware.RequestIDHeader = "X-Request-ID"
```

---

## Logger Package (`api/internal/logger`)

### Constructor

```go
logger.New(env string) *Logger  // "development" = console, "production" = JSON
```

### Methods

```go
log.Debug(msg string, fields map[string]interface{})
log.Info(msg string, fields map[string]interface{})
log.Warn(msg string, fields map[string]interface{})
log.Error(msg string, err error, fields map[string]interface{})
log.Fatal(msg string, err error, fields map[string]interface{})  // Exits program

// Create child loggers with context
log.With(fields map[string]interface{}) *Logger
log.WithRequestID(requestID string) *Logger  // Adds "request_id" field
```

**Note**: Logger automatically handles nil fields maps.

---

## Database Package (`api/internal/database`)

### Constructor

```go
database.NewPostgresPool(ctx context.Context, cfg config.DatabaseConfig) (*Database, error)
// - Creates connection pool
// - Configures timeouts (5s connect, 30s idle, 1h lifetime)
// - Tests connection immediately
// - Returns error if connection fails
```

### Methods

```go
db.Ping(ctx context.Context) error  // Check if DB is alive
db.Close()  // Gracefully close pool (safe to call multiple times)
db.Stats() *pgxpool.Stat  // Pool statistics (or nil)
db.Pool *pgxpool.Pool  // Direct access to pgx pool
```

**Usage**: Use `Ping()` for health checks, `Stats()` for monitoring.

---

## Config Package (`api/internal/config`)

### Loading Configuration

```go
config.Load() (*Config, error)  // Loads from .env file + env vars, validates, returns error if invalid
```

**Priority**: defaults < `.env` file < shell environment variables

### Config Structure

```go
type Config struct {
    Server   ServerConfig   // Port, Env
    Database DatabaseConfig // Host, Port, Name, User, Password, PoolMin, PoolMax
    CORS     CORSConfig     // Origins []string
}
```

### Environment Variables (with defaults)

```
PORT=8080 (default)
ENV=development (default)
DB_HOST=host.docker.internal (default)
DB_PORT=5432 (default)
DB_NAME=atlas (default)
DB_USER=postgres (default)
DB_PASSWORD=(REQUIRED - no default)
DB_POOL_MIN=2 (default)
DB_POOL_MAX=10 (default)
CORS_ORIGINS=http://localhost:3000,http://localhost:3001 (default, comma-separated)
```

**Notes**: 
- `Load()` validates all required fields and returns descriptive errors
- Create `.env` file from `api/env.example` for local development
- `.env` file is optional; defaults and shell env vars work without it

---

## Errors Package (`api/internal/errors`)

### Error Helper Functions

```go
errors.NotFound(c *gin.Context, message string)
errors.BadRequest(c *gin.Context, message string, details map[string]interface{})
errors.InternalServerError(c *gin.Context, message string, err error)
errors.ValidationError(c *gin.Context, validationErrors validator.ValidationErrors)
```

**Usage**: Always use these helpers for consistent error responses across the API.

### Error Code Constants

```go
errors.ErrNotFound           = "NOT_FOUND"
errors.ErrBadRequest         = "BAD_REQUEST"
errors.ErrInternalServer     = "INTERNAL_SERVER_ERROR"
errors.ErrValidation         = "VALIDATION_ERROR"
errors.ErrDatabaseConnection = "DATABASE_CONNECTION_ERROR"
```

### Error Response Structure

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

**Note**: 
- Automatically includes request ID from context
- Logs with appropriate level (Warn for 4xx, Error for 5xx)
- InternalServerError logs full error but returns generic message to client

---

## Handlers Package (`api/internal/handlers`)

### Constants

```go
handlers.APIVersion = "0.1.0"
handlers.HealthCheckTimeout = 2 * time.Second
```

### Health Handler

```go
handlers.NewHealthHandler(db *database.Database, env string) *HealthHandler

// Handler methods
handler.Health(c *gin.Context)  // GET /health - always 200 OK
handler.Ready(c *gin.Context)   // GET /health/ready - checks DB (200 or 503)
handler.Info(c *gin.Context)    // GET /api/v1/info - returns version, env, uptime
```

### Parcel Handler

```go
handlers.NewParcelHandler(service services.ParcelService) *ParcelHandler

// Handler methods
handler.AtPoint(c *gin.Context)  // GET /api/v1/parcels/at-point - find parcel by lat/lng
handler.Nearby(c *gin.Context)   // GET /api/v1/parcels/nearby - find parcels within radius
```

**Request DTOs**:
```go
type AtPointRequest struct {
    Lat float64 `form:"lat" binding:"required,min=-90,max=90"`
    Lng float64 `form:"lng" binding:"required,min=-180,max=180"`
}

type NearbyRequest struct {
    Lat    float64 `form:"lat" binding:"required,min=-90,max=90"`
    Lng    float64 `form:"lng" binding:"required,min=-180,max=180"`
    Radius int     `form:"radius,omitempty,min=1,max=5000"` // default: 1000m
}
```

**Response DTOs**:
```go
type ParcelResponse struct {
    Parcel *ParcelData `json:"parcel"`
}

type ParcelData struct {
    Geometry     map[string]interface{} `json:"geometry"`
    ParcelID     string                 `json:"parcel_id,omitempty"`
    OwnerName    string                 `json:"owner_name,omitempty"`
    SitusAddress string                 `json:"situs_address,omitempty"`
    PropType     string                 `json:"prop_type,omitempty"`
    LandUse      string                 `json:"land_use,omitempty"`
    CountyName   string                 `json:"county_name"`
    Acres        float64                `json:"acres,omitempty"`
    ID           uint                   `json:"id"`
}

type NearbyResponse struct {
    Parcels []ParcelWithDistance `json:"parcels"`
    Count   int                  `json:"count"`
}

type ParcelWithDistance struct {
    Geometry   map[string]interface{} `json:"geometry"`
    ParcelID   string                 `json:"parcel_id,omitempty"`
    OwnerName  string                 `json:"owner_name,omitempty"`
    CountyName string                 `json:"county_name"`
    Acres      float64                `json:"acres,omitempty"`
    Distance   float64                `json:"distance_meters"`
    ID         uint                   `json:"id"`
}
```

**Error Handling**:
- Returns 400 for validation errors (missing/invalid coordinates, invalid radius)
- Returns 404 when no parcel found at the given point (at-point only)
- Returns 200 with empty array when no parcels found (nearby only)
- Returns 500 for database or unexpected errors
- Uses `errors` package helpers for consistent responses

**Nearby Endpoint Specifics**:
- Default radius: 1000 meters (applied when radius=0 or not provided)
- Maximum radius: 5000 meters
- Returns empty array (count=0) when no parcels found
- Results ordered by distance ascending
- Distance values in meters

---

## Common Patterns

### Handler Pattern

```go
func (h *Handler) SomeEndpoint(c *gin.Context) {
    // Get logger from context (set by Logger middleware)
    log := middleware.GetLogger(c)
    if log != nil {
        log.Info("Processing request", map[string]interface{}{
            "param": c.Param("id"),
        })
    }
    
    // Get request ID if needed
    requestID := middleware.GetRequestID(c)
    
    // ... handler logic
    
    c.JSON(http.StatusOK, response)
}
```

### Error Handling Pattern

```go
func (h *Handler) GetResource(c *gin.Context) {
    id := c.Param("id")
    
    // Not found error
    resource, err := h.db.Find(id)
    if err == sql.ErrNoRows {
        errors.NotFound(c, "Resource not found")
        return
    }
    
    // Internal server error
    if err != nil {
        errors.InternalServerError(c, "Failed to retrieve resource", err)
        return
    }
    
    c.JSON(http.StatusOK, resource)
}

func (h *Handler) CreateResource(c *gin.Context) {
    var input CreateInput
    
    // Validation error
    if err := c.ShouldBindJSON(&input); err != nil {
        if validationErrors, ok := err.(validator.ValidationErrors); ok {
            errors.ValidationError(c, validationErrors)
            return
        }
        errors.BadRequest(c, "Invalid request body", nil)
        return
    }
    
    // ... create resource
}
```

### Middleware Stack Order

```go
router.Use(middleware.RequestID())      // 1. Generate request ID first
router.Use(middleware.Logger(log))      // 2. Logger uses request ID
router.Use(middleware.Recovery(log))    // 3. Recovery catches panics
router.Use(middleware.CORS(origins))    // 4. CORS last
```

### Error Response Format (standardized)

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": {
      "field": "value"
    },
    "request_id": "uuid"
  }
}
```

**Note**: Use `errors` package helpers instead of manually creating error responses.

### Dependencies

- `github.com/stretchr/testify/assert` - for assertions
- `github.com/stretchr/testify/require` - for required checks (fail fast)

---

## Models Package (`api/internal/models`)

### TaxParcel Model

```go
type TaxParcel struct {
    ID, ObjectID, PIN, PID           // Identifiers
    StateCd, Block, Lot, Tract        // Subdivision
    OwnerName, OwnerAddress           // Owner info
    Situs, AsCode, LegalDescription   // Property details
    ImprvActualYearBuilt, ImprvMainArea  // Building
    PYear, PVersion, TaxingUnits, Exemptions  // Tax info
    CountyName string                 // Default: "Montgomery"
    Geom MultiPolygon                 // PostGIS MultiPolygon, SRID 4326
    CreatedAt, UpdatedAt time.Time
}
```

**Key Points**: Nullable fields use pointers. `TableName()` returns "tax_parcels".

### Geometry Types

```go
type Polygon struct {
    Coordinates [][][2]float64  // [rings][points][lon,lat]
    SRID int                    // 4326 (WGS84)
}

type MultiPolygon struct {
    Coordinates [][][][2]float64  // [polygons][rings][points][lon,lat]
    SRID int                      // 4326 (WGS84)
}
```

Both implement `sql.Scanner`, `driver.Valuer`, `json.Marshaler/Unmarshaler` for PostGIS/GeoJSON.

---

## Repository Package (`api/internal/repository`)

### ParcelRepository

```go
type ParcelWithDistance struct {
    Parcel   models.TaxParcel
    Distance float64  // meters
}

type ParcelRepository interface {
    FindByPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error)
    FindNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]ParcelWithDistance, error)
}

repo := repository.NewParcelRepository(db)
```

**Usage**:
- `FindByPoint`: Returns `(nil, nil)` when no parcel found (not an error)
- `FindNearby`: Returns empty slice when no parcels found (not an error)
- Returns error only for database failures
- Uses PostGIS spatial functions with indexes (`ST_Contains`, `ST_DWithin`)
- Context-aware for timeouts/cancellation

**Example**:
```go
// Point query
parcel, err := repo.FindByPoint(ctx, 30.3477, -95.4502)
if err != nil { /* Database error */ }
if parcel == nil { /* Not found */ }

// Nearby query (1km radius, ordered by distance, max 20 results)
parcels, err := repo.FindNearby(ctx, 30.3477, -95.4502, 1000)
if err != nil { /* Database error */ }
// parcels slice is empty if none found
```

---

## Services Package (`api/internal/services`)

### ParcelService

```go
type ParcelService interface {
    GetParcelAtPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error)
    GetNearbyParcels(ctx context.Context, lat, lng float64, radiusMeters int) ([]repository.ParcelWithDistance, error)
}

service := services.NewParcelService(repo, log)
```

**Errors**:
```go
services.ErrInvalidCoordinates  // Coordinates out of valid range
services.ErrParcelNotFound      // No parcel at given point
services.ErrInvalidRadius       // Radius not between 1 and 5000 meters
```

**Validation Constants**:
```go
services.MinLatitude     = -90.0
services.MaxLatitude     = 90.0
services.MinLongitude    = -180.0
services.MaxLongitude    = 180.0
services.MinRadiusMeters = 1
services.MaxRadiusMeters = 5000
```

**Usage**:
- Validates coordinates and radius before querying repository
- Transforms repository `(nil, nil)` → `ErrParcelNotFound` for point queries
- Logs queries with structured fields
- Returns wrapped errors for database failures

**Example**:
```go
// Point query
parcel, err := service.GetParcelAtPoint(ctx, 30.3477, -95.4502)
if errors.Is(err, services.ErrInvalidCoordinates) { /* validation */ }
if errors.Is(err, services.ErrParcelNotFound) { /* not found */ }

// Nearby query (returns empty slice if none found)
parcels, err := service.GetNearbyParcels(ctx, 30.3477, -95.4502, 1000)
if errors.Is(err, services.ErrInvalidRadius) { /* invalid radius */ }
// Check len(parcels) for results
```

---

## Database Schema

### tax_parcels Table

- **GiST Index**: `idx_parcels_geom` on geom column (for fast spatial queries)
- **Indexes**: object_id (unique), pin, owner_name, situs
- **Geometry**: `GEOMETRY(MultiPolygon, 4326)` (allows Polygon or MultiPolygon)

**PostGIS Queries**:
```sql
-- Point-in-polygon (note: lng, lat order for PostGIS)
WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint(lng, lat), 4326))

-- Bounding box
WHERE geom && ST_MakeEnvelope(west, south, east, north, 4326)
```

---

## Scripts (`scripts/`)

### import-parcels.sh
```bash
./import-parcels.sh --file data.geojson --mapping config.json [--mode replace|append] [--dry-run] [--validate-geometries]
```
Imports GeoJSON/Shapefile → PostgreSQL. Uses ogr2ogr, staging table, field mapping, transaction-based.

### validate-geodata.sh
```bash
./validate-geodata.sh data.geojson
```
Pre-import validation: checks CRS, lists fields, counts records, shows samples.

### validate-geometries.sh
```bash
./validate-geometries.sh
```
Validates and repairs geometries in tax_parcels table (ST_IsValid, ST_MakeValid).

### post-import-validation.sh
```bash
./post-import-validation.sh
```
Post-import checks: record counts, NULL checks, SRID verification, spatial index test, VACUUM ANALYZE.

### Field Mappings
`scripts/mappings/*.json` - County-specific field mapping configs (maps source field names → DB columns)

---

## Key Design Decisions

1. **Logger in Context**: Middleware stores request-scoped logger in Gin context, handlers retrieve it
2. **Request ID**: Generated by middleware, available in context and response headers
3. **Database Ping**: Use `db.Ping(ctx)` for health checks (not `db.Pool.Ping()`)
4. **Configuration**: Single `Load()` call validates everything, fails fast with descriptive errors
5. **CORS**: Uses `gin-contrib/cors` package (not custom implementation)

---

## Quick Checklist for New Handlers

- [ ] Get logger from context using `middleware.GetLogger(c)`
- [ ] Use structured logging with fields map
- [ ] Use `errors` package helpers for error responses
- [ ] Use `c.JSON()` for success responses
- [ ] Handle validation errors with `errors.ValidationError()`
- [ ] Use `errors.InternalServerError()` for unexpected errors (never expose internals)
- [ ] Write unit tests (mocked) and integration tests (real DB)
- [ ] Follow existing response formats

---

## Files to Reference

**Backend API**:
- `/api/internal/middleware/logger.go` - Logger middleware and context utilities
- `/api/internal/middleware/request_id.go` - Request ID generation
- `/api/internal/logger/logger.go` - Logger methods
- `/api/internal/database/postgres.go` - Database operations
- `/api/internal/config/config.go` - Configuration loading
- `/api/internal/errors/errors.go` - Standardized error handling utilities
- `/api/internal/handlers/health.go` - Health check handler implementation
- `/api/internal/handlers/parcel_handler.go` - Parcel query handler implementation

**Data Models**:
- `/api/internal/models/tax_parcel.go` - TaxParcel model with GORM tags
- `/api/internal/models/geometry.go` - Polygon and MultiPolygon types with PostGIS integration

**Repositories**:
- `/api/internal/repository/parcel_repository.go` - Parcel data access layer

**Services**:
- `/api/internal/services/parcel_service.go` - Parcel business logic layer

**Database**:
- `/api/migrations/000002_create_tax_parcels_table.up.sql` - Main tax_parcels table schema
- `/docs/delivery/2/2-1-gorm-postgis-guide.md` - GORM + PostGIS integration guide

**Data Pipeline**:
- `/scripts/import-parcels.sh` - Main import script (ogr2ogr-based)
- `/scripts/mappings/` - Field mapping configurations per county
- `/docs/delivery/3/3-1-ogr2ogr-guide.md` - ogr2ogr usage guide

