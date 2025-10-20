# PBI-3: Data Import Pipeline

[View in Backlog](../backlog.md#user-content-3)

## Overview

Build a robust data import pipeline to load county tax parcel shapefile data into PostgreSQL, including coordinate transformation, geometry validation, and error handling.

## Problem Statement

County parcel data comes in various geospatial formats (GeoJSON, Shapefile, Geodatabase) with varying:
- Coordinate reference systems (may not be EPSG:4326)
- Attribute field names and structures
- Geometry types (Polygon vs MultiPolygon)
- Data quality issues (invalid geometries, null values)
- File sizes (500MB-2GB+)

We need a reliable pipeline that:
- Supports multiple input formats (GeoJSON, Shapefile, etc.)
- Transforms any CRS to EPSG:4326 (WGS84)
- Normalizes attribute names to our schema
- Validates and repairs geometries
- Provides clear error reporting
- Can handle 50k-500k parcels efficiently

## User Stories

- **US-DEV-9**: As a developer, I want to import county parcel data (GeoJSON/Shapefile) so that I can populate the database with real data
- **US-DEV-10**: As a developer, I want automatic CRS transformation so that all data is in WGS84
- **US-DEV-11**: As a developer, I want geometry validation so that invalid shapes don't break queries
- **US-DEV-12**: As a developer, I want clear error reporting so that I can fix data issues

## Technical Approach

### Import Tools Options

**Option 1: ogr2ogr + shp2pgsql (Recommended for MVP)**
- **Primary**: ogr2ogr (GDAL) for GeoJSON and flexibility
- **Secondary**: shp2pgsql for shapefiles when performance is critical
- Pros: Handles many formats (GeoJSON, Shapefile, Geodatabase), CRS transformation, field mapping
- Cons: Requires GDAL installation
- Command example (GeoJSON):
```bash
ogr2ogr -f PostgreSQL \
  PG:"host=localhost dbname=atlas user=postgres" \
  parcels.geojson \
  -nln tax_parcels_staging \
  -t_srs EPSG:4326
```

**Option 2: Custom Go Tool (Post-MVP)**
- Pros: Full control, single binary, type-safe field mapping
- Cons: More development time
- Consider for future enhancement (PBI-9 or similar)

**Why GeoJSON as Primary Format:**
- Modern, widely supported format
- 4-5x smaller than shapefiles (500MB vs 2GB+ for same data)
- Single file (vs shapefile's 4+ files)
- Human-readable, easier to debug
- Native JSON format integrates well with modern tools

### Import Pipeline Steps

1. **Pre-Import Validation**
   - Check file exists and is readable (GeoJSON, Shapefile, etc.)
   - Identify source CRS
   - Preview field names and sample data
   - Estimate record count
   - Report file size and format

2. **Field Mapping**
   - Map source file fields to tax_parcels columns
   - Handle different county naming conventions:
     - "OBJECTID" / "OBJECT_ID" / "OID" → object_id
     - "PIN" / "PARCEL_ID" / "PARCEL" / "APN" → pin
     - "OWNER" / "OWNERNAME" / "OWNER_NAME" → owner_name
     - "SITUS" / "ADDRESS" / "SITE_ADDR" → situs

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
SOURCE_FILE="./data/montgomery_parcels.geojson"  # or .shp, .gdb
SOURCE_CRS="EPSG:2278"  # Texas South Central NAD83 (often auto-detected)
TARGET_CRS="EPSG:4326"  # WGS84
DB_HOST="host.docker.internal"
DB_NAME="atlas"
DB_USER="postgres"
FILE_FORMAT="GeoJSON"  # or Shapefile, Geodatabase
```

## UX/UI Considerations

N/A - Backend data pipeline only

## Acceptance Criteria

1. ✅ Import script accepts GeoJSON/Shapefile path as input
2. ✅ Script automatically detects source CRS
3. ✅ All geometries transformed to EPSG:4326
4. ✅ Polygons converted to MultiPolygon format (if needed)
5. ✅ Invalid geometries are validated and fixed or logged
6. ✅ At least 10,000 parcels imported successfully
7. ✅ Import completes in reasonable time (< 5 minutes for 50k parcels)
8. ✅ Progress logging shows import status
9. ✅ Post-import validation confirms data integrity
10. ✅ Spatial index is functional after import (verify with EXPLAIN)
11. ✅ Sample point-in-polygon query returns results correctly
12. ✅ Documentation for running import with different counties and formats
13. ✅ Error log captures any problematic records

## Dependencies

- PBI-2: Database Schema and Spatial Indexing (table must exist)
- County parcel data downloaded (GeoJSON preferred, ~500MB for Montgomery County)
- GDAL tools (ogr2ogr, ogrinfo) installed
- PostGIS tools (shp2pgsql) optional for shapefile performance
- Database credentials and connection

## Open Questions

1. ~~Which county's data should we use for initial testing?~~ **RESOLVED**: Montgomery County, TX GeoJSON (~500MB, ~50k parcels)
2. Should we support incremental updates or only full imports? (MVP: full imports only)
3. Do we need to preserve original source geometry in a separate column? (Not for MVP)
4. Should we add metadata tracking (import_date, source_file) to records? (Consider for post-MVP)
5. How do we handle multi-county imports - separate tables or single table with county field? (MVP: single county, county_name field exists for future)

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Agreed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-20

