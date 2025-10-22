package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	apierrors "github.com/stwalsh4118/atlas/api/internal/errors"
	"github.com/stwalsh4118/atlas/api/internal/middleware"
	"github.com/stwalsh4118/atlas/api/internal/models"
	"github.com/stwalsh4118/atlas/api/internal/repository"
	"github.com/stwalsh4118/atlas/api/internal/services"
)

// ParcelHandler handles parcel-related HTTP requests.
type ParcelHandler struct {
	service services.ParcelService
}

// NewParcelHandler creates a new ParcelHandler instance.
func NewParcelHandler(service services.ParcelService) *ParcelHandler {
	return &ParcelHandler{
		service: service,
	}
}

// AtPointRequest represents the query parameters for the at-point endpoint.
type AtPointRequest struct {
	Lat float64 `form:"lat" binding:"required,min=-90,max=90"`
	Lng float64 `form:"lng" binding:"required,min=-180,max=180"`
}

// NearbyRequest represents the query parameters for the nearby endpoint.
type NearbyRequest struct {
	Lat    float64 `form:"lat" binding:"required,min=-90,max=90"`
	Lng    float64 `form:"lng" binding:"required,min=-180,max=180"`
	Radius int     `form:"radius,omitempty,min=1,max=5000"`
}

// ParcelResponse represents the response for parcel endpoints.
type ParcelResponse struct {
	Parcel *ParcelData `json:"parcel"`
}

// ParcelData represents the parcel data in the API response.
// This DTO includes only the fields needed by the frontend.
// Field order is optimized for memory alignment.
type ParcelData struct {
	Geometry     map[string]interface{} `json:"geometry"`
	ParcelID     string                 `json:"parcel_id,omitempty"`
	OwnerName    string                 `json:"owner_name,omitempty"`
	SitusAddress string                 `json:"situs_address,omitempty"`
	PropType     string                 `json:"prop_type,omitempty"`
	LandUse      string                 `json:"land_use,omitempty"`
	CountyName   string                 `json:"county_name"`
	Acres        float64                `json:"acres,omitempty"`
	ID           uint                   `json:"id"`
}

// NearbyResponse represents the response for the nearby endpoint.
type NearbyResponse struct {
	Parcels []ParcelWithDistance `json:"parcels"`
	Count   int                  `json:"count"`
}

// ParcelWithDistance represents a parcel with its distance from the query point.
// Field order is optimized for memory alignment.
type ParcelWithDistance struct {
	Geometry   map[string]interface{} `json:"geometry"`
	ParcelID   string                 `json:"parcel_id,omitempty"`
	OwnerName  string                 `json:"owner_name,omitempty"`
	CountyName string                 `json:"county_name"`
	Acres      float64                `json:"acres,omitempty"`
	Distance   float64                `json:"distance_meters"`
	ID         uint                   `json:"id"`
}

// AtPoint handles GET /api/v1/parcels/at-point endpoint.
// It retrieves the parcel that contains the given lat/lng point.
func (h *ParcelHandler) AtPoint(c *gin.Context) {
	log := middleware.GetLogger(c)

	// Bind and validate query parameters
	var req AtPointRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// Check if it's a validation error
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			apierrors.ValidationError(c, validationErrors)
			return
		}
		// Generic bad request for other binding errors
		apierrors.BadRequest(c, "Invalid query parameters", nil)
		return
	}

	if log != nil {
		log.Info("Processing at-point request", map[string]interface{}{
			"lat": req.Lat,
			"lng": req.Lng,
		})
	}

	// Call service layer
	parcel, err := h.service.GetParcelAtPoint(c.Request.Context(), req.Lat, req.Lng)
	if err != nil {
		// Handle service-level errors
		if errors.Is(err, services.ErrInvalidCoordinates) {
			apierrors.BadRequest(c, err.Error(), nil)
			return
		}
		if errors.Is(err, services.ErrParcelNotFound) {
			apierrors.NotFound(c, "No property found at this location")
			return
		}
		// Database or other unexpected errors
		apierrors.InternalServerError(c, "Failed to query parcel data", err)
		return
	}

	// Map TaxParcel model to ParcelData DTO
	response := ParcelResponse{
		Parcel: mapTaxParcelToDTO(parcel),
	}

	c.JSON(http.StatusOK, response)
}

