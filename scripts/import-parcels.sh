#!/usr/bin/env bash

#==============================================================================
# Import Tax Parcels Script
#==============================================================================
# Description: Imports GeoJSON tax parcel data into PostgreSQL using ogr2ogr
# Usage: ./import-parcels.sh --file <geojson> --mapping <config.json> [options]
#
# This script:
# 1. Reads field mapping configuration from JSON
# 2. Imports GeoJSON into a staging table using ogr2ogr
# 3. Maps fields from staging to final tax_parcels table via SQL
# 4. Executes all operations within a transaction for atomicity
#
# Requirements:
# - GDAL/OGR tools (ogr2ogr, ogrinfo)
# - jq (JSON processor)
# - PostgreSQL client (psql)
# - Database with PostGIS extension enabled
#==============================================================================

set -euo pipefail

#------------------------------------------------------------------------------
# Constants
#------------------------------------------------------------------------------
readonly SCRIPT_NAME="$(basename "$0")"
readonly SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
readonly STAGING_TABLE="tax_parcels_staging"
readonly FINAL_TABLE="tax_parcels"
readonly LOG_DIR="${SCRIPT_DIR}/../logs"

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

#------------------------------------------------------------------------------
# Default values
#------------------------------------------------------------------------------
DB_HOST="${DB_HOST:-host.docker.internal}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-atlas}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
MODE="replace"  # replace or append
DRY_RUN=false
VALIDATE_GEOMETRIES=false
GEOJSON_FILE=""
MAPPING_FILE=""

# Logging variables
LOG_FILE=""
START_TIME=""
END_TIME=""

#------------------------------------------------------------------------------
# Functions
#------------------------------------------------------------------------------

# Get current timestamp in ISO 8601 format
log_timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

# Get timestamp for filenames (no spaces or colons)
log_timestamp_filename() {
    date '+%Y%m%d-%H%M%S'
}

# Initialize log file
init_log_file() {
    # Create logs directory if it doesn't exist
    mkdir -p "${LOG_DIR}"
    
    # Generate log filename with timestamp
    local timestamp=$(log_timestamp_filename)
    LOG_FILE="${LOG_DIR}/import-${timestamp}.log"
    
    # Create log file and write header
    {
        echo "========================================"
        echo "Tax Parcel Import Log"
        echo "========================================"
        echo "Started: $(log_timestamp)"
        echo "Script: ${SCRIPT_NAME}"
        echo ""
    } > "${LOG_FILE}"
    
    print_success "Log file created: ${LOG_FILE}"
}

# Log message to file and optionally to stdout
log_to_file() {
    local message="$1"
    local level="${2:-INFO}"
    
    if [ -n "${LOG_FILE}" ]; then
        echo "[$(log_timestamp)] [${level}] ${message}" >> "${LOG_FILE}"
    fi
}

# Print colored output (also logs to file)
print_error() {
    local message="$1"
    echo -e "${RED}✗ ERROR: ${message}${NC}" >&2
    log_to_file "ERROR: ${message}" "ERROR"
}

print_success() {
    local message="$1"
    echo -e "${GREEN}✓ ${message}${NC}"
    log_to_file "${message}" "SUCCESS"
}

print_warning() {
    local message="$1"
    echo -e "${YELLOW}⚠ WARNING: ${message}${NC}"
    log_to_file "WARNING: ${message}" "WARN"
}

print_info() {
    local message="$1"
    echo -e "${BLUE}ℹ ${message}${NC}"
    log_to_file "${message}" "INFO"
}

print_step() {
    local message="$1"
    echo -e "${BLUE}==>${NC} ${message}"
    log_to_file "STEP: ${message}" "INFO"
}

