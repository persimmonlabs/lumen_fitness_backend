# Nutrition Integration Tests - Test Coverage Documentation

**Last Updated:** 2025-11-16
**Location:** `backend/internal/domain/nutrition/meals/*_test.go`

## Overview

This document describes the comprehensive integration test suite that verifies nutrition totals are always correct across all layers of the application. These tests ensure the **Single Source of Truth** architecture (migration 012) works correctly.

## Test Files

### 1. trigger_integration_test.go

**Purpose:** Verifies database triggers automatically calculate meal totals when meal_items change.

**Test Coverage:**

#### TestDatabaseTriggers_InsertItems
- ✅ Single item insert triggers total calculation
- ✅ Multiple items insert triggers sum calculation
- ✅ Empty meal has zero totals
- ✅ COALESCE handles NULL from SUM correctly

**Edge Cases:**
- Very small fractional values (0.2g protein)
- Many items with precision (10+ items)
- Zero-calorie items (water, black coffee)

#### TestDatabaseTriggers_UpdateItems
- ✅ Updating item recalculates meal totals
- ✅ Updating one item preserves other items in sum
- ✅ Updated values reflected immediately

#### TestDatabaseTriggers_DeleteItems
- ✅ Deleting item decreases meal totals
- ✅ Deleting all items sets totals to zero
- ✅ Remaining items still summed correctly

**Key Assertions:**
- Totals match SUM of all items (within 0.01 tolerance)
- Database triggers fire on INSERT, UPDATE, DELETE
- COALESCE returns 0 for meals with no items

---

### 2. api_integration_test.go

**Purpose:** Verifies API contracts return correct database-calculated totals.

**Test Coverage:**

#### TestAPI_CreateMeal
- ✅ POST /api/v1/meals returns auto-calculated totals
- ✅ Client-provided totals are ignored (database overrides)
- ✅ Response includes all items

#### TestAPI_UpdateMeal
- ✅ PUT /api/v1/meals/:id recalculates totals from new items
- ✅ Updating with fewer items adjusts totals down
- ✅ Old items completely replaced

#### TestAPI_GetMeal
- ✅ GET /api/v1/meals/:id returns database totals
- ✅ Manually inserted items reflected in totals

#### TestAPI_ListMeals
- ✅ GET /api/v1/meals shows correct totals for all meals
- ✅ Item count accurate
- ✅ Pagination preserves accuracy

#### TestAPI_DailyAnalytics
- ✅ Daily totals aggregate from all meals
- ✅ Deleted meals excluded from analytics
- ✅ SUM query matches expected values

**Key Assertions:**
- API responses contain database-calculated totals
- Client cannot override totals
- All CRUD operations maintain accuracy

---

### 3. macro_validation_test.go

**Purpose:** Verifies nutrition math accuracy using Atwater factors.

**Test Coverage:**

#### TestMacroMath_Calculation
- ✅ Standard meal: (P×4) + (C×4) + (F×9)
- ✅ High protein meals
- ✅ High fat meals
- ✅ Zero fat (pure protein/carbs)
- ✅ Fractional values (12.5g protein)
- ✅ Very small values (0.5g)

#### TestMacroMath_Tolerance
- ✅ Exact match (100 = 100)
- ✅ Within upper bound (109 vs 100 ±10%)
- ✅ Within lower bound (91 vs 100 ±10%)
- ✅ At limits (110, 90)
- ✅ Exceeds bounds (111, 89) → FAIL
- ✅ Small values skip validation (< 5 kcal)

#### TestMacroMath_ValidateMacroCalories
- ✅ Perfect match validates
- ✅ ±5% validates (acceptable rounding)
- ✅ ±10% validates (tolerance limit)
- ✅ ±15% fails validation
- ✅ Nutrition label rounding acceptable

#### TestMacroMath_DatabaseIntegration
- ✅ Real USDA food data validates
- ✅ Chicken breast (165 kcal)
- ✅ Brown rice (112 kcal)
- ✅ Banana, avocado, Greek yogurt
- ✅ Invalid data detected (1000 kcal vs 125 calculated)

