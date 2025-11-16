# Backend Development Rules - Fitness App Go Server

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Development Rules](#development-rules)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Performance Guidelines](#performance-guidelines)
- [Security Guidelines](#security-guidelines)
- [Git Workflow](#git-workflow)

## Overview

This is the backend Go server for the fitness tracking application. The server follows Clean Architecture principles with a focus on:

- Modular, testable code
- Type-safe error handling
- Comprehensive configuration management
- Feature flag support for graceful degradation
- Clear separation of concerns

**Technology Stack:**
- Go 1.24.4+
- Supabase (authentication, database, storage)
- Standard library with minimal dependencies

## Project Structure

```
backend/
├── cmd/
│   └── api/              # Application entry points
├── internal/
│   ├── adapters/         # External service adapters (Supabase, etc.)
│   ├── api/
│   │   └── handlers/     # HTTP request handlers
│   ├── config/           # Configuration management
│   ├── core/
│   │   └── errors/       # Domain error types
│   ├── database/         # Database layer
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Domain models
│   ├── pkg/
│   │   └── logger/       # Logging utilities
│   └── services/         # Business logic layer
├── pkg/                  # Public packages (reusable across projects)
│   └── logger/
└── storage/
    └── fake_supabase/    # In-memory implementations for testing
```

### Directory Organization Rules

**NEVER save files to the root folder unless they are:**
- `go.mod` / `go.sum`
- `README.md`
- `CLAUDE.md` (this file)
- `.env` / `.env.example`
- `.gitignore`

**ALWAYS organize files in appropriate subdirectories:**
- Source code → `internal/` or `pkg/`
- Tests → Same directory as source with `_test.go` suffix
- Documentation → `../docs/` (parent docs directory)
- Scripts → `scripts/`
- Configuration examples → `config/`

## Development Rules

### 1. Architecture Patterns

**Controller-Service-Repository Pattern**

```go
// Handler (Controller) - HTTP concerns only
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := h.DecodeJSON(r, &req); err != nil {
        h.BadRequestError(w, "Invalid request body", nil)
        return
    }

    // Delegate to service layer
    user, err := h.userService.Create(r.Context(), req)
    if err != nil {
        h.handleError(w, err)
        return
    }

    h.SuccessResponse(w, http.StatusCreated, user, nil)
}

// Service - Business logic
type UserService struct {
    repo UserRepository
    auth AuthProvider
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Validation
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // Business logic
    user := &User{...}

    // Delegate to repository
    return s.repo.Save(ctx, user)
}

// Repository - Data access
type UserRepository interface {
    Save(ctx context.Context, user *User) (*User, error)
    FindByID(ctx context.Context, id string) (*User, error)
}
```

### 2. File Size and Modularity

- **Maximum file size:** 500 lines
- **Split large files** by concern (e.g., `user_create.go`, `user_update.go`)
- **One public interface per file** (exceptions for tightly coupled types)
- **Group related private functions** after the public API

### 3. Feature Flags

Use feature flags for graceful degradation:

```go
// Check feature flag before using optional features
if cfg.Features.EnableAuth {
    // Full authentication flow
    user, err := authProvider.ValidateToken(ctx, token)
} else {
    // Development bypass
    user = &User{ID: "dev-user"}
}
```

### 4. Configuration Management

- **Never hardcode:** URLs, secrets, timeouts, or limits
- **Always use config:** Load from `config.Config`
- **Validate early:** Configuration validation happens at startup
- **Environment-aware:** Respect `APP_MODE` (development/testing/production)

```go
// ✅ CORRECT
timeout := cfg.Server.ReadTimeout
maxConns := cfg.Database.MaxOpenConns

// ❌ WRONG
timeout := 15 * time.Second
maxConns := 25
```

## Coding Standards

### 1. Package Documentation

Every package MUST have a package comment:

```go
// Package handlers provides HTTP request handlers for the fitness app API.
//
// This package contains the base handler functionality and common utilities
// used across all API endpoints including JSON response helpers, error
// formatting, and request validation utilities.
package handlers
```

### 2. Import Organization

Group imports in three sections with blank lines:

```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"
    "time"

    // External dependencies
    "github.com/joho/godotenv"

    // Internal packages
    "github.com/lumen/fitness-app/internal/config"
    "github.com/lumen/fitness-app/internal/core/errors"
)
```

### 3. Naming Conventions

**Packages:**
- Lowercase, single word (e.g., `handlers`, `errors`)
- No underscores or mixed caps
- Descriptive but concise

**Types:**
- PascalCase for exported (e.g., `UserService`)
- camelCase for unexported (e.g., `userCache`)
- Interface names: action + "er" suffix (e.g., `UserRepository`, `Validator`)

**Functions/Methods:**
- PascalCase for exported (e.g., `CreateUser`)
- camelCase for unexported (e.g., `validateEmail`)
- Be specific (e.g., `GetUserByID` not `Get`)

**Variables:**
- camelCase (e.g., `userID`, `maxRetries`)
- Acronyms: all caps if exported (e.g., `UserID`, `HTTPURL`)
- Avoid single-letter names except in short scopes (loop indices)

**Constants:**
- PascalCase or SCREAMING_SNAKE_CASE for exported
- Group related constants in blocks

```go
const (
    ModeDevelopment AppMode = "development"
    ModeTesting     AppMode = "testing"
    ModeProduction  AppMode = "production"
)
```

### 4. Comments

**Public APIs:**
- Every exported type, function, constant, and variable MUST have a comment
- Start with the name of the item
- Use complete sentences

```go
// UserService provides business logic for user management operations.
// It handles user creation, authentication, profile updates, and deletion.
type UserService struct {
    repo UserRepository
}

// CreateUser registers a new user account with email and password.
// Returns ErrDuplicate if the email already exists.
// Returns ErrValidation if the input data is invalid.
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Implementation
}
```

**Implementation Comments:**
- Explain "why", not "what"
- Comment complex logic
- Use TODO/FIXME/NOTE for markers

```go
// NOTE: We use bcrypt cost 12 for production security while allowing
// lower cost in testing for faster test execution
cost := bcrypt.DefaultCost
if cfg.IsTesting() {
    cost = bcrypt.MinCost
}
```

### 5. Error Handling

**Use typed domain errors from `internal/core/errors`:**

```go
// ✅ CORRECT - Use domain error constructors
if user == nil {
    return errors.NotFound("user", userID)
}

if !validPassword {
    return errors.InvalidCredentials()
}

valErr := errors.Validation("Invalid user data")
valErr.AddField("email", "must be a valid email address")
valErr.AddField("password", "must be at least 8 characters")
return valErr

// ❌ WRONG - Don't use generic errors
return fmt.Errorf("user not found")
return errors.New("invalid password")
```

**Error wrapping:**

```go
// ✅ CORRECT - Preserve error chain
user, err := s.repo.FindByID(ctx, id)
if err != nil {
    return nil, errors.Wrap(err, "failed to fetch user")
}

// With context
if err != nil {
    return nil, errors.Wrapf(err, "failed to fetch user %s", id)
}
```

**Error checking patterns:**

```go
// Use type checking functions
if errors.IsNotFound(err) {
    return errors.NotFound("workout", workoutID)
}

if errors.IsUnauthorized(err) {
    h.UnauthorizedError(w, "")
    return
}
```

### 6. Context Handling

**Always pass context as first parameter:**

```go
func (s *UserService) Create(ctx context.Context, user *User) error
```

**Propagate context through call chain:**

```go
func (s *WorkoutService) CreateWorkout(ctx context.Context, req CreateWorkoutRequest) (*Workout, error) {
    // Pass context to database calls
    user, err := s.userRepo.FindByID(ctx, req.UserID)

    // Pass context to external services
    analysis, err := s.aiService.AnalyzeWorkout(ctx, workout)

    return workout, nil
}
```

**Respect context cancellation:**

```go
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-ch:
    return result, nil
}
```

### 7. Struct Tags

Use consistent tag ordering:

```go
type User struct {
    ID        string    `json:"id" db:"id"`
    Email     string    `json:"email" db:"email" validate:"required,email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

### 8. Interfaces

**Define interfaces at usage point:**

```go
// ✅ CORRECT - Define in consumer package
package services

type UserRepository interface {
    Save(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
}

// ❌ WRONG - Don't define in implementation package
package database

type UserRepository interface { ... }
```

**Keep interfaces small:**

```go
// ✅ CORRECT - Single-method interfaces
type Validator interface {
    Validate() error
}

// ❌ WRONG - Large interface
type UserManager interface {
    Create(...) error
    Update(...) error
    Delete(...) error
    Find(...) error
    List(...) error
    // ... 10+ more methods
}
```

## Testing Requirements

### 1. Test Organization

- **Co-locate tests:** Place `*_test.go` files next to source files
- **Test package suffix:** Use `_test` package for black-box tests
- **Same package:** Test unexported functions in the same package

```go
// Black-box testing (external API)
package handlers_test

// White-box testing (internal implementation)
package handlers
```

### 2. Test Naming

```go
// Pattern: Test<FunctionName>
func TestCreateUser(t *testing.T) { ... }

// Table-driven tests
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"empty string", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 3. Test Coverage

- **Minimum coverage:** 80% for all packages
- **Critical paths:** 100% coverage for:
  - Error handling logic
  - Authentication/authorization
  - Payment processing
  - Data validation

### 4. Mocking

Use fake implementations in `storage/fake_supabase/`:

```go
func TestUserService_Create(t *testing.T) {
    // Use fake client for testing
    fakeClient := supabase.NewFakeClient()
    service := NewUserService(fakeClient.Auth())

    user, err := service.Create(context.Background(), req)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### 5. Test Cleanup

Always clean up resources:

```go
func TestDatabase(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close() // Cleanup

    t.Cleanup(func() {
        // Additional cleanup
        db.Truncate()
    })
}
```

## Performance Guidelines

### 1. Context Timeouts

Set appropriate timeouts for operations:

```go
// Database operations
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

// External API calls
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### 2. Connection Pooling

Use configuration for connection limits:

```go
db.SetMaxOpenConns(cfg.Database.MaxOpenConns)     // Default: 25
db.SetMaxIdleConns(cfg.Database.MaxIdleConns)     // Default: 5
db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime) // Default: 5m
```

### 3. Defer Placement

Close resources as soon as possible:

```go
// ✅ CORRECT
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()

// ❌ WRONG - defer in loop
for _, url := range urls {
    resp, _ := http.Get(url)
    defer resp.Body.Close() // Accumulates resources
}
```

### 4. Avoid Allocations

Reuse buffers and use sync.Pool for frequently allocated objects:

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processData(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()

    // Use buffer
}
```

### 5. Logging Performance

Use structured logging with appropriate levels:

```go
// Development - detailed logs
logger.Debug("processing request",
    slog.String("user_id", userID),
    slog.String("action", "create_workout"),
)

// Production - minimal logs
logger.Info("request completed",
    slog.Int("status", 200),
    slog.Duration("duration", elapsed),
)
```

## Security Guidelines

### 1. Secrets Management

**Never commit secrets:**

```go
// ✅ CORRECT - Use environment variables
supabaseURL := os.Getenv("SUPABASE_URL")
serviceKey := os.Getenv("SUPABASE_SERVICE_KEY")

// ❌ WRONG - Hardcoded secrets
supabaseURL := "https://xxxxx.supabase.co"
serviceKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 2. Input Validation

Validate all user input:

```go
// Validate request payload
func (r *CreateUserRequest) Validate() error {
    valErr := errors.Validation("Invalid user data")

    if r.Email == "" {
        valErr.AddField("email", "email is required")
    } else if !isValidEmail(r.Email) {
        valErr.AddField("email", "must be a valid email address")
    }

    if len(r.Password) < 8 {
        valErr.AddField("password", "must be at least 8 characters")
    }

    if valErr.HasErrors() {
        return valErr
    }
    return nil
}
```

### 3. SQL Injection Prevention

Use parameterized queries:

```go
// ✅ CORRECT - Parameterized query
query := "SELECT * FROM users WHERE email = $1"
row := db.QueryRowContext(ctx, query, email)

// ❌ WRONG - String concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
```

### 4. Authentication

Always validate tokens:

```go
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    // Extract token from header
    token := extractBearerToken(r)
    if token == "" {
        h.UnauthorizedError(w, "")
        return
    }

    // Validate token
    userID, err := h.auth.ValidateToken(r.Context(), token)
    if err != nil {
        if errors.IsTokenExpired(err) {
            h.ErrorResponse(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired", nil)
            return
        }
        h.UnauthorizedError(w, "Invalid token")
        return
    }

    // Process authenticated request
}
```

### 5. CORS Configuration

Use strict CORS in production:

```go
// Development - allow all origins
AllowedOrigins: []string{"*"}

// Production - specific origins only
AllowedOrigins: []string{
    "https://app.example.com",
    "https://www.example.com",
}
```

### 6. Rate Limiting

Implement rate limiting to prevent abuse:

```go
if cfg.Features.EnableRateLimit {
    // Apply rate limiting middleware
    handler = rateLimitMiddleware(handler)
}
```

## Git Workflow

### 1. Branch Naming

- Feature: `feature/user-authentication`
- Bug fix: `fix/login-token-validation`
- Hotfix: `hotfix/security-patch`
- Refactor: `refactor/error-handling`

### 2. Commit Messages

Follow conventional commits:

```
feat: add user profile endpoint
fix: correct token expiration validation
refactor: simplify error handling in handlers
test: add integration tests for auth flow
docs: update API documentation for workout endpoints
```

### 3. Pre-Commit Checks

Run before committing:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test ./... -race -cover

# Check for vulnerabilities
go vet ./...
```

### 4. Pull Request Requirements

- All tests passing
- Code coverage maintained or improved
- No linting errors
- Documentation updated
- Reviewed by at least one team member

## AI Agent Instructions

When working with this codebase:

1. **Read configuration first:** Check `internal/config/config.go` for available settings
2. **Use existing error types:** Don't create new error types; use `internal/core/errors`
3. **Follow existing patterns:** Look at similar code before implementing new features
4. **Test thoroughly:** Write tests for all new code
5. **Document changes:** Update this file and other docs as needed
6. **Respect feature flags:** Use configuration-driven features
7. **Never hardcode:** Always use configuration for environment-specific values
8. **Check file organization:** Never save files to root; use appropriate subdirectories

## Common Commands

```bash
# Run server
go run cmd/api/main.go

# Run tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./internal/handlers -run TestCreateUser -v

# Format code
go fmt ./...

# Lint code
golangci-lint run

# Build binary
go build -o bin/api cmd/api/main.go

# Run with environment
APP_MODE=development go run cmd/api/main.go
```

## Nutrition Calculation Rules

**Last Updated:** 2025-11-16

### Single Source of Truth

**CRITICAL: Database triggers (migration 012) are the ONLY place that calculates meal nutrition totals.**

- ✅ Database triggers automatically calculate `meals.total_*` columns when meal_items change
- ❌ Backend services NEVER manually calculate totals
- ❌ Frontend components NEVER calculate totals client-side

**Why:** This ensures data consistency across all layers and eliminates calculation bugs.

**Implementation:**
```sql
-- Database trigger (migration 012_nutrition_totals_trigger.up.sql)
CREATE OR REPLACE FUNCTION update_meal_nutrition_totals()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE meals
    SET
        total_calories = COALESCE((SELECT SUM(calories) FROM meal_items WHERE meal_id = NEW.meal_id), 0),
        total_protein_g = COALESCE((SELECT SUM(protein) FROM meal_items WHERE meal_id = NEW.meal_id), 0),
        total_carbs_g = COALESCE((SELECT SUM(carbs) FROM meal_items WHERE meal_id = NEW.meal_id), 0),
        total_fat_g = COALESCE((SELECT SUM(fat) FROM meal_items WHERE meal_id = NEW.meal_id), 0),
        total_fiber_g = COALESCE((SELECT SUM(fiber) FROM meal_items WHERE meal_id = NEW.meal_id), 0)
    WHERE id = NEW.meal_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

### Naming Standards

**AUTHORITATIVE REFERENCE:** See `backend/docs/naming-standards.md` for complete naming rules.

**Quick Reference:**

| Layer | Field Type | Format | Example |
|-------|-----------|---------|---------|
| Database (meals) | Total macros | `total_<macro>_g` | `total_protein_g` |
| Database (meal_items) | Individual macros | `<macro>` (no suffix) | `protein` |
| Go JSON tags | All macros | `<field>_g` | `"protein_g"` |
| Go DB tags | Match DB exactly | (varies) | `db:"protein"` or `db:"total_protein_g"` |
| TypeScript | All macros | `<field>_g` | `protein_g: number` |

**Key Rule:** JSON responses ALWAYS use `_g` suffix for clarity. Database columns vary by table.

**Example Go Struct:**
```go
type Meal struct {
    TotalCalories float64 `json:"total_calories" db:"total_calories"`
    TotalProteinG float64 `json:"total_protein_g" db:"total_protein_g"`  // Note: _g in both
}

type MealItem struct {
    Calories  float64 `json:"calories" db:"calories"`
    ProteinG  float64 `json:"protein_g" db:"protein"`  // ⚠️ JSON has _g, DB does NOT
}
```

### Macro Constants

**NEVER hardcode Atwater factors (4, 4, 9) in code.**

**ALWAYS import from:** `internal/domain/nutrition/constants/macros.go`

```go
import "github.com/lumen/fitness-app/internal/domain/nutrition/constants"

// ✅ CORRECT
calculatedCalories := constants.CalculateMacroCalories(proteinG, carbsG, fatG)

// ✅ CORRECT - Using constants
calories := (protein * constants.CaloriesPerGramProtein) +
           (carbs * constants.CaloriesPerGramCarbs) +
           (fat * constants.CaloriesPerGramFat)

// ❌ WRONG - Hardcoded values
calories := (protein * 4) + (carbs * 4) + (fat * 9)
```

**Available Constants:**
- `CaloriesPerGramProtein` = 4.0
- `CaloriesPerGramCarbs` = 4.0
- `CaloriesPerGramFat` = 9.0
- `CaloriesPerGramAlcohol` = 7.0 (future use)
- `MacroCalorieTolerancePercent` = 0.10 (10% validation tolerance)

**Validation:**
```go
// Use helper function for validation
isValid := constants.ValidateMacroCalories(statedCalories, proteinG, carbsG, fatG)
```

### Migration History

**Migration 012:** `012_nutrition_totals_trigger.up.sql`
- Created database triggers for automatic total calculation
- Renamed `food_name` → `name` in meal_items and template_items
- Dropped deprecated `meal_time` and `name` columns from meals table

**Migration 013:** `013_optimize_rpc_functions.up.sql`
- Optimized RPC functions to use pre-calculated `meals.total_*` columns
- Removed manual SUM operations from SQL queries
- **150x performance improvement** on large datasets

### Code Review Checklist

When reviewing nutrition-related code:

- [ ] NO manual total calculations in backend services
- [ ] NO manual total calculations in frontend components
- [ ] ALL JSON field names match `backend/docs/naming-standards.md`
- [ ] TypeScript interfaces match backend JSON tags exactly
- [ ] Atwater factors imported from `constants/macros.go` (no hardcoding)
- [ ] Database triggers remain intact (no ALTER TRIGGER DISABLE)

### Common Mistakes to Avoid

```go
// ❌ WRONG: Calculating totals in service layer
func (s *service) CreateMeal(...) {
    totalCalories := 0.0
    for _, item := range items {
        totalCalories += item.Calories  // Database should do this!
    }
}

// ❌ WRONG: Using wrong field names
type MealItem struct {
    FoodName string `json:"food_name"` // Should be "name"
    Grams    float64 `json:"grams"`     // Should be "quantity"
}

// ❌ WRONG: Hardcoding macro constants
calories := (protein * 4) + (carbs * 4) + (fat * 9)  // Use constants!
```

**Related Documentation:**
- Authoritative naming rules: `backend/docs/naming-standards.md`
- API field reference: `backend/docs/API-NUTRITION-CONTRACTS.md`
- Architecture details: `backend/docs/ARCHITECTURE-SINGLE-SOURCE-TRUTH.md`
- Migration analysis: `backend/docs/calculation-inventory.md`

---

## Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Project Layout](https://github.com/golang-standards/project-layout)