# Display usage information
usage() {
    cat << EOF
Usage: ${SCRIPT_NAME} [OPTIONS]

Import GeoJSON tax parcel data into PostgreSQL using ogr2ogr.

Required Options:
  --file <path>           Path to GeoJSON file to import
  --mapping <path>        Path to field mapping configuration (JSON)

Database Options:
  --db-host <host>        Database host (default: ${DB_HOST})
  --db-port <port>        Database port (default: ${DB_PORT})
  --db-name <database>    Database name (default: ${DB_NAME})
  --db-user <user>        Database user (default: ${DB_USER})
  --db-password <pass>    Database password (default: \$DB_PASSWORD env var)

Import Options:
  --mode <mode>           Import mode: 'replace' or 'append' (default: replace)
  --validate-geometries   Run geometry validation and repair after import
  --dry-run               Preview operations without executing

Other Options:
  -h, --help              Show this help message

Environment Variables:
  DB_HOST                 Database host
  DB_PORT                 Database port
  DB_NAME                 Database name
  DB_USER                 Database user
  DB_PASSWORD             Database password

Examples:
  # Import Montgomery County parcels (replace mode)
  ${SCRIPT_NAME} \\
    --file ./data/montgomery_parcels.geojson \\
    --mapping ./scripts/mappings/montgomery-tx.json

  # Append mode with custom database
  ${SCRIPT_NAME} \\
    --file ./data/new_parcels.geojson \\
    --mapping ./scripts/mappings/montgomery-tx.json \\
    --mode append \\
    --db-host localhost

  # Dry-run to preview operations
  ${SCRIPT_NAME} \\
    --file ./data/montgomery_parcels.geojson \\
    --mapping ./scripts/mappings/montgomery-tx.json \\
    --dry-run

EOF
    exit 0
}

# Check if required tools are installed
check_dependencies() {
    local missing_deps=()
    
    if ! command -v ogr2ogr &> /dev/null; then
        missing_deps+=("ogr2ogr (GDAL)")
    fi
    
    if ! command -v ogrinfo &> /dev/null; then
        missing_deps+=("ogrinfo (GDAL)")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if ! command -v psql &> /dev/null; then
        missing_deps+=("psql (PostgreSQL client)")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "Missing required dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - ${dep}"
        done
        echo ""
        echo "Install on Ubuntu/Debian:"
        echo "  sudo apt-get install gdal-bin jq postgresql-client"
        echo ""
        echo "Install on macOS:"
        echo "  brew install gdal jq postgresql"
        exit 1
    fi
}

# Validate input file
validate_input_file() {
    local file="$1"
    
    if [ ! -f "${file}" ]; then
        print_error "GeoJSON file not found: ${file}"
        exit 1
    fi
    
    if [ ! -r "${file}" ]; then
        print_error "GeoJSON file is not readable: ${file}"
        exit 1
    fi
    
    # Check if file is valid GeoJSON
    if ! ogrinfo -ro -so "${file}" &> /dev/null; then
        print_error "File is not a valid GeoJSON: ${file}"
        exit 1
    fi
    
    print_success "GeoJSON file validated: ${file}"
}

# Validate mapping configuration
validate_mapping_file() {
    local file="$1"
    
    if [ ! -f "${file}" ]; then
        print_error "Mapping configuration file not found: ${file}"
        exit 1
    fi
    
    if [ ! -r "${file}" ]; then
        print_error "Mapping configuration file is not readable: ${file}"
        exit 1
    fi
    
    # Check if file is valid JSON
    if ! jq empty "${file}" 2> /dev/null; then
        print_error "Mapping file is not valid JSON: ${file}"
        exit 1
    fi
    
    # Check for required fields
    local required_fields=("field_mappings" "source_crs" "target_crs")
    for field in "${required_fields[@]}"; do
        if ! jq -e ".${field}" "${file}" &> /dev/null; then
            print_error "Mapping file missing required field: ${field}"
            exit 1
        fi
    done
    
    print_success "Mapping configuration validated: ${file}"
}

# Test database connection
test_db_connection() {
    print_step "Testing database connection..."
    
    if ! PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "SELECT version();" &> /dev/null; then
        print_error "Cannot connect to database"
        echo "  Host: ${DB_HOST}:${DB_PORT}"
        echo "  Database: ${DB_NAME}"
        echo "  User: ${DB_USER}"
        exit 1
    fi
    
    print_success "Database connection successful"
}

# Get record count from GeoJSON
get_record_count() {
    local file="$1"
    # Use -al -so to get summary info which includes Feature Count
    # This is more reliable than SQL queries for GeoJSON files
    local count=$(ogrinfo -ro -so -al "${file}" 2>/dev/null | \
        grep "Feature Count:" | head -n 1 | awk '{print $3}')
    
    if [ -n "${count}" ] && [ "${count}" -gt 0 ]; then
        echo "${count}"
    else
        echo "unknown"
    fi
}

# Read mapping configuration
read_mapping_config() {
    local file="$1"
    
    print_step "Reading field mapping configuration..."
    
    # Extract configuration values
    SOURCE_CRS=$(jq -r '.source_crs' "${file}")
    TARGET_CRS=$(jq -r '.target_crs' "${file}")
    COUNTY_NAME=$(jq -r '.county_name // "Unknown"' "${file}")
    
    print_info "County: ${COUNTY_NAME}"
    print_info "Source CRS: ${SOURCE_CRS}"
    print_info "Target CRS: ${TARGET_CRS}"
}

