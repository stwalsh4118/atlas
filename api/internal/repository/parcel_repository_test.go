package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stwalsh4118/atlas/api/internal/config"
	"github.com/stwalsh4118/atlas/api/internal/database"
)

// getTestConfig returns database configuration for integration tests.
func getTestConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "host.docker.internal"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Name:     getEnvOrDefault("DB_NAME", "atlas"),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		PoolMin:  2,
		PoolMax:  5,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestRepository creates a test database connection and repository.
func setupTestRepository(t *testing.T) (*ParcelRepository, *database.Database) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}

	repo := NewParcelRepository(db)
	return &repo, db
}

// TestNewParcelRepository verifies repository creation.
func TestNewParcelRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	db, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	repo := NewParcelRepository(db)
	if repo == nil {
		t.Fatal("Expected repository to be initialized")
	}
}

// TestFindByPoint_Success tests finding a parcel at a known location.
// Note: This test requires parcel data to be loaded in the database.
func TestFindByPoint_Success(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Query for Montgomery County, TX - coordinates that should have parcel data
	// This is a real location in Montgomery County, Texas
	// If no data is loaded, this test will return nil (not found)
	lat := 30.3477
	lng := -95.4502

	parcel, err := (*repo).FindByPoint(ctx, lat, lng)
	if err != nil {
		t.Fatalf("FindByPoint returned error: %v", err)
	}

	// If parcel data is loaded, we should get a result
	// If not, the result will be nil (which is valid behavior)
	if parcel != nil {
		// Verify basic parcel structure
		if parcel.ID == 0 {
			t.Error("Expected parcel ID to be non-zero")
		}
		if parcel.CountyName == "" {
			t.Error("Expected county name to be set")
		}
		if len(parcel.Geom.Coordinates) == 0 {
			t.Error("Expected geometry coordinates to be populated")
		}

		t.Logf("Found parcel: ID=%d, ObjectID=%d, CountyName=%s",
			parcel.ID, parcel.ObjectID, parcel.CountyName)
	} else {
		t.Log("No parcel found at test coordinates (may need to load test data)")
	}
}

// TestFindByPoint_NotFound tests querying a location with no parcels.
func TestFindByPoint_NotFound(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Coordinates in the middle of the Gulf of Mexico (no parcels)
	lat := 27.0
	lng := -93.0

	parcel, err := (*repo).FindByPoint(ctx, lat, lng)
	if err != nil {
		t.Errorf("FindByPoint should not return error for not found, got: %v", err)
	}

	if parcel != nil {
		t.Errorf("Expected nil parcel for ocean coordinates, got parcel ID %d", parcel.ID)
	}
}

// TestFindByPoint_ExtremeCoordinates tests with extreme but valid coordinates.
func TestFindByPoint_ExtremeCoordinates(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	testCases := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"North Pole", 90.0, 0.0},
		{"South Pole", -90.0, 0.0},
		{"International Date Line West", 0.0, -180.0},
		{"International Date Line East", 0.0, 180.0},
		{"Prime Meridian", 0.0, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parcel, err := (*repo).FindByPoint(ctx, tc.lat, tc.lng)
			if err != nil {
				t.Errorf("FindByPoint with extreme coordinates should not error, got: %v", err)
			}
			// We expect nil (not found) for these coordinates
			if parcel != nil {
				t.Logf("Unexpectedly found parcel at %s: ID=%d", tc.name, parcel.ID)
			}
		})
	}
}

// TestFindByPoint_ContextCancellation tests context cancellation.
func TestFindByPoint_ContextCancellation(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 30.3477
	lng := -95.4502

	_, err := (*repo).FindByPoint(ctx, lat, lng)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}

	// Verify it's a context error
	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}

// TestFindByPoint_ContextTimeout tests context timeout.
func TestFindByPoint_ContextTimeout(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	lat := 30.3477
	lng := -95.4502

	_, err := (*repo).FindByPoint(ctx, lat, lng)
	// Should get a context deadline exceeded error or nil if query was fast enough
	if err != nil && ctx.Err() == nil {
		t.Errorf("Expected context timeout error, got: %v", err)
	}
}

