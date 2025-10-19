# PBI-7: Vue.js Frontend with Leaflet

[View in Backlog](../backlog.md#user-content-7)

## Overview

Build an alternative frontend implementation using Vue 3 and Leaflet to demonstrate the same property query functionality with a different technology stack, serving as both a learning exercise and a comparison point for framework trade-offs.

## Problem Statement

To demonstrate versatility and compare frontend frameworks, we need a second implementation that:
- Provides the same core functionality as the Next.js version
- Uses Vue 3 Composition API instead of React
- Uses Leaflet instead of Mapbox GL JS (open-source alternative)
- Shares the same API backend
- Has a similar but distinct visual design
- Runs on a different port (3001) to allow side-by-side comparison

This serves multiple purposes:
- Learning Vue 3 ecosystem and patterns
- Comparing Leaflet vs Mapbox for property boundary rendering
- Evaluating framework differences for interview discussions
- Providing fallback if Mapbox API limits are reached

## User Stories

- **US-1**: As a user, I want to see a map of my current area so that I can orient myself geographically
- **US-2**: As a user, I want to click anywhere on the map to see property information at that location
- **US-3**: As a user, I want to see property boundaries highlighted when I select them so I can understand the exact extent
- **US-DEV-18**: As a developer, I want to compare Vue and React implementations so that I can discuss framework trade-offs
- **US-DEV-19**: As a developer, I want to use Leaflet so that I have an open-source mapping alternative

## Technical Approach

### Technology Stack

- **Framework**: Vue 3.5+ (Composition API)
- **Language**: TypeScript 5.5+
- **Build Tool**: Vite 6.0+
- **Map Library**: Leaflet 1.9+ + Leaflet.GeoJSON
- **State Management**: Pinia
- **HTTP Client**: Axios
- **Styling**: Tailwind CSS 4.0+
- **Icons**: Heroicons Vue

### Project Structure

```
frontend-vue/
├── src/
│   ├── main.ts                 # Entry point
│   ├── App.vue                 # Root component
│   ├── components/
│   │   ├── MapView.vue         # Main map component
│   │   ├── PropertyPanel.vue   # Property details panel
│   │   ├── PropertyInfo.vue    # Property information display
│   │   ├── LoadingSpinner.vue  # Loading state
│   │   └── ErrorMessage.vue    # Error display
│   ├── composables/
│   │   ├── useParcelQuery.ts   # Parcel query logic
│   │   ├── useGeolocation.ts   # Geolocation hook
│   │   └── useLeafletMap.ts    # Leaflet map management
│   ├── stores/
│   │   └── mapStore.ts         # Pinia store
│   ├── services/
│   │   └── api.ts              # API client
│   ├── types/
│   │   └── index.ts            # TypeScript interfaces
│   └── assets/
│       └── styles.css
├── public/
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

### Core Features

**1. Map Display with Leaflet**
```typescript
import L from 'leaflet';

const map = L.map('map').setView([30.3477, -95.4502], 12);

// OpenStreetMap tiles (free)
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
  attribution: '© OpenStreetMap contributors',
  maxZoom: 19,
}).addTo(map);
```

**2. Click-to-Query with Composition API**
```vue
<script setup lang="ts">
import { ref } from 'vue';
import { useParcelQuery } from '@/composables/useParcelQuery';

const { queryParcel, loading, error } = useParcelQuery();
const selectedParcel = ref<Parcel | null>(null);

const handleMapClick = async (e: L.LeafletMouseEvent) => {
  const { lat, lng } = e.latlng;
  selectedParcel.value = await queryParcel(lat, lng);
};
</script>
```

**3. Property Boundary with GeoJSON**
```typescript
let currentLayer: L.GeoJSON | null = null;

function showParcelBoundary(geoJSON: GeoJSON.MultiPolygon) {
  // Remove previous layer
  if (currentLayer) {
    map.removeLayer(currentLayer);
  }
  
  // Add new layer
  currentLayer = L.geoJSON(geoJSON, {
    style: {
      color: '#3B82F6',
      weight: 2,
      fillColor: '#3B82F610',
      fillOpacity: 0.2
    }
  }).addTo(map);
  
  // Zoom to bounds
  map.fitBounds(currentLayer.getBounds());
}
```

**4. Pinia Store for State Management**
```typescript
// stores/mapStore.ts
import { defineStore } from 'pinia';

