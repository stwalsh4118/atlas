package models

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// TestPolygonImplementsInterfaces verifies Polygon implements required interfaces
func TestPolygonImplementsInterfaces(t *testing.T) {
	var _ driver.Valuer = Polygon{}
	var _ driver.Valuer = (*Polygon)(nil)

	// sql.Scanner requires a pointer receiver
	var p Polygon
	var scanner interface{} = &p
	if _, ok := scanner.(interface{ Scan(interface{}) error }); !ok {
		t.Error("Polygon does not implement sql.Scanner interface")
	}
}

// TestPolygonValue tests the Value method (writing to database)
func TestPolygonValue(t *testing.T) {
	tests := []struct {
		name      string
		polygon   Polygon
		wantNil   bool
		wantError bool
	}{
		{
			name: "valid polygon",
			polygon: Polygon{
				Coordinates: [][][2]float64{
					{{-95.5, 30.2}, {-95.4, 30.2}, {-95.4, 30.3}, {-95.5, 30.3}, {-95.5, 30.2}},
				},
				SRID: 4326,
			},
			wantNil:   false,
			wantError: false,
		},
		{
			name:      "empty polygon",
			polygon:   Polygon{},
			wantNil:   true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.polygon.Value()

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && val != nil {
				t.Errorf("expected nil value, got %v", val)
			}
			if !tt.wantNil && val == nil {
				t.Error("expected non-nil value, got nil")
			}

			// For valid polygon, verify it's valid GeoJSON
			if !tt.wantNil && !tt.wantError && val != nil {
				var geom map[string]interface{}
				if err := json.Unmarshal([]byte(val.(string)), &geom); err != nil {
					t.Errorf("Value() did not return valid JSON: %v", err)
				}
				if geom["type"] != "Polygon" {
					t.Errorf("expected type=Polygon, got %v", geom["type"])
				}
			}
		})
	}
}

// TestPolygonScan tests the Scan method (reading from database)
func TestPolygonScan(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantError bool
		wantNil   bool
	}{
		{
			name:      "nil value",
			input:     nil,
			wantError: false,
			wantNil:   true,
		},
		{
			name:      "valid GeoJSON",
			input:     []byte(`{"type":"Polygon","coordinates":[[[-95.5,30.2],[-95.4,30.2],[-95.4,30.3],[-95.5,30.3],[-95.5,30.2]]]}`),
			wantError: false,
			wantNil:   false,
		},
		{
			name:      "invalid JSON",
			input:     []byte(`{invalid}`),
			wantError: true,
			wantNil:   false,
		},
		{
			name:      "wrong type",
			input:     []byte(`{"type":"Point","coordinates":[0,0]}`),
			wantError: true,
			wantNil:   false,
		},
		{
			name:      "unsupported input type",
			input:     "not a byte slice",
			wantError: true,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Polygon
			err := p.Scan(tt.input)

			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantError && !tt.wantNil {
				if len(p.Coordinates) == 0 {
					t.Error("expected coordinates to be populated")
				}
				if p.SRID != 4326 {
					t.Errorf("expected SRID 4326, got %d", p.SRID)
				}
			}
		})
	}
}

// TestPolygonJSON tests JSON marshaling/unmarshaling
func TestPolygonJSON(t *testing.T) {
	original := Polygon{
		Coordinates: [][][2]float64{
			{{-95.5, 30.2}, {-95.4, 30.2}, {-95.4, 30.3}, {-95.5, 30.3}, {-95.5, 30.2}},
		},
		SRID: 4326,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var decoded Polygon
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify structure
	if len(decoded.Coordinates) != len(original.Coordinates) {
		t.Errorf("coordinate ring count mismatch: got %d, want %d",
			len(decoded.Coordinates), len(original.Coordinates))
	}
	if decoded.SRID != original.SRID {
		t.Errorf("SRID mismatch: got %d, want %d", decoded.SRID, original.SRID)
	}
}
