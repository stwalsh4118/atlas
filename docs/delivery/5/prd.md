# PBI-5: Point-in-Polygon Query API

[View in Backlog](../backlog.md#user-content-5)

## Overview

Implement the core spatial query functionality that allows users to click a location on the map and retrieve property information for that point, including optimized PostGIS queries returning results in under 100ms.

## Problem Statement

The primary user interaction is clicking anywhere on the map to discover "what property is here?" This requires:
- Fast spatial query using ST_Contains to find which parcel contains a lat/lng point
- Return property details (owner, acreage, parcel ID, etc.)
- Return GeoJSON geometry for highlighting the boundary
- Handle edge cases (no property found, multiple matches, invalid coordinates)
- Meet performance target of < 100ms response time
- Support nearby properties query for future features

## User Stories

- **US-2**: As a user, I want to click anywhere on the map to see property information at that location
- **US-3**: As a user, I want to see property boundaries highlighted when I select them so I can understand the exact extent
- **US-DEV-10**: As a developer, I want spatial queries to return results in under 100ms so the UX remains responsive
- **US-5**: As a user, I want to see the owner's name for a property so I can identify who owns the land
- **US-6**: As a user, I want to see the acreage/size of a property so I can understand its scale
- **US-7**: As a user, I want to distinguish between public and private land so I can understand access rights
- **US-8**: As a user, I want to see the parcel ID so I can reference it in county records

## Technical Approach

### Core Endpoint: Point-in-Polygon

**Endpoint**: `GET /api/v1/parcels/at-point`

**Query Parameters**:
- `lat` (required): Latitude in decimal degrees (WGS84)
- `lng` (required): Longitude in decimal degrees (WGS84)

**Example Request**:
```
GET /api/v1/parcels/at-point?lat=30.3477&lng=-95.4502
```

**Success Response** (200 OK):
```json
{
  "parcel": {
    "id": 12345,
    "parcel_id": "R123456",
    "owner_name": "John Doe",
    "situs_address": "123 Main St, Montgomery, TX",
    "acres": 5.25,
    "prop_type": "Residential",
    "land_use": "Single Family",
    "county_name": "Montgomery",
    "geometry": {
      "type": "MultiPolygon",
      "coordinates": [[[[...]]]
    }
  }
}
```

**Not Found Response** (404):
```json
{
  "error": {
    "code": "PARCEL_NOT_FOUND",
    "message": "No property found at this location"
  }
}
```

**Validation Error** (400):
```json
{
  "error": {
    "code": "INVALID_COORDINATES",
    "message": "Latitude must be between -90 and 90, longitude between -180 and 180"
  }
}
```

### PostGIS Query

```sql
SELECT 
    id,
    parcel_id,
    owner_name,
    situs_address,
    acres,
    prop_type,
    land_use,
    county_name,
    ST_AsGeoJSON(geom) as geometry
FROM tax_parcels
WHERE ST_Contains(
    geom, 
    ST_SetSRID(ST_MakePoint($1, $2), 4326)
)
LIMIT 1;
```

**Query Optimization**:
- GiST index on `geom` column is critical
- ST_Contains uses the spatial index automatically
- LIMIT 1 since a point should only be in one parcel
- Use EXPLAIN ANALYZE to verify index usage

### Secondary Endpoint: Nearby Properties

**Endpoint**: `GET /api/v1/parcels/nearby`

**Query Parameters**:
- `lat` (required): Latitude
- `lng` (required): Longitude  
- `radius` (optional): Radius in meters (default: 1000, max: 5000)

**PostGIS Query**:
```sql
SELECT 
    id,
    parcel_id,
    owner_name,
    acres,
    ST_Distance(
        geom::geography, 
        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
    ) as distance_meters,
    ST_AsGeoJSON(geom) as geometry
FROM tax_parcels
WHERE ST_DWithin(
    geom::geography,
    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
    $3
)
ORDER BY distance_meters
LIMIT 20;
```

### Tertiary Endpoint: Get Parcel by ID

**Endpoint**: `GET /api/v1/parcels/:id`

**Path Parameter**:
- `id`: Database ID (integer) or `parcel_id` (string)

**Example**: `GET /api/v1/parcels/R123456`

### Input Validation

- Latitude: -90 to 90
- Longitude: -180 to 180
- Radius: 1 to 5000 meters
- Parcel ID: Alphanumeric, max 50 chars

### Error Handling

1. **Invalid coordinates**: Return 400 with validation error
2. **No parcel found**: Return 404 with helpful message
3. **Database error**: Return 500, log error, don't expose internals
4. **Query timeout**: Set 5 second timeout, return 504
5. **Multiple results** (shouldn't happen): Return first, log warning

### Performance Monitoring

- Log query execution time for all spatial queries
- Alert if queries take > 200ms
- Track cache hit rates if caching added later
- Monitor database connection pool usage

### Repository Layer

```go
type ParcelRepository interface {
    FindByPoint(ctx context.Context, lat, lng float64) (*models.Parcel, error)
    FindNearby(ctx context.Context, lat, lng float64, radius int) ([]models.ParcelWithDistance, error)
    FindByID(ctx context.Context, id string) (*models.Parcel, error)
}
```

### Service Layer

```go
type ParcelService interface {
    GetParcelAtPoint(ctx context.Context, lat, lng float64) (*models.Parcel, error)
    GetNearbyParcels(ctx context.Context, lat, lng float64, radius int) ([]models.ParcelWithDistance, error)
    GetParcelByID(ctx context.Context, id string) (*models.Parcel, error)
}
```

## UX/UI Considerations

### Response Time
- Target: < 100ms for point queries
- Acceptable: < 200ms
- Timeout: 5 seconds

### Error Messages
- User-friendly messages for common errors
- Suggest actions: "Try clicking a different location"
- Distinguish between "no data" and "error"

### Property Type Display
- Use clear labels for prop_type
- Indicate public vs private land visually
- Format acres with 2 decimal places

## Acceptance Criteria

1. ✅ `GET /api/v1/parcels/at-point` endpoint implemented
2. ✅ Validates lat/lng parameters correctly
3. ✅ Returns 400 for invalid coordinates
4. ✅ Returns 404 when no parcel found
5. ✅ Returns 200 with parcel data when found
6. ✅ GeoJSON geometry included in response
7. ✅ Query completes in < 100ms for database with 50k parcels
8. ✅ EXPLAIN ANALYZE shows spatial index is used
9. ✅ `GET /api/v1/parcels/nearby` endpoint implemented
10. ✅ Nearby query respects radius parameter
11. ✅ Results sorted by distance
12. ✅ `GET /api/v1/parcels/:id` endpoint implemented
13. ✅ Can fetch parcel by database ID
14. ✅ Can fetch parcel by parcel_id string
15. ✅ All endpoints have proper error handling
16. ✅ Query execution time logged for each request
17. ✅ Integration tests cover happy path and error cases
18. ✅ API documentation updated with examples

## Dependencies

- PBI-2: Database Schema and Spatial Indexing (spatial index required)
- PBI-3: Data Import Pipeline (data must be loaded)
- PBI-4: Core Go API Backend (API server foundation)
- Sample parcel data imported for testing

## Open Questions

1. Should we cache frequent queries? (Probably not for MVP)
2. Do we need to support bulk point queries?
3. Should nearby query return full geometry or simplified?
4. How to handle properties that span county boundaries?
5. Should we expose raw SQL for power users?
6. Do we need audit logging for queries?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Agreed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-21

