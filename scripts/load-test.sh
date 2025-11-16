#!/bin/bash
# Load testing for Lumen Nutrition Tracker API
# Uses Apache Bench (ab) to test server performance under load

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
CONCURRENCY="${CONCURRENCY:-10}"
REQUESTS="${REQUESTS:-1000}"

# Colors for output
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "================================"
echo "Lumen API Load Tests"
echo "================================"
echo "Base URL: $BASE_URL"
echo "Concurrency: $CONCURRENCY"
echo "Total Requests: $REQUESTS"
echo ""

# Check if ab is installed
if ! command -v ab &> /dev/null; then
    echo "Error: Apache Bench (ab) is not installed"
    echo "Install it with:"
    echo "  Ubuntu/Debian: sudo apt-get install apache2-utils"
    echo "  macOS: brew install ab"
    echo "  Windows: Download Apache HTTP Server"
    exit 1
fi

# Test 1: Health endpoint (should be very fast)
echo -e "${YELLOW}Testing /health endpoint...${NC}"
ab -n $REQUESTS -c $CONCURRENCY -q "$BASE_URL/health"

echo ""
echo "================================"

# Test 2: API version endpoint
echo -e "${YELLOW}Testing /api/v1 endpoint...${NC}"
ab -n $REQUESTS -c $CONCURRENCY -q "$BASE_URL/api/v1"

echo ""
echo "================================"

# Alternative: Using wrk if available (faster and more detailed)
if command -v wrk &> /dev/null; then
    echo -e "${YELLOW}Using wrk for additional testing...${NC}"
    echo "Health endpoint (10s duration, 10 connections):"
    wrk -t10 -c10 -d10s "$BASE_URL/health"

    echo ""
    echo "API version endpoint (10s duration, 10 connections):"
    wrk -t10 -c10 -d10s "$BASE_URL/api/v1"
else
    echo "Tip: Install 'wrk' for more detailed load testing:"
    echo "  Ubuntu/Debian: sudo apt-get install wrk"
    echo "  macOS: brew install wrk"
fi

echo ""
echo -e "${GREEN}✓ Load tests completed!${NC}"
echo "Review the results above for performance metrics."
echo ""
echo "Key metrics to watch:"
echo "  - Requests per second (should be >500 for simple endpoints)"
echo "  - Time per request (should be <100ms p95)"
echo "  - Failed requests (should be 0)"
