# Development Guide

Complete guide for developing features in the Lumen Nutrition Tracker Backend.

## Table of Contents

- [Getting Started](#getting-started)
- [Project Architecture](#project-architecture)
- [Adding a New Feature](#adding-a-new-feature)
- [Testing Conventions](#testing-conventions)
- [Code Style Guide](#code-style-guide)
- [Domain Structure](#domain-structure)
- [Common Patterns](#common-patterns)
- [Database Migrations](#database-migrations)
- [AI Integration](#ai-integration)
- [Debugging](#debugging)

---

## Getting Started

### Prerequisites

- Go 1.24+
- Git
- IDE with Go support (VS Code, GoLand, etc.)
- PostgreSQL client (optional, for testing against real DB)

### Setup Development Environment

```bash
# Clone repository
git clone <repository-url>
cd backend

# Install dependencies
go mod download

# Setup development environment
cp .env.example .env
nano .env  # Edit with your API keys

# Set development mode
# In .env:
APP_MODE=development
DATABASE_MODE=memory  # Use in-memory for quick iteration
FEATURE_AUTH=false    # Disable auth for easier testing
LOG_LEVEL=debug       # Verbose logging

# Run server
go run cmd/api/main.go

# In another terminal, run tests with watch
go install github.com/cosmtrek/air@latest
air  # Hot reload on file changes
```

### Development Workflow

```bash
# Create feature branch
git checkout -b feature/meal-photos

# Make changes
# ... edit files ...

# Run tests
go test ./...

# Run specific package tests
go test ./internal/domain/nutrition/meals/... -v

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Lint (optional, install golangci-lint first)
golangci-lint run

# Commit
git add .
git commit -m "feat: add meal photo upload support"

# Push
git push origin feature/meal-photos
```

---

## Project Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────┐
│         Handlers (HTTP Layer)           │
│  - Parse HTTP requests                  │
│  - Return HTTP responses                │
│  - No business logic                    │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│        Services (Business Logic)        │
│  - Domain rules and validation          │
│  - Orchestrate operations               │
│  - Technology-agnostic                  │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│      Repositories (Data Access)         │
│  - Database queries                     │
│  - External API calls                   │
│  - Cache operations                     │
└─────────────────────────────────────────┘
```

### Directory Structure

```
internal/
├── config/              # Configuration management
│   ├── config.go        # Config struct and loader
│   └── config_test.go   # Config tests
│
├── domain/              # Business domains
│   └── nutrition/
│       ├── meals/       # Meal domain
│       │   ├── models.go      # Data models
│       │   ├── repository.go  # Repository interface
│       │   ├── service.go     # Business logic
│       │   ├── handler.go     # HTTP handlers
│       │   └── *_test.go      # Tests
│       ├── weight/      # Weight tracking domain
│       ├── goals/       # Goals domain
│       ├── analytics/   # Analytics domain
│       └── templates/   # Templates domain
│
├── errors/              # Domain error types
│   ├── errors.go        # Error definitions
│   └── errors_test.go
│
├── server/              # HTTP server setup
│   ├── handlers/        # Base handlers
│   ├── middleware/      # HTTP middleware
│   ├── router.go        # Route definitions
│   └── server.go        # Server initialization
│
├── services/            # Shared services
│   ├── ai/             # AI providers
│   ├── cache/          # Caching
│   ├── cost/           # Cost tracking
│   └── storage/        # File storage
│
├── supabase/           # Supabase client
└── testutil/           # Test helpers
```

---

## Adding a New Feature

Let's walk through adding a new feature: **Meal Ratings**.

### Step 1: Define Models

**File**: `internal/domain/nutrition/ratings/models.go`

```go
package ratings

import (
    "time"
    "github.com/google/uuid"
)

// MealRating represents a user's rating of a meal
type MealRating struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    MealID      uuid.UUID  `json:"meal_id" db:"meal_id"`
    UserID      uuid.UUID  `json:"user_id" db:"user_id"`
    Rating      int        `json:"rating" db:"rating"`           // 1-5 stars
    Taste       *int       `json:"taste,omitempty" db:"taste"`   // 1-5
    Satiety     *int       `json:"satiety,omitempty" db:"satiety"` // 1-5
    WouldEatAgain bool     `json:"would_eat_again" db:"would_eat_again"`
    Notes       string     `json:"notes" db:"notes"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateRatingRequest represents request to create a rating
type CreateRatingRequest struct {
    MealID        uuid.UUID `json:"meal_id" validate:"required"`
    Rating        int       `json:"rating" validate:"required,min=1,max=5"`
    Taste         *int      `json:"taste,omitempty" validate:"omitempty,min=1,max=5"`
    Satiety       *int      `json:"satiety,omitempty" validate:"omitempty,min=1,max=5"`
    WouldEatAgain bool      `json:"would_eat_again"`
    Notes         string    `json:"notes,omitempty" validate:"max=500"`
}

// UpdateRatingRequest represents request to update a rating
type UpdateRatingRequest struct {
    Rating        *int   `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
    Taste         *int   `json:"taste,omitempty" validate:"omitempty,min=1,max=5"`
    Satiety       *int   `json:"satiety,omitempty" validate:"omitempty,min=1,max=5"`
    WouldEatAgain *bool  `json:"would_eat_again,omitempty"`
    Notes         *string `json:"notes,omitempty" validate:"omitempty,max=500"`
}
```

### Step 2: Create Repository

**File**: `internal/domain/nutrition/ratings/repository.go`

```go
package ratings

import (
    "context"
    "github.com/google/uuid"
)

// Repository defines the interface for meal rating data access
type Repository interface {
    Create(ctx context.Context, rating *MealRating) error
    GetByID(ctx context.Context, userID, ratingID uuid.UUID) (*MealRating, error)
    GetByMealID(ctx context.Context, userID, mealID uuid.UUID) (*MealRating, error)
    Update(ctx context.Context, rating *MealRating) error
    Delete(ctx context.Context, userID, ratingID uuid.UUID) error
    ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]MealRating, int, error)
    GetAverageRatingForMeal(ctx context.Context, mealID uuid.UUID) (float64, int, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
    db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, rating *MealRating) error {
    query := `
        INSERT INTO meal_ratings (id, meal_id, user_id, rating, taste, satiety, would_eat_again, notes)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING created_at, updated_at
    `

    return r.db.QueryRowContext(
        ctx, query,
        rating.ID, rating.MealID, rating.UserID, rating.Rating,
        rating.Taste, rating.Satiety, rating.WouldEatAgain, rating.Notes,
    ).Scan(&rating.CreatedAt, &rating.UpdatedAt)
}

// ... implement other methods
```

### Step 3: Build Service

**File**: `internal/domain/nutrition/ratings/service.go`

```go
package ratings

import (
    "context"
    "github.com/google/uuid"
    "github.com/pradord/lumen_final/backend/internal/errors"
)

// Service provides business logic for meal ratings
type Service interface {
    CreateRating(ctx context.Context, userID uuid.UUID, req CreateRatingRequest) (*MealRating, error)
    GetRating(ctx context.Context, userID, ratingID uuid.UUID) (*MealRating, error)
    UpdateRating(ctx context.Context, userID, ratingID uuid.UUID, req UpdateRatingRequest) (*MealRating, error)
    DeleteRating(ctx context.Context, userID, ratingID uuid.UUID) error
    GetMealRatings(ctx context.Context, mealID uuid.UUID) ([]MealRating, error)
}

type service struct {
    repo      Repository
    mealRepo  meals.Repository  // To verify meal exists
}

func NewService(repo Repository, mealRepo meals.Repository) Service {
    return &service{
        repo:     repo,
        mealRepo: mealRepo,
    }
}

func (s *service) CreateRating(ctx context.Context, userID uuid.UUID, req CreateRatingRequest) (*MealRating, error) {
    // Validate meal exists and belongs to user
    meal, err := s.mealRepo.GetByID(ctx, userID, req.MealID)
    if err != nil {
        return nil, errors.Wrap(err, "meal not found")
    }

    // Check if rating already exists for this meal
    existing, _ := s.repo.GetByMealID(ctx, userID, req.MealID)
    if existing != nil {
        return nil, errors.Conflict("meal_rating", "rating already exists for this meal")
    }

    // Create rating
    rating := &MealRating{
        ID:            uuid.New(),
        MealID:        req.MealID,
        UserID:        userID,
        Rating:        req.Rating,
        Taste:         req.Taste,
        Satiety:       req.Satiety,
        WouldEatAgain: req.WouldEatAgain,
        Notes:         req.Notes,
    }

    if err := s.repo.Create(ctx, rating); err != nil {
        return nil, errors.Wrap(err, "failed to create rating")
    }

    return rating, nil
}

// ... implement other methods
```

### Step 4: Add Handler

**File**: `internal/domain/nutrition/ratings/handler.go`

```go
package ratings

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)

// Handler handles HTTP requests for meal ratings
type Handler struct {
    service Service
}

func NewHandler(service Service) *Handler {
    return &Handler{service: service}
}

// RegisterRoutes registers rating routes
func (h *Handler) RegisterRoutes(r chi.Router) {
    r.Post("/api/v1/ratings", h.CreateRating)
    r.Get("/api/v1/ratings/{id}", h.GetRating)
    r.Put("/api/v1/ratings/{id}", h.UpdateRating)
    r.Delete("/api/v1/ratings/{id}", h.DeleteRating)
    r.Get("/api/v1/meals/{mealId}/ratings", h.GetMealRatings)
}

// CreateRating handles POST /api/v1/ratings
func (h *Handler) CreateRating(w http.ResponseWriter, r *http.Request) {
    userID := getUserIDFromContext(r.Context())
    if userID == uuid.Nil {
        respondError(w, http.StatusUnauthorized, "user not authenticated")
        return
    }

    var req CreateRatingRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    rating, err := h.service.CreateRating(r.Context(), userID, req)
    if err != nil {
        handleServiceError(w, err)
        return
    }

    respondJSON(w, http.StatusCreated, rating)
}

// ... implement other handlers
```

### Step 5: Register Routes

**File**: `internal/server/router.go`

```go
func SetupRoutes(s *Server) {
    // ... existing routes ...

    // Meal ratings
    ratingsRepo := ratings.NewPostgresRepository(s.db)
    ratingsService := ratings.NewService(ratingsRepo, mealsRepo)
    ratingsHandler := ratings.NewHandler(ratingsService)
    ratingsHandler.RegisterRoutes(s.Router)
}
```

### Step 6: Write Tests

**File**: `internal/domain/nutrition/ratings/service_test.go`

```go
package ratings_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestService_CreateRating(t *testing.T) {
    tests := []struct {
        name    string
        userID  uuid.UUID
        req     ratings.CreateRatingRequest
        wantErr bool
        errMsg  string
    }{
        {
            name:   "valid rating",
            userID: uuid.New(),
            req: ratings.CreateRatingRequest{
                MealID: uuid.New(),
                Rating: 5,
            },
            wantErr: false,
        },
        {
            name:   "invalid rating too high",
            userID: uuid.New(),
            req: ratings.CreateRatingRequest{
                MealID: uuid.New(),
                Rating: 6,
            },
            wantErr: true,
            errMsg:  "validation",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            mockRepo := new(MockRepository)
            mockMealRepo := new(MockMealRepository)

            // Configure mock expectations
            if !tt.wantErr {
                mockMealRepo.On("GetByID", mock.Anything, tt.userID, tt.req.MealID).
                    Return(&meals.Meal{ID: tt.req.MealID}, nil)
                mockRepo.On("GetByMealID", mock.Anything, tt.userID, tt.req.MealID).
                    Return(nil, nil)
                mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*ratings.MealRating")).
                    Return(nil)
            }

            svc := ratings.NewService(mockRepo, mockMealRepo)

            // Execute
            result, err := svc.CreateRating(context.Background(), tt.userID, tt.req)

            // Assert
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errMsg != "" {
                    assert.Contains(t, err.Error(), tt.errMsg)
                }
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, tt.req.Rating, result.Rating)
            }

            mockRepo.AssertExpectations(t)
            mockMealRepo.AssertExpectations(t)
        })
    }
}
```

### Step 7: Create Migration

**File**: `migrations/006_meal_ratings.sql`

```sql
-- Create meal_ratings table
CREATE TABLE IF NOT EXISTS meal_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    taste INT CHECK (taste IS NULL OR (taste >= 1 AND taste <= 5)),
    satiety INT CHECK (satiety IS NULL OR (satiety >= 1 AND satiety <= 5)),
    would_eat_again BOOLEAN NOT NULL DEFAULT false,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(meal_id, user_id)
);

-- Create indexes
CREATE INDEX idx_meal_ratings_meal_id ON meal_ratings(meal_id);
CREATE INDEX idx_meal_ratings_user_id ON meal_ratings(user_id);
CREATE INDEX idx_meal_ratings_rating ON meal_ratings(rating);

-- Enable RLS
ALTER TABLE meal_ratings ENABLE ROW LEVEL SECURITY;

-- RLS policies
CREATE POLICY "Users can view own ratings"
ON meal_ratings FOR SELECT
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY "Users can create own ratings"
ON meal_ratings FOR INSERT
TO authenticated
WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update own ratings"
ON meal_ratings FOR UPDATE
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY "Users can delete own ratings"
ON meal_ratings FOR DELETE
TO authenticated
USING (user_id = auth.uid());

-- Update trigger
CREATE TRIGGER update_meal_ratings_updated_at
BEFORE UPDATE ON meal_ratings
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

### Step 8: Update Documentation

Add to `docs/API.md`:

```markdown
### Ratings

#### POST /api/v1/ratings
Create a meal rating.

**Request**:
```json
{
  "meal_id": "meal-uuid",
  "rating": 5,
  "taste": 5,
  "satiety": 4,
  "would_eat_again": true,
  "notes": "Delicious and filling!"
}
```

**Response** (201 Created): Full rating object
```

---

## Testing Conventions

### Test File Organization

- Place tests next to source files: `service.go` → `service_test.go`
- Use `_test` package for black-box tests: `package ratings_test`
- Use same package for white-box tests: `package ratings`

### Table-Driven Tests

```go
func TestValidateRating(t *testing.T) {
    tests := []struct {
        name    string
        rating  int
        wantErr bool
    }{
        {"valid min", 1, false},
        {"valid max", 5, false},
        {"invalid too low", 0, true},
        {"invalid too high", 6, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateRating(tt.rating)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateRating() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Mocking

Use testify/mock for interfaces:

```go
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, rating *MealRating) error {
    args := m.Called(ctx, rating)
    return args.Error(0)
}
```

### Test Helpers

Use helpers from `internal/testutil`:

```go
func TestHandler_CreateRating(t *testing.T) {
    req := testutil.NewRequest(t, "POST", "/api/v1/ratings", map[string]interface{}{
        "meal_id": "test-uuid",
        "rating": 5,
    })

    rr := testutil.ExecuteRequest(handler.CreateRating, req)

    testutil.AssertStatus(t, rr, http.StatusCreated)
    testutil.AssertJSONField(t, rr.Body, "rating", 5)
}
```

---

## Code Style Guide

### Naming Conventions

**Packages**: lowercase, single word
```go
package ratings  // ✓
package mealRatings  // ✗
```

**Types**: PascalCase
```go
type MealRating struct { }  // ✓
type mealRating struct { }  // ✗ (unexported)
```

**Functions**: PascalCase (exported), camelCase (unexported)
```go
func CreateRating() { }      // ✓ exported
func validateRating() { }    // ✓ unexported
func create_rating() { }     // ✗
```

**Variables**: camelCase
```go
var userID uuid.UUID        // ✓
var user_id uuid.UUID       // ✗
```

### Error Handling

Use typed errors from `internal/errors`:

```go
// ✓ Good
if user == nil {
    return errors.NotFound("user", userID.String())
}

// ✗ Bad
if user == nil {
    return fmt.Errorf("user not found")
}
```

Wrap errors for context:

```go
result, err := repo.Create(ctx, rating)
if err != nil {
    return nil, errors.Wrap(err, "failed to create rating")
}
```

### Comments

Document all exported items:

```go
// CreateRating creates a new meal rating for the specified user.
// Returns ErrConflict if a rating already exists for this meal.
// Returns ErrNotFound if the meal doesn't exist.
func CreateRating(ctx context.Context, userID uuid.UUID, req CreateRatingRequest) (*MealRating, error) {
    // ...
}
```

### Context Usage

Always pass context as first parameter:

```go
// ✓ Good
func GetRating(ctx context.Context, id uuid.UUID) (*MealRating, error)

// ✗ Bad
func GetRating(id uuid.UUID, ctx context.Context) (*MealRating, error)
```

---

## Domain Structure

### Organizing Domains

Each domain (meals, weight, goals, etc.) follows the same structure:

```
domain/
├── models.go          # Data models and DTOs
├── errors.go          # Domain-specific errors (optional)
├── repository.go      # Repository interface
├── repository_postgres.go  # PostgreSQL implementation
├── repository_test.go # Repository tests
├── service.go         # Business logic
├── service_test.go    # Service tests
├── handler.go         # HTTP handlers
├── handler_test.go    # Handler tests
└── README.md          # Domain documentation (optional)
```

### Cross-Domain Dependencies

Services can depend on other repositories:

```go
type RatingService struct {
    repo     RatingRepository
    mealRepo meals.Repository  // Cross-domain dependency
}
```

Avoid circular dependencies:
- ✓ ratings → meals (OK)
- ✗ meals → ratings → meals (circular, not OK)

---

## Common Patterns

### Repository Pattern

```go
type Repository interface {
    Create(ctx context.Context, entity *Entity) error
    GetByID(ctx context.Context, id uuid.UUID) (*Entity, error)
    Update(ctx context.Context, entity *Entity) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filters Filters) ([]Entity, int, error)
}
```

### Service Pattern

```go
type Service interface {
    CreateEntity(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Entity, error)
    GetEntity(ctx context.Context, userID, entityID uuid.UUID) (*Entity, error)
    UpdateEntity(ctx context.Context, userID, entityID uuid.UUID, req UpdateRequest) (*Entity, error)
    DeleteEntity(ctx context.Context, userID, entityID uuid.UUID) error
}
```

### Handler Pattern

```go
type Handler struct {
    service Service
}

func (h *Handler) CreateEntity(w http.ResponseWriter, r *http.Request) {
    // 1. Extract user ID from context
    userID := getUserIDFromContext(r.Context())

    // 2. Decode request
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // 3. Call service
    entity, err := h.service.CreateEntity(r.Context(), userID, req)
    if err != nil {
        handleServiceError(w, err)
        return
    }

    // 4. Return response
    respondJSON(w, http.StatusCreated, entity)
}
```

### Validation Pattern

```go
// Validate returns validation error if invalid
func (r *CreateRatingRequest) Validate() error {
    valErr := errors.Validation("Invalid rating data")

    if r.Rating < 1 || r.Rating > 5 {
        valErr.AddField("rating", "must be between 1 and 5")
    }

    if r.Taste != nil && (*r.Taste < 1 || *r.Taste > 5) {
        valErr.AddField("taste", "must be between 1 and 5")
    }

    if valErr.HasErrors() {
        return valErr
    }
    return nil
}
```

---

## Database Migrations

### Creating Migrations

1. Create file: `migrations/00X_feature_name.sql`
2. Use incremental numbering (006, 007, etc.)
3. Include rollback instructions in comments

```sql
-- Migration: Add meal ratings
-- Created: 2025-11-15
-- Rollback: DROP TABLE meal_ratings CASCADE;

CREATE TABLE meal_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- ... columns ...
);

