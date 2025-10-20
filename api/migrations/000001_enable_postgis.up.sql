-- Enable PostGIS extension
-- This extension provides spatial data types and functions for PostgreSQL
CREATE EXTENSION IF NOT EXISTS postgis;

-- Verify PostGIS is installed and working
-- This will return the version string (e.g., "3.6 USE_GEOS=1 USE_PROJ=1...")
SELECT PostGIS_Version();

