-- Drop standard B-tree indexes
-- Drop in reverse order of creation

DROP INDEX IF EXISTS idx_parcels_county;
DROP INDEX IF EXISTS idx_parcels_situs;
DROP INDEX IF EXISTS idx_parcels_owner_name;
DROP INDEX IF EXISTS idx_parcels_pin;

