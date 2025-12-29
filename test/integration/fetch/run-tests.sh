#!/bin/bash
# Integration tests for fetch command
# This script sets up a Docker SFTP server and runs the integration tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Fetch Command Integration Tests ==="
echo ""

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "Docker is required but not installed. Skipping integration tests."
    exit 0
fi

if ! docker info &> /dev/null; then
    echo "Docker is not running. Skipping integration tests."
    exit 0
fi

# Setup test data
echo "Setting up test data..."
mkdir -p testdata/testproject

# Run tests
echo "Running integration tests..."
go test -v -tags=integration ./...

echo ""
echo "=== Integration tests completed ==="
