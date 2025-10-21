package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stwalsh4118/atlas/api/internal/database"
	"github.com/stwalsh4118/atlas/api/internal/models"
)

// ParcelRepository defines the interface for parcel data access operations.
type ParcelRepository interface {
	// FindByPoint finds the parcel that contains the given lat/lng point.
	// Returns nil, nil if no parcel is found (not an error).
	// Returns error only for actual database failures.
	FindByPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error)
}

// parcelRepository is the concrete implementation of ParcelRepository.
type parcelRepository struct {
	db *database.Database
}

// NewParcelRepository creates a new instance of ParcelRepository.
func NewParcelRepository(db *database.Database) ParcelRepository {
	return &parcelRepository{
		db: db,
	}
}

// FindByPoint queries the database for a parcel that contains the given point.
// It uses PostGIS ST_Contains to perform a point-in-polygon spatial query.
// The spatial index on the geom column is automatically used by PostGIS.
//
// Note: PostGIS functions expect (longitude, latitude) order, not (lat, lng).
func (r *parcelRepository) FindByPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error) {
	query := `
		SELECT 
			id,
			object_id,
			pin,
			pid,
			state_cd,
			block,
			lot,
			tract,
			owner_name,
			owner_address,
			situs,
			as_code,
			legal_description,
			imprv_actual_year_built,
			imprv_main_area,
			market_area,
			p_year,
			p_version,
			p_roll_corr,
			taxing_units,
			exemptions,
			county_name,
			ST_AsGeoJSON(geom) as geometry,
			created_at,
			updated_at
		FROM tax_parcels
		WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		LIMIT 1
	`

	var parcel models.TaxParcel
	var geomJSON []byte

	// Execute query - note: PostGIS uses (lng, lat) order
	err := r.db.Pool.QueryRow(ctx, query, lng, lat).Scan(
		&parcel.ID,
		&parcel.ObjectID,
		&parcel.PIN,
		&parcel.PID,
		&parcel.StateCd,
		&parcel.Block,
		&parcel.Lot,
		&parcel.Tract,
		&parcel.OwnerName,
		&parcel.OwnerAddress,
		&parcel.Situs,
		&parcel.AsCode,
		&parcel.LegalDescription,
		&parcel.ImprvActualYearBuilt,
		&parcel.ImprvMainArea,
		&parcel.MarketArea,
		&parcel.PYear,
		&parcel.PVersion,
		&parcel.PRollCorr,
		&parcel.TaxingUnits,
		&parcel.Exemptions,
		&parcel.CountyName,
		&geomJSON,
		&parcel.CreatedAt,
		&parcel.UpdatedAt,
	)

	// Handle no rows found - this is not an error at the repository level
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query parcel at point (lat=%f, lng=%f): %w", lat, lng, err)
	}

	// Parse GeoJSON geometry into Polygon type using its Scanner
	if err := parcel.Geom.Scan(geomJSON); err != nil {
		return nil, fmt.Errorf("failed to parse geometry for parcel %d: %w", parcel.ID, err)
	}

	return &parcel, nil
}
