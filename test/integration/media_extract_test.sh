#!/bin/bash
# Integration test for media extraction with progress tracking
# This test:
# 1. Generates a test tarball with random files
# 2. Tests extraction with progress tracking
# 3. Verifies the extracted files
#
# Usage: ./media_extract_test.sh [--keep-files] [--size SIZE_MB]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FIXTURES_DIR="$PROJECT_ROOT/test/fixtures"
TMP_DIR="/tmp/magebox-media-test-$$"

# Configuration
TEST_SIZE_MB="${TEST_SIZE_MB:-10}"
KEEP_FILES=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-files)
            KEEP_FILES=true
            shift
            ;;
        --size)
            TEST_SIZE_MB="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if [ "$KEEP_FILES" = false ]; then
        log_info "Cleaning up..."
        rm -rf "$TMP_DIR" 2>/dev/null || true
        rm -f "$FIXTURES_DIR/test-media.tar.gz" 2>/dev/null || true
    else
        log_info "Keeping files in: $TMP_DIR"
    fi
}

trap cleanup EXIT

# Generate random file content
generate_random_file() {
    local FILE="$1"
    local SIZE_KB="$2"
    dd if=/dev/urandom of="$FILE" bs=1024 count="$SIZE_KB" 2>/dev/null
}

# Generate test media archive
generate_test_media() {
    log_info "Generating ${TEST_SIZE_MB}MB test media archive..."

    mkdir -p "$TMP_DIR/source"
    mkdir -p "$FIXTURES_DIR"

    # Create directory structure like Magento media
    local DIRS=(
        "catalog/product/a/b"
        "catalog/product/c/d"
        "catalog/category"
        "wysiwyg"
        "customer"
        "downloadable"
        "tmp"
    )

    for dir in "${DIRS[@]}"; do
        mkdir -p "$TMP_DIR/source/$dir"
    done

    # Calculate how many files to create
    local TARGET_KB=$((TEST_SIZE_MB * 1024))
    local GENERATED_KB=0
    local FILE_COUNT=0

    # Generate files of various sizes
    local SIZES=(10 50 100 200 500) # KB

    log_info "Creating test files..."

    while [ $GENERATED_KB -lt $TARGET_KB ]; do
        # Pick random directory
        local DIR="${DIRS[$((RANDOM % ${#DIRS[@]}))]}"

        # Pick random size
        local SIZE="${SIZES[$((RANDOM % ${#SIZES[@]}))]}"

        # Generate file
        local FILE="$TMP_DIR/source/$DIR/file_$FILE_COUNT.dat"
        generate_random_file "$FILE" "$SIZE"

        GENERATED_KB=$((GENERATED_KB + SIZE))
        FILE_COUNT=$((FILE_COUNT + 1))

        # Progress indicator
        local PERCENT=$((GENERATED_KB * 100 / TARGET_KB))
        printf "\rGenerating files: %d%% (%d files, %dKB)" $PERCENT $FILE_COUNT $GENERATED_KB
    done
    echo ""

    # Create the tarball
    log_info "Creating tar.gz archive..."
    cd "$TMP_DIR/source"
    tar -czf "$FIXTURES_DIR/test-media.tar.gz" .
    cd - > /dev/null

    local ARCHIVE_SIZE=$(ls -lh "$FIXTURES_DIR/test-media.tar.gz" | awk '{print $5}')
    log_info "Created archive: $ARCHIVE_SIZE ($FILE_COUNT files)"
}

# Test extraction with system tar
test_tar_extraction() {
    log_info "Testing tar extraction..."

    local EXTRACT_DIR="$TMP_DIR/extract-tar"
    mkdir -p "$EXTRACT_DIR"

    local START_TIME=$(date +%s.%N)

    tar -xzf "$FIXTURES_DIR/test-media.tar.gz" -C "$EXTRACT_DIR"

    local END_TIME=$(date +%s.%N)
    local DURATION=$(echo "$END_TIME - $START_TIME" | bc)

    # Count extracted files
    local FILE_COUNT=$(find "$EXTRACT_DIR" -type f | wc -l)

    if [ "$FILE_COUNT" -lt 1 ]; then
        log_error "Extraction verification failed: no files found"
        return 1
    fi

    log_info "tar extraction: $FILE_COUNT files in ${DURATION}s"
    return 0
}

