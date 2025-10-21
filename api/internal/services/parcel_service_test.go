package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stwalsh4118/atlas/api/internal/logger"
	"github.com/stwalsh4118/atlas/api/internal/models"
	"github.com/stwalsh4118/atlas/api/internal/repository"
)

// MockParcelRepository is a mock implementation of ParcelRepository for testing
type MockParcelRepository struct {
	mock.Mock
}

func (m *MockParcelRepository) FindByPoint(ctx context.Context, lat, lng float64) (*models.TaxParcel, error) {
	args := m.Called(ctx, lat, lng)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	parcel, ok := args.Get(0).(*models.TaxParcel)
	if !ok {
		return nil, args.Error(1)
	}
	return parcel, args.Error(1)
}

func (m *MockParcelRepository) FindNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]repository.ParcelWithDistance, error) {
	args := m.Called(ctx, lat, lng, radiusMeters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	parcels, ok := args.Get(0).([]repository.ParcelWithDistance)
	if !ok {
		return nil, args.Error(1)
	}
	return parcels, args.Error(1)
}

func TestGetParcelAtPoint_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502

	ownerName := "John Doe"
	expectedParcel := &models.TaxParcel{
		ID:         12345,
		ObjectID:   123456,
		OwnerName:  &ownerName,
		CountyName: "Montgomery",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockRepo.On("FindByPoint", ctx, lat, lng).Return(expectedParcel, nil)

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parcel)
	assert.Equal(t, expectedParcel.ID, parcel.ID)
	assert.Equal(t, expectedParcel.OwnerName, parcel.OwnerName)
	mockRepo.AssertExpectations(t)
}

func TestGetParcelAtPoint_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502

	// Repository returns nil, nil when no parcel found
	mockRepo.On("FindByPoint", ctx, lat, lng).Return(nil, nil)

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, ErrParcelNotFound)
	mockRepo.AssertExpectations(t)
}

func TestGetParcelAtPoint_InvalidLatitude_TooHigh(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 91.0, -95.4502 // Latitude > 90

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "latitude must be between")
	// Repository should not be called for validation errors
	mockRepo.AssertNotCalled(t, "FindByPoint")
}

func TestGetParcelAtPoint_InvalidLatitude_TooLow(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := -91.0, -95.4502 // Latitude < -90

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "latitude must be between")
	mockRepo.AssertNotCalled(t, "FindByPoint")
}

func TestGetParcelAtPoint_InvalidLongitude_TooHigh(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, 181.0 // Longitude > 180

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "longitude must be between")
	mockRepo.AssertNotCalled(t, "FindByPoint")
}

func TestGetParcelAtPoint_InvalidLongitude_TooLow(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -181.0 // Longitude < -180

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "longitude must be between")
	mockRepo.AssertNotCalled(t, "FindByPoint")
}

func TestGetParcelAtPoint_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502

	dbError := errors.New("database connection failed")
	mockRepo.On("FindByPoint", ctx, lat, lng).Return(nil, dbError)

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.Contains(t, err.Error(), "failed to query parcel")
	assert.ErrorIs(t, err, dbError)
	mockRepo.AssertExpectations(t)
}

func TestGetParcelAtPoint_ContextCancellation(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	lat, lng := 30.3477, -95.4502

	mockRepo.On("FindByPoint", ctx, lat, lng).Return(nil, context.Canceled)

	// Act
	parcel, err := service.GetParcelAtPoint(ctx, lat, lng)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcel)
	assert.ErrorIs(t, err, context.Canceled)
	mockRepo.AssertExpectations(t)
}

func TestGetParcelAtPoint_BoundaryValues(t *testing.T) {
	// Test boundary values for coordinates
	testCases := []struct {
		name      string
		lat       float64
		lng       float64
		expectErr bool
	}{
		{
			name:      "Min valid latitude",
			lat:       -90.0,
			lng:       0.0,
			expectErr: false,
		},
		{
			name:      "Max valid latitude",
			lat:       90.0,
			lng:       0.0,
			expectErr: false,
		},
		{
			name:      "Min valid longitude",
			lat:       0.0,
			lng:       -180.0,
			expectErr: false,
		},
		{
			name:      "Max valid longitude",
			lat:       0.0,
			lng:       180.0,
			expectErr: false,
		},
		{
			name:      "Equator and prime meridian",
			lat:       0.0,
			lng:       0.0,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockParcelRepository)
			log := logger.New("test")
			service := NewParcelService(mockRepo, log)

			ctx := context.Background()

			if !tc.expectErr {
				mockRepo.On("FindByPoint", ctx, tc.lat, tc.lng).Return(nil, nil)
			}

			// Act
			parcel, err := service.GetParcelAtPoint(ctx, tc.lat, tc.lng)

			// Assert
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, parcel)
			} else {
				// Should get ErrParcelNotFound since we mock nil return
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrParcelNotFound)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestCoordinateConstants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, -90.0, MinLatitude)
	assert.Equal(t, 90.0, MaxLatitude)
	assert.Equal(t, -180.0, MinLongitude)
	assert.Equal(t, 180.0, MaxLongitude)
}

