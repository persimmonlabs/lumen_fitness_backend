#!/bin/bash
# Smoke tests for Lumen Nutrition Tracker API
# Tests critical endpoints to ensure basic functionality

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS_COUNT=0
FAIL_COUNT=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================"
echo "Lumen API Smoke Tests"
echo "================================"
echo "Base URL: $BASE_URL"
echo ""

# Helper function to test endpoint
test_endpoint() {
    local name=$1
    local method=$2
    local path=$3
    local expected_code=$4
    local data=$5

    echo -n "Testing $name... "

    if [ -z "$data" ]; then
        response=$(curl -s -w "%{http_code}" -X "$method" "$BASE_URL$path" 2>&1)
    else
        response=$(curl -s -w "%{http_code}" -X "$method" "$BASE_URL$path" -H "Content-Type: application/json" -d "$data" 2>&1)
    fi

    http_code="${response: -3}"

    if [ "$http_code" == "$expected_code" ]; then
        echo -e "${GREEN}PASS${NC} (HTTP $http_code)"
        ((PASS_COUNT++))
        return 0
    else
        echo -e "${RED}FAIL${NC} (Expected $expected_code, got $http_code)"
        ((FAIL_COUNT++))
        return 1
    fi
}

# Test 1: Health check
test_endpoint "Health Check" "GET" "/health" "200"

# Test 2: Ready check
test_endpoint "Ready Check" "GET" "/ready" "200"

# Test 3: API version info
test_endpoint "API Version" "GET" "/api/v1" "200"

# Test 4: 404 for unknown route
test_endpoint "404 Response" "GET" "/nonexistent" "404"

# Test 5: CORS preflight (OPTIONS request)
echo -n "Testing CORS Preflight... "
cors_response=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS "$BASE_URL/api/v1" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: GET" 2>&1)

if [ "$cors_response" == "200" ] || [ "$cors_response" == "204" ]; then
    echo -e "${GREEN}PASS${NC} (HTTP $cors_response)"
    ((PASS_COUNT++))
else
    echo -e "${RED}FAIL${NC} (Expected 200/204, got $cors_response)"
    ((FAIL_COUNT++))
fi

# Test 6: Nutrition routes are registered (should return 401/403 without auth)
test_endpoint "Meals Endpoint (Unauth)" "GET" "/api/v1/meals" "401"

# Test 7: Analytics endpoint
test_endpoint "Analytics Endpoint (Unauth)" "GET" "/api/v1/analytics/daily" "401"

# Test 8: Weight endpoint
test_endpoint "Weight Endpoint (Unauth)" "GET" "/api/v1/weight" "401"

# Test 9: Goals endpoint
test_endpoint "Goals Endpoint (Unauth)" "GET" "/api/v1/goals" "401"

# Test 10: Templates endpoint
test_endpoint "Templates Endpoint (Unauth)" "GET" "/api/v1/templates" "401"

# Summary
echo ""
echo "================================"
echo "Test Results"
echo "================================"
echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
echo -e "${RED}Failed: $FAIL_COUNT${NC}"
echo "Total: $((PASS_COUNT + FAIL_COUNT))"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✓ All smoke tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
