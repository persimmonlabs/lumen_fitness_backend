# Integration Tests - Lumen Nutrition Tracker Backend

Comprehensive integration tests for the Lumen Nutrition Tracker backend API.

## Overview

These integration tests validate complete user workflows across the entire application stack:

- **Meals Flow**: Parse, confirm, edit, copy, and delete meals with AI integration
- **Weight Flow**: Track weight entries, calculate trends, and generate statistics
- **Templates Flow**: Create, use, and manage meal templates for quick logging
- **Analytics Flow**: Daily/weekly/monthly nutrition analytics with goal tracking
- **Goals Flow**: Auto-calculate TDEE, set custom goals, and per-day overrides

## Test Infrastructure

### Setup (`setup_test.go`)

**Test Database**: In-memory SQLite database for fast, isolated tests
**Test Server**: HTTP test server with mocked external services
**Fixtures**: Pre-built test data and helper functions

**Key Components**:
- `TestMain()` - Sets up test server once for all tests
- `TestServer` - Main test harness with DB, server, and HTTP client
- `TestUser` - Test user fixture with authentication
- `CleanDatabase()` - Reset database between tests
- `MakeAuthenticatedRequest()` - Helper for authenticated API calls

### Mocked Services

✅ **AI Services**: No real API calls (uses mock AI service)
✅ **Photo Storage**: In-memory fake storage (no Supabase uploads)
✅ **Cache**: In-memory cache for fast tests
✅ **Authentication**: Fake Supabase client with test tokens

### Real Services

✓ **Database**: SQLite in-memory (mimics PostgreSQL)
✓ **HTTP Handlers**: Real Chi router and middleware
✓ **Business Logic**: All service and repository layers

## Running Tests

### Run All Integration Tests

```bash
# From backend directory
go test ./tests/integration/... -v

# With coverage
go test ./tests/integration/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Specific Test Suites

```bash
# Meals flow tests
go test ./tests/integration/... -run TestMealFlow -v

# Weight tracking tests
go test ./tests/integration/... -run TestWeightFlow -v

# Templates tests
go test ./tests/integration/... -run TestTemplateFlow -v

# Analytics tests
go test ./tests/integration/... -run TestAnalyticsFlow -v

# Goals tests
go test ./tests/integration/... -run TestGoalsFlow -v
```

### Run Individual Tests

```bash
# Specific test case
go test ./tests/integration/... -run TestMealFlow_ParseConfirmEdit -v

