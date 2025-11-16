#!/bin/bash
# Production mode script for Lumen Nutrition Tracker
# Runs the compiled binary with production settings

set -e

# Set production environment
export APP_MODE=production
export LOG_LEVEL=info
export LOG_FORMAT=json

# Load environment variables from .env file
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
else
    echo "Error: .env file not found"
    echo "Production mode requires a .env file with proper configuration"
    exit 1
fi

# Validate required environment variables
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL is required in production mode"
    exit 1
fi

echo "================================"
echo "Lumen API - Production Mode"
echo "================================"
echo "Mode: $APP_MODE"
echo "Port: ${PORT:-8080}"
echo ""

# Build the binary if it doesn't exist or is out of date
if [ ! -f "bin/lumen-api" ] || [ "cmd/api/main.go" -nt "bin/lumen-api" ]; then
    echo "Building production binary..."
    mkdir -p bin
    go build -o bin/lumen-api cmd/api/main.go
    echo "Build complete!"
    echo ""
fi

# Run the binary
echo "Starting production server..."
./bin/lumen-api
