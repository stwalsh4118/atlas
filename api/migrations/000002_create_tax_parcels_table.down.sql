-- Drop tax_parcels table and its associated indexes
-- The GiST spatial index will be automatically dropped with CASCADE

DROP TABLE IF EXISTS tax_parcels CASCADE;

