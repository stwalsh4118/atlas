-- Change geometry column from Polygon to MultiPolygon
-- This allows the table to accept both Polygon and MultiPolygon geometries
-- Polygons will be automatically cast to MultiPolygon during import using ST_Multi()

-- Drop the existing geometry column constraint
ALTER TABLE tax_parcels 
    ALTER COLUMN geom TYPE GEOMETRY(MultiPolygon, 4326) 
    USING ST_Multi(geom);

-- Update the comment to reflect the change
COMMENT ON COLUMN tax_parcels.geom IS 'MultiPolygon geometry in SRID 4326 (WGS84) representing parcel boundary. Polygons are converted to MultiPolygon on import.';

