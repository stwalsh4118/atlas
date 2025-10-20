-- Revert geometry column from MultiPolygon back to Polygon
-- WARNING: This will fail if any MultiPolygon geometries with multiple parts exist

-- Convert back to Polygon (will fail if true MultiPolygons exist)
ALTER TABLE tax_parcels 
    ALTER COLUMN geom TYPE GEOMETRY(Polygon, 4326)
    USING ST_GeometryN(geom, 1);

-- Restore original comment
COMMENT ON COLUMN tax_parcels.geom IS 'Polygon geometry in SRID 4326 (WGS84) representing parcel boundary';

