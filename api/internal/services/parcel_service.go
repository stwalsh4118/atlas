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

// Service-level errors
var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrParcelNotFound     = errors.New("parcel not found")
)

// ParcelService defines the interface for parcel business logic operations.
type ParcelService interface {
	// GetParcelAtPoint retrieves the parcel that contains the given lat/lng point.
	// Returns ErrInvalidCoordinates if coordinates are out of valid range.
	// Returns ErrParcelNotFound if no parcel exists at the point.
	// Returns error for database failures.
	GetParcelAtPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error)
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
