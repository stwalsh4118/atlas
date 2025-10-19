# PBI-6: Next.js Frontend with Map Integration

[View in Backlog](../backlog.md#user-content-6)

## Overview

Build the primary frontend application using Next.js 15+ with Mapbox GL JS, providing an interactive map interface where users can click to query property information, see highlighted boundaries, and view property details in a responsive UI.

## Problem Statement

Users need an intuitive, performant web interface to:
- View an interactive map centered on their location or a default area
- Click anywhere on the map to query property information
- See property boundaries highlighted when selected
- View property details (owner, acreage, type) in a clean panel
- Experience fast, responsive interactions with loading states
- Use the application on desktop browsers (mobile as stretch goal)

## User Stories

- **US-1**: As a user, I want to see a map of my current area so that I can orient myself geographically
- **US-2**: As a user, I want to click anywhere on the map to see property information at that location
- **US-3**: As a user, I want to see property boundaries highlighted when I select them so I can understand the exact extent
- **US-4**: As a user, I want to see my current location on the map so I can understand properties near me
- **US-5**: As a user, I want to see the owner's name for a property so I can identify who owns the land
- **US-6**: As a user, I want to see the acreage/size of a property so I can understand its scale
- **US-7**: As a user, I want to distinguish between public and private land so I can understand access rights
- **US-8**: As a user, I want to see the parcel ID so I can reference it in county records

## Technical Approach

### Technology Stack

- **Framework**: Next.js 15+ (App Router)
- **Language**: TypeScript 5.5+
- **React**: 19.0+
- **Map Library**: Mapbox GL JS v3+
- **State Management**: Zustand (lightweight) or React Context
- **HTTP Client**: Fetch API with error handling
- **Styling**: Tailwind CSS 4.0+
- **Icons**: Lucide React or Heroicons

### Project Structure

```
frontend-nextjs/
├── app/
│   ├── layout.tsx              # Root layout
│   ├── page.tsx                # Home page with map
│   └── api/
│       └── proxy/              # API proxy if needed
├── components/
│   ├── Map/
│   │   ├── MapContainer.tsx    # Main map component
│   │   ├── PropertyLayer.tsx   # Property boundary layer
│   │   └── MapControls.tsx     # Zoom, location controls
│   ├── PropertyPanel/
│   │   ├── PropertyDetails.tsx # Property info display
│   │   ├── PropertyHeader.tsx  # Owner name, parcel ID
│   │   └── PropertyStats.tsx   # Acreage, type, etc.
│   └── UI/
│       ├── LoadingSpinner.tsx
│       ├── ErrorMessage.tsx
│       └── Button.tsx
├── lib/
│   ├── api.ts                  # API client functions
│   ├── mapbox.ts               # Mapbox utilities
│   └── types.ts                # TypeScript types
├── hooks/
│   ├── useParcelQuery.ts       # Query parcel by point
│   ├── useGeolocation.ts       # User location
│   └── useMapState.ts          # Map state management
├── store/
│   └── mapStore.ts             # Zustand store
└── styles/
    └── globals.css
```

### Core Features

**1. Map Display**
- Initialize Mapbox GL JS with base map style
- Default center: Montgomery County, TX (or user's location)
- Default zoom: 12
- Controls: Zoom, rotation, navigation
- Style: Light theme to not compete with overlays

**2. Click-to-Query Interaction**
```typescript
map.on('click', async (e) => {
  const { lng, lat } = e.lngLat;
  
  // Show loading state
  setLoading(true);
  
  try {
    // Query API
    const parcel = await fetchParcelAtPoint(lat, lng);
    
    // Add boundary to map
    addParcelBoundary(parcel.geometry);
    
    // Show property panel
    setSelectedParcel(parcel);
  } catch (error) {
    if (error.status === 404) {
      showMessage("No property found at this location");
    } else {
      showError("Failed to load property data");
    }
  } finally {
    setLoading(false);
  }
});
```

**3. Property Boundary Display**
- Add GeoJSON layer for selected property
- Style:
  - Fill: Transparent or light blue (#3B82F610)
  - Stroke: Bright blue (#3B82F6)
  - Stroke width: 2px
- Remove previous boundary when new property selected
- Zoom to fit boundary extent

**4. Property Details Panel**
- Slide-in panel from right side
- Sections:
  - Owner Information
  - Property Metrics (acres, type)
  - Location (address)
  - Identifiers (parcel ID, county)
- Actions:
  - Close button
  - "View on County Site" link (if available)
  - Copy coordinates button

**5. Geolocation**
- Request user's location on load
- Center map on user's position if granted
- Add user location marker
- Fallback to default location if denied

**6. Loading States**
- Map loading skeleton
- Query loading spinner on map
- Shimmer effect in property panel
- Disable map clicks during query

**7. Error Handling**
- Network errors: "Unable to connect to server"
- No property found: "No property data at this location"
- Invalid location: "Invalid coordinates"
- Mapbox API errors: "Map failed to load"

### API Integration

**API Client** (`lib/api.ts`):
```typescript
export async function fetchParcelAtPoint(
  lat: number, 
  lng: number
): Promise<Parcel> {
  const response = await fetch(
    `${API_BASE_URL}/api/v1/parcels/at-point?lat=${lat}&lng=${lng}`
  );
  
  if (!response.ok) {
    if (response.status === 404) {
      throw new NotFoundError("No parcel found");
    }
    throw new Error("Failed to fetch parcel");
  }
  
  const data = await response.json();
  return data.parcel;
}
```

### Styling & UX

**Color Palette**:
- Primary: Blue (#3B82F6)
- Success: Green (#10B981)
- Warning: Yellow (#FCD34D)
- Error: Red (#EF4444)
- Text: Gray-900 (#111827)
- Background: White (#FFFFFF)

**Typography**:
- Headers: font-bold text-xl
- Body: font-normal text-base
- Metadata: font-normal text-sm text-gray-600

**Spacing**:
- Panel padding: p-6
- Section spacing: space-y-4
- Component spacing: 8px base unit

**Responsive Design**:
- Desktop: Panel width 384px (w-96)
- Tablet: Full width panel overlay
- Mobile: Bottom sheet (stretch goal)

### Configuration

**Environment Variables**:
```bash
NEXT_PUBLIC_MAPBOX_TOKEN=pk.xxxxx
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_DEFAULT_LAT=30.3477
NEXT_PUBLIC_DEFAULT_LNG=-95.4502
NEXT_PUBLIC_DEFAULT_ZOOM=12
```

### Performance Considerations

- Code split map component (client-only)
- Lazy load Mapbox GL JS
- Debounce rapid clicks (300ms)
- Optimize GeoJSON rendering for large parcels
- Cache API responses (optional)

## UX/UI Considerations

### Primary User Flow

1. **Page Load**
   - Map loads with loading skeleton
   - Center on user location or default
   - Show help tooltip: "Click anywhere to explore properties"

2. **User Clicks Map**
   - Immediate visual feedback (cursor change)
   - Loading spinner appears on map
   - Panel slides in when data loads
   - Boundary highlights simultaneously

3. **Viewing Property Details**
   - Clean, scannable layout
   - Important info at top (owner, acres)
   - Secondary info below (IDs, type)
   - Clear action buttons

4. **Exploring More Properties**
   - Click elsewhere to query new property
   - Previous boundary removed smoothly
   - Panel content updates with transition
   - Loading states during transition

### Error States

- **No Property Found**: Show friendly message, suggest clicking different area
- **API Error**: Show error banner, "Try again" button
- **Map Error**: Show full-page error with refresh option
- **Network Offline**: Show offline banner

### Accessibility

- Keyboard navigation support
- ARIA labels on interactive elements
- Focus management for panel
- Screen reader announcements for property changes

## Acceptance Criteria

1. ✅ Next.js 15+ application initialized with TypeScript 5.5+
2. ✅ Mapbox GL JS integrated and displaying base map
3. ✅ Map centered on default location or user's geolocation
4. ✅ User can click on map to query property
5. ✅ Loading spinner shows during API request
6. ✅ Property boundary displayed on map when found
7. ✅ Property details panel slides in with data
8. ✅ Panel shows: owner, acres, address, parcel ID, type
9. ✅ Public vs private land indicated visually
10. ✅ Acres formatted with 2 decimal places
11. ✅ Previous boundary removed when new property selected
12. ✅ "No property found" message when clicking empty area
13. ✅ Error handling for API failures
14. ✅ Error handling for map loading failures
15. ✅ Responsive layout works on desktop (1920x1080+)
16. ✅ Close button dismisses property panel
17. ✅ Application runs on localhost:3000
18. ✅ Code passes TypeScript checks and ESLint
19. ✅ README documents how to run the application
20. ✅ Environment variables documented in .env.example

## Dependencies

- PBI-5: Point-in-Polygon Query API (API must be functional)
- Mapbox API key (free tier acceptable)
- Node.js 20+ (LTS) and pnpm installed [[memory:2879566]]
- API server running on localhost:8080

## Open Questions

1. Should we use Mapbox free tier or OSM with open map tiles?
2. Do we need to support map style switching (light/dark)?
3. Should we add keyboard shortcuts (e.g., Escape to close panel)?
4. Do we need analytics tracking for user interactions?
5. Should we show adjacent parcels on hover?
6. Do we need to support URL parameters for sharing specific locations?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

