# shp2pgsql Import Tool Guide

**Created**: 2025-10-20  
**Tool Version**: PostGIS 3.5.x  
**Documentation**: https://postgis.net/docs/using_postgis_dbmanagement.html#shp2pgsql_usage

## Overview

`shp2pgsql` is a command-line utility included with PostGIS that converts ESRI Shapefiles into SQL suitable for insertion into a PostgreSQL/PostGIS database. It's the recommended tool for our MVP import pipeline.

## Installation

`shp2pgsql` is included with PostGIS:

```bash
# Verify installation
which shp2pgsql
shp2pgsql --help

# If not installed (Ubuntu/Debian)
sudo apt-get install postgis

# If not installed (macOS with Homebrew)
brew install postgis
```

## Key Features

- ✅ Built-in CRS/SRID transformation
- ✅ Automatic spatial index creation
- ✅ Handles Polygon → MultiPolygon conversion
- ✅ Fast bulk loading via SQL
- ✅ Works seamlessly with psql pipeline
- ✅ Stable and battle-tested

## Command Syntax

```bash
shp2pgsql [OPTIONS] <shapefile> <table_name> | psql [PSQL_OPTIONS]
```

## Essential Options for Our Use Case

### `-I` - Create spatial index
Creates a GIST spatial index on the geometry column after import.

```bash
shp2pgsql -I parcels.shp tax_parcels_staging
```