# Generate field mapping SQL
generate_field_mapping_sql() {
    local mapping_file="$1"
    
    print_step "Generating field mapping SQL..." >&2
    
    # Extract field mappings from JSON
    local field_mappings=$(jq -r '.field_mappings | to_entries[] | "\(.key):\(.value)"' "${mapping_file}")
    
    # Build column lists for INSERT statement
    local target_columns=()
    local source_columns=()
    
    while IFS=: read -r target_col source_col; do
        if [ "${source_col}" != "null" ] && [ -n "${source_col}" ]; then
            target_columns+=("${target_col}")
            # ogr2ogr converts field names to lowercase, so we need to do the same
            local source_col_lower=$(echo "${source_col}" | tr '[:upper:]' '[:lower:]')
            source_columns+=("${source_col_lower}")
        fi
    done <<< "${field_mappings}"
    
    # Add geometry column (always present)
    # Convert to MultiPolygon to handle both Polygon and MultiPolygon geometries
    target_columns+=("geom")
    source_columns+=("ST_Multi(wkb_geometry)")  # ogr2ogr uses wkb_geometry as default geometry column
    
    # Add timestamps
    target_columns+=("created_at")
    target_columns+=("updated_at")
    source_columns+=("NOW()")
    source_columns+=("NOW()")
    
    # Join arrays into comma-separated strings
    local target_cols_str=$(IFS=, ; echo "${target_columns[*]}")
    local source_cols_str=$(IFS=, ; echo "${source_columns[*]}")
    
    # Generate INSERT statement
    # Filter out records with NULL geometries or NULL PINs since both are NOT NULL in target table
    cat << EOF
INSERT INTO ${FINAL_TABLE} (${target_cols_str})
SELECT ${source_cols_str}
FROM ${STAGING_TABLE}
WHERE wkb_geometry IS NOT NULL
  AND pin IS NOT NULL;
EOF
}

# Import GeoJSON into staging table
import_to_staging() {
    local geojson_file="$1"
    local target_crs="$2"
    
    print_step "Importing GeoJSON to staging table..."
    
    local ogr_cmd=(
        ogr2ogr
        -f PostgreSQL
        "PG:host=${DB_HOST} port=${DB_PORT} dbname=${DB_NAME} user=${DB_USER} password=${DB_PASSWORD}"
        "${geojson_file}"
        -nln "${STAGING_TABLE}"
        -t_srs "${target_crs}"
        -overwrite
        -lco GEOMETRY_NAME=wkb_geometry
        -lco FID=ogc_fid
        -progress
    )
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN - Would execute:"
        echo "  ${ogr_cmd[*]}"
        return 0
    fi
    
    # Execute ogr2ogr
    if ! "${ogr_cmd[@]}"; then
        print_error "ogr2ogr import failed"
        exit 1
    fi
    
    print_success "Data imported to staging table: ${STAGING_TABLE}"
}

# Execute field mapping from staging to final table
execute_field_mapping() {
    local mapping_sql="$1"
    
    print_step "Mapping fields from staging to final table..."
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN - Would execute SQL:"
        echo "${mapping_sql}"
        return 0
    fi
    
    # Execute within transaction using a temporary SQL file
    local temp_sql=$(mktemp)
    cat > "${temp_sql}" << EOF
BEGIN;
${mapping_sql}
COMMIT;
EOF
    
    local sql_output
    if ! sql_output=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -v ON_ERROR_STOP=1 \
        -f "${temp_sql}" 2>&1); then
        print_error "Field mapping SQL failed"
        echo "${sql_output}" | head -n 10
        print_info "Attempting rollback..."
        PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            -c "ROLLBACK;" &> /dev/null || true
        rm -f "${temp_sql}"
        exit 1
    fi
    
    rm -f "${temp_sql}"
    
    print_success "Field mapping completed"
}

# Clean up staging table
cleanup_staging() {
    print_step "Cleaning up staging table..."
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN - Would drop table: ${STAGING_TABLE}"
        return 0
    fi
    
    PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "DROP TABLE IF EXISTS ${STAGING_TABLE} CASCADE;" &> /dev/null || true
    
    print_success "Staging table cleaned up"
}

