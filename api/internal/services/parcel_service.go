package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/stwalsh4118/atlas/api/internal/logger"
	"github.com/stwalsh4118/atlas/api/internal/models"
	"github.com/stwalsh4118/atlas/api/internal/repository"
)

// Coordinate validation constants
const (
	MinLatitude  = -90.0
	MaxLatitude  = 90.0
	MinLongitude = -180.0
	MaxLongitude = 180.0
)

// Radius validation constants
const (
	MinRadiusMeters = 1
	MaxRadiusMeters = 5000
)

// Service-level errors
var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrParcelNotFound     = errors.New("parcel not found")
	ErrInvalidRadius      = errors.New("radius must be between 1 and 5000 meters")
)

// ParcelService defines the interface for parcel business logic operations.
type ParcelService interface {
	// GetParcelAtPoint retrieves the parcel that contains the given lat/lng point.
	// Returns ErrInvalidCoordinates if coordinates are out of valid range.
	// Returns ErrParcelNotFound if no parcel exists at the point.
	// Returns error for database failures.
	GetParcelAtPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error)

	// GetNearbyParcels retrieves all parcels within the specified radius of the given point.
	// Returns ErrInvalidCoordinates if coordinates are out of valid range.
	// Returns ErrInvalidRadius if radius is not between 1 and 5000 meters.
	// Returns empty slice if no parcels found (not an error).
	// Returns error for database failures.
	GetNearbyParcels(ctx context.Context, lat, lng float64, radiusMeters int) ([]repository.ParcelWithDistance, error)
}

// parcelService is the concrete implementation of ParcelService.
type parcelService struct {
	repo repository.ParcelRepository
	log  *logger.Logger
}

// NewParcelService creates a new instance of ParcelService.
func NewParcelService(repo repository.ParcelRepository, log *logger.Logger) ParcelService {
	return &parcelService{
		repo: repo,
		log:  log,
	}
}

// GetParcelAtPoint retrieves the parcel containing the given point.
// It validates the coordinates, logs the query, and transforms repository
// responses into appropriate business-level errors.
func (s *parcelService) GetParcelAtPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error) {
	// Validate latitude range
	if lat < MinLatitude || lat > MaxLatitude {
		s.log.Warn("Invalid latitude provided", map[string]interface{}{
			"lat": lat,
			"lng": lng,
		})
		return nil, fmt.Errorf("%w: latitude must be between %f and %f, got %f",
			ErrInvalidCoordinates, MinLatitude, MaxLatitude, lat)
	}

	// Validate longitude range
	if lng < MinLongitude || lng > MaxLongitude {
		s.log.Warn("Invalid longitude provided", map[string]interface{}{
			"lat": lat,
			"lng": lng,
		})
		return nil, fmt.Errorf("%w: longitude must be between %f and %f, got %f",
			ErrInvalidCoordinates, MinLongitude, MaxLongitude, lng)
	}

	// Log the query
	s.log.Info("Querying parcel at point", map[string]interface{}{
		"lat": lat,
		"lng": lng,
	})

	// Query repository
	parcel, err := s.repo.FindByPoint(ctx, lat, lng)
	if err != nil {
		s.log.Error("Failed to query parcel at point", err, map[string]interface{}{
			"lat": lat,
			"lng": lng,
		})
		return nil, fmt.Errorf("failed to query parcel: %w", err)
	}

	// Repository returns nil, nil when no parcel found - transform to domain error
	if parcel == nil {
		s.log.Debug("No parcel found at point", map[string]interface{}{
			"lat": lat,
			"lng": lng,
		})
		return nil, ErrParcelNotFound
	}

	// Success - log and return parcel
	s.log.Info("Parcel found at point", map[string]interface{}{
		"lat":       lat,
		"lng":       lng,
		"parcel_id": parcel.ID,
		"owner":     parcel.OwnerName,
	})

	return parcel, nil
}

// GetNearbyParcels retrieves all parcels within the specified radius of the given point.
// It validates coordinates and radius, logs the query, and returns results ordered by distance.
func (s *parcelService) GetNearbyParcels(ctx context.Context, lat, lng float64, radiusMeters int) ([]repository.ParcelWithDistance, error) {
	// Validate latitude range
	if lat < MinLatitude || lat > MaxLatitude {
		s.log.Warn("Invalid latitude provided", map[string]interface{}{
			"lat":    lat,
			"lng":    lng,
			"radius": radiusMeters,
		})
		return nil, fmt.Errorf("%w: latitude must be between %f and %f, got %f",
			ErrInvalidCoordinates, MinLatitude, MaxLatitude, lat)
	}

	// Validate longitude range
	if lng < MinLongitude || lng > MaxLongitude {
		s.log.Warn("Invalid longitude provided", map[string]interface{}{
			"lat":    lat,
			"lng":    lng,
			"radius": radiusMeters,
		})
		return nil, fmt.Errorf("%w: longitude must be between %f and %f, got %f",
			ErrInvalidCoordinates, MinLongitude, MaxLongitude, lng)
	}

	// Validate radius range
	if radiusMeters < MinRadiusMeters || radiusMeters > MaxRadiusMeters {
		s.log.Warn("Invalid radius provided", map[string]interface{}{
			"lat":    lat,
			"lng":    lng,
			"radius": radiusMeters,
		})
		return nil, fmt.Errorf("%w: got %d", ErrInvalidRadius, radiusMeters)
	}

	// Log the query
	s.log.Info("Querying nearby parcels", map[string]interface{}{
		"lat":    lat,
		"lng":    lng,
		"radius": radiusMeters,
	})

	// Query repository
	parcels, err := s.repo.FindNearby(ctx, lat, lng, radiusMeters)
	if err != nil {
		s.log.Error("Failed to query nearby parcels", err, map[string]interface{}{
			"lat":    lat,
			"lng":    lng,
			"radius": radiusMeters,
		})
		return nil, fmt.Errorf("failed to query nearby parcels: %w", err)
	}

	// Log results
	s.log.Info("Nearby parcels found", map[string]interface{}{
		"lat":    lat,
		"lng":    lng,
		"radius": radiusMeters,
		"count":  len(parcels),
	})

	return parcels, nil
}