**Validation Rules:**
- **Tolerance:** ±10% (MacroCalorieTolerancePercent = 0.10)
- **Formula:** Calories = (Protein × 4) + (Carbs × 4) + (Fat × 9)
- **Skip Threshold:** < 5 kcal (MinimumCaloriesForValidation)
- **Constants:** `internal/domain/nutrition/constants/macros.go`

#### TestMacroMath_EdgeCases
- ✅ Zero calories with zero macros
- ✅ Very large values (99,999.9g)
- ✅ Negative values (math works, DB should prevent)
- ✅ Alcohol calories NOT in base formula (expected behavior)

---

## Test Infrastructure

### Setup & Cleanup

**File:** `integration_test.go`

**TestMain:**
- Connects to test database via `SUPABASE_TEST_DATABASE_URL`
- Sets connection pool limits (5 max open, 2 idle)
- Initializes repository
- Runs all tests
- Skips if no test database configured

**setupTestFixtures:**
- Creates unique test user (`auth.users`)
- Generates UUID for user_id
- Tracks created meals for cleanup
- Provides setupTime for consistent timestamps

**cleanupTestFixtures:**
- Deletes meal_items (child records first)
- Deletes meals
- Deletes test user
- Prevents test data leakage

---

## Running Tests

### All Integration Tests
```bash
cd backend
go test -v ./internal/domain/nutrition/meals -run "^Test" -timeout 60s
```

### Specific Test Suite
```bash
# Database triggers only
go test -v ./internal/domain/nutrition/meals -run "^TestDatabaseTriggers" -timeout 30s

# API contracts only
go test -v ./internal/domain/nutrition/meals -run "^TestAPI" -timeout 30s

# Macro validation only
go test -v ./internal/domain/nutrition/meals -run "^TestMacroMath" -timeout 10s
```

### Unit Tests (No Database)
```bash
go test -v ./internal/domain/nutrition/meals -run "^TestMacroMath_Calculation$" -short
```

### Skip Integration Tests
```bash
go test -v ./internal/domain/nutrition/meals -short
```

---

## Test Environment Variables

**Required:**
```bash
export SUPABASE_TEST_DATABASE_URL="postgresql://postgres:password@localhost:5432/test_db"
```

**If not set:**
- Tests print: "Skipping integration tests: SUPABASE_TEST_DATABASE_URL not set"
- Exit code 0 (not a failure)

---

## Test Assertions & Tolerances

### Exact Equality
```go
assert.Equal(t, 105.0, meal.TotalCalories)  // Exact match
```

### Float Comparison
```go
assert.InDelta(t, 308.0, meal.TotalCalories, 0.01)  // ±0.01 tolerance
```

### Macro Validation
```go
isValid := constants.ValidateMacroCalories(stated, protein, carbs, fat)
assert.True(t, isValid)  // Within ±10% tolerance
```

---

## Coverage Metrics

### Lines of Test Code
- **trigger_integration_test.go:** ~500 lines
- **api_integration_test.go:** ~600 lines
- **macro_validation_test.go:** ~520 lines
- **Total:** ~1,620 lines of integration tests

### Test Scenarios
- **Database Trigger Tests:** 12 scenarios
- **API Contract Tests:** 9 scenarios
- **Macro Validation Tests:** 9 scenarios
- **Total:** 30+ test scenarios

### Edge Cases Covered
- ✅ Empty meals (0 items)
- ✅ Single item meals
- ✅ Many items (10+)
- ✅ Very small values (< 1.0)
- ✅ Zero-calorie items
- ✅ Fractional quantities (0.5 fruit)
- ✅ Concurrent operations
- ✅ Invalid nutrition data
- ✅ USDA real-world data
- ✅ Rounding on nutrition labels

---

## Critical Test Validations

### 1. Single Source of Truth
**Verified:** Database triggers are the ONLY calculation point.

```go
// ✅ Test passes: Client sends wrong totals, DB overrides
meal.TotalCalories = 999.0  // Wrong value
result := CreateMealWithItems(meal, items)
assert.Equal(t, 350.0, result.TotalCalories)  // DB calculated correctly
```

### 2. Macro Math Accuracy
**Verified:** All calculations use Atwater factors from constants.