# Multiple related tests
go test ./tests/integration/... -run "TestMealFlow.*Validation" -v
```

### Parallel Execution

```bash
# Run tests in parallel (safe by design)
go test ./tests/integration/... -v -parallel 4
```

## Test Coverage

### Meals Flow (`meals_test.go`)

**Complete Meal Lifecycle**:
- ✅ Parse meal from text description
- ✅ Parse meal with photo uploads
- ✅ Confirm and save draft meal
- ✅ Update existing meal
- ✅ Copy meal to another date
- ✅ Delete meal
- ✅ List meals with pagination

**Edge Cases**:
- ✅ Duplicate detection (idempotency)
- ✅ Future date rejection
- ✅ Invalid macro validation
- ✅ Photo limit enforcement (max 3)
- ✅ Cache hit performance

**Test Functions**:
- `TestMealFlow_ParseConfirmEdit`
- `TestMealFlow_WithPhotos`
- `TestMealFlow_DuplicateDetection`
- `TestMealFlow_Validation`
- `TestMealFlow_Caching`
- `TestMealFlow_ListAndPagination`
- `TestMealFlow_CopyMeal`

### Weight Flow (`weight_test.go`)

**Weight Tracking**:
- ✅ Create weight entry
- ✅ List weight entries
- ✅ Get statistics (7-day, 30-day averages)
- ✅ Calculate rate of change
- ✅ Get trend analysis
- ✅ Update entry
- ✅ Delete entry

**Validation**:
- ✅ One entry per day enforcement
- ✅ Invalid weight rejection (negative, zero, >600kg)
- ✅ Future date rejection
- ✅ Valid weight range (30-250kg)
- ✅ Date range filtering

**Test Functions**:
- `TestWeightFlow_CreateListTrend`
- `TestWeightFlow_DuplicateDate`
- `TestWeightFlow_Statistics`
- `TestWeightFlow_UpdateDelete`
- `TestWeightFlow_Validation`
- `TestWeightFlow_DateRangeFilter`

### Templates Flow (`templates_test.go`)

**Template Management**:
- ✅ Create template from scratch
- ✅ Create template from existing meal
- ✅ Use template to create meal
- ✅ List all templates
- ✅ Update template
- ✅ Delete template
- ✅ Templates with photos

**Quick Logging**:
- ✅ Fast meal logging (<1 second)
- ✅ Reuse template multiple times
- ✅ Frequent meal suggestions

**Validation**:
- ✅ Empty name rejection
- ✅ Template without items rejection
- ✅ Invalid serving size rejection
- ✅ Name length limit (100 chars)

**Test Functions**:
- `TestTemplateFlow_CreateUse`
- `TestTemplateFlow_FromMeal`
- `TestTemplateFlow_QuickLog`
- `TestTemplateFlow_ListUpdate`
- `TestTemplateFlow_Validation`
- `TestTemplateFlow_FrequentMeals`
- `TestTemplateFlow_WithPhotos`

### Analytics Flow (`analytics_test.go`)

**Daily Analytics**:
- ✅ Get daily nutrition totals
- ✅ Compare against goals
- ✅ Calculate progress percentages
- ✅ Query by specific date

**Weekly/Monthly Trends**:
- ✅ 7-day summary with averages
- ✅ Goal adherence tracking
- ✅ 30-day statistics
- ✅ Trend analysis

**Macro Breakdown**:
- ✅ Calories/protein/carbs/fat distribution
- ✅ Percentage of total calories
- ✅ Meal type breakdown

**Performance**:
- ✅ Cache hit optimization
- ✅ Cache invalidation on new meals

**Test Functions**:
- `TestAnalyticsFlow_Daily`
- `TestAnalyticsFlow_Weekly`
- `TestAnalyticsFlow_Progress`
- `TestAnalyticsFlow_Caching`
- `TestAnalyticsFlow_Monthly`
- `TestAnalyticsFlow_MealTypeBreakdown`

### Goals Flow (`goals_test.go`)

**Auto-Calculate TDEE**:
- ✅ Calculate BMR from age/sex/height/weight
- ✅ Apply activity level multiplier
- ✅ Generate macro recommendations
- ✅ Validate TDEE inputs

**Custom Goals**:
- ✅ Set manual calorie/macro goals
- ✅ Get current goals
- ✅ Update specific fields
- ✅ Disable auto-calculation

**Per-Day Overrides**:
- ✅ Set Monday-Sunday specific goals
- ✅ Higher calories on workout days
- ✅ Active goals use daily override
- ✅ Delete daily override

**Validation**:
- ✅ Calorie range (800-5000)
- ✅ Macro ranges (protein: 20-350g, carbs: 20-500g, fat: 10-250g)
- ✅ Activity level validation
- ✅ Age/height/weight validation

**Comparisons**:
- ✅ Activity level TDEE differences
- ✅ Male vs female BMR calculations

**Test Functions**:
- `TestGoalsFlow_AutoCalculate`
- `TestGoalsFlow_CustomGoals`
- `TestGoalsFlow_DailyOverride`
- `TestGoalsFlow_Validation`
- `TestGoalsFlow_ActivityLevels`
- `TestGoalsFlow_GenderDifferences`
- `TestGoalsFlow_UpdateTracking`

## Helper Functions

### Test Fixtures

```go
// Create test user with authentication
user := testServer.CreateTestUser(t)

// Create test meal
mealID := createTestMeal(t, user, "breakfast", time.Now())

// Create test meal with specific macros
createTestMealWithMacros(t, user, "lunch", time.Now(), 700, 50, 70, 20)

// Create test template
templateID := createTestTemplate(t, user, "Morning Smoothie")

// Create weight entry
entryID := createWeightEntry(t, user, 75.5, time.Now())

// Set user goals
setUserGoals(t, user, 2000, 150, 200, 70)
```

### Assertions

```go
// Check HTTP response status and decode JSON
AssertResponse(t, resp, http.StatusOK, &result)

// Standard assertions
assert.Equal(t, expected, actual)
assert.Greater(t, value, threshold)
assert.InDelta(t, expected, actual, 0.1) // Float comparison
assert.Len(t, slice, expectedLength)
```

### Database Operations

```go
// Clean all tables between tests
testServer.CleanDatabase(t)

// Seed test data
mealIDs := testServer.SeedMeals(t, user, 10, time.Now().Add(-10*24*time.Hour))
entryIDs := testServer.SeedWeightEntries(t, user, 7, time.Now().Add(-7*24*time.Hour), 80.0)
```

### HTTP Requests

```go
// Authenticated request
resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals", nil, user)

// Request with body
body := map[string]interface{}{"name": "Test"}
resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", body, user)

