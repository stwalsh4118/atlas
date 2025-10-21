package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/atlas/api/internal/config"
	"github.com/stwalsh4118/atlas/api/internal/database"
	apierrors "github.com/stwalsh4118/atlas/api/internal/errors"
	"github.com/stwalsh4118/atlas/api/internal/logger"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
	"github.com/stwalsh4118/atlas/api/internal/models"
	"github.com/stwalsh4118/atlas/api/internal/repository"
	"github.com/stwalsh4118/atlas/api/internal/services"
)

// setupParcelTestRouter creates a test router with middleware and parcel handlers.
func setupParcelTestRouter(handler *ParcelHandler, log *logger.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))

	// Register routes
	v1 := router.Group("/api/v1")
	{
		parcels := v1.Group("/parcels")
		{
			parcels.GET("/at-point", handler.AtPoint)
		}
	}

	return router
}

// setupTestDB creates a test database connection.
// This requires a real PostgreSQL database with the test schema.
func setupTestDB(t *testing.T) *database.Database {
	t.Helper()

	cfg := config.DatabaseConfig{
		Host:     "host.docker.internal",
		Port:     "5432",
		Name:     "atlas",
		User:     "postgres",
		Password: "postgres",
		PoolMin:  2,
		PoolMax:  5,
	}

	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg)
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

// insertTestParcel inserts a test parcel into the database for testing.
func insertTestParcel(t *testing.T, db *database.Database) *models.TaxParcel {
	t.Helper()

	ctx := context.Background()

	// Create a simple square polygon around Montgomery, TX
	// Coordinates: [lng, lat] in WKT format
	wkt := "POLYGON((-95.4510 30.3485, -95.4490 30.3485, -95.4490 30.3470, -95.4510 30.3470, -95.4510 30.3485))"

	ownerName := "Test Owner"
	situs := "123 Test St, Montgomery, TX"
	asCode := "Residential"

	query := `
		INSERT INTO tax_parcels (
			object_id, pin, owner_name, situs, as_code, 
			county_name, geom, created_at, updated_at
		) VALUES (
			999999, 123456, $1, $2, $3,
			'Montgomery', ST_GeomFromText($4, 4326), NOW(), NOW()
		) RETURNING id, object_id, pin, owner_name, situs, as_code, county_name, 
		ST_AsGeoJSON(geom) as geom_json, created_at, updated_at
	`

	var parcel models.TaxParcel
	var geomJSON string

	err := db.Pool.QueryRow(ctx, query, ownerName, situs, asCode, wkt).Scan(
		&parcel.ID,
		&parcel.ObjectID,
		&parcel.PIN,
		&parcel.OwnerName,
		&parcel.Situs,
		&parcel.AsCode,
		&parcel.CountyName,
		&geomJSON,
		&parcel.CreatedAt,
		&parcel.UpdatedAt,
	)
	require.NoError(t, err, "Failed to insert test parcel")

	// Parse GeoJSON into MultiPolygon
	err = json.Unmarshal([]byte(geomJSON), &parcel.Geom)
	require.NoError(t, err, "Failed to parse geometry JSON")

	return &parcel
}

// cleanupTestParcel removes the test parcel from the database.
func cleanupTestParcel(t *testing.T, db *database.Database, objectID int) {
	t.Helper()

	ctx := context.Background()
	query := "DELETE FROM tax_parcels WHERE object_id = $1"

	_, err := db.Pool.Exec(ctx, query, objectID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test parcel: %v", err)
	}
}

func TestAtPoint_Success(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	testParcel := insertTestParcel(t, db)
	defer cleanupTestParcel(t, db, testParcel.ObjectID)

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with coordinates inside the test parcel
	// Point: (-95.4500, 30.3477) should be inside the test polygon
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response ParcelResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response.Parcel)
	assert.Equal(t, testParcel.ID, response.Parcel.ID)
	assert.Equal(t, "Test Owner", response.Parcel.OwnerName)
	assert.Equal(t, "123 Test St, Montgomery, TX", response.Parcel.SitusAddress)
	assert.Equal(t, "Montgomery", response.Parcel.CountyName)
	assert.NotNil(t, response.Parcel.Geometry)
	assert.Equal(t, "MultiPolygon", response.Parcel.Geometry["type"])

	// Verify response headers
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestAtPoint_NotFound(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with coordinates far from any parcels
	// Using coordinates in the middle of the ocean
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=0.0&lng=0.0", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrNotFound, response.Error.Code)
	assert.Equal(t, "No property found at this location", response.Error.Message)
	assert.NotEmpty(t, response.Error.RequestID)
}

func TestAtPoint_MissingLatitude(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request without lat parameter
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
	assert.NotNil(t, response.Error.Details)
}

func TestAtPoint_MissingLongitude(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request without lng parameter
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
	assert.NotNil(t, response.Error.Details)
}

func TestAtPoint_InvalidLatitude_TooLow(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with latitude < -90
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=-91.0&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
}

func TestAtPoint_InvalidLatitude_TooHigh(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with latitude > 90
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=91.0&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
}

func TestAtPoint_InvalidLongitude_TooLow(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with longitude < -180
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=-181.0", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
}

func TestAtPoint_InvalidLongitude_TooHigh(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with longitude > 180
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=181.0", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, apierrors.ErrValidation, response.Error.Code)
}

func TestAtPoint_InvalidParameterType(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request with non-numeric latitude
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=abc&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response apierrors.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Either validation error or bad request
	assert.Contains(t, []string{apierrors.ErrValidation, apierrors.ErrBadRequest}, response.Error.Code)
}

func TestAtPoint_ResponseFormat(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	testParcel := insertTestParcel(t, db)
	defer cleanupTestParcel(t, db, testParcel.ObjectID)

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response structure
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response ParcelResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify required fields
	assert.NotNil(t, response.Parcel)
	assert.Greater(t, response.Parcel.ID, uint(0))
	assert.NotEmpty(t, response.Parcel.CountyName)
	assert.NotNil(t, response.Parcel.Geometry)
	assert.NotEmpty(t, response.Parcel.Geometry["type"])
	assert.NotEmpty(t, response.Parcel.Geometry["coordinates"])
}

func TestAtPoint_RequestIDHeader(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=0.0&lng=0.0", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify X-Request-ID header is present
	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)

	// Verify it's a valid UUID format (basic check)
	assert.Len(t, requestID, 36, "Request ID should be UUID format")
	assert.Contains(t, requestID, "-", "Request ID should contain hyphens")
}

func TestAtPoint_Logging(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	testParcel := insertTestParcel(t, db)
	defer cleanupTestParcel(t, db, testParcel.ObjectID)

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Make request
	req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=-95.4500", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Just verify the request completed successfully
	// In a real scenario, you'd capture log output and verify specific log messages
	assert.Equal(t, http.StatusOK, w.Code)
}

// Benchmark test for performance validation
func BenchmarkAtPoint(b *testing.B) {
	// Setup
	cfg := config.DatabaseConfig{
		Host:     "host.docker.internal",
		Port:     "5432",
		Name:     "atlas",
		User:     "postgres",
		Password: "postgres",
		PoolMin:  2,
		PoolMax:  10,
	}

	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log := logger.New("test")
	repo := repository.NewParcelRepository(db)
	service := services.NewParcelService(repo, log)
	handler := NewParcelHandler(service)
	router := setupParcelTestRouter(handler, log)

	// Reset timer after setup
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodGet, "/api/v1/parcels/at-point?lat=30.3477&lng=-95.4500", nil)
		if err != nil {
			b.Fatal(err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
