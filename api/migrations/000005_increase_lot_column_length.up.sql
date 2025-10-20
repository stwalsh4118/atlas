-- Increase lot column length from VARCHAR(50) to VARCHAR(100)
-- Montgomery County data has lot values up to 78 characters

ALTER TABLE tax_parcels 
    ALTER COLUMN lot TYPE VARCHAR(100);

COMMENT ON COLUMN tax_parcels.lot IS 'Lot identifier (increased to 100 chars to accommodate real data)';

