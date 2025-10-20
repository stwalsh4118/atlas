# GORM + PostGIS Integration Guide

**Created**: 2025-10-20  
**For**: Task 2-1 (PBI 2: Database Schema and Spatial Indexing)  
**Source Research**: GORM documentation, PostGIS documentation, gormgis library, community patterns

## Overview

This guide documents the approach for integrating GORM with PostGIS in the Atlas property boundary viewer backend. GORM does not have native PostGIS support, so we use a hybrid approach:

- **GORM** for standard CRUD operations on non-spatial columns
- **Custom Geometry Types** with Scanner/Valuer interfaces for geometry columns
- **Raw SQL** for spatial queries (ST_Contains, ST_Intersects, etc.)

## Approach Summary

1. **Custom `Polygon` type** with Scanner/Valuer interfaces for geometry handling
2. **GORM Raw SQL** for all spatial operations (ST_GeomFromGeoJSON, ST_Contains, etc.)
3. **GORM ORM methods** for standard CRUD on non-spatial columns
4. **golang-migrate** for database migrations (GORM-agnostic)
5. **GeoJSON parsing** from Montgomery County data format

This approach provides full control, no external geometry libraries, and transparent SQL.

## Custom Polygon Type (Primary Approach)

We'll implement a custom Polygon type that works seamlessly with GORM and PostGIS using raw SQL.

### Custom Type Implementation

```go
package models

import (
    "database/sql/driver"
    "encoding/json"
    "fmt"
)

// Polygon represents a PostGIS Polygon geometry
type Polygon struct {
    Coordinates [][][2]float64 // [rings][points][lon,lat]
    SRID        int             // Spatial Reference ID (4326 for WGS84)
}

// Scan implements sql.Scanner interface for reading from database
func (p *Polygon) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    
    // PostGIS returns geometry as WKB (Well-Known Binary) or EWKB
    // We'll use ST_AsGeoJSON in queries to get JSON format
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("failed to scan Polygon: expected []byte, got %T", value)
    }
    
    // Parse GeoJSON geometry
    var geom struct {
        Type        string          `json:"type"`
        Coordinates [][][2]float64 `json:"coordinates"`
    }
    
    if err := json.Unmarshal(bytes, &geom); err != nil {
        return fmt.Errorf("failed to unmarshal polygon geometry: %w", err)
    }
    
    p.Coordinates = geom.Coordinates
    p.SRID = 4326 // Default to WGS84
    
    return nil
}

// Value implements driver.Valuer interface for writing to database
func (p Polygon) Value() (driver.Value, error) {
    if p.Coordinates == nil || len(p.Coordinates) == 0 {
        return nil, nil
    }
    
    // Convert to GeoJSON string for ST_GeomFromGeoJSON
    geom := map[string]interface{}{
        "type":        "Polygon",
        "coordinates": p.Coordinates,
    }
    
    geoJSON, err := json.Marshal(geom)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal polygon to GeoJSON: %w", err)
    }
    
    // Return as string for use with ST_GeomFromGeoJSON in raw SQL
    return string(geoJSON), nil
}
```

### TaxParcel Model with Custom Polygon Type

```go
package models

import (
    "time"
)

type TaxParcel struct {
    ID                    uint      `gorm:"primaryKey"`
    ObjectID              int       `gorm:"uniqueIndex;not null;column:object_id"`
    PIN                   int       `gorm:"index;not null;column:pin"`
    PID                   *int      `gorm:"column:pid"`
    
    // Parcel subdivision info
    StateCd               *string   `gorm:"size:10;column:state_cd"`
    Block                 *int      `gorm:"column:block"`
    Lot                   *string   `gorm:"size:50;column:lot"`
    Tract                 *string   `gorm:"size:50;column:tract"`
    
    // Owner information
    OwnerName             *string   `gorm:"size:500;index;column:owner_name"`
    OwnerAddress          *string   `gorm:"type:text;column:owner_address"`
    
    // Property details
    Situs                 *string   `gorm:"size:500;index;column:situs"`
    AsCode                *string   `gorm:"size:50;column:as_code"`
    LegalDescription      *string   `gorm:"type:text;column:legal_description"`
    
    // Improvement/building info
    ImprvActualYearBuilt  *int      `gorm:"column:imprv_actual_year_built"`
    ImprvMainArea         *int      `gorm:"column:imprv_main_area"`
    
    // Tax information
    PYear                 *int      `gorm:"column:p_year"`
    PVersion              *int      `gorm:"column:p_version"`
    PRollCorr             *int      `gorm:"column:p_roll_corr"`
    TaxingUnits           *string   `gorm:"size:255;column:taxing_units"`
    Exemptions            *string   `gorm:"size:255;column:exemptions"`
    MarketArea            *string   `gorm:"size:50;column:market_area"`
    
    // County metadata
    CountyName            string    `gorm:"size:100;default:'Montgomery';index;column:county_name"`
    
    // Spatial data - custom type
    Geom                  Polygon   `gorm:"type:geometry(Polygon,4326);not null;column:geom"`
    
    // Timestamps
    CreatedAt             time.Time `gorm:"column:created_at"`
    UpdatedAt             time.Time `gorm:"column:updated_at"`
}

func (TaxParcel) TableName() string {
    return "tax_parcels"
}
```

