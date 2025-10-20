# ogr2ogr Import Tool Guide

**Created**: 2025-10-20  
**Tool Version**: GDAL 3.x  
**Documentation**: https://gdal.org/programs/ogr2ogr.html

## Overview

`ogr2ogr` is a command-line utility from the GDAL/OGR library that converts vector data between different formats. It provides more flexibility than `shp2pgsql`, including built-in field renaming and SQL-based transformations during import.

## Installation

`ogr2ogr` requires GDAL installation:

```bash
# Verify installation
which ogr2ogr
ogr2ogr --version

# Ubuntu/Debian
sudo apt-get install gdal-bin

# macOS with Homebrew
brew install gdal

# Verify PostGIS driver support
ogrinfo --formats | grep -i postgres
```

## Key Features

- ✅ Built-in field renaming during import
- ✅ SQL SELECT statements for transformation
- ✅ CRS transformation
- ✅ Many input format support (not just shapefiles)
- ✅ Direct database connection
- ✅ Flexible data filtering and transformation

## Command Syntax

```bash
ogr2ogr -f PostgreSQL \
  PG:"connection_string" \
  source_file \
  [options]
```

## PostgreSQL Connection String

```bash
PG:"host=localhost port=5432 dbname=atlas user=postgres password=secret"

# Or using environment variables
export PGHOST=localhost
export PGDATABASE=atlas
export PGUSER=postgres
PG:"dbname=atlas"
```

## Essential Options for Our Use Case

### `-f PostgreSQL` - Output format
Specifies PostgreSQL as the target format.

### `-nln <table_name>` - New layer name
Sets the target table name.

```bash
ogr2ogr -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels
```

### `-t_srs EPSG:4326` - Target CRS
Transforms geometries to target coordinate reference system.

```bash
ogr2ogr -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels \
  -t_srs EPSG:4326
```

### `-s_srs EPSG:2278` - Source CRS
Explicitly sets source CRS (if .prj file is missing or incorrect).

```bash
ogr2ogr -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels \
  -s_srs EPSG:2278 \
  -t_srs EPSG:4326
```

### `-sql "SELECT ..."` - Transform during import
**THIS IS THE KEY FEATURE**: Rename fields and transform data during import.

```bash
ogr2ogr -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels \
  -t_srs EPSG:4326 \
  -sql "SELECT 
    PARCEL_ID as parcel_id,
    OWNER as owner_name,
    SITUS_ADDR as situs_address,
    ACRES as acres,
    geometry
  FROM parcels"
```

### `-lco` - Layer creation options
Controls table creation behavior.

```bash
# Create spatial index
-lco SPATIAL_INDEX=GIST

# Set geometry column name
-lco GEOMETRY_NAME=geom

# Drop table if exists
-lco OVERWRITE=YES

# Use MultiPolygon instead of Polygon
-lco GEOM_TYPE=MultiPolygon
```

### Mode Options

- `-append` - Add to existing table
- `-overwrite` - Drop and recreate table
- `-update` - Update existing database (used with -append)

## Complete Import Example with Field Mapping

### Montgomery County, TX Import with Field Renaming

```bash
#!/bin/bash

# Configuration
SHAPEFILE="./data/montgomery_county_parcels.shp"
TABLE="tax_parcels"
SOURCE_CRS="EPSG:2278"  # Texas South Central NAD83 (feet)
TARGET_CRS="EPSG:4326"  # WGS84
DB_HOST="localhost"
DB_NAME="atlas"
DB_USER="postgres"

# Single-step import with field mapping
ogr2ogr \
  -f PostgreSQL \
  PG:"host=${DB_HOST} dbname=${DB_NAME} user=${DB_USER}" \
  "${SHAPEFILE}" \
  -nln ${TABLE} \
  -s_srs ${SOURCE_CRS} \
  -t_srs ${TARGET_CRS} \
  -lco GEOMETRY_NAME=geom \
  -lco GEOM_TYPE=MultiPolygon \
  -lco SPATIAL_INDEX=GIST \
  -lco OVERWRITE=YES \
  -sql "SELECT 
    PARCEL_ID as parcel_id,
    OWNER as owner_name,
    SITUS_ADDR as situs_address,
    ACRES as acres,
    ASSESSED_VALUE as assessed_value,
    PROP_CLASS as property_class,
    geometry
  FROM 'montgomery_county_parcels'"

# Note: The FROM clause uses the layer name (filename without .shp extension)
```

## Field Mapping with Data Transformation

You can also transform data during import:

```bash
ogr2ogr -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels \
  -t_srs EPSG:4326 \
  -sql "SELECT 
    PARCEL_ID as parcel_id,
    UPPER(OWNER) as owner_name,          -- Convert to uppercase
    TRIM(SITUS_ADDR) as situs_address,   -- Trim whitespace
    CAST(ACRES AS REAL) as acres,        -- Ensure numeric type
    CASE 
      WHEN PROP_CLASS = 'R' THEN 'Residential'
      WHEN PROP_CLASS = 'C' THEN 'Commercial'
      ELSE 'Other'
    END as property_class,
    geometry
  FROM parcels"
```

## Handling Missing Fields

Use COALESCE for default values:

```bash
-sql "SELECT 
  PARCEL_ID as parcel_id,
  COALESCE(OWNER, 'Unknown') as owner_name,
  COALESCE(ACRES, 0.0) as acres,
  geometry
FROM parcels"
```