func TestGetNearbyParcels_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502
	radiusMeters := 1000

	ownerName := "John Doe"
	expectedParcels := []repository.ParcelWithDistance{
		{
			Parcel: models.TaxParcel{
				ID:         1,
				ObjectID:   101,
				OwnerName:  &ownerName,
				CountyName: "Montgomery",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			Distance: 100.5,
		},
		{
			Parcel: models.TaxParcel{
				ID:         2,
				ObjectID:   102,
				OwnerName:  &ownerName,
				CountyName: "Montgomery",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			Distance: 250.3,
		},
	}

	mockRepo.On("FindNearby", ctx, lat, lng, radiusMeters).Return(expectedParcels, nil)

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parcels)
	assert.Len(t, parcels, 2)
	assert.Equal(t, expectedParcels[0].Parcel.ID, parcels[0].Parcel.ID)
	assert.Equal(t, expectedParcels[0].Distance, parcels[0].Distance)
	mockRepo.AssertExpectations(t)
}

func TestGetNearbyParcels_EmptyResults(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502
	radiusMeters := 1000

	emptyResults := []repository.ParcelWithDistance{}
	mockRepo.On("FindNearby", ctx, lat, lng, radiusMeters).Return(emptyResults, nil)

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parcels)
	assert.Len(t, parcels, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetNearbyParcels_InvalidLatitude_TooHigh(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 91.0, -95.4502 // Latitude > 90
	radiusMeters := 1000

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "latitude must be between")
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_InvalidLatitude_TooLow(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := -91.0, -95.4502 // Latitude < -90
	radiusMeters := 1000

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "latitude must be between")
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_InvalidLongitude_TooHigh(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, 181.0 // Longitude > 180
	radiusMeters := 1000

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "longitude must be between")
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_InvalidLongitude_TooLow(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -181.0 // Longitude < -180
	radiusMeters := 1000

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidCoordinates)
	assert.Contains(t, err.Error(), "longitude must be between")
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_InvalidRadius_TooSmall(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502
	radiusMeters := 0 // Radius < 1

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidRadius)
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_InvalidRadius_TooLarge(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502
	radiusMeters := 5001 // Radius > 5000

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, ErrInvalidRadius)
	mockRepo.AssertNotCalled(t, "FindNearby")
}

func TestGetNearbyParcels_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx := context.Background()
	lat, lng := 30.3477, -95.4502
	radiusMeters := 1000

	dbError := errors.New("database connection failed")
	mockRepo.On("FindNearby", ctx, lat, lng, radiusMeters).Return(nil, dbError)

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.Contains(t, err.Error(), "failed to query nearby parcels")
	assert.ErrorIs(t, err, dbError)
	mockRepo.AssertExpectations(t)
}

func TestGetNearbyParcels_ContextCancellation(t *testing.T) {
	// Arrange
	mockRepo := new(MockParcelRepository)
	log := logger.New("test")
	service := NewParcelService(mockRepo, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	lat, lng := 30.3477, -95.4502
	radiusMeters := 1000

	mockRepo.On("FindNearby", ctx, lat, lng, radiusMeters).Return(nil, context.Canceled)

	// Act
	parcels, err := service.GetNearbyParcels(ctx, lat, lng, radiusMeters)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parcels)
	assert.ErrorIs(t, err, context.Canceled)
	mockRepo.AssertExpectations(t)
}

func TestGetNearbyParcels_BoundaryValues(t *testing.T) {
	// Test boundary values for coordinates and radius
	//nolint:govet // fieldalignment - test struct, optimization not critical
	testCases := []struct {
		name         string
		errType      error
		lat          float64
		lng          float64
		radiusMeters int
		expectErr    bool
	}{
		{
			name:         "Min valid latitude",
			lat:          -90.0,
			lng:          0.0,
			radiusMeters: 1000,
			expectErr:    false,
		},
		{
			name:         "Max valid latitude",
			lat:          90.0,
			lng:          0.0,
			radiusMeters: 1000,
			expectErr:    false,
		},
		{
			name:         "Min valid longitude",
			lat:          0.0,
			lng:          -180.0,
			radiusMeters: 1000,
			expectErr:    false,
		},
		{
			name:         "Max valid longitude",
			lat:          0.0,
			lng:          180.0,
			radiusMeters: 1000,
			expectErr:    false,
		},
		{
			name:         "Min valid radius",
			lat:          30.3477,
			lng:          -95.4502,
			radiusMeters: 1,
			expectErr:    false,
		},
		{
			name:         "Max valid radius",
			lat:          30.3477,
			lng:          -95.4502,
			radiusMeters: 5000,
			expectErr:    false,
		},
		{
			name:         "Zero radius (invalid)",
			lat:          30.3477,
			lng:          -95.4502,
			radiusMeters: 0,
			expectErr:    true,
			errType:      ErrInvalidRadius,
		},
		{
			name:         "Negative radius (invalid)",
			lat:          30.3477,
			lng:          -95.4502,
			radiusMeters: -100,
			expectErr:    true,
			errType:      ErrInvalidRadius,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockParcelRepository)
			log := logger.New("test")
			service := NewParcelService(mockRepo, log)

			ctx := context.Background()

			if !tc.expectErr {
				mockRepo.On("FindNearby", ctx, tc.lat, tc.lng, tc.radiusMeters).
					Return([]repository.ParcelWithDistance{}, nil)
			}

			// Act
			parcels, err := service.GetNearbyParcels(ctx, tc.lat, tc.lng, tc.radiusMeters)

			// Assert
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, parcels)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parcels)
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestRadiusConstants(t *testing.T) {
	// Verify radius constants are set correctly
	assert.Equal(t, 1, MinRadiusMeters)
	assert.Equal(t, 5000, MaxRadiusMeters)
}
