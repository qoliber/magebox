#!/bin/bash
# Generate test SQL files for integration testing
# Usage: ./generate-test-sql.sh [size_mb] [output_file]
#
# Examples:
#   ./generate-test-sql.sh 10 test-10mb.sql      # 10 MB SQL file
#   ./generate-test-sql.sh 100 test-100mb.sql    # 100 MB SQL file

set -e

SIZE_MB="${1:-10}"
OUTPUT_FILE="${2:-test-data.sql}"

echo "Generating ${SIZE_MB}MB SQL test file: ${OUTPUT_FILE}"

# Calculate approximate rows needed (each row is ~200 bytes)
ROWS_PER_MB=5000
TOTAL_ROWS=$((SIZE_MB * ROWS_PER_MB))
BATCH_SIZE=1000

cat > "$OUTPUT_FILE" << 'EOF'
-- MageBox Test SQL File
-- Generated for integration testing

DROP DATABASE IF EXISTS magebox_test;
CREATE DATABASE magebox_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE magebox_test;

-- Test table with various column types
CREATE TABLE test_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    uuid CHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2),
    quantity INT,
    is_active TINYINT(1) DEFAULT 1,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_uuid (uuid),
    INDEX idx_email (email),
    INDEX idx_created (created_at)
) ENGINE=InnoDB;

EOF

echo "Generating $TOTAL_ROWS rows in batches of $BATCH_SIZE..."

# Generate data in batches
ROWS_GENERATED=0
BATCH_NUM=0

while [ $ROWS_GENERATED -lt $TOTAL_ROWS ]; do
    BATCH_NUM=$((BATCH_NUM + 1))
    BATCH_ROWS=$BATCH_SIZE

    # Don't exceed total
    if [ $((ROWS_GENERATED + BATCH_ROWS)) -gt $TOTAL_ROWS ]; then
        BATCH_ROWS=$((TOTAL_ROWS - ROWS_GENERATED))
    fi

    # Start INSERT statement
    echo "INSERT INTO test_data (uuid, name, email, description, price, quantity, is_active, metadata) VALUES" >> "$OUTPUT_FILE"

    # Generate rows
    for i in $(seq 1 $BATCH_ROWS); do
        ROW_ID=$((ROWS_GENERATED + i))
        UUID=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || uuidgen 2>/dev/null || echo "$(printf '%08x-%04x-%04x-%04x-%012x' $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM)")
        NAME="Test User $ROW_ID"
        EMAIL="user${ROW_ID}@example.com"
        DESC="This is a test description for row $ROW_ID with some additional text to make it longer and more realistic for testing purposes."
        PRICE=$(echo "scale=2; $RANDOM / 100" | bc 2>/dev/null || echo "99.99")
        QTY=$((RANDOM % 1000))
        ACTIVE=$((RANDOM % 2))

        # JSON metadata
        META="{\"row\": $ROW_ID, \"batch\": $BATCH_NUM, \"timestamp\": $(date +%s)}"

        # Add comma except for last row
        if [ $i -eq $BATCH_ROWS ]; then
            COMMA=";"
        else
            COMMA=","
        fi

        echo "('$UUID', '$NAME', '$EMAIL', '$DESC', $PRICE, $QTY, $ACTIVE, '$META')$COMMA" >> "$OUTPUT_FILE"
    done

    ROWS_GENERATED=$((ROWS_GENERATED + BATCH_ROWS))

    # Progress indicator
    PERCENT=$((ROWS_GENERATED * 100 / TOTAL_ROWS))
    printf "\rProgress: %d%% (%d/%d rows)" $PERCENT $ROWS_GENERATED $TOTAL_ROWS
done

echo ""

# Add some verification queries as comments
cat >> "$OUTPUT_FILE" << 'EOF'

-- Verification queries (commented out)
-- SELECT COUNT(*) as total_rows FROM test_data;
-- SELECT MIN(id), MAX(id), AVG(price) FROM test_data;
-- SHOW TABLE STATUS LIKE 'test_data';

EOF

# Show file size
FILE_SIZE=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')
echo "Generated: $OUTPUT_FILE ($FILE_SIZE)"
echo "Total rows: $ROWS_GENERATED"

# Optionally create gzipped version
if command -v gzip &> /dev/null; then
    gzip -k -f "$OUTPUT_FILE"
    GZ_SIZE=$(ls -lh "${OUTPUT_FILE}.gz" | awk '{print $5}')
    echo "Compressed: ${OUTPUT_FILE}.gz ($GZ_SIZE)"
fi
