# Lumen Nutrition Tracker - Naming Standards

**Version:** 1.0
**Date:** 2025-11-16
**Status:** ✅ DEFINITIVE - All code MUST follow these standards

---

## Executive Summary

This document establishes the SINGLE authoritative naming convention for all nutrition-related data across database, backend, and frontend layers. Every decision is final and must be applied consistently everywhere.

---

## Database Layer (PostgreSQL)

### Table: `meals`

| Column Name | Type | Rationale |
|-------------|------|-----------|
| `consumed_at` | TIMESTAMPTZ | REST/Rails convention; clearer than `meal_time` or `eaten_at` |
| `meal_type` | TEXT | Enum constraint (breakfast/lunch/dinner/snack) |
| `total_calories` | DECIMAL(7,1) | No suffix - kilocalories (kcal) is implicit standard |
| `total_protein_g` | DECIMAL(6,1) | Suffix `_g` explicitly indicates grams (avoids ambiguity) |
| `total_carbs_g` | DECIMAL(6,1) | Suffix `_g` for consistency with protein |
| `total_fat_g` | DECIMAL(6,1) | Suffix `_g` for consistency |
| `total_fiber_g` | DECIMAL(5,1) | Suffix `_g` for consistency (nullable) |

**Decision:** `meal_type` replaces legacy `name` column. Drop `name` column entirely.

### Table: `meal_items`

| Column Name | Type | Rationale |
|-------------|------|-----------|
| `name` | TEXT | Simple, RESTful; renamed from `food_name` for consistency |
| `quantity` | DECIMAL(8,2) | Neutral - works with any unit |
| `unit` | TEXT | Stores "g", "oz", "serving", "cup", etc. |
| `calories` | DECIMAL(7,1) | No suffix - kcal implicit |
| `protein` | DECIMAL(6,1) | No suffix - grams implicit from context (NOT `protein_g`) |
| `carbs` | DECIMAL(6,1) | No suffix - grams implicit |
| `fat` | DECIMAL(6,1) | No suffix - grams implicit |
| `fiber` | DECIMAL(5,1) | No suffix - grams implicit (nullable) |

**Rationale for NO `_g` suffix on items:**
- Individual items have `unit` column providing context
- Meals aggregate totals need explicit `_g` suffix for clarity
- Reduces verbosity while maintaining clarity

### Table: `template_items`

Same naming as `meal_items` - uses `name` and nutrition columns without `_g` suffix.

---

## Backend Layer (Go)

### Struct Field Naming Convention

```go
type Meal struct {
    ConsumedAt    time.Time `json:"consumed_at" db:"consumed_at"`
    MealType      string    `json:"meal_type" db:"meal_type"`
    TotalCalories float64   `json:"total_calories" db:"total_calories"`
    TotalProteinG float64   `json:"total_protein_g" db:"total_protein_g"`  // Note: _g in JSON and DB
    TotalCarbsG   float64   `json:"total_carbs_g" db:"total_carbs_g"`
    TotalFatG     float64   `json:"total_fat_g" db:"total_fat_g"`
    TotalFiberG   float64   `json:"total_fiber_g" db:"total_fiber_g"`
}

type MealItem struct {
    Name      string  `json:"name" db:"name"`  // Matches DB column
    Quantity  float64 `json:"quantity" db:"quantity"`
    Unit      string  `json:"unit" db:"unit"`
    Calories  float64 `json:"calories" db:"calories"`
    ProteinG  float64 `json:"protein_g" db:"protein"`  // ⚠️ JSON has _g, DB does NOT
    CarbsG    float64 `json:"carbs_g" db:"carbs"`      // ⚠️ JSON has _g, DB does NOT
    FatG      float64 `json:"fat_g" db:"fat"`          // ⚠️ JSON has _g, DB does NOT
    FiberG    float64 `json:"fiber_g" db:"fiber"`      // ⚠️ JSON has _g, DB does NOT
}
```

**Key Rule:** JSON tags have `_g` suffix for clarity, DB tags match EXACT column names.

### Macro Calculation Constants

**File:** `backend/internal/domain/nutrition/constants/macros.go` (NEW)