### CRUD Operations with Raw SQL

```go
package repository

import (
    "fmt"
    "your-project/internal/models"
    "gorm.io/gorm"
)

type ParcelRepository struct {
    db *gorm.DB
}

func NewParcelRepository(db *gorm.DB) *ParcelRepository {
    return &ParcelRepository{db: db}
}

// Create - Insert parcel with geometry using raw SQL
func (r *ParcelRepository) Create(parcel *models.TaxParcel) error {
    // Convert Polygon to GeoJSON string for ST_GeomFromGeoJSON
    geomJSON, err := parcel.Geom.Value()
    if err != nil {
        return fmt.Errorf("failed to convert geometry: %w", err)
    }
    
    result := r.db.Exec(`
        INSERT INTO tax_parcels (
            object_id, pin, pid, state_cd, block, lot, tract,
            owner_name, owner_address, situs, as_code, legal_description,
            imprv_actual_year_built, imprv_main_area,
            p_year, p_version, p_roll_corr,
            taxing_units, exemptions, market_area, county_name,
            geom, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7,
            $8, $9, $10, $11, $12,
            $13, $14,
            $15, $16, $17,
            $18, $19, $20, $21,
            ST_GeomFromGeoJSON($22), NOW(), NOW()
        )
        RETURNING id, created_at, updated_at
    `,
        parcel.ObjectID, parcel.PIN, parcel.PID, parcel.StateCd, parcel.Block, parcel.Lot, parcel.Tract,
        parcel.OwnerName, parcel.OwnerAddress, parcel.Situs, parcel.AsCode, parcel.LegalDescription,
        parcel.ImprvActualYearBuilt, parcel.ImprvMainArea,
        parcel.PYear, parcel.PVersion, parcel.PRollCorr,
        parcel.TaxingUnits, parcel.Exemptions, parcel.MarketArea, parcel.CountyName,
        geomJSON,
    ).Scan(&parcel.ID, &parcel.CreatedAt, &parcel.UpdatedAt)
    
    return result.Error
}

// FindByPIN - Query by PIN using GORM's Raw with geometry as GeoJSON
func (r *ParcelRepository) FindByPIN(pin int) (*models.TaxParcel, error) {
    var parcel models.TaxParcel
    
    err := r.db.Raw(`
        SELECT 
            id, object_id, pin, pid, state_cd, block, lot, tract,
            owner_name, owner_address, situs, as_code, legal_description,
            imprv_actual_year_built, imprv_main_area,
            p_year, p_version, p_roll_corr,
            taxing_units, exemptions, market_area, county_name,
            ST_AsGeoJSON(geom) as geom,
            created_at, updated_at
        FROM tax_parcels
        WHERE pin = $1
    `, pin).Scan(&parcel).Error
    
    if err != nil {
        return nil, err
    }
    
    return &parcel, nil
}

// Update - Update non-spatial fields using GORM ORM methods
func (r *ParcelRepository) Update(parcel *models.TaxParcel) error {
    // For non-spatial updates, use GORM's Update method
    // This will automatically handle updated_at timestamp
    return r.db.Model(parcel).Updates(map[string]interface{}{
        "owner_name":    parcel.OwnerName,
        "owner_address": parcel.OwnerAddress,
        "situs":         parcel.Situs,
        // ... other fields as needed
    }).Error
}

// Delete - Use GORM ORM for deletion
func (r *ParcelRepository) Delete(id uint) error {
    return r.db.Delete(&models.TaxParcel{}, id).Error
}
```

## Parsing Montgomery County GeoJSON

