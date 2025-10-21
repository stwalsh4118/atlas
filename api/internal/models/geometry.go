package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Polygon represents a PostGIS Polygon geometry.
// It stores coordinates in GeoJSON format: [rings][points][lon,lat]
// SRID 4326 (WGS84) is used for lat/lng coordinates.
type Polygon struct {
	Coordinates [][][2]float64 // GeoJSON coordinate structure
	SRID        int            // Spatial Reference ID (default: 4326)
}

// Scan implements sql.Scanner interface for reading polygon geometry from database.
// PostGIS returns geometry data which we parse as GeoJSON.
// This is typically called when GORM reads from the database with ST_AsGeoJSON.
func (p *Polygon) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	// PostGIS with ST_AsGeoJSON returns JSON as []byte
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Polygon: expected []byte, got %T", value)
	}

	// Parse GeoJSON geometry structure
	var geom struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}

	if err := json.Unmarshal(bytes, &geom); err != nil {
		return fmt.Errorf("failed to unmarshal polygon geometry: %w", err)
	}

	if geom.Type != "Polygon" {
		return fmt.Errorf("expected Polygon type, got %s", geom.Type)
	}

	p.Coordinates = geom.Coordinates
	p.SRID = 4326 // Default to WGS84

	return nil
}

// Value implements driver.Valuer interface for writing polygon geometry to database.
// Returns GeoJSON string to be used with ST_GeomFromGeoJSON in raw SQL queries.
func (p Polygon) Value() (driver.Value, error) {
	if len(p.Coordinates) == 0 {
		return nil, nil
	}

	// Convert to GeoJSON format
	geom := map[string]interface{}{
		"type":        "Polygon",
		"coordinates": p.Coordinates,
	}

	geoJSON, err := json.Marshal(geom)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal polygon to GeoJSON: %w", err)
	}

	// Return as string for use with ST_GeomFromGeoJSON
	return string(geoJSON), nil
}

// MarshalJSON implements json.Marshaler for API responses.
// Returns GeoJSON-compliant format for frontend consumption.
func (p Polygon) MarshalJSON() ([]byte, error) {
	geom := struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}{
		Type:        "Polygon",
		Coordinates: p.Coordinates,
	}
	return json.Marshal(geom)
}

// UnmarshalJSON implements json.Unmarshaler for parsing GeoJSON input.
// Used when parsing Montgomery County GeoJSON data.
func (p *Polygon) UnmarshalJSON(data []byte) error {
	var geom struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}

	if err := json.Unmarshal(data, &geom); err != nil {
		return fmt.Errorf("failed to unmarshal polygon: %w", err)
	}

	if geom.Type != "" && geom.Type != "Polygon" {
		return fmt.Errorf("expected Polygon type, got %s", geom.Type)
	}

	p.Coordinates = geom.Coordinates
	p.SRID = 4326

	return nil
}

// MultiPolygon represents a PostGIS MultiPolygon geometry.
// It stores coordinates in GeoJSON format: [polygons][rings][points][lon,lat]
// SRID 4326 (WGS84) is used for lat/lng coordinates.
// This is used for parcels that consist of multiple separate polygons.
type MultiPolygon struct {
	Coordinates [][][][2]float64 // GeoJSON coordinate structure for MultiPolygon
	SRID        int              // Spatial Reference ID (default: 4326)
}

// Scan implements sql.Scanner interface for reading multipolygon geometry from database.
// PostGIS returns geometry data which we parse as GeoJSON.
func (mp *MultiPolygon) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	// PostGIS with ST_AsGeoJSON returns JSON as []byte
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan MultiPolygon: expected []byte, got %T", value)
	}

	// Parse GeoJSON geometry structure
	var geom struct {
		Type        string           `json:"type"`
		Coordinates [][][][2]float64 `json:"coordinates"`
	}

	if err := json.Unmarshal(bytes, &geom); err != nil {
		return fmt.Errorf("failed to unmarshal multipolygon geometry: %w", err)
	}

	if geom.Type != "MultiPolygon" {
		return fmt.Errorf("expected MultiPolygon type, got %s", geom.Type)
	}

	mp.Coordinates = geom.Coordinates
	mp.SRID = 4326 // Default to WGS84

	return nil
}

// Value implements driver.Valuer interface for writing multipolygon geometry to database.
// Returns GeoJSON string to be used with ST_GeomFromGeoJSON in raw SQL queries.
func (mp MultiPolygon) Value() (driver.Value, error) {
	if len(mp.Coordinates) == 0 {
		return nil, nil
	}

	// Convert to GeoJSON format
	geom := map[string]interface{}{
		"type":        "MultiPolygon",
		"coordinates": mp.Coordinates,
	}

	geoJSON, err := json.Marshal(geom)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal multipolygon to GeoJSON: %w", err)
	}

	// Return as string for use with ST_GeomFromGeoJSON
	return string(geoJSON), nil
}

// MarshalJSON implements json.Marshaler for API responses.
// Returns GeoJSON-compliant format for frontend consumption.
func (mp MultiPolygon) MarshalJSON() ([]byte, error) {
	geom := struct {
		Type        string           `json:"type"`
		Coordinates [][][][2]float64 `json:"coordinates"`
	}{
		Type:        "MultiPolygon",
		Coordinates: mp.Coordinates,
	}
	return json.Marshal(geom)
}

// UnmarshalJSON implements json.Unmarshaler for parsing GeoJSON input.
// Used when parsing Montgomery County GeoJSON data.
func (mp *MultiPolygon) UnmarshalJSON(data []byte) error {
	var geom struct {
		Type        string           `json:"type"`
		Coordinates [][][][2]float64 `json:"coordinates"`
	}

	if err := json.Unmarshal(data, &geom); err != nil {
		return fmt.Errorf("failed to unmarshal multipolygon: %w", err)
	}

	if geom.Type != "" && geom.Type != "MultiPolygon" {
		return fmt.Errorf("expected MultiPolygon type, got %s", geom.Type)
	}

	mp.Coordinates = geom.Coordinates
	mp.SRID = 4326

	return nil
}
