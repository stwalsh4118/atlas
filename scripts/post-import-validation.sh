#!/usr/bin/env bash

#==============================================================================
# Post-Import Validation Script
#==============================================================================
# Description: Validates imported tax parcel data integrity and spatial functionality
# Usage: ./post-import-validation.sh [options]
#
# This script:
# 1. Counts total records imported
# 2. Checks for NULL geometries and required fields
# 3. Verifies all geometries are in EPSG:4326
# 4. Verifies all geometries are MultiPolygon type
# 5. Tests spatial index with EXPLAIN
# 6. Runs sample spatial queries (point-in-polygon, bounding box)
# 7. Calculates data statistics (acreage, owners, extent)
# 8. Runs VACUUM ANALYZE
# 9. Generates validation report
#
# Requirements:
# - PostgreSQL client (psql)
# - PostGIS extension enabled in database
# - Database with tax_parcels table
#==============================================================================

set -euo pipefail

#------------------------------------------------------------------------------
# Constants
#------------------------------------------------------------------------------
readonly SCRIPT_NAME="$(basename "$0")"
readonly SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
readonly TABLE_NAME="tax_parcels"
readonly LOG_DIR="${SCRIPT_DIR}/../logs"
readonly EXPECTED_SRID=4326
readonly EXPECTED_GEOM_TYPE="MULTIPOLYGON"

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

# Logging variables
LOG_FILE=""
START_TIME=""
END_TIME=""

# Validation tracking
CRITICAL_ISSUES=0
WARNINGS=0

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
    LOG_FILE="${LOG_DIR}/post-import-validation-${timestamp}.log"
    
    # Create log file and write header
    {
        echo "========================================"
        echo "Post-Import Validation Report"
        echo "========================================"
        echo "Started: $(log_timestamp)"
        echo "Script: ${SCRIPT_NAME}"
        echo "Table: ${TABLE_NAME}"
        echo ""
    } > "${LOG_FILE}"
    
    print_success "Log file created: ${LOG_FILE}"
}

# Log message to file
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
    ((CRITICAL_ISSUES++))
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
    ((WARNINGS++))
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

Validate imported tax parcel data integrity and spatial functionality.

Database Options:
  --db-host <host>        Database host (default: ${DB_HOST})
  --db-port <port>        Database port (default: ${DB_PORT})
  --db-name <database>    Database name (default: ${DB_NAME})
  --db-user <user>        Database user (default: ${DB_USER})
  --db-password <pass>    Database password (default: \$DB_PASSWORD env var)

Other Options:
  -h, --help              Show this help message

Examples:
  # Basic validation with default database connection
  ${SCRIPT_NAME}

  # Validate with custom database
  ${SCRIPT_NAME} --db-host localhost --db-name my_atlas

Exit Codes:
  0 - All validation checks passed
  1 - Critical issues found (data integrity problems)

EOF
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
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
            -h|--help)
                usage
                ;;
            *)
                echo "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# Test database connection
test_db_connection() {
    print_step "Testing database connection..."
    
    if PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "SELECT version();" &> /dev/null; then
        print_success "Database connection successful"
        return 0
    else
        print_error "Failed to connect to database"
        print_error "Host: ${DB_HOST}, Port: ${DB_PORT}, Database: ${DB_NAME}, User: ${DB_USER}"
        exit 1
    fi
}

# Check if table exists
check_table_exists() {
    print_step "Checking if ${TABLE_NAME} table exists..."
    
    local table_exists=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '${TABLE_NAME}');" | tr -d ' ')
    
    if [ "${table_exists}" = "t" ]; then
        print_success "Table ${TABLE_NAME} exists"
        return 0
    else
        print_error "Table ${TABLE_NAME} does not exist"
        exit 1
    fi
}

# Count total records
count_records() {
    print_step "Counting total records..."
    
    local count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME};" | tr -d ' ')
    
    if [ "${count}" -eq 0 ]; then
        print_error "No records found in ${TABLE_NAME}"
        return 1
    else
        print_success "Total records: ${count}"
        return 0
    fi
}

# Check for NULL geometries
check_null_geometries() {
    print_step "Checking for NULL geometries..."
    
    local null_count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME} WHERE geom IS NULL;" | tr -d ' ')
    
    if [ "${null_count}" -gt 0 ]; then
        print_error "Found ${null_count} records with NULL geometries"
        return 1
    else
        print_success "No NULL geometries found"
        return 0
    fi
}

# Check for NULL pin (required identifier)
check_null_pins() {
    print_step "Checking for NULL pin values (required identifier)..."
    
    local null_count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME} WHERE pin IS NULL;" | tr -d ' ')
    
    if [ "${null_count}" -gt 0 ]; then
        print_error "Found ${null_count} records with NULL pin"
        return 1
    else
        print_success "No NULL pin values found"
        return 0
    fi
}

