#!/bin/bash
# Linting script for Lumen Nutrition Tracker
# Runs code quality checks and linters

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "================================"
echo "Lumen API Code Quality Checks"
echo "================================"
echo ""

# 1. Format check
echo -e "${YELLOW}Checking code formatting...${NC}"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    echo -e "${RED}✗ The following files need formatting:${NC}"
    echo "$unformatted"
    echo ""
    echo "Run 'gofmt -w .' to fix"
    exit 1
else
    echo -e "${GREEN}✓ All files are properly formatted${NC}"
fi

echo ""

# 2. Go vet
echo -e "${YELLOW}Running go vet...${NC}"
if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet found issues${NC}"
    exit 1
fi

echo ""

# 3. golangci-lint (if installed)
if command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}Running golangci-lint...${NC}"
    if golangci-lint run; then
        echo -e "${GREEN}✓ golangci-lint passed${NC}"
    else
        echo -e "${RED}✗ golangci-lint found issues${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ golangci-lint not installed, skipping${NC}"
    echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

echo ""

# 4. Check for common issues
echo -e "${YELLOW}Checking for common issues...${NC}"

# Check for TODO comments
todos=$(grep -r "TODO" --include="*.go" . | wc -l)
if [ "$todos" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Found $todos TODO comments${NC}"
fi

# Check for fmt.Println (should use logger)
printlns=$(grep -r "fmt.Println" --include="*.go" . | wc -l)
if [ "$printlns" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Found $printlns fmt.Println statements (use logger instead)${NC}"
fi

# Check for hardcoded secrets patterns
secrets=$(grep -rE "(password|secret|key|token).*=.*\"" --include="*.go" . | grep -v "// " | wc -l)
if [ "$secrets" -gt 0 ]; then
    echo -e "${RED}⚠ Found $secrets potential hardcoded secrets${NC}"
fi

echo ""
echo -e "${GREEN}✓ Code quality checks completed!${NC}"
