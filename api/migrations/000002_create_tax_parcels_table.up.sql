-- Create tax_parcels table with all Montgomery County GeoJSON fields
-- This table stores property tax parcel data with spatial boundaries

CREATE TABLE tax_parcels (
    -- Primary identifiers
    id BIGSERIAL PRIMARY KEY,
    object_id INTEGER UNIQUE NOT NULL,
    pin INTEGER NOT NULL,
    pid INTEGER,
    
    -- Parcel subdivision info
    state_cd VARCHAR(10),
    block INTEGER,
    lot VARCHAR(50),
    tract VARCHAR(50),
    
    -- Owner information
    owner_name VARCHAR(500),
    owner_address TEXT,
    
    -- Property details
    situs VARCHAR(500),
    as_code VARCHAR(50),
    legal_description TEXT,
    
    -- Improvement/building info
    imprv_actual_year_built INTEGER,
    imprv_main_area INTEGER,
    
    -- Tax information
    p_year INTEGER,
    p_version INTEGER,
    p_roll_corr INTEGER,
    taxing_units VARCHAR(255),
    exemptions VARCHAR(255),
    market_area VARCHAR(50),
    
    -- County metadata
    county_name VARCHAR(100) DEFAULT 'Montgomery',
    
    -- Spatial data - Polygon geometry with SRID 4326 (WGS84)
    geom GEOMETRY(Polygon, 4326) NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create GiST spatial index on geometry column
-- This is critical for efficient point-in-polygon queries (target < 100ms)
CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);

-- Add comment for documentation
COMMENT ON TABLE tax_parcels IS 'Property tax parcels with spatial boundaries from Montgomery County, Texas';
COMMENT ON COLUMN tax_parcels.geom IS 'Polygon geometry in SRID 4326 (WGS84) representing parcel boundary';
COMMENT ON INDEX idx_parcels_geom IS 'GiST spatial index for efficient point-in-polygon queries';

