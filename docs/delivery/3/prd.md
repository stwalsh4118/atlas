# PBI-3: Data Import Pipeline

[View in Backlog](../backlog.md#user-content-3)

## Overview

Build a robust data import pipeline to load county tax parcel shapefile data into PostgreSQL, including coordinate transformation, geometry validation, and error handling.

## Problem Statement

County parcel data comes in shapefile format with varying:
- Coordinate reference systems (may not be EPSG:4326)
- Attribute field names and structures
- Geometry types (Polygon vs MultiPolygon)
- Data quality issues (invalid geometries, null values)

We need a reliable pipeline that:
- Transforms any CRS to EPSG:4326 (WGS84)
- Normalizes attribute names to our schema
- Validates and repairs geometries
- Provides clear error reporting
- Can handle 50k-500k parcels efficiently

## User Stories

- **US-DEV-9**: As a developer, I want to import county parcel shapefiles so that I can populate the database with real data
- **US-DEV-10**: As a developer, I want automatic CRS transformation so that all data is in WGS84
- **US-DEV-11**: As a developer, I want geometry validation so that invalid shapes don't break queries
- **US-DEV-12**: As a developer, I want clear error reporting so that I can fix data issues

## Technical Approach

### Import Tools Options

**Option 1: PostGIS shp2pgsql (Recommended for MVP)**
- Pros: Built-in, fast, reliable, handles CRS transformation
- Cons: Command-line only, less flexibility
- Command example:
```bash
shp2pgsql -I -s 2278:4326 -g geom parcels.shp tax_parcels | \
  psql -h localhost -d atlas -U postgres
```

**Option 2: ogr2ogr (GDAL)**
- Pros: Very flexible, handles many formats
- Cons: Requires GDAL installation
- Command example:
```bash
ogr2ogr -f PostgreSQL \
  PG:"host=localhost dbname=atlas user=postgres" \
  parcels.shp \
  -nln tax_parcels \
  -t_srs EPSG:4326
```

**Option 3: Custom Go Tool**
- Pros: Full control, can add business logic
- Cons: More development time, need to handle CRS transformation
- Consider for post-MVP if needed

### Import Pipeline Steps

1. **Pre-Import Validation**
   - Check shapefile exists and is readable
   - Identify source CRS
   - Preview field names and sample data
   - Estimate record count

2. **Field Mapping**
   - Map shapefile fields to tax_parcels columns
   - Handle different county naming conventions:
     - "OWNER" / "OWNERNAME" / "OWNER_NAME" → owner_name
     - "PARCEL" / "PARCEL_ID" / "APN" → parcel_id
     - "SITUS" / "ADDRESS" / "SITE_ADDR" → situs_address
     - "ACRES" / "ACREAGE" / "AREA_AC" → acres

3. **Geometry Processing**
   - Transform to EPSG:4326
   - Convert Polygon to MultiPolygon if needed
   - Validate geometries with ST_IsValid
   - Attempt to fix invalid geometries with ST_MakeValid

4. **Import Execution**
   - Use transactions for atomicity
   - Batch inserts for performance
   - Log progress (every 1000 records)

5. **Post-Import Validation**
   - Count imported records
   - Check for NULL geometries
   - Run VACUUM ANALYZE
   - Test sample spatial queries
   - Verify spatial index is being used

### Error Handling

- Invalid geometries: Log parcel_id and attempt ST_MakeValid
- Duplicate parcel_ids: Skip or update based on configuration
- Missing required fields: Use NULL or default values
- CRS transformation errors: Abort import with clear message

### Configuration

```bash
# Example configuration
SOURCE_SHAPEFILE="./data/montgomery_parcels.shp"
SOURCE_CRS="EPSG:2278"  # Texas South Central NAD83
TARGET_CRS="EPSG:4326"  # WGS84
DB_HOST="localhost"
DB_NAME="atlas"
DB_USER="postgres"
```

## UX/UI Considerations

N/A - Backend data pipeline only

## Acceptance Criteria

1. ✅ Import script accepts shapefile path as input
2. ✅ Script automatically detects source CRS
3. ✅ All geometries transformed to EPSG:4326
4. ✅ Polygons converted to MultiPolygon format
5. ✅ Invalid geometries are validated and fixed or logged
6. ✅ At least 10,000 parcels imported successfully
7. ✅ Import completes in reasonable time (< 5 minutes for 50k parcels)
8. ✅ Progress logging shows import status
9. ✅ Post-import validation confirms data integrity
10. ✅ Spatial index is functional after import (verify with EXPLAIN)
11. ✅ Sample point-in-polygon query returns results correctly
12. ✅ Documentation for running import with different counties
13. ✅ Error log captures any problematic records

## Dependencies

- PBI-2: Database Schema and Spatial Indexing (table must exist)
- County parcel shapefile data downloaded
- PostGIS tools (shp2pgsql) or GDAL (ogr2ogr) available
- Database credentials and connection

## Open Questions

1. Which county's data should we use for initial testing? Montgomery County, TX?
2. Should we support incremental updates or only full imports?
3. Do we need to preserve original source geometry in a separate column?
4. Should we add metadata tracking (import_date, source_file) to records?
5. How do we handle multi-county imports - separate tables or single table with county field?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