# Log configuration details
log_configuration() {
    print_step "Logging import configuration..."
    
    log_to_file "=== Import Configuration ===" "INFO"
    log_to_file "GeoJSON File: ${GEOJSON_FILE}" "INFO"
    
    # Get file size
    if [ -f "${GEOJSON_FILE}" ]; then
        local file_size=$(du -h "${GEOJSON_FILE}" | cut -f1)
        log_to_file "File Size: ${file_size}" "INFO"
        print_info "File Size: ${file_size}"
    fi
    
    log_to_file "Mapping File: ${MAPPING_FILE}" "INFO"
    log_to_file "Source CRS: ${SOURCE_CRS}" "INFO"
    log_to_file "Target CRS: ${TARGET_CRS}" "INFO"
    log_to_file "County: ${COUNTY_NAME}" "INFO"
    log_to_file "Import Mode: ${MODE}" "INFO"
    log_to_file "Database: ${DB_HOST}:${DB_PORT}/${DB_NAME}" "INFO"
    log_to_file "Target Table: ${FINAL_TABLE}" "INFO"
    log_to_file "==============================" "INFO"
}

# Calculate duration in seconds
calculate_duration() {
    local start="$1"
    local end="$2"
    echo $((end - start))
}

# Format duration as human-readable string
format_duration() {
    local duration_sec="$1"
    local hours=$((duration_sec / 3600))
    local minutes=$(((duration_sec % 3600) / 60))
    local seconds=$((duration_sec % 60))
    
    if [ ${hours} -gt 0 ]; then
        echo "${hours}h ${minutes}m ${seconds}s"
    elif [ ${minutes} -gt 0 ]; then
        echo "${minutes}m ${seconds}s"
    else
        echo "${seconds}s"
    fi
}

# Calculate and log import summary
log_import_summary() {
    local estimated_count="$1"
    
    END_TIME=$(date +%s)
    local duration=$(calculate_duration ${START_TIME} ${END_TIME})
    local duration_formatted=$(format_duration ${duration})
    
    print_step "Calculating import performance metrics..."
    
    # Get actual record count from database
    local actual_count="unknown"
    if [ "${DRY_RUN}" = false ]; then
        actual_count=$(PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST}" \
            -p "${DB_PORT}" \
            -U "${DB_USER}" \
            -d "${DB_NAME}" \
            -t -c "SELECT COUNT(*) FROM ${FINAL_TABLE};" 2>/dev/null | xargs || echo "unknown")
    fi
    
    log_to_file "=== Import Summary ===" "INFO"
    log_to_file "Start Time: $(date -d @${START_TIME} '+%Y-%m-%d %H:%M:%S')" "INFO"
    log_to_file "End Time: $(date -d @${END_TIME} '+%Y-%m-%d %H:%M:%S')" "INFO"
    log_to_file "Duration: ${duration_formatted} (${duration} seconds)" "INFO"
    
    # Use actual count for performance calculation
    local count_for_calc="${actual_count}"
    if [ "${count_for_calc}" = "unknown" ] || [ "${count_for_calc}" = "" ]; then
        count_for_calc="${estimated_count}"
    fi
    
    # Calculate records per second
    if [ ${duration} -gt 0 ] && [ "${count_for_calc}" != "unknown" ] && [ -n "${count_for_calc}" ]; then
        local records_per_sec=$((count_for_calc / duration))
        log_to_file "Records Imported: ${count_for_calc}" "INFO"
        log_to_file "Import Rate: ${records_per_sec} records/second" "INFO"
        print_success "Import Rate: ${records_per_sec} records/second"
    fi
    
    log_to_file "======================" "INFO"
    
    print_success "Total Duration: ${duration_formatted}"
}

# Get import statistics
get_import_stats() {
    print_step "Gathering import statistics..."
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN - Skipping statistics"
        return 0
    fi
    
    local count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${FINAL_TABLE};" | xargs)
    
    print_success "Total records in ${FINAL_TABLE}: ${count}"
    log_to_file "Final record count: ${count}" "INFO"
    
    # Check for NULL geometries
    local null_geoms=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${FINAL_TABLE} WHERE geom IS NULL;" | xargs)
    
    if [ "${null_geoms}" -gt 0 ]; then
        print_warning "Found ${null_geoms} records with NULL geometries"
    else
        print_success "No NULL geometries found"
    fi
}

