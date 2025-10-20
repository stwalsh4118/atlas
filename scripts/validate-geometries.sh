#!/usr/bin/env bash

#==============================================================================
# Geometry Validation and Repair Script
#==============================================================================
# Description: Validates geometries in tax_parcels table using PostGIS functions
# Usage: ./validate-geometries.sh [options]
#
# This script:
# 1. Checks all geometries with ST_IsValid
# 2. Identifies invalid geometries with ST_IsValidReason
# 3. Attempts to repair invalid geometries with ST_MakeValid
# 4. Logs all results to timestamped log file
# 5. Generates summary statistics
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
DRY_RUN=false
AUTO_REPAIR=true

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
    LOG_FILE="${LOG_DIR}/geometry-validation-${timestamp}.log"
    
    # Create log file and write header
    {
        echo "========================================"
        echo "Geometry Validation and Repair Log"
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

Validate and repair geometries in the tax_parcels table using PostGIS.

Database Options:
  --db-host <host>        Database host (default: ${DB_HOST})
  --db-port <port>        Database port (default: ${DB_PORT})
  --db-name <database>    Database name (default: ${DB_NAME})
  --db-user <user>        Database user (default: ${DB_USER})
  --db-password <pass>    Database password (default: \$DB_PASSWORD env var)

Validation Options:
  --no-repair             Skip automatic repair of invalid geometries
  --dry-run               Preview operations without making changes

Other Options:
  -h, --help              Show this help message

Environment Variables:
  DB_HOST                 Database host
  DB_PORT                 Database port
  DB_NAME                 Database name
  DB_USER                 Database user
  DB_PASSWORD             Database password

Examples:
  # Validate and repair geometries
  ${SCRIPT_NAME}

  # Check only (no repairs)
  ${SCRIPT_NAME} --no-repair

  # Dry-run mode
  ${SCRIPT_NAME} --dry-run

  # Custom database
  ${SCRIPT_NAME} --db-host localhost --db-name mydb

EOF
    exit 0
}

# Check if required tools are installed
check_dependencies() {
    if ! command -v psql &> /dev/null; then
        print_error "psql (PostgreSQL client) not found"
        echo ""
        echo "Install on Ubuntu/Debian:"
        echo "  sudo apt-get install postgresql-client"
        echo ""
        echo "Install on macOS:"
        echo "  brew install postgresql"
        exit 1
    fi
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

# Check if PostGIS is available
check_postgis() {
    print_step "Checking PostGIS availability..."
    
    local postgis_version
    postgis_version=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT PostGIS_Version();" 2>/dev/null | xargs)
    
    if [ -z "${postgis_version}" ]; then
        print_error "PostGIS extension not found or not enabled"
        echo "  Please enable PostGIS: CREATE EXTENSION postgis;"
        exit 1
    fi
    
    print_success "PostGIS ${postgis_version} available"
    log_to_file "PostGIS version: ${postgis_version}" "INFO"
}

# Get total geometry count
get_total_geometries() {
    local count
    count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME};" 2>/dev/null | xargs)
    
    echo "${count}"
}

# Find invalid geometries
find_invalid_geometries() {
    print_step "Checking geometries with ST_IsValid..."
    
    local sql="
SELECT 
    id,
    object_id,
    pin,
    ST_IsValidReason(geom) as reason
FROM ${TABLE_NAME}
WHERE NOT ST_IsValid(geom)
ORDER BY id;
"
    
    local invalid_geoms
    invalid_geoms=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "${sql}" 2>&1)
    
    echo "${invalid_geoms}"
}

# Get count of invalid geometries
count_invalid_geometries() {
    local count
    count=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -t -c "SELECT COUNT(*) FROM ${TABLE_NAME} WHERE NOT ST_IsValid(geom);" 2>/dev/null | xargs)
    
    echo "${count}"
}