```go
calculated := constants.CalculateMacroCalories(protein, carbs, fat)
// Formula: (protein × 4) + (carbs × 4) + (fat × 9)
```

### 3. Data Integrity
**Verified:** Totals always equal SUM of items.

```go
// After INSERT/UPDATE/DELETE of meal_items
SELECT total_calories, (SELECT SUM(calories) FROM meal_items WHERE meal_id = $1)
// These MUST match (within 0.01 tolerance)
```

---

## Test Data Examples

### Valid Meal (Passes Validation)
```go
Chicken Breast: 165 kcal
  Protein: 31g × 4 = 124 kcal
  Carbs:    0g × 4 =   0 kcal
  Fat:    3.6g × 9 =  32.4 kcal
  Calculated: 156.4 kcal
  Stated: 165 kcal
  Difference: 5.2% ✅ (within ±10%)
```

### Invalid Data (Fails Validation)
```go
Invalid Item: 1000 kcal (claimed)
  Protein: 10g × 4 = 40 kcal
  Carbs:   10g × 4 = 40 kcal
  Fat:      5g × 9 = 45 kcal
  Calculated: 125 kcal
  Difference: 700% ❌ (exceeds ±10%)
```

---

## Dependencies

### Go Packages
- `testing` - Test framework
- `github.com/stretchr/testify/assert` - Assertions
- `github.com/stretchr/testify/require` - Fatal assertions
- `github.com/jmoiron/sqlx` - Database access
- `github.com/google/uuid` - UUID generation

### Database Requirements
- PostgreSQL with Supabase schema
- Migration 012 applied (database triggers)
- `auth.users` table
- `meals` and `meal_items` tables

### Internal Packages
- `backend/internal/domain/nutrition/meals` - Domain models
- `backend/internal/domain/nutrition/constants` - Atwater factors

---

## Troubleshooting

### Tests Fail with "connection refused"
```bash
# Check database is running
psql $SUPABASE_TEST_DATABASE_URL -c "SELECT 1"

# Verify migrations applied
psql $SUPABASE_TEST_DATABASE_URL -c "\dt"  # Should see meals, meal_items
```

### Tests Fail with "triggers not found"
```bash
# Verify migration 012 applied
psql $SUPABASE_TEST_DATABASE_URL -c "\df calculate_meal_totals"
```

### Totals Don't Match
```sql
-- Check if triggers are enabled
SELECT tgname, tgenabled FROM pg_trigger WHERE tgname LIKE 'meal_items%';
-- tgenabled should be 'O' (enabled)
```

---

## Future Test Additions

### Planned Coverage
- [ ] Concurrent meal updates (stress test)
- [ ] Very large meals (1000+ items)
- [ ] Performance benchmarks (1M meals)
- [ ] Transaction rollback scenarios
- [ ] Trigger failure recovery
- [ ] Database replication lag

### Integration with Other Services
- [ ] Analytics service integration
- [ ] Goals service integration
- [ ] Templates service integration

---

## Related Documentation

- **Trigger Implementation:** `backend/migrations/012_establish_single_source_truth.up.sql`
- **Macro Constants:** `backend/internal/domain/nutrition/constants/macros.go`
- **Naming Standards:** `backend/docs/naming-standards.md`
- **Architecture:** `backend/docs/ARCHITECTURE-SINGLE-SOURCE-TRUTH.md`
- **API Contracts:** `backend/docs/API-NUTRITION-CONTRACTS.md`

---

## Maintenance

**When to Update These Tests:**
- ✏️ Adding new nutrition fields (e.g., sugar, sodium)
- ✏️ Changing database schema
- ✏️ Modifying trigger logic
- ✏️ Updating Atwater factors
- ✏️ Adding new API endpoints

**Test Review Checklist:**
- [ ] All tests pass locally
- [ ] All tests pass in CI
- [ ] Coverage > 80%
- [ ] No flaky tests
- [ ] Edge cases documented
- [ ] Test data realistic (USDA values)

---

**Maintained by:** Backend Team
**Questions:** See `backend/CLAUDE.md` for code review guidelines
