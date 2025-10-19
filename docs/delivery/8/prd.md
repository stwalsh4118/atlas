# PBI-8: Search and Discovery Features

[View in Backlog](../backlog.md#user-content-8)

## Overview

Implement advanced search and discovery features that allow users to find properties by address, owner name, parcel ID, and discover nearby properties - enhancing the application beyond basic point-and-click interaction.

## Problem Statement

After MVP, users need more powerful ways to find properties:
- Search by specific address without needing to know exact map location
- Find all properties owned by a specific person or entity
- Look up properties by parcel ID from county records
- Discover properties within a certain distance of a point
- Get autocomplete suggestions while typing

These features transform the app from a simple query tool to a comprehensive property search platform.

## User Stories

- **US-12**: As a user, I want to search for a property by address so I can find specific locations quickly
- **US-13**: As a user, I want to find all properties within a certain distance so I can explore nearby parcels
- **US-14**: As a user, I want to search by owner name so I can find all properties owned by someone
- **US-SEARCH-1**: As a user, I want autocomplete suggestions so I can find properties faster
- **US-SEARCH-2**: As a user, I want to see search results on the map so I can visualize property locations
- **US-SEARCH-3**: As a user, I want to filter search results by property type so I can narrow down results

## Technical Approach

### New API Endpoints

**1. General Search Endpoint**

`GET /api/v1/parcels/search`

**Query Parameters**:
- `q` (required): Search query string
- `type` (optional): Search type - `address`, `owner`, `parcel_id`, or `all` (default)
- `limit` (optional): Max results (default: 20, max: 100)
- `offset` (optional): Pagination offset

**Example Requests**:
```
GET /api/v1/parcels/search?q=main+street&type=address
GET /api/v1/parcels/search?q=john+doe&type=owner
GET /api/v1/parcels/search?q=R123456&type=parcel_id
```

**Response**:
```json
{
  "results": [
    {
      "id": 12345,
      "parcel_id": "R123456",
      "owner_name": "John Doe",
      "situs_address": "123 Main St, Montgomery, TX",
      "acres": 5.25,
      "prop_type": "Residential",
      "county_name": "Montgomery",
      "center_point": {
        "lat": 30.3477,
        "lng": -95.4502
      }
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**PostGIS Queries**:

*Address Search* (using ILIKE for case-insensitive partial match):
```sql
SELECT 
    id, parcel_id, owner_name, situs_address, 
    acres, prop_type, county_name,
    ST_Y(ST_Centroid(geom)) as lat,
    ST_X(ST_Centroid(geom)) as lng
FROM tax_parcels
WHERE situs_address ILIKE '%' || $1 || '%'
ORDER BY situs_address
LIMIT $2 OFFSET $3;
```

*Owner Search*:
```sql
SELECT 
    id, parcel_id, owner_name, situs_address, 
    acres, prop_type, county_name,
    ST_Y(ST_Centroid(geom)) as lat,
    ST_X(ST_Centroid(geom)) as lng
FROM tax_parcels
WHERE owner_name ILIKE '%' || $1 || '%'
ORDER BY owner_name, situs_address
LIMIT $2 OFFSET $3;
```

*Parcel ID Search*:
```sql
SELECT 
    id, parcel_id, owner_name, situs_address, 
    acres, prop_type, county_name,
    ST_Y(ST_Centroid(geom)) as lat,
    ST_X(ST_Centroid(geom)) as lng
FROM tax_parcels
WHERE parcel_id ILIKE $1 || '%'
ORDER BY parcel_id
LIMIT $2 OFFSET $3;
```

**Performance Optimization**:
- Existing indexes on `owner_name` and `parcel_id` help
- Consider adding GIN index with pg_trgm for fuzzy text search:
```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_parcels_owner_trgm ON tax_parcels USING GIN(owner_name gin_trgm_ops);
CREATE INDEX idx_parcels_address_trgm ON tax_parcels USING GIN(situs_address gin_trgm_ops);
```

**2. Autocomplete Endpoint**

`GET /api/v1/parcels/autocomplete`

**Query Parameters**:
- `q` (required): Partial query string (min 2 chars)
- `type` (required): `address` or `owner`
- `limit` (optional): Max suggestions (default: 10)

**Example**:
```
GET /api/v1/parcels/autocomplete?q=main&type=address&limit=10
```

**Response**:
```json
{
  "suggestions": [
    "123 Main St, Montgomery, TX",
    "456 Main St, Montgomery, TX",
    "789 Main Street, Montgomery, TX"
  ]
}
```

**Query**:
```sql
SELECT DISTINCT situs_address
FROM tax_parcels
WHERE situs_address ILIKE $1 || '%'
ORDER BY situs_address
LIMIT $2;
```

**3. Enhanced Nearby Endpoint**

Already exists from PBI-5, but add:
- Property type filtering
- Acreage range filtering
- Owner type filtering (individual vs corporate)

`GET /api/v1/parcels/nearby`

**Additional Query Parameters**:
- `prop_type`: Filter by property type
- `min_acres`, `max_acres`: Acreage range
- `exclude_public`: Boolean to exclude public land

### Frontend Changes

**Next.js Components**:
```
components/
├── Search/
│   ├── SearchBar.tsx           # Main search input
│   ├── SearchResults.tsx       # Results list
│   ├── SearchFilters.tsx       # Type, acreage filters
│   └── AutocompleteList.tsx    # Dropdown suggestions
```

**Search Bar Features**:
- Debounced input (300ms) for autocomplete
- Search type selector (Address/Owner/Parcel ID)
- Clear button
- Keyboard navigation (arrow keys, enter, escape)

**Search Results Display**:
- List view of matching properties
- Click to center map and show boundary
- Show result count and pagination
- Display all results as markers on map
- Cluster markers when zoomed out

**Map Integration**:
- Show search result markers
- Clicking marker shows property details
- Different marker colors by property type
- Fit map bounds to show all results

### Vue.js Implementation

Similar implementation with Vue-specific patterns:
- v-model for search input
- Computed properties for filtered results
- Vue transitions for results panel

## UX/UI Considerations

### Search Experience

**Search Bar Placement**:
- Top-left corner of map
- Always visible
- Overlays map
- Width: 400px
- Mobile: Full width at top

**Search Flow**:
1. User types in search bar
2. Autocomplete shows after 2 characters
3. User selects suggestion or presses Enter
4. Results panel slides in from right
5. Map shows result markers
6. Click result to see property details

**Results Display**:
- Max 20 results per page
- "Load more" button for pagination
- Each result shows:
  - Owner name (bold)
  - Address
  - Acres and property type
  - Distance (if nearby search)

**Visual Feedback**:
- Loading spinner in search bar while querying
- "No results found" message
- Result count: "Found 15 properties"
- Highlight hovered result on map

### Error States

- "Please enter at least 2 characters"
- "No properties found matching your search"
- "Search failed, please try again"

### Performance

- Debounce autocomplete requests
- Cache recent searches
- Limit autocomplete to 10 results
- Limit search results to 20 per page
- Cancel previous requests on new input

## Acceptance Criteria

### Backend

1. ✅ `GET /api/v1/parcels/search` endpoint implemented
2. ✅ Search by address works with partial matches
3. ✅ Search by owner name works with partial matches
4. ✅ Search by parcel ID works with prefix match
5. ✅ Results limited to specified limit (default 20)
6. ✅ Pagination with offset parameter works
7. ✅ `GET /api/v1/parcels/autocomplete` endpoint implemented
8. ✅ Autocomplete returns distinct suggestions
9. ✅ Autocomplete limited to 10 results
10. ✅ Queries use indexes effectively (verify with EXPLAIN)
11. ✅ Response includes center point coordinates
12. ✅ pg_trgm extension installed for fuzzy search
13. ✅ GIN indexes created for text search

### Frontend (Next.js)

14. ✅ Search bar component in top-left of map
15. ✅ Search type selector (Address/Owner/Parcel ID)
16. ✅ Autocomplete shows suggestions after 2 characters
17. ✅ Autocomplete debounced to 300ms
18. ✅ Keyboard navigation in autocomplete
19. ✅ Search results panel displays matching properties
20. ✅ Results show: owner, address, acres, type
21. ✅ Click result centers map and shows property
22. ✅ Search result markers displayed on map
23. ✅ Marker clustering when zoomed out
24. ✅ "No results" message when appropriate
25. ✅ Loading states during search
26. ✅ Clear button empties search and hides results
27. ✅ Pagination for results (load more)

### Frontend (Vue.js)

28. ✅ Search functionality implemented with Vue patterns
29. ✅ Same features as Next.js version
30. ✅ Vue-specific transitions and animations

## Dependencies

- PBI-5: Point-in-Polygon Query API (API backend must exist)
- PBI-6: Next.js Frontend (frontend components)
- PBI-7: Vue.js Frontend (Vue components)
- pg_trgm extension available in PostgreSQL
- Sufficient test data for meaningful search results

## Open Questions

1. Should we support full-text search with ranking?
2. Do we need fuzzy matching for misspellings?
3. Should we cache popular searches?
4. Do we need search history per user?
5. Should we add export functionality for search results?
6. Do we need advanced filters (property value, year built, etc.)?
7. Should nearby search show distance in miles or meters?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