# Verify CRS (SRID)
verify_crs() {
    print_step "Verifying coordinate reference system (SRID)..."
    
    local srids=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT DISTINCT ST_SRID(geom) FROM ${TABLE_NAME} WHERE geom IS NOT NULL;" | tr -d ' ')
    
    local has_error=false
    while IFS= read -r srid; do
        if [ -n "${srid}" ]; then
            if [ "${srid}" -eq "${EXPECTED_SRID}" ]; then
                print_success "All geometries are in EPSG:${EXPECTED_SRID}"
            else
                print_error "Found geometries with SRID ${srid} (expected ${EXPECTED_SRID})"
                has_error=true
            fi
        fi
    done <<< "${srids}"
    
    if [ "${has_error}" = true ]; then
        return 1
    else
        return 0
    fi
}

# Verify geometry types
verify_geometry_types() {
    print_step "Verifying geometry types..."
    
    local geom_types=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT DISTINCT GeometryType(geom) FROM ${TABLE_NAME} WHERE geom IS NOT NULL;")
    
    local has_error=false
    while IFS= read -r geom_type; do
        # Trim whitespace
        geom_type=$(echo "${geom_type}" | tr -d ' ')
        if [ -n "${geom_type}" ]; then
            if [ "${geom_type}" = "${EXPECTED_GEOM_TYPE}" ]; then
                print_success "All geometries are ${EXPECTED_GEOM_TYPE} type"
            else
                print_error "Found ${geom_type} geometries (expected ${EXPECTED_GEOM_TYPE})"
                has_error=true
            fi
        fi
    done <<< "${geom_types}"
    
    if [ "${has_error}" = true ]; then
        return 1
    else
        return 0
    fi
}

# Test spatial index
test_spatial_index() {
    print_step "Testing spatial index with EXPLAIN..."
    
    # Run EXPLAIN to see if index is used
    local explain_output=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "EXPLAIN SELECT * FROM ${TABLE_NAME} WHERE ST_Intersects(geom, ST_MakePoint(-95.5, 30.2));" 2>&1)
    
    # Log the full EXPLAIN output
    log_to_file "EXPLAIN output:" "INFO"
    log_to_file "${explain_output}" "INFO"
    
    # Check if the output contains "Index Scan" or "Bitmap Index Scan"
    if echo "${explain_output}" | grep -qi "Index Scan"; then
        print_success "Spatial index is being used (Index Scan detected)"
        return 0
    elif echo "${explain_output}" | grep -qi "Bitmap"; then
        print_success "Spatial index is being used (Bitmap Index Scan detected)"
        return 0
    else
        print_warning "Spatial index may not be in use (check EXPLAIN output in log)"
        return 1
    fi
}

# Run sample point-in-polygon query
test_point_in_polygon() {
    print_step "Running sample point-in-polygon query..."
    
    # Use a point in Montgomery County, TX
    local result=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME} WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint(-95.5, 30.2), ${EXPECTED_SRID}));" | tr -d ' ')
    
    if [ -n "${result}" ]; then
        print_success "Point-in-polygon query returned ${result} result(s)"
        return 0
    else
        print_warning "Point-in-polygon query returned no results"
        return 1
    fi
}

# Run sample bounding box query
test_bounding_box_query() {
    print_step "Running sample bounding box query..."
    
    # Create a small bounding box in Montgomery County, TX
    local result=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME} WHERE ST_Intersects(geom, ST_MakeEnvelope(-95.6, 30.1, -95.4, 30.3, ${EXPECTED_SRID}));" | tr -d ' ')
    
    if [ -n "${result}" ] && [ "${result}" -gt 0 ]; then
        print_success "Bounding box query returned ${result} result(s)"
        return 0
    else
        print_warning "Bounding box query returned no results"
        return 1
    fi
}

# Calculate statistics
calculate_statistics() {
    print_step "Calculating data statistics..."
    
    echo ""
    print_info "=== Data Statistics ==="
    
    # Calculate area from geometry (in square meters, then convert to acres)
    # 1 acre = 4046.86 square meters
    local area_stats=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT MIN(ST_Area(geom::geography) / 4046.86), MAX(ST_Area(geom::geography) / 4046.86), AVG(ST_Area(geom::geography) / 4046.86) FROM ${TABLE_NAME};")
    
    if [ -n "${area_stats}" ]; then
        local min_area=$(echo "${area_stats}" | cut -d'|' -f1 | tr -d ' ')
        local max_area=$(echo "${area_stats}" | cut -d'|' -f2 | tr -d ' ')
        local avg_area=$(echo "${area_stats}" | cut -d'|' -f3 | tr -d ' ')
        
        print_info "  Calculated Area (acres) - Min: ${min_area}, Max: ${max_area}, Avg: ${avg_area}"
        log_to_file "Area statistics (calculated from geometry) - Min: ${min_area}, Max: ${max_area}, Avg: ${avg_area}" "INFO"
    fi
    
    # Number of unique owners
    local unique_owners=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(DISTINCT owner_name) FROM ${TABLE_NAME} WHERE owner_name IS NOT NULL;" | tr -d ' ')
    
    if [ -n "${unique_owners}" ]; then
        print_info "  Unique owners: ${unique_owners}"
        log_to_file "Unique owners: ${unique_owners}" "INFO"
    fi
    
    # Geographic extent (bounding box)
    local extent=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT ST_AsText(ST_Extent(geom)) FROM ${TABLE_NAME};")
    
    if [ -n "${extent}" ]; then
        print_info "  Geographic extent: ${extent}"
        log_to_file "Geographic extent: ${extent}" "INFO"
    fi
    
    echo ""
}

