-- Create standard B-tree indexes on commonly queried columns
-- These indexes improve performance for lookups and searches on non-spatial fields
-- Note: object_id already has a UNIQUE index from the constraint

-- Index for PIN lookups (very common - Parcel Identification Number)
CREATE INDEX idx_parcels_pin ON tax_parcels(pin);

-- Index for owner name searches and filtering
CREATE INDEX idx_parcels_owner_name ON tax_parcels(owner_name);

-- Index for address searches (situs = street address)
CREATE INDEX idx_parcels_situs ON tax_parcels(situs);

-- Index for county filtering (supports multi-county expansion)
CREATE INDEX idx_parcels_county ON tax_parcels(county_name);

-- Add comments for documentation
COMMENT ON INDEX idx_parcels_pin IS 'B-tree index for PIN lookups';
COMMENT ON INDEX idx_parcels_owner_name IS 'B-tree index for owner name searches';
COMMENT ON INDEX idx_parcels_situs IS 'B-tree index for address searches';
COMMENT ON INDEX idx_parcels_county IS 'B-tree index for county filtering';

