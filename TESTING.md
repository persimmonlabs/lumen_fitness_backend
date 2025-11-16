# Testing Infrastructure - Fitness App Go Server

Comprehensive testing infrastructure with unit and integration test support.

## Overview

This testing infrastructure provides:

- **Unit Test Helpers** - Fake clients, fixtures, and assertions for fast isolated tests
- **Integration Test Helpers** - Real database setup, cleanup, and fixtures
- **Makefile Targets** - Convenient commands for running tests
- **Build Tags** - Isolated test execution (unit vs integration)
- **CI/CD Ready** - Coverage checks and quality gates

## Quick Start

### Run Unit Tests (Fast)

```bash
make test
# or
go test ./...
```

**Characteristics**:
- No external dependencies
- Fast execution (< 100ms per package)
- Uses fake/memory adapters
- Default test run

### Run Integration Tests (Requires Database)

```bash
make test-integration
# or
go test -tags=integration ./tests/integration/...
```

**Prerequisites**:
1. PostgreSQL database running
2. Configure `tests/integration/.env.test`
3. Run database migrations

### Run All Tests

```bash
make test-all
```

## Directory Structure

```
backend/
├── tests/
│   ├── unit/
│   │   ├── test_helpers.go      # Fake clients and utilities
│   │   └── example_test.go      # Example unit tests
│   ├── integration/
│   │   ├── test_helpers.go      # Database helpers and fixtures
│   │   ├── example_test.go      # Example integration tests
│   │   └── .env.test            # Integration test config
│   └── README.md                # Detailed testing guide
├── Makefile                      # Build and test automation
└── .env.example                  # Environment configuration template
```

## Unit Test Helpers

### Fake Supabase Client

Complete fake implementation for testing without external dependencies:

```go
import "github.com/lumen/fitness-app/tests/unit"

func TestMyService(t *testing.T) {
    // Create fake client
    client := unit.NewFakeSupabaseClient()
    ctx := unit.TestContext(t)

    // Use fake auth
    auth := client.Auth()
    userID, token, err := auth.SignUp(ctx, "test@example.com", "password")
    unit.AssertNoError(t, err, "SignUp should succeed")

    // Use fake storage
    storage := client.Storage()
    path, err := storage.Upload(ctx, "bucket", "file.txt", data, "text/plain")
    unit.AssertNoError(t, err, "Upload should succeed")
}
```

### Fake Auth Provider

Standalone fake authentication:

```go
auth := unit.NewFakeAuthProvider()

// Sign up
userID, token, _ := auth.SignUp(ctx, "user@example.com", "password")

// Sign in
userID, token, _ := auth.SignIn(ctx, "user@example.com", "password")

// Validate token
userID, _ := auth.ValidateToken(ctx, token)
```

### Fake Storage Provider

In-memory file storage:

```go
storage := unit.NewFakeStorageProvider()

// Upload file
path, _ := storage.Upload(ctx, "bucket", "path/file.txt", reader, "text/plain")

// Download file
reader, _ := storage.Download(ctx, "bucket", "path/file.txt")

// Delete file
storage.Delete(ctx, "bucket", "path/file.txt")
```

### HTTP Test Helper

Utilities for HTTP testing:

```go
helper := unit.NewHTTPTestHelper(t)

// Make request
w := helper.MakeRequest("POST", "/api/users", requestBody)

// Assert status code
helper.AssertStatusCode(w, http.StatusCreated)

// Parse JSON response
var response map[string]interface{}
helper.AssertJSONResponse(w, &response)
```

### Test Fixtures

Common test data generation:

```go
fixtures := unit.NewTestFixtures()

// Create test user
user := fixtures.CreateTestUser()

// Create test session
session := fixtures.CreateTestSession(userID)
```

### Assertion Helpers

Clean assertion functions:

```go
// Assert no error
unit.AssertNoError(t, err, "Operation should succeed")

// Assert error exists
unit.AssertError(t, err, "Operation should fail")

// Assert equality
unit.AssertEqual(t, expected, actual, "Values should match")

// Assert not nil
unit.AssertNotNil(t, value, "Value should not be nil")

// Assert contains
unit.AssertContains(t, slice, "value", "Slice should contain value")
```

## Integration Test Helpers

### Database Connection

Setup real database connection:

```go
import "github.com/lumen/fitness-app/tests/integration"

func TestDatabaseOperation(t *testing.T) {
    integration.SkipIfShort(t)

    // Create database connection
    db, err := integration.NewTestDatabase(t)
    integration.AssertNoError(t, err, "DB connection should succeed")

    ctx := integration.TestContext(t)

    // Use database
    // ...

    // Cleanup is automatic via t.Cleanup()
}
```

### Transaction Management

Use transactions for test isolation:

```go
db, _ := integration.NewTestDatabase(t)
ctx := integration.TestContext(t)

// Begin transaction
tx, err := db.BeginTransaction(ctx)
integration.AssertNoError(t, err, "Transaction should start")

// Rollback at end (automatic cleanup)
defer db.RollbackTransaction(tx)

// Perform operations within transaction
```

### Test Fixtures

Create and cleanup test data:

```go
db, _ := integration.NewTestDatabase(t)
ctx := integration.TestContext(t)
fixtures := integration.NewTestFixtures(t, db)

// Create test user
userID, err := fixtures.CreateTestUser(ctx, "test@example.com")
integration.AssertNoError(t, err, "User creation should succeed")

// Create test workout
workoutID, err := fixtures.CreateTestWorkout(ctx, userID, "Test Workout")

// Verify exists
fixtures.AssertRowExists(ctx, "users", "id", userID)

// Cleanup (manual)
fixtures.DeleteTestWorkout(ctx, workoutID)
fixtures.DeleteTestUser(ctx, userID)

// Or use automatic cleanup
defer db.CleanupTestData(ctx)
```