```go
package constants

// MacroCalorieCoefficients define the standard Atwater factors for
// macronutrient energy conversion. These are the scientifically
// accepted values used in nutrition labeling worldwide.
const (
    // CaloriesPerGramProtein is the Atwater factor for protein (4 kcal/g)
    CaloriesPerGramProtein = 4.0

    // CaloriesPerGramCarbs is the Atwater factor for carbohydrates (4 kcal/g)
    CaloriesPerGramCarbs = 4.0

    // CaloriesPerGramFat is the Atwater factor for fat (9 kcal/g)
    CaloriesPerGramFat = 9.0

    // CaloriesPerGramAlcohol is the Atwater factor for alcohol (7 kcal/g)
    // Currently unused but defined for future alcohol tracking support
    CaloriesPerGramAlcohol = 7.0
)

// ValidationTolerances define acceptable variance ranges for data quality checks
const (
    // MacroCalorieTolerancePercent allows for ±10% variance between
    // calculated calories (from macros) and stated calories.
    // This accounts for:
    // - Rounding in nutrition labels
    // - Fiber energy availability variation (2-4 kcal/g)
    // - Measurement precision
    MacroCalorieTolerancePercent = 0.10

    // MinimumCaloriesForValidation sets the threshold below which
    // macro validation is skipped (very small portions have higher relative error)
    MinimumCaloriesForValidation = 5.0
)

// DisplayPrecision defines decimal places for user-facing nutrition values
const (
    // CalorieDisplayDecimals - show calories with 1 decimal place (e.g., 123.5)
    CalorieDisplayDecimals = 1

    // MacroDisplayDecimals - show macros with 1 decimal place (e.g., 12.3g)
    MacroDisplayDecimals = 1
)
```

**Rationale:**
- Atwater factors are scientific standards - never change
- 10% tolerance accounts for label rounding and fiber variation
- Constants centralized for easy testing and modification

---

## Frontend Layer (TypeScript)

### Interface Naming - Match Backend EXACTLY

**File:** `frontend/types/nutrition.ts` (NEW)

```typescript
/**
 * Nutrition type definitions matching backend API contracts
 *
 * CRITICAL: Field names MUST match backend JSON tags character-for-character.
 * DO NOT modify these without corresponding backend changes.
 *
 * Generated from: backend/internal/domain/nutrition/meals/models.go
 * Last synced: 2025-11-16
 */

export interface Meal {
  id: string
  user_id: string

  /** Meal category (breakfast/lunch/dinner/snack) */
  meal_type: 'breakfast' | 'lunch' | 'dinner' | 'snack'

  /** When the meal was consumed (ISO 8601 timestamp) */
  consumed_at: string  // NOT eaten_at, NOT meal_time

  /** Aggregated nutrition totals (auto-calculated by database triggers) */
  total_calories: number   // kcal
  total_protein_g: number  // grams (note the _g suffix)
  total_carbs_g: number    // grams
  total_fat_g: number      // grams
  total_fiber_g?: number   // grams (optional)

  /** Metadata */
  is_draft: boolean
  draft_status?: 'analyzing' | 'ready' | 'error'
  photos?: string[]
  notes?: string

  created_at: string
  updated_at: string
}

export interface MealItem {
  id: string
  meal_id: string

  /** Food name (NOT description, NOT food_name) */
  name: string

  /** Amount (NOT grams) - interpret using 'unit' */
  quantity: number

  /** Unit of measurement (g, oz, serving, cup, etc.) */
  unit: string

  /** Nutrition per serving */
  calories: number     // kcal
  protein_g: number    // grams (JSON has _g even though DB column is 'protein')
  carbs_g: number      // grams (JSON has _g even though DB column is 'carbs')
  fat_g: number        // grams (JSON has _g even though DB column is 'fat')
  fiber_g?: number     // grams (JSON has _g even though DB column is 'fiber')

  created_at: string
}

/**
 * Nutrition totals calculation result
 * Used internally for displaying aggregated nutrition
 */
export interface NutritionTotals {
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  fiber_g?: number
}

/**
 * Macro calculation constants (must match backend)
 */
export const MACRO_CONSTANTS = {
  CALORIES_PER_GRAM_PROTEIN: 4,
  CALORIES_PER_GRAM_CARBS: 4,
  CALORIES_PER_GRAM_FAT: 9,
  CALORIES_PER_GRAM_ALCOHOL: 7, // Future use
  TOLERANCE_PERCENT: 0.10,
} as const
```