-- Always include indexes for foreign keys
CREATE INDEX idx_meal_ratings_meal_id ON meal_ratings(meal_id);

-- Always enable RLS
ALTER TABLE meal_ratings ENABLE ROW LEVEL SECURITY;

-- Always add RLS policies
CREATE POLICY "Users can view own ratings"
ON meal_ratings FOR SELECT
TO authenticated
USING (user_id = auth.uid());
```

### Testing Migrations

```bash
# Apply migration
psql $DATABASE_URL -f migrations/006_meal_ratings.sql

# Verify tables created
psql $DATABASE_URL -c "\dt"

# Test RLS (should only return own data)
psql $DATABASE_URL -c "SELECT * FROM meal_ratings;"
```

---

## AI Integration

### Using AI Services

```go
import "github.com/pradord/lumen_final/backend/internal/services/ai"

// In service
type MealService struct {
    repo        Repository
    aiCoordinator ai.Coordinator
}

func (s *MealService) ParseMeal(ctx context.Context, description string) (*ParseResult, error) {
    // AI coordinator handles provider selection and failover
    result, err := s.aiCoordinator.ParseNutrition(ctx, ai.ParseRequest{
        Description: description,
        UserID:      userID.String(),
    })

    if err != nil {
        return nil, errors.Wrap(err, "AI parsing failed")
    }

    return result, nil
}
```

### Adding New AI Provider

1. Implement `ai.Provider` interface
2. Add configuration in `config.go`
3. Register in coordinator
4. Add tests

---

## Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=debug go run cmd/api/main.go
```

### Use Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug cmd/api/main.go

# Set breakpoint
(dlv) break internal/domain/nutrition/meals/service.go:42
(dlv) continue
```

### Common Debug Techniques

**Print debugging**:
```go
log.Printf("DEBUG: userID=%s, mealID=%s", userID, mealID)
```

**Error inspection**:
```go
if err != nil {
    log.Printf("Error type: %T, value: %+v", err, err)
}
```

**SQL query logging**:
```bash
LOG_LEVEL=debug  # Logs all SQL queries
```

---

## Best Practices

### DO

- ✓ Write tests for all new code
- ✓ Use typed errors from `internal/errors`
- ✓ Pass context as first parameter
- ✓ Document all exported functions
- ✓ Keep files under 500 lines
- ✓ Use table-driven tests
- ✓ Validate input at service layer
- ✓ Return errors, don't panic

### DON'T

- ✗ Hardcode configuration values
- ✗ Skip error handling
- ✗ Use generic error messages
- ✗ Put business logic in handlers
- ✗ Skip writing tests
- ✗ Commit `.env` file
- ✗ Use `panic` in production code
- ✗ Ignore linter warnings

---

**Last Updated**: 2025-11-15