### Environment Configuration

Load test configuration:

```go
cfg, err := integration.LoadTestConfig()
integration.AssertNoError(t, err, "Config should load")

// Access configuration
dbURL := cfg.DatabaseURL
apiKey := cfg.APIKey
timeout := cfg.Timeout
```

## Makefile Targets

### Testing Commands

```bash
# Run unit tests (default)
make test

# Run unit tests explicitly
make test-unit

# Run integration tests
make test-integration

# Run all tests
make test-all

# Run in short mode (skip slow tests)
make test-short
```

### Coverage Commands

```bash
# Generate HTML coverage report
make coverage

# Display coverage in terminal
make coverage-text

# Check coverage threshold (80%)
make coverage-check
```

### Code Quality Commands

```bash
# Format code
make fmt

# Check formatting
make fmt-check

# Run linter
make lint

# Fix linting issues
make lint-fix

# Type check
make typecheck

# Run go vet
make vet

# Run all checks
make check
```

### Build Commands

```bash
# Build binary
make build

# Build for Linux
make build-linux

# Build for Windows
make build-windows

# Build all platforms
make build-all
```

### Development Commands

```bash
# Run server
make run

# Run with auto-reload (requires air)
make run-watch

# Clean build artifacts
make clean

# Install development tools
make install-tools
```

### CI/CD Commands

```bash
# Run all CI checks
make ci

# Run pre-commit checks (fast)
make pre-commit
```

## Writing Tests

### Unit Test Example

```go
package services_test

import (
    "testing"
    "github.com/lumen/fitness-app/tests/unit"
)

func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    client := unit.NewFakeSupabaseClient()
    service := NewUserService(client.Auth())
    ctx := unit.TestContext(t)

    // Act
    user, err := service.CreateUser(ctx, "test@example.com", "password123")

    // Assert
    unit.AssertNoError(t, err, "CreateUser should succeed")
    unit.AssertNotNil(t, user, "User should not be nil")
    unit.AssertEqual(t, "test@example.com", user.Email, "Email should match")
}
```

### Integration Test Example

```go
//go:build integration

package integration_test

import (
    "testing"
    "github.com/lumen/fitness-app/tests/integration"
)

func TestUserRepository_Create(t *testing.T) {
    integration.SkipIfShort(t)

    // Arrange
    db, err := integration.NewTestDatabase(t)
    integration.AssertNoError(t, err, "DB connection should succeed")

    ctx := integration.TestContext(t)
    fixtures := integration.NewTestFixtures(t, db)

    // Act
    userID, err := fixtures.CreateTestUser(ctx, "test@example.com")

    // Assert
    integration.AssertNoError(t, err, "User creation should succeed")
    fixtures.AssertRowExists(ctx, "users", "id", userID)

    // Cleanup
    defer fixtures.DeleteTestUser(ctx, userID)
}
```

### Table-Driven Test Example

```go
func TestEmailValidation(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid email", "not-an-email", true},
        {"empty email", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            hasErr := err != nil

            if hasErr != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v",
                    tt.input, err, tt.wantErr)
            }
        })
    }
}
```

## Environment Setup

### Unit Tests

No setup required! Unit tests use fake implementations.

### Integration Tests

1. **Create test database**:
   ```bash
   createdb fitness_test
   ```

2. **Configure environment**:
   ```bash
   cp tests/integration/.env.test.example tests/integration/.env.test
   # Edit .env.test with your database credentials
   ```

3. **Run migrations**:
   ```bash
   DATABASE_URL="postgresql://user:pass@localhost/fitness_test" make db-migrate
   ```

4. **Run tests**:
   ```bash
   make test-integration
   ```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: fitness_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependencies
        run: make deps

      - name: Run unit tests
        run: make test-unit

      - name: Run integration tests
        env:
          TEST_DATABASE_URL: postgresql://postgres:postgres@localhost:5432/fitness_test
          TEST_SUPABASE_API_KEY: test-key
          TEST_JWT_SECRET: test-secret
        run: make test-integration

      - name: Check coverage
        run: make coverage-check

      - name: Run linter
        run: make lint
```

## Best Practices

1. **Test Isolation**: Each test should be independent
2. **Use t.Cleanup()**: For automatic resource cleanup
3. **Table-Driven Tests**: For multiple test cases
4. **Descriptive Names**: Clear test and case names
5. **Fast Unit Tests**: Keep unit tests under 100ms
6. **Parallel Tests**: Use `t.Parallel()` when safe
7. **Context Timeout**: Always use context with timeout
8. **Skip Long Tests**: Use `t.Skip()` or `SkipIfShort()`

## Coverage Requirements

- **Domain**: 90%+ coverage
- **Services**: 85%+ coverage
- **Handlers**: 80%+ coverage
- **Overall**: 80%+ coverage

Check with:
```bash
make coverage-check
```

## Troubleshooting

### Integration Tests Fail

**Error**: "TEST_DATABASE_URL environment variable is required"
**Solution**: Configure `tests/integration/.env.test`

**Error**: "failed to connect to database"
**Solution**: Ensure PostgreSQL is running

**Error**: "table does not exist"
**Solution**: Run migrations on test database

### Unit Tests Slow

**Issue**: Tests taking longer than expected
**Solution**: Ensure using fake clients, not real connections

## Additional Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- Architecture Documentation: `docs/ARCHITECTURE.md`
- Full Testing Guide: `tests/README.md`

---

**Created**: 2025-01-15
**Version**: 1.0.0