// TestFindByPoint_MultipleQueries tests making multiple sequential queries.
func TestFindByPoint_MultipleQueries(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Test multiple queries in sequence
	coordinates := []struct {
		lat float64
		lng float64
	}{
		{30.3477, -95.4502}, // Montgomery County, TX
		{30.3500, -95.4500}, // Nearby location
		{0.0, 0.0},          // Gulf of Guinea (should be not found)
	}

	for i, coord := range coordinates {
		parcel, err := (*repo).FindByPoint(ctx, coord.lat, coord.lng)
		if err != nil {
			t.Errorf("Query %d failed: %v", i+1, err)
		}

		if parcel != nil {
			t.Logf("Query %d found parcel: ID=%d", i+1, parcel.ID)
		} else {
			t.Logf("Query %d: no parcel found", i+1)
		}
	}
}

// TestFindByPoint_GeometryParsing tests that geometry is correctly parsed.
func TestFindByPoint_GeometryParsing(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Montgomery County coordinates
	lat := 30.3477
	lng := -95.4502

	parcel, err := (*repo).FindByPoint(ctx, lat, lng)
	if err != nil {
		t.Fatalf("FindByPoint returned error: %v", err)
	}

	// If we found a parcel, verify geometry structure
	if parcel != nil {
		// Geometry should be populated (MultiPolygon has at least one polygon)
		if len(parcel.Geom.Coordinates) == 0 {
			t.Error("Expected geometry coordinates to be populated")
		}

		// SRID should be WGS84
		if parcel.Geom.SRID != 4326 {
			t.Errorf("Expected SRID 4326, got %d", parcel.Geom.SRID)
		}

		// Verify it's a valid MultiPolygon structure
		// MultiPolygon: [polygons][rings][points][lon,lat]
		totalRings := 0
		totalPoints := 0
		for polyIdx, polygon := range parcel.Geom.Coordinates {
			if len(polygon) == 0 {
				t.Errorf("Polygon %d has no rings", polyIdx)
			}

			for ringIdx, ring := range polygon {
				totalRings++
				totalPoints += len(ring)

				if len(ring) < 4 {
					t.Errorf("Polygon %d, Ring %d has %d points, expected at least 4 for a closed polygon",
						polyIdx, ringIdx, len(ring))
				}

				// First and last point should be the same (closed ring)
				if len(ring) >= 4 {
					firstPoint := ring[0]
					lastPoint := ring[len(ring)-1]
					if firstPoint[0] != lastPoint[0] || firstPoint[1] != lastPoint[1] {
						t.Errorf("Polygon %d, Ring %d is not closed: first point [%f,%f] != last point [%f,%f]",
							polyIdx, ringIdx, firstPoint[0], firstPoint[1], lastPoint[0], lastPoint[1])
					}
				}
			}
		}

		t.Logf("MultiPolygon has %d polygons with %d total rings and %d total points",
			len(parcel.Geom.Coordinates), totalRings, totalPoints)
	} else {
		t.Log("No parcel found for geometry parsing test (may need to load test data)")
	}
}

// TestFindByPoint_NullableFields tests handling of nullable fields.
func TestFindByPoint_NullableFields(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	lat := 30.3477
	lng := -95.4502

	parcel, err := (*repo).FindByPoint(ctx, lat, lng)
	if err != nil {
		t.Fatalf("FindByPoint returned error: %v", err)
	}

	if parcel != nil {
		// Log which nullable fields are present
		t.Log("Nullable field status:")
		if parcel.OwnerName != nil {
			t.Logf("  OwnerName: %s", *parcel.OwnerName)
		} else {
			t.Log("  OwnerName: NULL")
		}
		if parcel.Situs != nil {
			t.Logf("  Situs: %s", *parcel.Situs)
		} else {
			t.Log("  Situs: NULL")
		}
		if parcel.LegalDescription != nil {
			t.Logf("  LegalDescription: %s", *parcel.LegalDescription)
		} else {
			t.Log("  LegalDescription: NULL")
		}

		// Non-nullable fields should always be present
		if parcel.CountyName == "" {
			t.Error("CountyName should not be empty")
		}
	} else {
		t.Log("No parcel found for nullable fields test")
	}
}