### GeoJSON Feature Structure

```json
{
    "type": "Feature",
    "id": 31045124,
    "geometry": {
        "type": "Polygon",
        "coordinates": [[
            [-95.2298408276366, 30.0650740615456],
            [-95.2301885654377, 30.0650735884546],
            [-95.2301888621678, 30.0652385718926],
            [-95.2298411237739, 30.0652390449844],
            [-95.2298408276366, 30.0650740615456]
        ]]
    },
    "properties": {
        "OBJECTID": 31045124,
        "PIN": 334882,
        "stateCd": "A1",
        "Block": 2,
        "Lot": "15",
        "ownerName": "SMITH, STERLING RON",
        ...
    }
}
```

### Parsing Code

```go
package importer

import (
    "encoding/json"
    "os"
)

type GeoJSONFeature struct {
    Type       string                 `json:"type"`
    ID         int                    `json:"id"`
    Geometry   GeoJSONGeometry        `json:"geometry"`
    Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
    Type        string          `json:"type"`
    Coordinates [][][2]float64  `json:"coordinates"`
}

type GeoJSONFeatureCollection struct {
    Type     string             `json:"type"`
    CRS      map[string]interface{} `json:"crs"`
    Features []GeoJSONFeature   `json:"features"`
}

// ParseGeoJSONFile reads Montgomery County GeoJSON file
func ParseGeoJSONFile(filename string) (*GeoJSONFeatureCollection, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var collection GeoJSONFeatureCollection
    if err := json.Unmarshal(data, &collection); err != nil {
        return nil, err
    }
    
    return &collection, nil
}

// ConvertFeatureToTaxParcel converts a GeoJSON feature to TaxParcel model
func ConvertFeatureToTaxParcel(feature GeoJSONFeature) *models.TaxParcel {
    props := feature.Properties
    
    parcel := &models.TaxParcel{
        ObjectID:             getInt(props, "OBJECTID"),
        PIN:                  getInt(props, "PIN"),
        PID:                  getIntPtr(props, "pid"),
        StateCd:              getStringPtr(props, "stateCd"),
        Block:                getIntPtr(props, "Block"),
        Lot:                  getStringPtr(props, "Lot"),
        Tract:                getStringPtr(props, "Tract"),
        OwnerName:            getStringPtr(props, "ownerName"),
        OwnerAddress:         getStringPtr(props, "ownerAddress"),
        Situs:                getStringPtr(props, "situs"),
        AsCode:               getStringPtr(props, "asCode"),
        LegalDescription:     getStringPtr(props, "legalDescription"),
        ImprvActualYearBuilt: getIntPtr(props, "imprvActualYearBuilt"),
        ImprvMainArea:        getIntPtr(props, "imprvMainArea"),
        PYear:                getIntPtr(props, "pYear"),
        PVersion:             getIntPtr(props, "pVersion"),
        PRollCorr:            getIntPtr(props, "pRollCorr"),
        TaxingUnits:          getStringPtr(props, "taxingUnits"),
        Exemptions:           getStringPtr(props, "exemptions"),
        MarketArea:           getStringPtr(props, "marketArea"),
        CountyName:           "Montgomery",
        Geom: models.Polygon{
            Coordinates: feature.Geometry.Coordinates,
            SRID:        4326,
        },
    }
    
    return parcel
}

// Helper functions to safely extract values from map
func getInt(m map[string]interface{}, key string) int {
    if val, ok := m[key]; ok {
        if intVal, ok := val.(float64); ok {
            return int(intVal)
        }
    }
    return 0
}

func getIntPtr(m map[string]interface{}, key string) *int {
    if val, ok := m[key]; ok && val != nil {
        if intVal, ok := val.(float64); ok {
            result := int(intVal)
            return &result
        }
    }
    return nil
}

func getStringPtr(m map[string]interface{}, key string) *string {
    if val, ok := m[key]; ok && val != nil {
        if strVal, ok := val.(string); ok && strVal != "" {
            return &strVal
        }
    }
    return nil
}
```

## Spatial Queries with Raw SQL

GORM is used for CRUD, but spatial queries require raw SQL with PostGIS functions.

### Example: Point-in-Polygon Query

