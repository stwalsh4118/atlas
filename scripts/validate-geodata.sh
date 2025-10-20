#!/bin/bash
#
# validate-geodata.sh
# Pre-import validation script for geospatial data files
#
# This script validates GeoJSON, Shapefile, or Geodatabase files before import by:
# - Checking file existence and readability
# - Detecting CRS/SRID
# - Listing all attribute fields
# - Reporting record count
# - Displaying sample data
# - Checking for common required fields
#
# Usage: ./validate-geodata.sh <path/to/file.geojson|.shp|.gdb>
#

set -e  # Exit on error

# Constants
readonly SCRIPT_NAME=$(basename "$0")
readonly EXIT_SUCCESS=0
readonly EXIT_ERROR=1
readonly SAMPLE_SIZE=5

# Color codes for output
readonly COLOR_RESET='\033[0m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[1;33m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_BLUE='\033[0;34m'

# Function: Print usage
usage() {
    cat << EOF
Usage: ${SCRIPT_NAME} <geodata_file_path>

Validates geospatial data files before import into the database.

Supported Formats:
    - GeoJSON (.geojson, .json)
    - Shapefile (.shp)
    - File Geodatabase (.gdb)
    - KML (.kml)

Arguments:
    geodata_file_path    Path to the geospatial file to validate

Examples:
    ${SCRIPT_NAME} ./data/montgomery_parcels.geojson
    ${SCRIPT_NAME} ./data/montgomery_parcels.shp
    ${SCRIPT_NAME} /path/to/county/parcels.gdb

Exit Codes:
    0    Success - file is valid
    1    Error - validation failed

Requirements:
    - ogrinfo (GDAL) must be installed
    - For shapefiles: must include .shp, .shx, .dbf, .prj files

EOF
}

# Function: Print colored output
print_section() {
    echo -e "\n${COLOR_BLUE}==== $1 ====${COLOR_RESET}"
}

print_success() {
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} $1"
}

print_warning() {
    echo -e "${COLOR_YELLOW}⚠${COLOR_RESET} $1"
}

print_error() {
    echo -e "${COLOR_RED}✗${COLOR_RESET} $1"
}

# Function: Check dependencies
check_dependencies() {
    if ! command -v ogrinfo &> /dev/null; then
        print_error "ogrinfo not found. Please install GDAL/OGR tools:"
        echo "  Ubuntu/Debian: sudo apt-get install gdal-bin"
        echo "  macOS:         brew install gdal"
        exit ${EXIT_ERROR}
    fi
}

# Function: Validate geodata file path
validate_file_path() {
    local geodata_file="$1"
    local extension="${geodata_file##*.}"
    
    # Check if file exists
    if [[ ! -e "${geodata_file}" ]]; then
        print_error "File not found: ${geodata_file}"
        exit ${EXIT_ERROR}
    fi
    
    print_success "Found: ${geodata_file}"
    
    # Check file type and additional requirements
    case "${extension}" in
        shp)
            # Check for required shapefile companion files
            local base_path="${geodata_file%.shp}"
            local missing_files=()
            
            [[ ! -f "${base_path}.shx" ]] && missing_files+=(".shx")
            [[ ! -f "${base_path}.dbf" ]] && missing_files+=(".dbf")
            [[ ! -f "${base_path}.prj" ]] && missing_files+=(".prj")
            
            if [[ ${#missing_files[@]} -gt 0 ]]; then
                print_warning "Missing shapefile companion files: ${missing_files[*]}"
                echo "  Import may fail or CRS detection may not work"
            else
                print_success "All required shapefile files present (.shp, .shx, .dbf, .prj)"
            fi
            ;;
        geojson|json)
            print_success "GeoJSON format detected"
            # Check file size
            local file_size_mb=$(du -m "${geodata_file}" | cut -f1)
            echo "  File size: ${file_size_mb}MB"
            if [[ ${file_size_mb} -gt 1000 ]]; then
                print_warning "Large file (${file_size_mb}MB) - import may take several minutes"
            fi
            ;;
        gdb)
            print_success "File Geodatabase format detected"
            ;;
        kml)
            print_success "KML format detected"
            ;;
        *)
            print_warning "Unknown format: .${extension}"
            echo "  Will attempt to process anyway"
            ;;
    esac
}

# Function: Get layer name from geodata file
get_layer_name() {
    local geodata_file="$1"
    ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep "^Layer name:" | sed 's/Layer name: //' | xargs
}

