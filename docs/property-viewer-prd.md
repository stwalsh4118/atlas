# Property Boundary Viewer - Product Requirements Document

**Version**: 1.0  
**Date**: October 19, 2025  
**Status**: Draft  
**Author**: Project Owner  

## Executive Summary

This project creates a geospatial web application that allows users to explore property boundaries and ownership information by clicking on a map. The application demonstrates core competencies in PostGIS spatial queries, full-stack development with Go and modern JavaScript frameworks, and mirrors key features of onX Maps' property boundary functionality. This serves both as a portfolio piece for interview preparation and a practical learning project for geospatial development.

## Problem Statement

Users need an intuitive way to understand property boundaries and ownership information in their area. Currently, accessing this information requires navigating complex county GIS portals, understanding technical terminology, and dealing with inconsistent user interfaces across different counties. 

For developers interviewing at geospatial companies like onX, there's a need to demonstrate practical understanding of:
- Spatial database queries and optimization
- Real-time geospatial data visualization
- Full-stack integration with spatial data types
- Modern web development practices with spatial libraries

## Target Users

### Primary Persona: The Developer
- **Goal**: Learn geospatial development patterns and build portfolio project
- **Needs**: Hands-on experience with PostGIS, spatial queries, and map rendering
- **Pain Points**: Limited practical examples of full-stack spatial applications

### Secondary Persona: The Property Explorer
- **Goal**: Quickly understand property boundaries and ownership in their area
- **Needs**: Simple, visual interface to property information
- **Pain Points**: Complex county GIS systems, slow query interfaces

## Product Vision

Build a lightweight, performant web application that demonstrates mastery of geospatial technologies while providing real utility for exploring property data. The application will showcase:
- Fast spatial queries against real parcel data
- Clean, modern UI with multiple framework implementations
- Proper architecture separating concerns between frontend, API, and database
- Best practices in spatial indexing and query optimization

## Core Features

### Must Have (MVP)
1. **Interactive Map Display**
   - Display base map using Mapbox/Leaflet
   - Show user's current location
   - Pan and zoom functionality
   - Display property boundaries as overlays

2. **Point-in-Polygon Query**
   - Click anywhere on map to query property information
   - Return property details: owner name, acreage, parcel ID, property type
   - Highlight selected property boundary
   - Display loading states during query

3. **Property Information Display**
   - Show property details in sidebar/panel
   - Format acres/square footage appropriately
   - Distinguish between private and public land
   - Clear visual indication of selected property

4. **Data Import Pipeline**
   - Import county tax parcel shapefile data
   - Transform to EPSG:4326 (WGS84)
   - Create spatial indexes
   - Handle MultiPolygon geometry types

### Should Have (Post-MVP)
5. **Search Functionality**
   - Search by address
   - Search by parcel ID
   - Search by owner name
   - Autocomplete suggestions

6. **Nearby Properties Query**
   - Find properties within radius of point
   - Display multiple results on map
   - Sort by distance

7. **Property Boundary Export**
   - Export selected property as GeoJSON
   - Export visible properties as GeoJSON
   - Copy coordinates to clipboard

### Nice to Have (Future)
8. **Multi-County Support**
   - Import data from multiple counties
   - Switch between county datasets
   - Aggregate queries across counties

9. **Historical Data**
   - Track ownership changes
   - Show previous boundaries
   - Display change history

10. **User Annotations**
    - Allow users to save notes on properties
    - Mark properties as favorites
    - Create custom property collections

## User Stories

### Epic 1: Core Map Interaction
- **US-1**: As a user, I want to see a map of my current area so that I can orient myself geographically
- **US-2**: As a user, I want to click anywhere on the map to see property information at that location
- **US-3**: As a user, I want to see property boundaries highlighted when I select them so I can understand the exact extent
- **US-4**: As a user, I want to see my current location on the map so I can understand properties near me

### Epic 2: Property Information
- **US-5**: As a user, I want to see the owner's name for a property so I can identify who owns the land
- **US-6**: As a user, I want to see the acreage/size of a property so I can understand its scale
- **US-7**: As a user, I want to distinguish between public and private land so I can understand access rights
- **US-8**: As a user, I want to see the parcel ID so I can reference it in county records

### Epic 3: Data Management
- **US-9**: As a developer, I want to import county parcel data efficiently so I can populate the database
- **US-10**: As a developer, I want spatial queries to return results in under 100ms so the UX remains responsive
- **US-11**: As a developer, I want proper error handling for malformed geometries so the system remains stable

### Epic 4: Search and Discovery
- **US-12**: As a user, I want to search for a property by address so I can find specific locations quickly
- **US-13**: As a user, I want to find all properties within a certain distance so I can explore nearby parcels
- **US-14**: As a user, I want to search by owner name so I can find all properties owned by someone

## Technical Approach

