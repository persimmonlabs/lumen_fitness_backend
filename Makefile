# Lumen Nutrition Tracker - Backend Makefile
# Comprehensive build, test, and deployment automation

.PHONY: help build test run clean lint fmt vet docker dev prod smoke-test load-test coverage install-tools

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=lumen-api
BUILD_DIR=bin
COVERAGE_FILE=coverage.out
MAIN_PATH=cmd/api/main.go

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

## help: Display this help message
help:
	@echo "Lumen Nutrition Tracker - Backend Makefile"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""

## build: Build the server binary
build:
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-linux: Build for Linux (useful for Docker/deployment)
build-linux:
	@echo "$(YELLOW)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)
	@echo "$(GREEN)✓ Linux build complete$(NC)"

## build-windows: Build for Windows
build-windows:
	@echo "$(YELLOW)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "$(GREEN)✓ Windows build complete$(NC)"

## test: Run all tests
test:
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test ./... -v
	@echo "$(GREEN)✓ Tests complete$(NC)"

## test-short: Run tests with -short flag (skip slow tests)
test-short:
	@echo "$(YELLOW)Running short tests...$(NC)"
	@go test ./... -short
	@echo "$(GREEN)✓ Short tests complete$(NC)"

## test-race: Run tests with race detector
test-race:
	@echo "$(YELLOW)Running tests with race detector...$(NC)"
	@go test -race ./... -short
	@echo "$(GREEN)✓ Race detector tests complete$(NC)"

## coverage: Generate test coverage report
coverage:
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	@go test ./... -coverprofile=$(COVERAGE_FILE) -covermode=atomic
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "$(GREEN)✓ Coverage report generated: $(COVERAGE_FILE)$(NC)"

## coverage-html: Generate and open HTML coverage report
coverage-html: coverage
	@echo "$(YELLOW)Generating HTML coverage report...$(NC)"
	@go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "$(GREEN)✓ HTML coverage report: coverage.html$(NC)"

## run: Run the server (development mode)
run:
	@echo "$(YELLOW)Starting server (development mode)...$(NC)"
	@go run $(MAIN_PATH)

## dev: Run server with hot reload (requires air)
dev:
	@echo "$(YELLOW)Starting development server with hot reload...$(NC)"
	@bash scripts/dev.sh

## prod: Run server in production mode
prod: build
	@echo "$(YELLOW)Starting production server...$(NC)"
	@bash scripts/prod.sh

## clean: Remove build artifacts and coverage files
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	@echo "$(GREEN)✓ Clean complete$(NC)"

## fmt: Format all Go files
fmt:
	@echo "$(YELLOW)Formatting code...$(NC)"
	@gofmt -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

## vet: Run go vet
vet:
	@echo "$(YELLOW)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ go vet passed$(NC)"

## lint: Run all linters
lint:
	@echo "$(YELLOW)Running linters...$(NC)"
	@bash scripts/lint.sh
	@echo "$(GREEN)✓ Linting complete$(NC)"

## tidy: Run go mod tidy
tidy:
	@echo "$(YELLOW)Tidying dependencies...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

## smoke-test: Run smoke tests against running server
smoke-test:
	@echo "$(YELLOW)Running smoke tests...$(NC)"
	@bash scripts/smoke-test.sh

## load-test: Run load tests (requires ab or wrk)
load-test:
	@echo "$(YELLOW)Running load tests...$(NC)"
	@bash scripts/load-test.sh

## install-tools: Install development tools
install-tools:
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ Tools installed$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t lumen-api:latest .
	@echo "$(GREEN)✓ Docker image built$(NC)"

## docker-run: Run Docker container
docker-run:
	@echo "$(YELLOW)Starting Docker container...$(NC)"
	@docker run -p 8080:8080 --env-file .env lumen-api:latest

## ci: Run CI checks (format, vet, lint, test)
ci: fmt vet lint test
	@echo "$(GREEN)✓ All CI checks passed$(NC)"

## all: Build and test everything
all: clean tidy fmt vet lint test build
	@echo "$(GREEN)✓ All tasks completed successfully$(NC)"