# Function: Detect CRS
detect_crs() {
    local geodata_file="$1"
    
    print_section "Coordinate Reference System (CRS)"
    
    # Get CRS info
    local crs_info
    crs_info=$(ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep -A 10 "Layer SRS WKT:" || true)
    
    # For GeoJSON files, try simple text search for CRS (faster than parsing entire JSON)
    local extension="${geodata_file##*.}"
    if [[ "${extension}" == "geojson" || "${extension}" == "json" ]]; then
        # CRS is typically in first 20 lines, search for "EPSG:####" pattern
        local json_crs
        json_crs=$(head -n 20 "${geodata_file}" | grep -oE 'EPSG:[0-9]+' | head -n 1 || echo "")
        if [[ -n "${json_crs}" && "${json_crs}" =~ EPSG:([0-9]+) ]]; then
            local epsg="${BASH_REMATCH[1]}"
            print_success "EPSG:${epsg} (from GeoJSON crs property)"
            
            # Provide context for common CRS
            case "${epsg}" in
                2278)
                    echo "  → NAD83 / Texas South Central (feet)"
                    echo "  → Will need transformation to EPSG:4326 (WGS84)"
                    ;;
                4326)
                    echo "  → WGS 84 (lat/lon)"
                    echo "  → Already in target CRS, no transformation needed! ✓"
                    ;;
                *)
                    echo "  → Will need transformation to EPSG:4326 (WGS84)"
                    ;;
            esac
            return
        fi
    fi
    
    if [[ -z "${crs_info}" ]]; then
        print_warning "CRS not detected in file metadata"
        echo "  GeoJSON files without explicit CRS are assumed to be EPSG:4326 (WGS84)"
        echo "  Or you may need to specify source CRS manually during import"
        return
    fi
    
    # Try to extract EPSG code (handles both formats: ID["EPSG",4326] and EPSG","4326")
    local epsg
    epsg=$(echo "${crs_info}" | grep -oE 'EPSG[",]+[0-9]+' | grep -oE '[0-9]+' | head -n 1 || echo "")
    
    if [[ -n "${epsg}" ]]; then
        print_success "EPSG:${epsg}"
        
        # Provide context for common Texas CRS
        case "${epsg}" in
            2278)
                echo "  → NAD83 / Texas South Central (feet)"
                echo "  → Will need transformation to EPSG:4326 (WGS84)"
                ;;
            3081)
                echo "  → NAD83(HARN) / Texas South Central"
                echo "  → Will need transformation to EPSG:4326 (WGS84)"
                ;;
            4326)
                echo "  → WGS 84 (lat/lon)"
                echo "  → Already in target CRS, no transformation needed"
                ;;
            *)
                echo "  → Will need transformation to EPSG:4326 (WGS84)"
                ;;
        esac
    else
        print_warning "EPSG code not found in CRS definition"
        echo "  CRS exists but you may need to specify SRID manually"
    fi
    
    # Show projection name
    local proj_name
    proj_name=$(echo "${crs_info}" | grep 'PROJCS\|GEOGCS' | head -n 1 | sed 's/.*"\(.*\)".*/\1/' || echo "")
    if [[ -n "${proj_name}" ]]; then
        echo "  Projection: ${proj_name}"
    fi
}

# Function: Get field list
list_fields() {
    local geodata_file="$1"
    local extension="${geodata_file##*.}"
    
    print_section "Attribute Fields"
    
    # For GeoJSON, try jq on a valid JSON snippet
    if [[ "${extension}" == "geojson" || "${extension}" == "json" ]] && command -v jq &> /dev/null; then
        echo "Extracting fields using jq (reading first complete feature)..."
        local fields
        # Use jq in streaming mode to only parse first feature's properties
        fields=$(jq -r '.features[0].properties | keys | .[]' "${geodata_file}" 2>/dev/null || true)
        
        if [[ -n "${fields}" ]]; then
            echo "Fields found in file:"
            echo "${fields}" | while IFS= read -r field; do
                echo "  ${field}: (use jq to inspect type)"
            done
        else
            print_error "Could not extract field information with jq"
            return ${EXIT_ERROR}
        fi
    else
        # Fall back to ogrinfo for other formats
        local fields
        fields=$(ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep -E "^[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*:" || true)
        
        if [[ -z "${fields}" ]]; then
            print_error "Could not extract field information"
            return ${EXIT_ERROR}
        fi
        
        echo "Fields found in file:"
        echo "${fields}" | while IFS= read -r field; do
            echo "  ${field}"
        done
    fi
    
    # Check for common required fields (case-insensitive)
    local fields_upper
    fields_upper=$(echo "${fields}" | tr '[:lower:]' '[:upper:]')
    
    echo ""
    echo "Common field mappings for our schema:"
    
    check_field_mapping "OBJECTID|OBJECT_ID|OID" "object_id" "${fields_upper}"
    check_field_mapping "PIN|PARCEL_ID|PARCEL|APN" "pin" "${fields_upper}"
    check_field_mapping "OWNER|OWNERNAME|OWNER_NAME" "owner_name" "${fields_upper}"
    check_field_mapping "SITUS|ADDRESS|SITE_ADDR|SITUS_ADDR" "situs" "${fields_upper}"
}