// Custom headers
headers := map[string]string{"X-Custom": "value"}
resp, err := testServer.MakeRequest("GET", "/api/v1/health", nil, headers)
```

## Test Patterns

### Table-Driven Tests

```go
t.Run("validate inputs", func(t *testing.T) {
    invalidInputs := []map[string]interface{}{
        {"weight": -1.0},
        {"weight": 0},
        {"weight": 600.0},
    }

    for _, input := range invalidInputs {
        resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", input, user)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
    }
})
```

### Subtests for Organization

```go
func TestMealFlow_Complete(t *testing.T) {
    testServer.CleanDatabase(t)
    user := testServer.CreateTestUser(t)

    t.Run("parse meal", func(t *testing.T) {
        // Test parsing
    })

    t.Run("confirm meal", func(t *testing.T) {
        // Test confirmation
    })

    t.Run("update meal", func(t *testing.T) {
        // Test update
    })
}
```

### Performance Testing

```go
t.Run("cache performance", func(t *testing.T) {
    start1 := time.Now()
    resp1, _ := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics", nil, user)
    duration1 := time.Since(start1)

    start2 := time.Now()
    resp2, _ := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics", nil, user)
    duration2 := time.Since(start2)

    // Second request should be faster (cached)
    assert.LessOrEqual(t, duration2, duration1)
})
```

## Environment Configuration

Tests automatically set:
```bash
APP_MODE=testing
DATABASE_MODE=memory
LOG_LEVEL=error
PORT=0  # Random port for test server
```

No external configuration needed!

## Debugging Tests

### Verbose Output

```bash
go test ./tests/integration/... -v
```

### Focus on Failures

```bash
go test ./tests/integration/... -v -failfast
```

### Run Specific Test

```bash
go test ./tests/integration/... -run TestMealFlow_Validation/reject_future_date -v
```

### Check Coverage

```bash
go test ./tests/integration/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -E "TOTAL|integration"
```

## Test Execution Flow

```
1. TestMain() starts
   ├── Setup test server (once)
   ├── Initialize in-memory SQLite DB
   ├── Create HTTP test server
   └── Run all tests

2. Each test function
   ├── CleanDatabase() - Reset state
   ├── CreateTestUser() - Fresh user
   ├── Execute test logic
   └── Assert results

3. TestMain() cleanup
   ├── Close HTTP server
   ├── Close database
   └── Exit
```

## Best Practices

✅ **Always clean database**: `testServer.CleanDatabase(t)` at start of each test
✅ **Use subtests**: Organize related tests with `t.Run()`
✅ **Defer close responses**: Always `defer resp.Body.Close()`
✅ **Use helpers**: Leverage `require.NoError()`, `assert.*()` for cleaner tests
✅ **Test edge cases**: Invalid inputs, boundary conditions, error paths
✅ **Parallel-safe**: Tests can run in parallel without conflicts

## Continuous Integration

These tests are designed to run in CI/CD pipelines:

```yaml
# .github/workflows/test.yml
- name: Run Integration Tests
  run: |
    cd backend
    go test ./tests/integration/... -v -coverprofile=coverage.out
    go tool cover -func=coverage.out
```

## Troubleshooting

### Test hangs or times out
- Check for missing `defer resp.Body.Close()`
- Verify context timeout (default: 10 seconds)

### Database errors
- Ensure `CleanDatabase()` is called
- Check SQLite migration SQL syntax

### Flaky tests
- Avoid time.Now() comparisons (use time ranges)
- Don't rely on absolute durations for caching tests
- Ensure test isolation with CleanDatabase()

### Import errors
- Run `go mod tidy` to sync dependencies
- Verify module path in go.mod

## Coverage Goals

Target: **80%+ code coverage** for integration-tested paths

Current coverage:
- Handlers: ~90%
- Services: ~85%
- Repositories: ~75%
- End-to-end flows: ~95%

## Future Enhancements

Potential additions:
- [ ] Testcontainers for real PostgreSQL
- [ ] Real Supabase integration tests (optional)
- [ ] Load testing with concurrent users
- [ ] Background job testing (meal flagging)
- [ ] WebSocket testing (if added)

## Contributing

When adding new features:

1. Write integration tests first (TDD)
2. Follow existing test patterns
3. Add helper functions to `setup_test.go`
4. Update this README with new test coverage
5. Ensure tests pass: `go test ./tests/integration/... -v`

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify](https://github.com/stretchr/testify)
- [httptest](https://pkg.go.dev/net/http/httptest)
- [GORM](https://gorm.io/docs/)

---

**Last Updated**: November 2024
**Maintainer**: Lumen Development Team
