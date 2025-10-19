# PBI-2: Database Schema and Spatial Indexing

[View in Backlog](../backlog.md#user-content-2)

## Overview

Design and implement the PostgreSQL database schema with PostGIS spatial types, including the tax_parcels table, spatial indexes, and a migration system to manage schema changes over time.

## Problem Statement

To store and efficiently query property boundary data, we need:
- A properly designed table schema that can handle MultiPolygon geometries
- Spatial indexes (GiST) to enable fast point-in-polygon queries (< 100ms)
- Standard indexes for common query patterns (owner name, parcel ID)
- A migration system to version-control schema changes
- Validation that PostGIS is properly configured and performant

## User Stories

- **US-DEV-5**: As a developer, I want a tax_parcels table with spatial geometry support so that I can store property boundaries
- **US-DEV-6**: As a developer, I want spatial indexes created so that queries complete in under 100ms
- **US-DEV-7**: As a developer, I want a migration system so that schema changes are version-controlled
- **US-DEV-8**: As a developer, I want standard indexes on owner_name and parcel_id so that search queries are fast

## Technical Approach

### Schema Design

**tax_parcels table**
```sql
CREATE TABLE tax_parcels (
    id BIGSERIAL PRIMARY KEY,
    parcel_id VARCHAR(50) UNIQUE NOT NULL,
    owner_name VARCHAR(255),
    situs_address VARCHAR(500),
    acres NUMERIC(10, 2),
    prop_type VARCHAR(50),
    land_use VARCHAR(100),
    county_name VARCHAR(100),
    geom GEOMETRY(MultiPolygon, 4326) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Spatial index (most critical for performance)
CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);

-- Standard indexes for queries
CREATE INDEX idx_parcels_parcel_id ON tax_parcels(parcel_id);
CREATE INDEX idx_parcels_owner ON tax_parcels(owner_name);
CREATE INDEX idx_parcels_county ON tax_parcels(county_name);
```

### Migration Strategy

Options to evaluate:
1. **golang-migrate/migrate** - Popular, database-agnostic
2. **goose** - Simple, supports SQL and Go migrations
3. **atlas** - Modern, declarative approach
4. **Custom SQL scripts** - Simple numbered files

Recommendation: Start with golang-migrate for industry-standard approach

### Performance Considerations

- GiST index on geometry column is critical
- VACUUM ANALYZE after bulk imports
- Consider partial indexes if querying specific counties frequently
- Monitor query plans with EXPLAIN ANALYZE

### Spatial Reference System

- **SRID 4326** (WGS84): Standard lat/lng coordinates
- All imported data must be transformed to SRID 4326
- Consistent with GeoJSON standard for frontend

## UX/UI Considerations

N/A - Database layer only

## Acceptance Criteria

1. ✅ PostGIS extension is enabled in the database
2. ✅ tax_parcels table created with correct schema
3. ✅ Geometry column uses SRID 4326 (WGS84)
4. ✅ Geometry column enforces MultiPolygon type
5. ✅ GiST spatial index created on geom column
6. ✅ Standard indexes created on parcel_id, owner_name, county_name
7. ✅ Migration system configured and documented
8. ✅ Initial migration file creates tax_parcels table
9. ✅ Can insert test geometry data successfully
10. ✅ Can query using ST_Contains with < 10ms response (empty table)
11. ✅ Documentation of schema design decisions
12. ✅ VACUUM ANALYZE runs successfully on table

## Dependencies

- PBI-1: Project Infrastructure Setup (PostgreSQL 18.0 must be running)
- PostGIS 3.5 extension available
- Migration tool selected and installed

## Open Questions

1. Should we support multiple geometry types (Polygon + MultiPolygon) or enforce MultiPolygon only?
2. Do we need a separate table for county metadata?
3. Should we add a full-text search index on addresses?
4. Do we need soft-delete support or is hard delete acceptable?
5. Should we track data source/import metadata per parcel?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