**Note**: For our project, the final `tax_parcels` table already has a spatial index created by migrations, so we only use `-I` if we want an index on the temporary staging table (usually not necessary since it's short-lived).

### `-s [FROM_SRID:]TO_SRID` - Set SRID / Transform CRS
Transforms coordinates from source CRS to target CRS.

```bash
# If shapefile has SRID defined in .prj file
shp2pgsql -s 4326 parcels.shp tax_parcels

# Explicit transformation (Texas South Central NAD83 → WGS84)
shp2pgsql -s 2278:4326 parcels.shp tax_parcels
```

**Common Texas CRS Codes:**
- `EPSG:2278` - NAD83 / Texas South Central (feet)
- `EPSG:3081` - NAD83(HARN) / Texas South Central
- `EPSG:32614` - WGS 84 / UTM zone 14N
- `EPSG:4326` - WGS 84 (lat/lon) - **Our target**

### `-g <geocolumn>` - Specify geometry column name
Names the geometry column (default is 'geom').

```bash
shp2pgsql -g geom parcels.shp tax_parcels
```

### `-D` - Use PostgreSQL "dump" format
Uses COPY instead of INSERT for better performance.

```bash
shp2pgsql -D -I -s 2278:4326 parcels.shp tax_parcels
```

### Mode Options

- `-a` - **Append mode**: Add to existing table
- `-c` - **Create mode**: Create new table (default)
- `-d` - **Drop mode**: Drop table then create
- `-p` - **Prepare mode**: Only create table schema (no data)

```bash
# Drop and recreate table
shp2pgsql -d -I -s 2278:4326 parcels.shp tax_parcels

# Append to existing table
shp2pgsql -a -s 2278:4326 new_parcels.shp tax_parcels
```

## Complete Import Example

### Montgomery County, TX Import to Existing Table

**Important**: Our `tax_parcels` table already exists with indexes from migrations. We import to a staging table first, then map fields to the final table.

```bash
#!/bin/bash

# Configuration
SHAPEFILE="./data/montgomery_county_parcels.shp"
STAGING_TABLE="tax_parcels_staging"
FINAL_TABLE="tax_parcels"
SOURCE_CRS="2278"  # Texas South Central NAD83 (feet)
TARGET_CRS="4326"  # WGS84
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="atlas"
DB_USER="postgres"

# Step 1: Import shapefile to temporary staging table
# Note: -I flag is optional here since staging table is temporary
# The final tax_parcels table already has spatial index from migrations
shp2pgsql \
  -D \
  -s ${SOURCE_CRS}:${TARGET_CRS} \
  -g geom \
  -d \
  "${SHAPEFILE}" \
  ${STAGING_TABLE} | \
psql \
  -h ${DB_HOST} \
  -p ${DB_PORT} \
  -d ${DB_NAME} \
  -U ${DB_USER}

# Step 2: Map fields from staging to final table
# (See Field Mapping section below for complete SQL)

# Step 3: Drop staging table
psql -h ${DB_HOST} -d ${DB_NAME} -U ${DB_USER} -c \
  "DROP TABLE IF EXISTS ${STAGING_TABLE};"
```

## Field Name Handling

**Important**: `shp2pgsql` preserves shapefile field names exactly as they appear in the .dbf file, but:
- Converts to lowercase by default
- Preserves underscores
- May quote field names with special characters

### Example:
**Shapefile fields**: `PARCEL_ID`, `OWNER`, `SITUS_ADDR`  
**Database columns**: `parcel_id`, `owner`, `situs_addr`

### Problem with Our Schema
Our table expects: `owner_name`, `situs_address`, etc.

### Solution: Two-Stage Import
1. Import to staging table with original field names (shp2pgsql creates this)
2. Map fields via SQL INSERT INTO our existing table (created by migrations)

```sql
-- Stage 1: Import to staging (done by shp2pgsql above)

-- Stage 2: Map to final table (which already exists with proper schema & indexes)
INSERT INTO tax_parcels (
  object_id,
  pin,
  owner_name,
  situs,
  geom
  -- Map other fields as needed based on shapefile
)
SELECT 
  "OBJECTID" as object_id,
  "PIN" as pin,
  "OWNER" as owner_name,
  "SITUS" as situs,
  geom
FROM tax_parcels_staging;

-- Stage 3: Cleanup
DROP TABLE tax_parcels_staging;
```

**Note**: The `tax_parcels` table already has a spatial index from migrations (`idx_parcels_geom`), so newly inserted geometries are automatically indexed. We don't need to create indexes.

## Geometry Type Conversion

`shp2pgsql` automatically handles geometry types:

- Shapefiles store **Polygon** or **MultiPolygon**
- Output SQL creates appropriate geometry type
- Use `ST_Multi()` if you need to force MultiPolygon:

```sql
-- Ensure all geometries are MultiPolygon
UPDATE tax_parcels 
SET geom = ST_Multi(geom) 
WHERE GeometryType(geom) = 'POLYGON';
```

## Performance Considerations

### Large Datasets (50k-500k records)

**Use `-D` flag**: COPY is much faster than INSERT
```bash
shp2pgsql -D -I -s 2278:4326 parcels.shp tax_parcels
```

**Disable autocommit in psql**:
```bash
shp2pgsql ... | psql -1 -d atlas  # -1 runs as single transaction
```

**Expected Performance**:
- 50k parcels: ~30-90 seconds
- 500k parcels: ~3-8 minutes
- Depends on geometry complexity and hardware

### Memory Considerations
- `shp2pgsql` is memory-efficient (streams data)
- PostgreSQL may use significant RAM for spatial index creation
- Monitor with `psql -c "SELECT * FROM pg_stat_activity;"`

## Error Handling

### Common Errors

**1. SRID not found**
```
ERROR: SRID not found
```
**Solution**: Explicitly specify source SRID with `-s FROM:TO` syntax

**2. Invalid geometry**
```
ERROR: Geometry has invalid SRID
```
**Solution**: Run ST_MakeValid() after import (see Task 3-5)

**3. File not found**
```
Unable to open shapefile
```
**Solution**: Verify shapefile path, ensure .shp, .shx, .dbf, .prj files exist

### Validation Queries

```sql
-- Check SRID
SELECT DISTINCT ST_SRID(geom) FROM tax_parcels;

-- Check geometry types
SELECT GeometryType(geom), COUNT(*) 
FROM tax_parcels 
GROUP BY GeometryType(geom);

-- Check for invalid geometries
SELECT COUNT(*) 
FROM tax_parcels 
WHERE NOT ST_IsValid(geom);

-- Check spatial index exists
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'tax_parcels' AND indexdef LIKE '%GIST%';
```

## Integration with Our Pipeline

### Import Script Flow

1. **Pre-validation** (Task 3-2)
   - Verify shapefile exists
   - Detect source CRS from .prj file
   - Preview field names

2. **Import to staging** (Task 3-4)
   ```bash
   shp2pgsql -D -I -s ${SRC_CRS}:4326 -g geom ${FILE} staging
   ```

3. **Field mapping** (Task 3-4)
   ```sql
   INSERT INTO tax_parcels (...) SELECT ... FROM staging;
   ```

4. **Geometry validation** (Task 3-5)
   ```sql
   UPDATE tax_parcels SET geom = ST_MakeValid(geom) WHERE NOT ST_IsValid(geom);
   ```

5. **Post-validation** (Task 3-7)
   - Verify record counts
   - Test spatial queries
   - Confirm index usage

## Advantages for Our Use Case

✅ **Built-in**: No additional dependencies  
✅ **Fast**: Optimized for PostgreSQL  
✅ **Reliable**: Mature, stable tool  
✅ **Simple**: Straightforward command syntax  
✅ **CRS**: Handles transformation natively  
✅ **Indexing**: Creates spatial indexes automatically  

## Limitations

⚠️ **Field Names**: No built-in field renaming (requires staging table approach)  
⚠️ **Validation**: No built-in geometry validation (requires post-processing)  
⚠️ **Progress**: No progress bar (can pipe through `pv` for monitoring)  

## Alternative: ogr2ogr

For cases where field renaming is needed, see `3-1-ogr2ogr-guide.md`.

---

**Next Steps**: See Task 3-2 for pre-import validation script implementation.