// Nearby handles GET /api/v1/parcels/nearby endpoint.
// It retrieves parcels within the specified radius of the given lat/lng point.
func (h *ParcelHandler) Nearby(c *gin.Context) {
	log := middleware.GetLogger(c)

	// Bind and validate query parameters
	var req NearbyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// Check if it's a validation error
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			apierrors.ValidationError(c, validationErrors)
			return
		}
		// Generic bad request for other binding errors
		apierrors.BadRequest(c, "Invalid query parameters", nil)
		return
	}

	// Set default radius if not provided
	const defaultRadiusMeters = 1000
	if req.Radius == 0 {
		req.Radius = defaultRadiusMeters
	}

	if log != nil {
		log.Info("Processing nearby request", map[string]interface{}{
			"lat":    req.Lat,
			"lng":    req.Lng,
			"radius": req.Radius,
		})
	}

	// Call service layer
	parcels, err := h.service.GetNearbyParcels(c.Request.Context(), req.Lat, req.Lng, req.Radius)
	if err != nil {
		// Handle service-level errors
		if errors.Is(err, services.ErrInvalidCoordinates) {
			apierrors.BadRequest(c, err.Error(), nil)
			return
		}
		if errors.Is(err, services.ErrInvalidRadius) {
			apierrors.BadRequest(c, err.Error(), nil)
			return
		}
		// Database or other unexpected errors
		apierrors.InternalServerError(c, "Failed to query nearby parcels", err)
		return
	}

	// Map repository results to response DTOs
	responseParcels := make([]ParcelWithDistance, 0, len(parcels))
	for _, p := range parcels {
		responseParcels = append(responseParcels, mapParcelWithDistanceToDTO(&p))
	}

	response := NearbyResponse{
		Parcels: responseParcels,
		Count:   len(responseParcels),
	}

	c.JSON(http.StatusOK, response)
}

// mapTaxParcelToDTO converts a TaxParcel model to a ParcelData DTO.
// It handles nil pointer fields and converts geometry to GeoJSON map.
func mapTaxParcelToDTO(parcel *models.TaxParcel) *ParcelData {
	if parcel == nil {
		return nil
	}

	dto := &ParcelData{
		ID:         parcel.ID,
		CountyName: parcel.CountyName,
	}

	// Handle optional string fields
	if parcel.OwnerName != nil {
		dto.OwnerName = *parcel.OwnerName
	}
	if parcel.Situs != nil {
		dto.SitusAddress = *parcel.Situs
	}
	if parcel.AsCode != nil {
		dto.LandUse = *parcel.AsCode
	}

	// Note: The current database schema doesn't have all fields from the PRD
	// - ParcelID: Could use PIN or ObjectID when needed
	// - Acres: Would need to be calculated from geometry or added to schema
	// - PropType: Not yet in schema
	// For now, leaving these as zero values

	// Convert geometry to GeoJSON map
	// The MultiPolygon type already implements json.Marshaler for GeoJSON format
	geojson := make(map[string]interface{})
	geojson["type"] = "MultiPolygon"
	geojson["coordinates"] = parcel.Geom.Coordinates

	dto.Geometry = geojson

	return dto
}

// mapParcelWithDistanceToDTO converts a repository ParcelWithDistance to a handler ParcelWithDistance DTO.
func mapParcelWithDistanceToDTO(pwd *repository.ParcelWithDistance) ParcelWithDistance {
	dto := ParcelWithDistance{
		ID:         pwd.Parcel.ID,
		CountyName: pwd.Parcel.CountyName,
		Distance:   pwd.Distance,
	}

	// Handle optional string fields
	if pwd.Parcel.OwnerName != nil {
		dto.OwnerName = *pwd.Parcel.OwnerName
	}

	// Convert geometry to GeoJSON map
	geojson := make(map[string]interface{})
	geojson["type"] = "MultiPolygon"
	geojson["coordinates"] = pwd.Parcel.Geom.Coordinates

	dto.Geometry = geojson

	return dto
}