# Repair invalid geometries
repair_invalid_geometries() {
    print_step "Attempting to repair invalid geometries with ST_MakeValid..."
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN - Would execute repair operations"
        return 0
    fi
    
    local sql="
UPDATE ${TABLE_NAME}
SET geom = ST_MakeValid(geom)
WHERE NOT ST_IsValid(geom);
"
    
    local result
    if result=$(PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "${sql}" 2>&1); then
        
        local updated_count=$(echo "${result}" | grep "UPDATE" | awk '{print $2}')
        print_success "Attempted repair on ${updated_count} geometries"
        log_to_file "Repaired ${updated_count} geometries" "INFO"
        echo "${updated_count}"
    else
        print_error "Failed to repair geometries"
        log_to_file "Repair failed: ${result}" "ERROR"
        echo "0"
    fi
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

# Generate validation summary
generate_summary() {
    local total="$1"
    local invalid_before="$2"
    local invalid_after="$3"
    local repaired="$4"
    
    END_TIME=$(date +%s)
    local duration=$(calculate_duration ${START_TIME} ${END_TIME})
    local duration_formatted=$(format_duration ${duration})
    
    print_step "Validation Summary"
    
    log_to_file "=== Validation Summary ===" "INFO"
    log_to_file "Total Geometries: ${total}" "INFO"
    log_to_file "Invalid Before Repair: ${invalid_before}" "INFO"
    
    if [ "${AUTO_REPAIR}" = true ] && [ "${DRY_RUN}" = false ]; then
        log_to_file "Geometries Repaired: ${repaired}" "INFO"
        log_to_file "Invalid After Repair: ${invalid_after}" "INFO"
        
        if [ "${invalid_after}" -gt 0 ]; then
            print_warning "Found ${invalid_after} geometries that could not be repaired"
            log_to_file "WARNING: ${invalid_after} geometries remain invalid" "WARN"
        else
            print_success "All invalid geometries successfully repaired"
        fi
    else
        log_to_file "Auto-repair: disabled" "INFO"
    fi
    
    log_to_file "Duration: ${duration_formatted} (${duration} seconds)" "INFO"
    log_to_file "=========================" "INFO"
    
    echo ""
    echo "Summary:"
    echo "  Total geometries checked: ${total}"
    echo "  Invalid geometries found: ${invalid_before}"
    
    if [ "${AUTO_REPAIR}" = true ] && [ "${DRY_RUN}" = false ]; then
        echo "  Successfully repaired: ${repaired}"
        echo "  Still invalid: ${invalid_after}"
    fi
    
    echo "  Duration: ${duration_formatted}"
    echo ""
}

#------------------------------------------------------------------------------
# Parse command-line arguments
#------------------------------------------------------------------------------
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
        --no-repair)
            AUTO_REPAIR=false
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
    echo "  Geometry Validation and Repair"
    echo "=========================================="
    echo ""
    
    # Record start time
    START_TIME=$(date +%s)
    
    # Initialize log file
    init_log_file
    
    if [ "${DRY_RUN}" = true ]; then
        print_warning "DRY RUN MODE - No changes will be made"
        echo ""
    fi
    
    # Check dependencies
    check_dependencies
    
    # Test database connection
    test_db_connection
    
    # Check PostGIS
    check_postgis
    
    # Get total geometry count
    print_step "Counting total geometries..."
    local total_geoms=$(get_total_geometries)
    print_info "Total geometries in ${TABLE_NAME}: ${total_geoms}"
    log_to_file "Total geometries: ${total_geoms}" "INFO"
    
    if [ "${total_geoms}" -eq 0 ]; then
        print_warning "No geometries found in ${TABLE_NAME}"
        exit 0
    fi
    
    # Find invalid geometries
    local invalid_before=$(count_invalid_geometries)
    
    if [ "${invalid_before}" -eq 0 ]; then
        print_success "All geometries are valid! No repairs needed."
        log_to_file "All ${total_geoms} geometries are valid" "SUCCESS"
        
        # Generate summary
        generate_summary "${total_geoms}" "0" "0" "0"
        
        print_success "Validation completed successfully!"
        print_info "Log file: ${LOG_FILE}"
        echo ""
        exit 0
    fi
    
    print_warning "Found ${invalid_before} invalid geometries"
    log_to_file "Found ${invalid_before} invalid geometries" "WARN"
    
    # Log details of invalid geometries
    print_step "Logging invalid geometry details..."
    local invalid_details=$(find_invalid_geometries)
    
    if [ -n "${invalid_details}" ]; then
        log_to_file "=== Invalid Geometry Details ===" "INFO"
        while IFS= read -r line; do
            if [ -n "${line}" ]; then
                log_to_file "${line}" "INFO"
            fi
        done <<< "${invalid_details}"
        log_to_file "================================" "INFO"
        print_success "Logged details of ${invalid_before} invalid geometries"
    fi
    
    # Repair if enabled
    local repaired=0
    local invalid_after="${invalid_before}"
    
    if [ "${AUTO_REPAIR}" = true ]; then
        repaired=$(repair_invalid_geometries)
        invalid_after=$(count_invalid_geometries)
    else
        print_info "Auto-repair disabled - skipping ST_MakeValid"
        log_to_file "Auto-repair disabled" "INFO"
    fi
    
    # Generate summary
    generate_summary "${total_geoms}" "${invalid_before}" "${invalid_after}" "${repaired}"
    
    # Write completion to log
    log_to_file "Validation completed" "SUCCESS"
    log_to_file "Log file: ${LOG_FILE}" "INFO"
    
    print_success "Validation completed successfully!"
    print_info "Log file: ${LOG_FILE}"
    echo ""
    
    # Exit with error if there are still invalid geometries
    if [ "${invalid_after}" -gt 0 ]; then
        exit 1
    fi
    
    exit 0
}

# Run main function
main