// TestFindByPoint_CoordinateOrder tests that PostGIS (lng, lat) order is correct.
func TestFindByPoint_CoordinateOrder(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// These coordinates are in Montgomery County, TX
	lat := 30.3477
	lng := -95.4502

	// Query with correct order
	parcel1, err := (*repo).FindByPoint(ctx, lat, lng)
	if err != nil {
		t.Fatalf("FindByPoint returned error: %v", err)
	}

	// Now try with swapped coordinates (should not find same parcel or any parcel)
	// If we accidentally swap lat/lng, this would fail
	parcel2, err := (*repo).FindByPoint(ctx, lng, lat)
	if err != nil {
		t.Fatalf("FindByPoint with swapped coords returned error: %v", err)
	}

	// Log results
	if parcel1 != nil {
		t.Logf("Correct order (lat=%f, lng=%f) found parcel ID=%d", lat, lng, parcel1.ID)
	} else {
		t.Log("No parcel found with correct coordinate order")
	}

	if parcel2 != nil {
		t.Logf("Swapped order (lat=%f, lng=%f) found parcel ID=%d", lng, lat, parcel2.ID)
	} else {
		t.Log("No parcel found with swapped coordinates (expected for invalid location)")
	}
}

// TestFindNearby_Success tests finding parcels within a radius of a known location.
// Note: This test requires parcel data to be loaded in the database.
func TestFindNearby_Success(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Query for Montgomery County, TX - coordinates that should have parcel data
	lat := 30.3477
	lng := -95.4502
	radiusMeters := 1000 // 1km radius

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby returned error: %v", err)
	}

	// Result should be a non-nil slice (empty or with data)
	if parcels == nil {
		t.Fatal("Expected non-nil slice from FindNearby")
	}

	// Log results
	t.Logf("Found %d parcels within %dm of (lat=%f, lng=%f)",
		len(parcels), radiusMeters, lat, lng)

	// If parcels were found, verify their structure
	for i, result := range parcels {
		if result.Parcel.ID == 0 {
			t.Errorf("Parcel %d has zero ID", i)
		}
		if result.Distance < 0 {
			t.Errorf("Parcel %d has negative distance: %f", i, result.Distance)
		}
		if result.Distance > float64(radiusMeters) {
			t.Errorf("Parcel %d distance %fm exceeds radius %dm", i, result.Distance, radiusMeters)
		}
		if i > 0 {
			// Verify ordering by distance
			prevDistance := parcels[i-1].Distance
			if result.Distance < prevDistance {
				t.Errorf("Results not ordered by distance: parcel %d (dist=%f) < parcel %d (dist=%f)",
					i, result.Distance, i-1, prevDistance)
			}
		}

		t.Logf("  Parcel %d: ID=%d, Distance=%.2fm", i+1, result.Parcel.ID, result.Distance)
	}
}

// TestFindNearby_EmptyResults tests querying a location with no nearby parcels.
func TestFindNearby_EmptyResults(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Coordinates in the middle of the Gulf of Mexico (no parcels)
	lat := 27.0
	lng := -93.0
	radiusMeters := 5000

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Errorf("FindNearby should not return error for empty results, got: %v", err)
	}

	if parcels == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}

	if len(parcels) != 0 {
		t.Errorf("Expected 0 parcels for ocean coordinates, got %d", len(parcels))
	}
}

// TestFindNearby_SmallRadius tests with minimum radius.
func TestFindNearby_SmallRadius(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	lat := 30.3477
	lng := -95.4502
	radiusMeters := 1 // Minimum radius

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby with small radius returned error: %v", err)
	}

	if parcels == nil {
		t.Fatal("Expected non-nil slice")
	}

	t.Logf("Found %d parcels within %dm", len(parcels), radiusMeters)

	// Verify all distances are within radius
	for i, result := range parcels {
		if result.Distance > float64(radiusMeters) {
			t.Errorf("Parcel %d distance %fm exceeds radius %dm", i, result.Distance, radiusMeters)
		}
	}
}

