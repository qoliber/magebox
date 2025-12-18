#!/bin/bash
#
# Run MageBox Team Server Integration Tests
#
# This script starts the Docker environment, runs the tests,
# and cleans up afterwards.
#
# Usage:
#   ./run-tests.sh
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

cd "$SCRIPT_DIR"

echo "=================================="
echo "MageBox Team Server Integration Tests"
echo "=================================="
echo ""

# Check Docker
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running"
    exit 1
fi

# Check docker-compose
if ! docker-compose version > /dev/null 2>&1; then
    echo "ERROR: docker-compose is not available"
    exit 1
fi

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    docker-compose down -v --remove-orphans 2>/dev/null || true
}

# Trap cleanup on exit
trap cleanup EXIT

# Build and start containers
echo "Building and starting containers..."
docker-compose up -d --build

# Wait for server to be ready
echo "Waiting for server to be ready..."
for i in {1..60}; do
    if curl -s http://localhost:17443/health > /dev/null 2>&1; then
        echo "Server is ready!"
        break
    fi
    echo "  Waiting... ($i/60)"
    sleep 2
done

# Check if server is ready
if ! curl -s http://localhost:17443/health > /dev/null 2>&1; then
    echo "ERROR: Server did not become ready in time"
    docker-compose logs server
    exit 1
fi

# Run tests
echo ""
echo "Running integration tests..."
cd "$PROJECT_ROOT"
go test -v -tags=integration ./test/integration/teamserver/...
TEST_EXIT_CODE=$?

# Show logs if tests failed
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "Tests failed! Server logs:"
    cd "$SCRIPT_DIR"
    docker-compose logs server
fi

exit $TEST_EXIT_CODE