# Run VACUUM ANALYZE
run_vacuum_analyze() {
    print_step "Running VACUUM ANALYZE on ${TABLE_NAME}..."
    
    if PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "VACUUM ANALYZE ${TABLE_NAME};" &> /dev/null; then
        print_success "VACUUM ANALYZE completed successfully"
        return 0
    else
        print_warning "VACUUM ANALYZE failed"
        return 1
    fi
}

# Calculate duration
calculate_duration() {
    local start_seconds=$(date -d "${START_TIME}" +%s)
    local end_seconds=$(date -d "${END_TIME}" +%s)
    local duration=$((end_seconds - start_seconds))
    echo "${duration}"
}

# Format duration as human-readable string
format_duration() {
    local total_seconds=$1
    local minutes=$((total_seconds / 60))
    local seconds=$((total_seconds % 60))
    
    if [ ${minutes} -gt 0 ]; then
        echo "${minutes}m ${seconds}s"
    else
        echo "${seconds}s"
    fi
}

# Generate validation summary
generate_summary() {
    END_TIME=$(log_timestamp)
    local duration=$(calculate_duration)
    local duration_formatted=$(format_duration ${duration})
    
    echo ""
    print_step "Validation Summary"
    echo ""
    
    log_to_file "=== Validation Summary ===" "INFO"
    log_to_file "Started: ${START_TIME}" "INFO"
    log_to_file "Completed: ${END_TIME}" "INFO"
    log_to_file "Duration: ${duration_formatted} (${duration} seconds)" "INFO"
    log_to_file "Critical Issues: ${CRITICAL_ISSUES}" "INFO"
    log_to_file "Warnings: ${WARNINGS}" "INFO"
    
    print_info "Started: ${START_TIME}"
    print_info "Completed: ${END_TIME}"
    print_info "Duration: ${duration_formatted} (${duration} seconds)"
    echo ""
    
    if [ ${CRITICAL_ISSUES} -eq 0 ]; then
        print_success "All validation checks passed!"
        if [ ${WARNINGS} -gt 0 ]; then
            print_warning "Found ${WARNINGS} warning(s) - check log for details"
        fi
        log_to_file "Validation completed successfully with no critical issues" "SUCCESS"
        return 0
    else
        print_error "Validation failed with ${CRITICAL_ISSUES} critical issue(s)"
        if [ ${WARNINGS} -gt 0 ]; then
            print_warning "Also found ${WARNINGS} warning(s)"
        fi
        log_to_file "Validation completed with ${CRITICAL_ISSUES} critical issues" "ERROR"
        return 1
    fi
}

#------------------------------------------------------------------------------
# Main
#------------------------------------------------------------------------------
main() {
    # Parse command line arguments
    parse_args "$@"
    
    # Initialize logging
    START_TIME=$(log_timestamp)
    init_log_file
    
    echo ""
    print_step "Post-Import Validation for ${TABLE_NAME}"
    print_info "Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}"
    echo ""
    
    # Log configuration
    log_to_file "Configuration:" "INFO"
    log_to_file "  Database Host: ${DB_HOST}" "INFO"
    log_to_file "  Database Port: ${DB_PORT}" "INFO"
    log_to_file "  Database Name: ${DB_NAME}" "INFO"
    log_to_file "  Database User: ${DB_USER}" "INFO"
    log_to_file "  Table Name: ${TABLE_NAME}" "INFO"
    log_to_file "  Expected SRID: ${EXPECTED_SRID}" "INFO"
    log_to_file "  Expected Geometry Type: ${EXPECTED_GEOM_TYPE}" "INFO"
    
    # Run validation checks
    test_db_connection
    check_table_exists
    
    echo ""
    print_step "Running Data Integrity Checks"
    echo ""
    
    count_records
    check_null_geometries
    check_null_pins
    verify_crs
    verify_geometry_types
    
    echo ""
    print_step "Running Spatial Index and Query Tests"
    echo ""
    
    test_spatial_index
    test_point_in_polygon
    test_bounding_box_query
    
    echo ""
    calculate_statistics
    
    echo ""
    run_vacuum_analyze
    
    echo ""
    generate_summary
    
    echo ""
    print_info "Full validation report: ${LOG_FILE}"
    echo ""
    
    # Exit with appropriate code
    if [ ${CRITICAL_ISSUES} -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function
main "$@"

