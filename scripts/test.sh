#!/bin/bash
# Comprehensive test script for Lumen Nutrition Tracker
# Runs all tests with coverage reporting

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "================================"
echo "Lumen API Test Suite"
echo "================================"
echo ""

# Run tests with coverage
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test ./... -v -coverprofile=coverage.out -covermode=atomic

echo ""
echo "================================"
echo -e "${YELLOW}Coverage Report${NC}"
echo "================================"

# Generate coverage summary
go tool cover -func=coverage.out | tail -1

echo ""
echo "Detailed coverage by package:"
go tool cover -func=coverage.out | grep -v "total:" | column -t

# Generate HTML coverage report
if command -v go &> /dev/null; then
    echo ""
    echo -e "${YELLOW}Generating HTML coverage report...${NC}"
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}✓ HTML coverage report generated: coverage.html${NC}"

    # Try to open in browser
    if command -v xdg-open &> /dev/null; then
        xdg-open coverage.html
    elif command -v open &> /dev/null; then
        open coverage.html
    fi
fi

# Run race detector on critical packages
echo ""
echo "================================"
echo -e "${YELLOW}Running race detector...${NC}"
echo "================================"
go test -race ./internal/services/... ./internal/domain/... -short

echo ""
echo -e "${GREEN}✓ All tests completed!${NC}"