# Function: Check for field mapping
check_field_mapping() {
    local pattern="$1"
    local target_field="$2"
    local fields="$3"
    
    IFS='|' read -ra field_options <<< "${pattern}"
    
    for field_option in "${field_options[@]}"; do
        if echo "${fields}" | grep -q "${field_option}"; then
            print_success "${target_field} ← ${field_option}"
            return
        fi
    done
    
    print_warning "${target_field} ← NO MATCH (expected one of: ${pattern})"
}

# Function: Get record count
get_record_count() {
    local geodata_file="$1"
    
    print_section "Record Count"
    
    local count
    count=$(ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep "Feature Count:" | grep -o '[0-9]*' || echo "0")
    
    if [[ "${count}" -gt 0 ]]; then
        print_success "${count} features/records"
        
        # Provide time estimate
        if [[ "${count}" -lt 10000 ]]; then
            echo "  Estimated import time: < 30 seconds"
        elif [[ "${count}" -lt 100000 ]]; then
            echo "  Estimated import time: 30-90 seconds"
        else
            echo "  Estimated import time: 2-5 minutes"
        fi
    else
        print_warning "No features found or count unavailable"
    fi
}

# Function: Get geometry type
get_geometry_type() {
    local geodata_file="$1"
    local extension="${geodata_file##*.}"
    
    print_section "Geometry Information"
    
    # For GeoJSON, search for geometry type in first feature (text search, no JSON parsing)
    local geom_type
    if [[ "${extension}" == "geojson" || "${extension}" == "json" ]]; then
        # Look for "type": "Polygon" or "type": "MultiPolygon" pattern in first 100 lines
        geom_type=$(head -n 100 "${geodata_file}" | grep -oE '"type"[[:space:]]*:[[:space:]]*"(Multi)?Polygon"' | head -n 1 | grep -oE '(Multi)?Polygon' || echo "Unknown")
    else
        geom_type=$(ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep "Geometry:" | sed 's/Geometry: //' | xargs || echo "Unknown")
    fi
    
    echo "Geometry Type: ${geom_type}"
    
    case "${geom_type}" in
        "Polygon")
            print_success "Polygon type detected"
            echo "  → Will be converted to MultiPolygon during import"
            ;;
        "Multi Polygon"|"MultiPolygon")
            print_success "MultiPolygon type detected"
            echo "  → Already in correct format"
            ;;
        *)
            print_warning "Unexpected geometry type: ${geom_type}"
            echo "  → Expected Polygon or MultiPolygon for parcel data"
            ;;
    esac
    
    # Get extent/bounding box
    local extent
    extent=$(ogrinfo -so -al "${geodata_file}" 2>/dev/null | grep "Extent:" | sed 's/Extent: //' || echo "")
    if [[ -n "${extent}" ]]; then
        echo "Extent (bounding box): ${extent}"
    fi
}

# Function: Show sample data
show_sample_data() {
    local geodata_file="$1"
    
    print_section "Sample Data (first ${SAMPLE_SIZE} records)"
    
    ogrinfo -al "${geodata_file}" 2>/dev/null | head -n 100 | grep -A 20 "^  " | head -n 50 || true
}

# Main function
main() {
    # Check arguments
    if [[ $# -ne 1 ]]; then
        usage
        exit ${EXIT_ERROR}
    fi
    
    local geodata_file="$1"
    
    # Show help if requested
    if [[ "${geodata_file}" == "-h" ]] || [[ "${geodata_file}" == "--help" ]]; then
        usage
        exit ${EXIT_SUCCESS}
    fi
    
    # Banner
    echo "========================================"
    echo "  Geospatial Data Validation"
    echo "========================================"
    echo "File: ${geodata_file}"
    
    # Run validation steps
    check_dependencies
    validate_file_path "${geodata_file}"
    detect_crs "${geodata_file}"
    get_geometry_type "${geodata_file}"
    list_fields "${geodata_file}"
    get_record_count "${geodata_file}"
    show_sample_data "${geodata_file}"
    
    # Summary
    print_section "Validation Complete"
    print_success "File is ready for import"
    echo ""
    echo "Next steps:"
    echo "  1. Create field mapping configuration (Task 3-3)"
    echo "  2. Run import script with this shapefile (Task 3-4)"
    
    exit ${EXIT_SUCCESS}
}

# Run main
main "$@"