## Progress Monitoring

```bash
# Add progress reporting
ogr2ogr \
  -progress \
  -f PostgreSQL \
  PG:"dbname=atlas" \
  parcels.shp \
  -nln tax_parcels
```

## Performance Considerations

### Large Datasets

**Use transactions**:
```bash
ogr2ogr \
  -lco PG_USE_COPY=YES \
  -f PostgreSQL \
  ...
```

**Batch size**:
```bash
-gt 65536  # Set transaction size (default is 20000)
```

**Expected Performance**:
- Generally slower than shp2pgsql (10-30% overhead)
- 50k parcels: ~45-120 seconds
- 500k parcels: ~4-12 minutes
- Trade-off: convenience vs speed

## Advantages Over shp2pgsql

✅ **Field Renaming**: Built-in SQL SELECT transformation  
✅ **Data Transformation**: UPPER, TRIM, CASE, COALESCE, etc.  
✅ **Flexibility**: Many input formats beyond shapefiles  
✅ **Single Step**: No staging table needed for field mapping  
✅ **Filtering**: WHERE clauses to import subsets  

## Limitations vs shp2pgsql

⚠️ **Speed**: 10-30% slower for large imports  
⚠️ **Dependencies**: Requires GDAL installation  
⚠️ **Complexity**: More complex command syntax  
⚠️ **Debugging**: Harder to troubleshoot than SQL-based approach  

## When to Use ogr2ogr vs shp2pgsql

### Use ogr2ogr when:
- You want single-step import with field renaming
- You need data transformations during import
- You're importing non-shapefile formats
- Convenience outweighs performance needs

### Use shp2pgsql when:
- Maximum performance is critical
- You prefer SQL-based field mapping (more visible/debuggable)
- You don't need complex transformations
- You want to minimize dependencies

## Error Handling

### Common Errors

**1. PostgreSQL driver not found**
```
Unable to find driver 'PostgreSQL'
```
**Solution**: Install GDAL with PostgreSQL support

**2. Layer name mismatch in SQL**
```
ERROR: Layer 'parcels' not found
```
**Solution**: Use filename without .shp extension in FROM clause

**3. Field name not found**
```
ERROR: Field 'OWNER' not found
```
**Solution**: Check actual field names with `ogrinfo -al parcels.shp | grep`

### Inspection Commands

```bash
# List available layers
ogrinfo parcels.shp

# Show all fields and sample data
ogrinfo -al parcels.shp

# Show just field names
ogrinfo -al parcels.shp | grep -E '^\s+\w+:'

# Show CRS
ogrinfo parcels.shp -al -so | grep -A5 "Layer SRS"
```

## Validation After Import

```bash
# Verify import
psql -d atlas -c "SELECT COUNT(*) FROM tax_parcels;"

# Check SRID
psql -d atlas -c "SELECT DISTINCT ST_SRID(geom) FROM tax_parcels;"

# Check geometry types
psql -d atlas -c "SELECT DISTINCT GeometryType(geom) FROM tax_parcels;"

# Check spatial index
psql -d atlas -c "SELECT indexname FROM pg_indexes WHERE tablename = 'tax_parcels';"
```

## Recommendation for Our Project

### MVP (Task 3-4): Use shp2pgsql + staging table
**Rationale**:
- Faster performance
- Better debuggability (SQL is visible)
- Fewer dependencies
- Easier to validate each step

### Post-MVP: Consider ogr2ogr
**Rationale**:
- If field mapping becomes complex
- If we add support for non-shapefile formats
- If single-step convenience is preferred

## Integration with Our Pipeline

### Alternative Flow Using ogr2ogr

1. **Pre-validation** (Task 3-2)
   ```bash
   ogrinfo -al parcels.shp
   ```

2. **Read field mapping config** (Task 3-3)
   ```json
   {
     "parcel_id": ["PARCEL_ID", "PARCEL"],
     "owner_name": ["OWNER", "OWNERNAME"]
   }
   ```

3. **Generate SQL SELECT from mapping**
   ```bash
   SELECT 
     PARCEL_ID as parcel_id,
     OWNER as owner_name,
     ...
   ```

4. **Single import with ogr2ogr** (Task 3-4)
   ```bash
   ogr2ogr -f PostgreSQL ... -sql "..."
   ```

5. **Geometry validation** (Task 3-5)
   ```sql
   UPDATE tax_parcels SET geom = ST_MakeValid(geom) ...
   ```

6. **Post-validation** (Task 3-7)

## Example: Multi-County Configuration

```bash
# Montgomery County
ogr2ogr ... -sql "SELECT PARCEL_ID as parcel_id, OWNER as owner_name, ..."

# Harris County (different field names)
ogr2ogr ... -sql "SELECT APN as parcel_id, OWNERNAME as owner_name, ..."

# Travis County (yet different names)
ogr2ogr ... -sql "SELECT PIN as parcel_id, OWNER_NAME as owner_name, ..."
```

---

**Decision**: For MVP implementation in Task 3-4, we'll use **shp2pgsql** with staging table approach for simplicity and performance. This guide documents ogr2ogr as a reference for future enhancements.

**Next Steps**: See Task 3-2 for pre-import validation using `ogrinfo`.