### Architecture Overview
```
┌─────────────────┐
│   Next.js UI    │  (Primary Frontend - Port 3000)
│  - Mapbox GL JS │
│  - React        │
└────────┬────────┘
         │ HTTP/REST
         │
┌────────┴────────┐
│    Vue.js UI    │  (Learning Frontend - Port 3001)
│  - Leaflet      │
│  - Vue 3        │
└────────┬────────┘
         │ HTTP/REST
         │
┌────────┴────────┐
│  Go Gin API     │  (Backend - Port 8080)
│  - GORM         │
│  - Gin Router   │
└────────┬────────┘
         │ SQL
         │
┌────────┴────────┐
│   PostgreSQL    │  (Database - Port 5432)
│   + PostGIS     │
│  - Spatial Idx  │
└─────────────────┘
```

### Technology Stack

#### Backend
- **Language**: Go 1.25+
- **Framework**: Gin (HTTP routing)
- **ORM**: GORM (with custom spatial query handling)
- **Database Driver**: pgx v5

#### Database
- **Database**: PostgreSQL 18
- **Extension**: PostGIS 3.5
- **Deployment**: Docker container (development), managed service (production)
- **Connection Pooling**: Built-in pgx pooling

#### Frontend (Next.js)
- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript
- **Map Library**: Mapbox GL JS v3
- **State Management**: React Context / Zustand
- **Styling**: Tailwind CSS
- **HTTP Client**: Fetch API / axios

#### Frontend (Vue.js)
- **Framework**: Vue 3 (Composition API)
- **Language**: TypeScript
- **Map Library**: Leaflet + Leaflet.VectorGrid
- **State Management**: Pinia
- **Styling**: Tailwind CSS
- **HTTP Client**: axios

#### Development Tools
- **Containerization**: Docker & Docker Compose
- **API Documentation**: OpenAPI/Swagger
- **Testing**: Go testing package, Vitest (frontend)
- **Linting**: golangci-lint, ESLint

### Data Model

#### Core Tables

**tax_parcels**
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

CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);
CREATE INDEX idx_parcels_parcel_id ON tax_parcels(parcel_id);
CREATE INDEX idx_parcels_owner ON tax_parcels(owner_name);
```

### API Endpoints

#### Core Endpoints
- `GET /api/v1/parcels/at-point` - Find parcel containing lat/lng
  - Query params: `lat`, `lng`
  - Returns: Parcel details + GeoJSON geometry
  
- `GET /api/v1/parcels/nearby` - Find parcels within radius
  - Query params: `lat`, `lng`, `radius` (meters)
  - Returns: Array of parcels with distance
  
- `GET /api/v1/parcels/:id` - Get parcel by ID
  - Returns: Full parcel details + GeoJSON geometry

- `GET /api/v1/parcels/search` - Search parcels
  - Query params: `q` (query string), `type` (address|owner|parcel_id)
  - Returns: Array of matching parcels

#### Utility Endpoints
- `GET /api/health` - Health check
- `GET /api/v1/stats` - Database statistics

### Key Spatial Queries

**Point-in-Polygon (Core Query)**
```sql
SELECT 
    id, parcel_id, owner_name, situs_address, 
    acres, prop_type, land_use,
    ST_AsGeoJSON(geom) as geometry
FROM tax_parcels
WHERE ST_Contains(
    geom, 
    ST_SetSRID(ST_MakePoint($1, $2), 4326)
)
LIMIT 1;
```

**Distance Query**
```sql
SELECT 
    id, parcel_id, owner_name, acres,
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

### Performance Requirements

- **Query Response Time**: < 100ms for point-in-polygon queries
- **Map Tile Load Time**: < 500ms for initial map load
- **API Availability**: 99.9% uptime (development target)
- **Concurrent Users**: Support 10 simultaneous users (development)
- **Database Size**: Handle 50k-500k parcels efficiently

### Security Considerations

- **API Rate Limiting**: 100 requests/minute per IP
- **Input Validation**: Validate all lat/lng inputs
- **SQL Injection Prevention**: Use parameterized queries only
- **CORS Configuration**: Restrict to known frontend origins
- **No Authentication**: Public read-only access (MVP)

## UX/UI Considerations

### Design Principles
1. **Performance First**: Instant feedback for all interactions
2. **Visual Clarity**: Clear distinction between selected/unselected properties
3. **Mobile Responsive**: Work on mobile devices (stretch goal)
4. **Progressive Disclosure**: Show basic info immediately, details on demand

### Key Interactions

**Primary Flow: Property Lookup**
1. User opens application → sees map centered on their location
2. User clicks anywhere on map
3. Loading indicator appears immediately
4. Property boundary highlights on map
5. Property details slide in from right
6. User can click elsewhere to query new property

