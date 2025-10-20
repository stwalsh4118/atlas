# Field Mapping Configuration

This directory contains field mapping configurations for importing county parcel data into the `tax_parcels` database table.

## Overview

Different counties use different naming conventions for their parcel data fields. For example:
- Montgomery County, TX uses: `OBJECTID`, `PIN`, `ownerName`, `situs`
- Other counties might use: `OBJECT_ID`, `PARCEL_ID`, `OWNER_NAME`, `ADDRESS`

Field mapping configurations provide a flexible way to map source data field names to our standardized database schema columns.

## Configuration Format

Each mapping configuration is a JSON file with the following structure:

```json
{
  "county": "county-name-state",
  "county_name": "County Name",
  "state": "TX",
  "source_file": "./data/county_parcels.geojson",
  "source_format": "GeoJSON",
  "source_crs": "EPSG:4326",
  "target_crs": "EPSG:4326",
  "record_count": 325071,
  "file_size_mb": 837,
  
  "field_mappings": {
    "object_id": "OBJECTID",
    "pin": "PIN",
    "owner_name": "ownerName",
    "situs": "situs",
    ...
  }
}
```

### Key Fields

- **county**: Unique identifier for the county (lowercase-with-dashes format)
- **county_name**: Human-readable county name
- **state**: Two-letter state code
- **source_file**: Path to the source geospatial data file
- **source_format**: File format (GeoJSON, Shapefile, Geodatabase)
- **source_crs**: Source coordinate reference system (EPSG code)
- **target_crs**: Target coordinate reference system (always EPSG:4326 for our database)
- **record_count**: Number of parcel records in the source file
- **file_size_mb**: File size in megabytes
- **field_mappings**: Object mapping database column names (left) to source field names (right)

## Database Schema

Our `tax_parcels` table has the following columns that need to be mapped:

### Primary Identifiers
- `object_id` (INTEGER, required) - Unique object identifier
- `pin` (INTEGER, required) - Parcel Identification Number
- `pid` (INTEGER, optional) - Secondary property ID

### Parcel Subdivision Info
- `state_cd` (VARCHAR) - State/parcel classification code
- `block` (INTEGER) - Block number
- `lot` (VARCHAR) - Lot identifier
- `tract` (VARCHAR) - Tract identifier

### Owner Information
- `owner_name` (VARCHAR) - Property owner name
- `owner_address` (TEXT) - Owner mailing address

### Property Details
- `situs` (VARCHAR) - Property street address
- `as_code` (VARCHAR) - Subdivision/area code
- `legal_description` (TEXT) - Legal property description

### Improvement/Building Info
- `imprv_actual_year_built` (INTEGER) - Year building was built
- `imprv_main_area` (INTEGER) - Square footage of main structure

### Tax Information
- `p_year` (INTEGER) - Tax year
- `p_version` (INTEGER) - Tax version
- `p_roll_corr` (INTEGER) - Roll correction number
- `taxing_units` (VARCHAR) - Taxing districts (comma-separated)
- `exemptions` (VARCHAR) - Tax exemptions
- `market_area` (VARCHAR) - Market/appraisal area code

### Metadata
- `county_name` (VARCHAR) - County name (set automatically to county_name from config)

### Geometry
- `geom` (GEOMETRY) - Parcel boundary geometry (handled automatically)

## Creating a New Mapping

### Step 1: Validate the Source Data

First, run the validation script to discover the field names in your source data:

```bash
./scripts/validate-geodata.sh ./data/your-county-parcels.geojson
```

This will output all available field names and their types.

### Step 2: Copy the Template

Create a new mapping file based on the template:

```bash
cp ./scripts/mappings/template.json ./scripts/mappings/yourcounty-tx.json
```

### Step 3: Update Metadata

Edit the new file and update the metadata fields:

```json
{
  "county": "yourcounty-tx",
  "county_name": "Your County",
  "state": "TX",
  "source_file": "./data/yourcounty_parcels.geojson",
  "source_crs": "EPSG:2278",  // or whatever the source CRS is
  "record_count": 50000,
  "file_size_mb": 250
}
```

### Step 4: Map the Fields

For each database column in `field_mappings`, replace the value with the actual field name from your source data:

```json
"field_mappings": {
  "object_id": "OBJECTID",     // Match your source field name exactly
  "pin": "PARCEL_ID",          // Could be PIN, PARCEL_ID, APN, etc.
  "owner_name": "OWNER",       // Could be OWNER, OWNERNAME, OWNER_NAME, etc.
  "situs": "SITE_ADDR",        // Could be SITUS, ADDRESS, SITE_ADDR, etc.
  ...
}
```

**Important Notes:**
- Field names are **case-sensitive** and must **exactly match** the source data
- Use `null` for fields that don't exist in your source data
- The template includes common field name variations in comments to help you identify the correct source field

### Step 5: Validate the Mapping

Verify your mapping is valid JSON:

```bash
cat ./scripts/mappings/yourcounty-tx.json | jq '.'
```

If jq outputs the JSON without errors, your mapping is syntactically correct.

## Common Field Name Variations

Here are common variations you might encounter:

| Database Column | Common Source Names |
|----------------|---------------------|
| object_id | OBJECTID, OBJECT_ID, OID, FID |
| pin | PIN, PARCEL_ID, PARCEL, APN, PARCEL_NUM |
| owner_name | OWNER, OWNERNAME, OWNER_NAME, OWNER_FULL_NAME |
| situs | SITUS, ADDRESS, SITE_ADDR, SITE_ADDRESS, PROPERTY_ADDRESS |
| state_cd | stateCd, STATE_CD, STATE_CODE, PARCEL_STATE |
| legal_description | legalDescription, LEGAL_DESC, LEGAL_DESCRIPTION |
| imprv_actual_year_built | YEAR_BUILT, YR_BUILT, BUILD_YEAR, ACTUAL_YEAR_BUILT |
| imprv_main_area | SQFT, SQUARE_FEET, BUILDING_AREA, LIVING_AREA |

## Using a Mapping Configuration

Once you've created a mapping configuration, you can use it with the import script:

```bash
./scripts/import-parcels.sh --config ./scripts/mappings/yourcounty-tx.json
```

The import script will:
1. Read the field mappings from the configuration file
2. Transform the source CRS to the target CRS if needed
3. Map source field names to database column names
4. Import the data into the `tax_parcels` table

## Examples

### Example 1: Montgomery County, TX (Included)

Montgomery County data uses mixed-case field names and is already in EPSG:4326:

```json
{
  "county": "montgomery-tx",
  "source_file": "./data/montgomery_parcels.geojson",
  "source_crs": "EPSG:4326",
  "field_mappings": {
    "object_id": "OBJECTID",
    "pin": "PIN",
    "owner_name": "ownerName",
    "situs": "situs"
  }
}
```

### Example 2: Hypothetical County with Different Names

```json
{
  "county": "example-tx",
  "source_file": "./data/example_parcels.shp",
  "source_crs": "EPSG:2278",
  "field_mappings": {
    "object_id": "FID",
    "pin": "PARCEL_NUM",
    "owner_name": "OWNER_NAME",
    "situs": "SITE_ADDRESS"
  }
}
```

## Troubleshooting

### Field Name Not Found

If the import script reports that a field name doesn't exist:
1. Run the validation script to see the actual field names
2. Compare with your mapping configuration
3. Remember: field names are case-sensitive

### CRS Transformation Issues

If you encounter CRS transformation errors:
1. Verify the source CRS using the validation script
2. Ensure you've specified the correct EPSG code in `source_crs`
3. The target CRS should always be `EPSG:4326`

### Missing Required Fields

If your source data is missing required fields (`object_id` or `pin`):
1. Check if the field exists under a different name
2. Look at the common variations table above
3. If truly missing, you may need to generate unique values (contact dev team)

## Files

- `template.json` - Template configuration with all possible fields and common variations
- `montgomery-tx.json` - Montgomery County, Texas mapping (reference implementation)
- `README.md` - This documentation file

## Related Documentation

- `../../docs/delivery/3/prd.md` - PBI 3: Data Import Pipeline
- `../../docs/delivery/3/3-3.md` - Task 3-3: Create field mapping configuration
- `../validate-geodata.sh` - Validation script to discover source field names
- `../import-parcels.sh` - Import script that uses these mappings (Task 3-4)