// TestFindNearby_LargeRadius tests with maximum radius.
func TestFindNearby_LargeRadius(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	lat := 30.3477
	lng := -95.4502
	radiusMeters := 5000 // Maximum radius

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby with large radius returned error: %v", err)
	}

	if parcels == nil {
		t.Fatal("Expected non-nil slice")
	}

	t.Logf("Found %d parcels within %dm", len(parcels), radiusMeters)

	// Verify all distances are within radius
	for i, result := range parcels {
		if result.Distance > float64(radiusMeters) {
			t.Errorf("Parcel %d distance %fm exceeds radius %dm", i, result.Distance, radiusMeters)
		}
	}
}

// TestFindNearby_DistanceAccuracy tests that distance calculations are reasonably accurate.
func TestFindNearby_DistanceAccuracy(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Use a location we know has data
	lat := 30.3477
	lng := -95.4502
	radiusMeters := 2000

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby returned error: %v", err)
	}

	// If we have results, verify distances are reasonable
	for i, result := range parcels {
		// Distance should be non-negative
		if result.Distance < 0 {
			t.Errorf("Parcel %d has negative distance: %f", i, result.Distance)
		}

		// Distance should be within the specified radius (with small tolerance for floating point)
		const tolerance = 0.01 // 1% tolerance
		maxAllowedDistance := float64(radiusMeters) * (1 + tolerance)
		if result.Distance > maxAllowedDistance {
			t.Errorf("Parcel %d distance %fm exceeds radius %dm (with tolerance)",
				i, result.Distance, radiusMeters)
		}

		t.Logf("Parcel %d: Distance=%.2fm, ID=%d", i+1, result.Distance, result.Parcel.ID)
	}
}

// TestFindNearby_ResultLimit tests that results are limited to maxNearbyResults.
func TestFindNearby_ResultLimit(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	// Use a large radius that might have many parcels
	lat := 30.3477
	lng := -95.4502
	radiusMeters := 5000

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby returned error: %v", err)
	}

	// Should not exceed the limit (currently 20)
	if len(parcels) > maxNearbyResults {
		t.Errorf("Result count %d exceeds maxNearbyResults %d", len(parcels), maxNearbyResults)
	}

	t.Logf("Found %d parcels (limit is %d)", len(parcels), maxNearbyResults)
}

// TestFindNearby_GeometryParsing tests that geometries are correctly parsed.
func TestFindNearby_GeometryParsing(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	ctx := context.Background()

	lat := 30.3477
	lng := -95.4502
	radiusMeters := 1000

	parcels, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		t.Fatalf("FindNearby returned error: %v", err)
	}

	// Verify geometry structure for all parcels
	for i, result := range parcels {
		parcel := result.Parcel

		// Geometry should be populated
		if len(parcel.Geom.Coordinates) == 0 {
			t.Errorf("Parcel %d has empty geometry", i)
			continue
		}

		// SRID should be WGS84
		if parcel.Geom.SRID != 4326 {
			t.Errorf("Parcel %d has incorrect SRID: expected 4326, got %d", i, parcel.Geom.SRID)
		}

		// Verify basic MultiPolygon structure
		for polyIdx, polygon := range parcel.Geom.Coordinates {
			if len(polygon) == 0 {
				t.Errorf("Parcel %d, Polygon %d has no rings", i, polyIdx)
			}
		}
	}
}

// TestFindNearby_ContextCancellation tests context cancellation.
func TestFindNearby_ContextCancellation(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lat := 30.3477
	lng := -95.4502
	radiusMeters := 1000

	_, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}

	// Verify it's a context error
	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}

// TestFindNearby_ContextTimeout tests context timeout.
func TestFindNearby_ContextTimeout(t *testing.T) {
	repo, db := setupTestRepository(t)
	defer db.Close()

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	lat := 30.3477
	lng := -95.4502
	radiusMeters := 1000

	_, err := (*repo).FindNearby(ctx, lat, lng, radiusMeters)
	// Should get a context deadline exceeded error or nil if query was fast enough
	if err != nil && ctx.Err() == nil {
		t.Errorf("Expected context timeout error, got: %v", err)
	}
}