**Visual Design**
- **Color Palette**:
  - Private land: Blue outline (#3B82F6)
  - Public land: Green outline (#10B981)
  - Selected property: Yellow highlight (#FCD34D)
- **Typography**: System fonts for performance
- **Spacing**: 8px base unit
- **Map Style**: Light/neutral to not compete with property overlays

### Error States
- **No property found**: "No property data at this location"
- **Query timeout**: "Query taking longer than expected, please try again"
- **Network error**: "Unable to connect to server"
- **Invalid coordinates**: "Invalid location selected"

## Acceptance Criteria

### Minimum Viable Product (MVP)
- [ ] User can see an interactive map
- [ ] User can click on map to query property at that location
- [ ] Property details display within 100ms of click
- [ ] Property boundary highlights on map when selected
- [ ] At least 10,000 parcels imported and queryable
- [ ] Both Next.js and Vue frontends functional
- [ ] Spatial index improves query performance by 100x vs no index
- [ ] Application runs in Docker containers locally
- [ ] API documented with example requests
- [ ] Code repository includes README with setup instructions

### Quality Criteria
- [ ] No SQL injection vulnerabilities
- [ ] All API endpoints return proper HTTP status codes
- [ ] Frontend handles loading and error states gracefully
- [ ] Geometry data validates before database insertion
- [ ] Queries log execution time for performance monitoring

## Dependencies

### External Dependencies
- County tax parcel data (shapefile format)
- Mapbox API key (free tier) OR OpenStreetMap tiles
- Docker for containerization
- PostgreSQL 18 with PostGIS 3.5

### Data Dependencies
- Montgomery County (or local county) GIS data
- Base map tiles from Mapbox or OSM

### Technical Dependencies
- Go 1.25+
- Node.js 20+ (LTS) for frontend builds
- Docker & Docker Compose

## Open Questions

1. **Data Source**: Which county's parcel data will be used? Montgomery County, TX confirmed?
2. **Map Provider**: Mapbox (requires API key) vs OpenStreetMap (free, self-hosted tiles)?
3. **Deployment Target**: Will this be deployed publicly or remain local-only?
4. **Data Updates**: How will parcel data be refreshed? Manual re-import acceptable for MVP?
5. **Multi-County**: Should the schema support multiple counties from day one?
6. **Testing Scope**: What level of test coverage is required? Integration tests only or unit tests too?
7. **Mobile Support**: Is mobile responsiveness required for MVP or post-MVP?

## Success Metrics

### Technical Success
- Query response time consistently < 100ms
- Application can handle 10 concurrent users without degradation
- Zero SQL injection vulnerabilities
- Code demonstrates understanding of spatial concepts

### Learning Success
- Successfully demonstrate understanding of PostGIS spatial queries
- Can explain query optimization strategies in interview context
- Understand trade-offs between different spatial index types
- Can articulate full-stack architecture decisions

### Portfolio Success
- Deployable demo that can be shown in interviews
- Clean, documented code that demonstrates best practices
- README that explains technical decisions and architecture
- Can discuss scaling considerations and limitations

## Timeline Estimate

**Weekend Project Breakdown** (Aggressive but achievable):

**Saturday Morning (3 hours)**
- Set up Docker environment with PostGIS
- Import and validate county parcel data
- Create spatial indexes

**Saturday Afternoon (4 hours)**
- Build Go API with core endpoints
- Implement point-in-polygon query
- Test API with sample queries

**Sunday Morning (3 hours)**
- Build Next.js frontend with map
- Implement click-to-query functionality
- Style property detail panel

**Sunday Afternoon (3 hours)**
- Build Vue.js frontend (simpler version)
- Test both frontends
- Write README and documentation

**Buffer** (2 hours)
- Debugging, polish, deployment prep

**Total: ~15 hours**

## Future Enhancements (Post-MVP)

1. **Advanced Search**: Full-text search on owner names and addresses
2. **Property Comparison**: Compare multiple properties side-by-side
3. **Route Planning**: Draw routes and see all properties crossed
4. **Historical Data**: Show property sales history
5. **Mobile App**: React Native version
6. **Offline Mode**: Cache tiles and property data for offline use
7. **User Accounts**: Save favorite properties
8. **Notifications**: Alert when properties change ownership
9. **3D View**: Visualize property elevations
10. **Analytics Dashboard**: Property statistics and trends

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| County data format incompatible | High | Low | Preview data in QGIS before import, have backup county |
| PostGIS queries too slow | High | Medium | Implement proper spatial indexes, query optimization |
| Frontend map library learning curve | Medium | Medium | Follow official examples, use simpler library if needed |
| Time estimate too aggressive | Medium | High | Prioritize core features, cut Vue.js if needed |
| Docker environment issues | Low | Low | Document setup steps, use Docker Compose |

## Appendix

### Relevant onX Technologies to Demonstrate
- PostGIS spatial queries (ST_Contains, ST_DWithin, ST_Intersects)
- Spatial indexing strategies (GiST)
- GeoJSON serialization for frontend
- Map tile rendering and interaction
- Real-time spatial queries
- Property boundary visualization

### Learning Resources Used
- PostGIS documentation
- onX Maps product research
- Mapbox GL JS examples
- Gin framework documentation

### County Data Sources
- Montgomery County, TX: [URL to be confirmed]
- Alternative counties if needed: Harris County, Boulder County

---

**Document Control**
- **Last Updated**: October 19, 2025
- **Next Review**: Upon MVP completion
- **Approval Required From**: Project Owner
