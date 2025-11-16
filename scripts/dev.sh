#!/bin/bash
# Development mode script for Lumen Nutrition Tracker
# Runs the server with hot reload and development settings

set -e

# Set development environment
export APP_MODE=development
export LOG_LEVEL=debug
export LOG_FORMAT=console

# Load environment variables from .env file if it exists
if [ -f ".env" ]; then
    echo "Loading environment from .env file..."
    set -a
    source .env
    set +a
else
    echo "Warning: .env file not found. Using defaults."
    echo "Copy .env.example to .env and configure your settings."
fi

echo "================================"
echo "Lumen API - Development Mode"
echo "================================"
echo "Mode: $APP_MODE"
echo "Port: ${PORT:-8080}"
echo ""

# Check if air is installed for hot reload
if command -v air &> /dev/null; then
    echo "Starting server with hot reload (air)..."
    echo "Edit files and they will auto-reload!"
    echo ""
    air
else
    echo "Starting server (no hot reload)..."
    echo "Install 'air' for hot reload: go install github.com/cosmtrek/air@latest"
    echo ""
    go run cmd/api/main.go
fi
