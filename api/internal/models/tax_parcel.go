package models

import (
	"time"
)

// TaxParcel represents a property tax parcel with boundary geometry.
// Based on Montgomery County, Texas GeoJSON data format.
// All nullable fields use pointers to distinguish between zero values and NULL.
type TaxParcel struct {
	CreatedAt            time.Time    `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt            time.Time    `gorm:"column:updated_at" json:"updatedAt"`
	LegalDescription     *string      `gorm:"type:text;column:legal_description" json:"legalDescription,omitempty"`
	Situs                *string      `gorm:"size:500;index;column:situs" json:"situs,omitempty"`
	StateCd              *string      `gorm:"size:10;column:state_cd" json:"stateCd,omitempty"`
	Block                *int         `gorm:"column:block" json:"block,omitempty"`
	Lot                  *string      `gorm:"size:50;column:lot" json:"lot,omitempty"`
	Tract                *string      `gorm:"size:50;column:tract" json:"tract,omitempty"`
	OwnerName            *string      `gorm:"size:500;index;column:owner_name" json:"ownerName,omitempty"`
	ImprvMainArea        *int         `gorm:"column:imprv_main_area" json:"imprvMainArea,omitempty"`
	ImprvActualYearBuilt *int         `gorm:"column:imprv_actual_year_built" json:"imprvActualYearBuilt,omitempty"`
	AsCode               *string      `gorm:"size:50;column:as_code" json:"asCode,omitempty"`
	PID                  *int         `gorm:"column:pid" json:"pid,omitempty"`
	MarketArea           *string      `gorm:"size:50;column:market_area" json:"marketArea,omitempty"`
	OwnerAddress         *string      `gorm:"type:text;column:owner_address" json:"ownerAddress,omitempty"`
	PYear                *int         `gorm:"column:p_year" json:"pYear,omitempty"`
	PVersion             *int         `gorm:"column:p_version" json:"pVersion,omitempty"`
	PRollCorr            *int         `gorm:"column:p_roll_corr" json:"pRollCorr,omitempty"`
	TaxingUnits          *string      `gorm:"size:255;column:taxing_units" json:"taxingUnits,omitempty"`
	Exemptions           *string      `gorm:"size:255;column:exemptions" json:"exemptions,omitempty"`
	CountyName           string       `gorm:"size:100;default:'Montgomery';index;column:county_name" json:"countyName"`
	Geom                 MultiPolygon `gorm:"type:geometry(MultiPolygon,4326);not null;column:geom" json:"geometry"`
	ID                   uint         `gorm:"primaryKey" json:"id"`
	PIN                  int          `gorm:"index;not null;column:pin" json:"pin"`
	ObjectID             int          `gorm:"uniqueIndex;not null;column:object_id" json:"objectId"`
}

// TableName specifies the table name for GORM.
// This matches the tax_parcels table created in migrations.
func (TaxParcel) TableName() string {
	return "tax_parcels"
}