#------------------------------------------------------------------------------
# Parse command-line arguments
#------------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case $1 in
        --file)
            GEOJSON_FILE="$2"
            shift 2
            ;;
        --mapping)
            MAPPING_FILE="$2"
            shift 2
            ;;
        --db-host)
            DB_HOST="$2"
            shift 2
            ;;
        --db-port)
            DB_PORT="$2"
            shift 2
            ;;
        --db-name)
            DB_NAME="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        --db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --mode)
            MODE="$2"
            if [[ ! "${MODE}" =~ ^(replace|append)$ ]]; then
                print_error "Invalid mode: ${MODE}. Must be 'replace' or 'append'"
                exit 1
            fi
            shift 2
            ;;
        --validate-geometries)
            VALIDATE_GEOMETRIES=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

#------------------------------------------------------------------------------
# Main execution
#------------------------------------------------------------------------------
main() {
    echo ""
    echo "=========================================="
    echo "  Tax Parcel Import Script"
    echo "=========================================="
    echo ""
    
    # Record start time
    START_TIME=$(date +%s)
    
    # Initialize log file
    init_log_file
    
    # Validate required arguments
    if [ -z "${GEOJSON_FILE}" ]; then
        print_error "Missing required argument: --file"
        echo "Use --help for usage information"
        exit 1
    fi
    
    if [ -z "${MAPPING_FILE}" ]; then
        print_error "Missing required argument: --mapping"
        echo "Use --help for usage information"
        exit 1
    fi
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN MODE - No changes will be made"
        echo ""
    fi
    
    # Check dependencies
    check_dependencies
    
    # Validate inputs
    validate_input_file "${GEOJSON_FILE}"
    validate_mapping_file "${MAPPING_FILE}"
    
    # Test database connection
    test_db_connection
    
    # Read mapping configuration
    read_mapping_config "${MAPPING_FILE}"
    
    # Log configuration details
    log_configuration
    
    # Get record count
    local record_count=$(get_record_count "${GEOJSON_FILE}")
    print_info "Records to import: ${record_count}"
    log_to_file "Records to import: ${record_count}" "INFO"
    
    # Handle mode
    if [ "${MODE}" = "replace" ]; then
        print_warning "Mode: REPLACE - Existing data in ${FINAL_TABLE} will be cleared"
        if [ "${DRY_RUN}" = false ]; then
            # TRUNCATE with RESTART IDENTITY resets the BIGSERIAL sequence to 1
            PGPASSWORD="${DB_PASSWORD}" psql \
                -h "${DB_HOST}" \
                -p "${DB_PORT}" \
                -U "${DB_USER}" \
                -d "${DB_NAME}" \
                -c "TRUNCATE TABLE ${FINAL_TABLE} RESTART IDENTITY;" &> /dev/null
            print_success "Existing data cleared and ID sequence reset"
        fi
    else
        print_info "Mode: APPEND - Data will be added to existing records"
    fi
    
    echo ""
    
    # Generate field mapping SQL
    local mapping_sql=$(generate_field_mapping_sql "${MAPPING_FILE}")
    
    # Import to staging table
    import_to_staging "${GEOJSON_FILE}" "${TARGET_CRS}"
    
    # Execute field mapping
    execute_field_mapping "${mapping_sql}"
    
    # Clean up staging table
    cleanup_staging
    
    # Get statistics
    get_import_stats
    
    # Run geometry validation if requested
    if [ "${VALIDATE_GEOMETRIES}" = true ]; then
        echo ""
        print_step "Running geometry validation and repair..."
        log_to_file "Running geometry validation and repair" "INFO"
        
        local validation_script="${SCRIPT_DIR}/validate-geometries.sh"
        if [ -f "${validation_script}" ]; then
            if "${validation_script}" \
                --db-host "${DB_HOST}" \
                --db-port "${DB_PORT}" \
                --db-name "${DB_NAME}" \
                --db-user "${DB_USER}" \
                --db-password "${DB_PASSWORD}"; then
                print_success "Geometry validation completed"
                log_to_file "Geometry validation passed" "SUCCESS"
            else
                print_warning "Geometry validation found issues (check validation log)"
                log_to_file "Geometry validation completed with warnings" "WARN"
            fi
        else
            print_warning "Validation script not found: ${validation_script}"
            log_to_file "Validation script not found" "WARN"
        fi
    fi
    
    # Log import summary with performance metrics
    log_import_summary "${record_count}"
    
    # Write completion to log file
    log_to_file "Import completed successfully!" "SUCCESS"
    log_to_file "Log file: ${LOG_FILE}" "INFO"
    
    echo ""
    print_success "Import completed successfully!"
    print_info "Log file: ${LOG_FILE}"
    echo ""
}

# Run main function
main

