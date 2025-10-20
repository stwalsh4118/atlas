-- Disable PostGIS extension
-- CASCADE will remove all dependent objects (geometry columns, spatial indexes, etc.)
DROP EXTENSION IF EXISTS postgis CASCADE;