**Key Rules:**
- NO client-side calculation of totals
- Display backend-provided values only
- Use `_g` suffix to match backend JSON
- Constants mirror backend for validation only

---

## Migration Strategy

### Phase 1: Schema Alignment (Migration 012)

```sql
-- Rename columns to match standards
DO $$
BEGIN
    -- Rename food_name to name in meal_items
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meal_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE meal_items RENAME COLUMN food_name TO name;
    END IF;

    -- Rename food_name to name in template_items
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='template_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE template_items RENAME COLUMN food_name TO name;
    END IF;

    -- Drop deprecated meal_time column (use consumed_at)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='meal_time'
    ) THEN
        ALTER TABLE meals DROP COLUMN meal_time;
    END IF;

    -- Drop deprecated name column (use meal_type)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='name'
    ) THEN
        ALTER TABLE meals DROP COLUMN name;
    END IF;
END $$;
```

**Safety:** Uses IF EXISTS checks for idempotency - safe to run multiple times.

---

## Consistency Rules

### Rule 1: Single Source of Truth for Totals
- **Totals calculated by:** Database triggers ONLY
- **Backend services:** NEVER calculate totals manually
- **Frontend components:** Display backend values, never recalculate

### Rule 2: Naming Suffix Pattern
- **Database `meals` table:** Use `_g` suffix for all macro totals
- **Database `meal_items` table:** NO `_g` suffix (grams implicit)
- **Go JSON tags:** ALWAYS use `_g` suffix for clarity
- **Go DB tags:** Match database column names EXACTLY
- **TypeScript:** Match Go JSON tags EXACTLY

### Rule 3: Time Field Naming
- **Always use:** `consumed_at` (past tense, clear meaning)
- **Never use:** `meal_time` (ambiguous), `eaten_at` (informal)

### Rule 4: Food Identifier Naming
- **Always use:** `name` (simple, RESTful)
- **Never use:** `food_name` (verbose), `description` (wrong semantic)

---

## Examples

### Correct Usage ✅

```go
// Backend service - NO manual calculation
func (s *service) UpdateMeal(ctx context.Context, req *UpdateMealRequest) (*Meal, error) {
    // Database trigger will calculate totals automatically
    meal := &Meal{
        MealType:   req.MealType,
        ConsumedAt: req.ConsumedAt,
        // NO TotalCalories = ... calculation here!
    }
    return s.repo.Update(ctx, meal, req.Items)
}

// Frontend component - use backend totals
function MealSummary({ meal }: { meal: Meal }) {
    return (
        <div>
            <p>Calories: {meal.total_calories}</p>  {/* From API */}
            <p>Protein: {meal.total_protein_g}g</p> {/* From API */}
        </div>
    )
    // ❌ DON'T: const totals = meal.items.reduce(...)
}
```

### Incorrect Usage ❌

```go
// ❌ WRONG: Manual calculation in service
func (s *service) CreateMeal(...) {
    totalCalories := 0.0
    for _, item := range items {
        totalCalories += item.Calories  // ❌ Don't do this!
    }
}

// ❌ WRONG: Hardcoded macro constants
calories := (protein * 4) + (carbs * 4) + (fat * 9)  // Use constants!

// ❌ WRONG: Wrong field names
type MealItem struct {
    FoodName string `json:"food_name"` // Should be "name"
    Grams    float64 `json:"grams"`     // Should be "quantity"
}
```

---

## Enforcement

1. **Code Review:** All PRs must follow these standards
2. **Linting:** Add custom linter rules to catch violations
3. **Tests:** Integration tests verify totals match across layers
4. **Documentation:** Link to this document in CLAUDE.md

---

## Changelog

- **2025-11-16:** Initial version - established definitive standards based on comprehensive codebase analysis

---

**Questions?** See `backend/docs/schema-ground-truth.md` and `backend/docs/calculation-inventory.md` for detailed analysis that led to these decisions.
