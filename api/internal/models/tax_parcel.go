package models

import (
	"time"
)

// TaxParcel represents a property tax parcel with boundary geometry.
// Based on Montgomery County, Texas GeoJSON data format.
// All nullable fields use pointers to distinguish between zero values and NULL.
type TaxParcel struct {
	// Primary identifiers
	ID       uint `gorm:"primaryKey" json:"id"`
	ObjectID int  `gorm:"uniqueIndex;not null;column:object_id" json:"objectId"`
	PIN      int  `gorm:"index;not null;column:pin" json:"pin"`
	PID      *int `gorm:"column:pid" json:"pid,omitempty"`

	// Parcel subdivision info
	StateCd *string `gorm:"size:10;column:state_cd" json:"stateCd,omitempty"`
	Block   *int    `gorm:"column:block" json:"block,omitempty"`
	Lot     *string `gorm:"size:50;column:lot" json:"lot,omitempty"`
	Tract   *string `gorm:"size:50;column:tract" json:"tract,omitempty"`

	// Owner information
	OwnerName    *string `gorm:"size:500;index;column:owner_name" json:"ownerName,omitempty"`
	OwnerAddress *string `gorm:"type:text;column:owner_address" json:"ownerAddress,omitempty"`

	// Property details
	Situs            *string `gorm:"size:500;index;column:situs" json:"situs,omitempty"`
	AsCode           *string `gorm:"size:50;column:as_code" json:"asCode,omitempty"`
	LegalDescription *string `gorm:"type:text;column:legal_description" json:"legalDescription,omitempty"`

	// Improvement/building info
	ImprvActualYearBuilt *int `gorm:"column:imprv_actual_year_built" json:"imprvActualYearBuilt,omitempty"`
	ImprvMainArea        *int `gorm:"column:imprv_main_area" json:"imprvMainArea,omitempty"`

	// Tax information
	PYear       *int    `gorm:"column:p_year" json:"pYear,omitempty"`
	PVersion    *int    `gorm:"column:p_version" json:"pVersion,omitempty"`
	PRollCorr   *int    `gorm:"column:p_roll_corr" json:"pRollCorr,omitempty"`
	TaxingUnits *string `gorm:"size:255;column:taxing_units" json:"taxingUnits,omitempty"`
	Exemptions  *string `gorm:"size:255;column:exemptions" json:"exemptions,omitempty"`
	MarketArea  *string `gorm:"size:50;column:market_area" json:"marketArea,omitempty"`

	// County metadata
	CountyName string `gorm:"size:100;default:'Montgomery';index;column:county_name" json:"countyName"`

	// Spatial data - custom Polygon type with PostGIS integration
	Geom Polygon `gorm:"type:geometry(Polygon,4326);not null;column:geom" json:"geometry"`

	// Timestamps - managed by GORM automatically
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName specifies the table name for GORM.
// This matches the tax_parcels table created in migrations.
func (TaxParcel) TableName() string {
	return "tax_parcels"
}
