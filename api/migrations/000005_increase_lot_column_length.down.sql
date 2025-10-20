-- Revert lot column length from VARCHAR(100) back to VARCHAR(50)
-- WARNING: This will truncate any values longer than 50 characters

ALTER TABLE tax_parcels 
    ALTER COLUMN lot TYPE VARCHAR(50);

COMMENT ON COLUMN tax_parcels.lot IS 'Lot identifier';