# Test extraction with piped input (how magebox does it)
test_piped_extraction() {
    log_info "Testing piped extraction..."

    local EXTRACT_DIR="$TMP_DIR/extract-piped"
    mkdir -p "$EXTRACT_DIR"

    local START_TIME=$(date +%s.%N)

    # This is how the progress reader pipes data
    cat "$FIXTURES_DIR/test-media.tar.gz" | tar -xz -C "$EXTRACT_DIR"

    local END_TIME=$(date +%s.%N)
    local DURATION=$(echo "$END_TIME - $START_TIME" | bc)

    # Count extracted files
    local FILE_COUNT=$(find "$EXTRACT_DIR" -type f | wc -l)

    if [ "$FILE_COUNT" -lt 1 ]; then
        log_error "Piped extraction verification failed: no files found"
        return 1
    fi

    log_info "Piped extraction: $FILE_COUNT files in ${DURATION}s"
    return 0
}

# Test extraction with pv (if available) for comparison
test_pv_extraction() {
    if ! command -v pv &> /dev/null; then
        log_warn "pv not installed, skipping pv test"
        return 0
    fi

    log_info "Testing extraction with pv..."

    local EXTRACT_DIR="$TMP_DIR/extract-pv"
    mkdir -p "$EXTRACT_DIR"

    local START_TIME=$(date +%s.%N)

    pv "$FIXTURES_DIR/test-media.tar.gz" | tar -xz -C "$EXTRACT_DIR"

    local END_TIME=$(date +%s.%N)
    local DURATION=$(echo "$END_TIME - $START_TIME" | bc)

    local FILE_COUNT=$(find "$EXTRACT_DIR" -type f | wc -l)

    log_info "pv extraction: $FILE_COUNT files in ${DURATION}s"
    return 0
}

# Verify file integrity
test_file_integrity() {
    log_info "Verifying file integrity..."

    local SOURCE_DIR="$TMP_DIR/source"
    local EXTRACT_DIR="$TMP_DIR/extract-tar"

    if [ ! -d "$SOURCE_DIR" ] || [ ! -d "$EXTRACT_DIR" ]; then
        log_warn "Source or extract directory missing, skipping integrity check"
        return 0
    fi

    # Compare file counts
    local SOURCE_COUNT=$(find "$SOURCE_DIR" -type f | wc -l)
    local EXTRACT_COUNT=$(find "$EXTRACT_DIR" -type f | wc -l)

    if [ "$SOURCE_COUNT" -ne "$EXTRACT_COUNT" ]; then
        log_error "File count mismatch: source=$SOURCE_COUNT, extract=$EXTRACT_COUNT"
        return 1
    fi

    # Compare total size
    local SOURCE_SIZE=$(du -sb "$SOURCE_DIR" | cut -f1)
    local EXTRACT_SIZE=$(du -sb "$EXTRACT_DIR" | cut -f1)

    if [ "$SOURCE_SIZE" -ne "$EXTRACT_SIZE" ]; then
        log_error "Size mismatch: source=$SOURCE_SIZE, extract=$EXTRACT_SIZE"
        return 1
    fi

    log_info "File integrity: OK ($SOURCE_COUNT files, $SOURCE_SIZE bytes)"
    return 0
}

# Run all tests
run_tests() {
    local PASSED=0
    local FAILED=0

    echo ""
    echo "========================================="
    echo "  MageBox Media Extract Integration Tests"
    echo "========================================="
    echo ""

    generate_test_media

    echo ""
    log_info "Running tests..."
    echo ""

    if test_tar_extraction; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    if test_piped_extraction; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    if test_pv_extraction; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    if test_file_integrity; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    echo ""
    echo "========================================="
    echo "  Results: $PASSED passed, $FAILED failed"
    echo "========================================="

    if [ $FAILED -gt 0 ]; then
        exit 1
    fi
}

# Main
run_tests