export const useMapStore = defineStore('map', {
  state: () => ({
    selectedParcel: null as Parcel | null,
    loading: false,
    error: null as string | null,
  }),
  
  actions: {
    async selectParcelAtPoint(lat: number, lng: number) {
      this.loading = true;
      this.error = null;
      
      try {
        const response = await api.fetchParcelAtPoint(lat, lng);
        this.selectedParcel = response.parcel;
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },
    
    clearSelection() {
      this.selectedParcel = null;
      this.error = null;
    }
  }
});
```

**5. Property Panel Component**
```vue
<template>
  <div 
    v-if="selectedParcel" 
    class="property-panel"
  >
    <div class="panel-header">
      <h2>{{ selectedParcel.owner_name }}</h2>
      <button @click="close">×</button>
    </div>
    
    <div class="property-stats">
      <div class="stat">
        <span class="label">Acres</span>
        <span class="value">{{ formatAcres(selectedParcel.acres) }}</span>
      </div>
      
      <div class="stat">
        <span class="label">Property Type</span>
        <span class="value">{{ selectedParcel.prop_type }}</span>
      </div>
      
      <div class="stat">
        <span class="label">Parcel ID</span>
        <span class="value">{{ selectedParcel.parcel_id }}</span>
      </div>
    </div>
    
    <div class="property-address">
      {{ selectedParcel.situs_address }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useMapStore } from '@/stores/mapStore';

const store = useMapStore();
const selectedParcel = computed(() => store.selectedParcel);

const formatAcres = (acres: number) => acres.toFixed(2);
const close = () => store.clearSelection();
</script>
```

### Leaflet vs Mapbox Differences

| Feature | Mapbox GL JS | Leaflet |
|---------|-------------|---------|
| License | Proprietary (free tier) | Open source (BSD) |
| Tile Source | Mapbox tiles | OSM or any tile server |
| Vector Tiles | Native support | Requires plugin |
| 3D Support | Yes | Limited |
| Performance | Better for complex maps | Good for basic maps |
| Bundle Size | ~500KB | ~150KB |
| API Complexity | More complex | Simpler |

### API Integration

Same API backend as Next.js version:
- `GET /api/v1/parcels/at-point`
- `GET /api/v1/parcels/nearby`
- `GET /api/v1/parcels/:id`

**Axios Client**:
```typescript
import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 5000,
});

export async function fetchParcelAtPoint(
  lat: number, 
  lng: number
): Promise<ParcelResponse> {
  const { data } = await apiClient.get('/api/v1/parcels/at-point', {
    params: { lat, lng }
  });
  return data;
}
```

### Visual Design

**Differences from Next.js version**:
- Panel slides from left instead of right
- Different color scheme (purple accent instead of blue)
- Slightly different layout for variety
- Same information displayed

**Color Palette**:
- Primary: Purple (#8B5CF6)
- Success: Green (#10B981)
- Background: White (#FFFFFF)
- Text: Gray-900 (#111827)

### Configuration

**Environment Variables** (`.env`):
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_DEFAULT_LAT=30.3477
VITE_DEFAULT_LNG=-95.4502
VITE_DEFAULT_ZOOM=12
```

### Development Setup

```json
{
  "scripts": {
    "dev": "vite --port 3001",
    "build": "vite build",
    "preview": "vite preview",
    "type-check": "vue-tsc --noEmit",
    "lint": "eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts"
  }
}
```

## UX/UI Considerations

### Simplified Scope

This implementation focuses on:
- Core click-to-query functionality
- Property boundary display
- Basic property information panel
- Error and loading states

**Out of scope for MVP**:
- Advanced search
- Multiple property selection
- Property comparison
- Export features

### Performance

- Leaflet is lighter weight than Mapbox
- May be slower with very complex geometries
- Optimize by simplifying geometries if needed

## Acceptance Criteria

1. ✅ Vue 3.5+ application initialized with Vite 6.0+ and TypeScript 5.5+
2. ✅ Leaflet map integrated and displaying OSM tiles
3. ✅ Map centered on default location
4. ✅ User can click on map to query property
5. ✅ Loading spinner shows during API request
6. ✅ Property boundary displayed as GeoJSON layer
7. ✅ Property details panel shows data
8. ✅ Panel shows: owner, acres, address, parcel ID, type
9. ✅ Acres formatted with 2 decimal places
10. ✅ Previous boundary removed when new property selected
11. ✅ "No property found" message when clicking empty area
12. ✅ Error handling for API failures
13. ✅ Close button dismisses property panel
14. ✅ Application runs on localhost:3001
15. ✅ Code passes TypeScript checks and ESLint
16. ✅ README documents how to run the application
17. ✅ Can run simultaneously with Next.js version
18. ✅ Same API endpoints as Next.js version

## Dependencies

- PBI-5: Point-in-Polygon Query API (API must be functional)
- Node.js 20+ (LTS) and pnpm installed [[memory:2879566]]
- API server running on localhost:8080
- OpenStreetMap tiles (no API key needed)

## Open Questions

1. Should we use Leaflet.VectorGrid for better performance?
2. Do we need to implement the same level of polish as Next.js version?
3. Should we add Vue-specific features (e.g., transitions)?
4. Is it worth adding Leaflet.Markercluster for nearby properties?
5. Should we create a shared TypeScript types package for both frontends?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Proposed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