```go
// Find parcel containing a specific point (lat/lng from map click)
func FindParcelByPoint(db *gorm.DB, longitude, latitude float64) (*TaxParcel, error) {
    var parcel TaxParcel
    
    err := db.Raw(`
        SELECT 
            id, object_id, pin, owner_name, situs,
            imprv_main_area, imprv_actual_year_built,
            ST_AsGeoJSON(geom) as geom
        FROM tax_parcels
        WHERE ST_Contains(
            geom, 
            ST_SetSRID(ST_Point($1, $2), 4326)
        )
        LIMIT 1
    `, longitude, latitude).Scan(&parcel).Error
    
    if err != nil {
        return nil, err
    }
    
    return &parcel, nil
}
```

### Example: Bounding Box Query

```go
// Find parcels within a bounding box (map viewport)
func FindParcelsInBounds(db *gorm.DB, minLon, minLat, maxLon, maxLat float64) ([]TaxParcel, error) {
    var parcels []TaxParcel
    
    err := db.Raw(`
        SELECT 
            id, object_id, pin, owner_name, situs,
            ST_AsGeoJSON(geom) as geom
        FROM tax_parcels
        WHERE ST_Intersects(
            geom,
            ST_MakeEnvelope($1, $2, $3, $4, 4326)
        )
        LIMIT 100
    `, minLon, minLat, maxLon, maxLat).Scan(&parcels).Error
    
    if err != nil {
        return nil, err
    }
    
    return parcels, nil
}
```

## Migrations with golang-migrate

golang-migrate is GORM-agnostic and works directly with SQL files.

### Migration Structure

```
api/migrations/
├── 000001_enable_postgis.up.sql
├── 000001_enable_postgis.down.sql
├── 000002_create_tax_parcels_table.up.sql
├── 000002_create_tax_parcels_table.down.sql
├── 000003_create_parcel_indexes.up.sql
└── 000003_create_parcel_indexes.down.sql
```

### Example Migration (Create Table)

```sql
-- 000002_create_tax_parcels_table.up.sql
CREATE TABLE tax_parcels (
    id BIGSERIAL PRIMARY KEY,
    object_id INTEGER UNIQUE NOT NULL,
    pin INTEGER NOT NULL,
    -- ... other columns
    geom GEOMETRY(Polygon, 4326) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);
```

### Running Migrations

```bash
# Create new migration
migrate create -ext sql -dir api/migrations -seq migration_name

# Run migrations
migrate -path api/migrations -database "postgresql://user:pass@localhost:5432/atlas?sslmode=disable" up

# Rollback
migrate -path api/migrations -database "postgresql://user:pass@localhost:5432/atlas?sslmode=disable" down 1
```

## Performance Considerations

1. **GiST Index**: Critical for spatial queries
   ```sql
   CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);
   ```

2. **Partial Indexes**: For frequently queried counties
   ```sql
   CREATE INDEX idx_parcels_montgomery ON tax_parcels (county_name) 
   WHERE county_name = 'Montgomery';
   ```

3. **VACUUM ANALYZE**: After bulk imports
   ```sql
   VACUUM ANALYZE tax_parcels;
   ```

4. **Query Planning**: Always check with EXPLAIN ANALYZE
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM tax_parcels
   WHERE ST_Contains(geom, ST_Point(-95.228, 30.065, 4326));
   ```

## References

- [GORM Documentation](https://gorm.io/docs/)
- [PostGIS Documentation](https://postgis.net/documentation/)
- [gormgis Library](https://pkg.go.dev/github.com/9ssi7/gormgis)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Atlas GORM Extensions Guide](https://atlasgo.io/guides/orms/gorm/extensions)
- [Montgomery County Data Format](../../../data/montgomery-county-sample.geojson)

## Summary

**Primary Approach: Custom Polygon Type + GORM Raw SQL**

This approach provides:
- ✅ **Full control** over PostGIS queries with transparent SQL
- ✅ **No external dependencies** beyond GORM itself
- ✅ **Flexibility** to use any PostGIS function (ST_Contains, ST_Intersects, ST_Buffer, etc.)
- ✅ **GORM benefits** for non-spatial operations (migrations, connection pooling, etc.)
- ✅ **Type safety** with custom Polygon type implementing Scanner/Valuer

**When to use GORM ORM vs Raw SQL:**
- **GORM ORM**: Non-spatial CRUD (Create with only attributes, Update non-spatial fields, Delete, simple queries)
- **Raw SQL**: Any operation involving geometry (Insert with geom, spatial queries, spatial joins)

This hybrid approach gives the best of both worlds: GORM's convenience for standard operations and raw SQL power for spatial operations.

