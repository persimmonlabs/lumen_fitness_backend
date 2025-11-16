# Nutrition Calculation Inventory

**Generated**: 2025-11-16
**Purpose**: Document all locations where nutrition totals are calculated across the codebase

## Executive Summary

- **Total Calculation Locations Found**: 11
- **Backend Calculations**: 7 locations
- **Frontend Calculations**: 2 locations
- **Database Calculations**: 2 locations (SQL SUM operations)
- **Naming Convention Issues**: Moderate inconsistency between database, Go, and TypeScript

---

## Backend Calculations

### Location 1: `meals/service.go:403-413`

**Purpose**: Calculate nutrition totals from DraftMealItem array
**Function**: `calculateTotals(items []DraftMealItem)`

**Formula**:
```go
func (s *service) calculateTotals(items []DraftMealItem) NutritionTotals {
    var totals NutritionTotals
    for _, item := range items {
        totals.Calories += item.Calories
        totals.ProteinG += item.ProteinG
        totals.CarbsG += item.CarbsG
        totals.FatG += item.FatG
        totals.FiberG += item.FiberG
    }
    return totals
}
```

**Used By**:
- `ParseMeal()` - Line 163
- `calculateTotalsFromDraft()` - Line 416 (wrapper)

**Naming Convention**:
- `Calories` (no suffix)
- `ProteinG`, `CarbsG`, `FatG`, `FiberG` (with `G` suffix for grams)

---

### Location 2: `meals/service.go:208-222`

**Purpose**: Calculate totals when confirming a meal
**Function**: `ConfirmMeal()`

**Formula**:
```go
// Line 208
totals := s.calculateTotalsFromDraft(req.Items)

// Line 211-222
meal := &Meal{
    TotalCalories: totals.Calories,
    TotalProteinG: totals.ProteinG,
    TotalCarbsG:   totals.CarbsG,
    TotalFatG:     totals.FatG,
    TotalFiberG:   totals.FiberG,
}
```

**Used By**: API endpoint `/meals/confirm`

**Naming Convention**:
- Meal struct: `TotalCalories`, `TotalProteinG`, `TotalCarbsG`, `TotalFatG`, `TotalFiberG`
- NutritionTotals: `Calories`, `ProteinG`, `CarbsG`, `FatG`, `FiberG`

---

### Location 3: `meals/service.go:309-325`

**Purpose**: Calculate totals when updating a meal
**Function**: `UpdateMeal()`

**Formula**:
```go
// Line 310
totals := s.calculateTotalsFromDraft(req.Items)

// Line 313-325
meal := &Meal{
    TotalCalories: totals.Calories,
    TotalProteinG: totals.ProteinG,
    TotalCarbsG:   totals.CarbsG,
    TotalFatG:     totals.FatG,
    TotalFiberG:   totals.FiberG,
}
```

**Used By**: API endpoint `/meals/{id}` (PUT)

---

### Location 4: `meals/service_draft.go:184-192`

**Purpose**: Calculate totals for draft meal status
**Function**: `GetDraftStatus()`

**Formula**:
```go
totals := &NutritionTotals{}
for _, item := range items {
    totals.Calories += item.Calories
    totals.ProteinG += item.ProteinG
    totals.CarbsG += item.CarbsG
    totals.FatG += item.FatG
    totals.FiberG += item.FiberG
}
response.Total = totals
```

**Used By**: API endpoint `/meals/draft/{id}/status`

**Note**: This is a DUPLICATE of calculateTotals logic but inline

---

### Location 5: `meals/repository.go:327`

**Purpose**: Calculate totals when querying meal items (appears to be legacy/unused)
**Function**: Database query loop

**Formula**:
```go
for rows.Next() {
    // ... scan row
    totalCalories += item.Calories
    // (only calories shown in grep, likely similar for other macros)
}
```

**Status**: Legacy code, should verify if this is still used

---

### Location 6: AI Services - `ai/groq.go:316`, `ai/openrouter.go:357`, `ai/mock.go:81`

**Purpose**: Aggregate nutrition from AI-parsed meal items
**Pattern**: IDENTICAL across all 3 AI service implementations

**Formula**:
```go
totalNutrition := NutritionData{}
for _, item := range parsed.Items {
    totalNutrition.Calories += item.Nutrition.Calories
    totalNutrition.Protein += item.Nutrition.Protein
    totalNutrition.Carbohydrates += item.Nutrition.Carbohydrates
    totalNutrition.Fat += item.Nutrition.Fat
    totalNutrition.Fiber += item.Nutrition.Fiber
    totalNutrition.Sugar += item.Nutrition.Sugar
    totalNutrition.Sodium += item.Nutrition.Sodium
}
```

**Naming Convention**:
- `Calories` (no suffix)
- `Protein` (no suffix, NOT `ProteinG`)
- `Carbohydrates` (NOT `CarbsG`)
- `Fat` (no suffix)
- `Fiber` (no suffix)
- Additional: `Sugar`, `Sodium`

**Files**:
- `backend/internal/services/ai/groq.go:316-323`
- `backend/internal/services/ai/openrouter.go:357-364`
- `backend/internal/services/ai/mock.go:81-88`

---

### Location 7: Analytics - `analytics/service.go:274-305`

**Purpose**: Calculate averages and totals for analytics
**Functions**: `calculateAverages()`, `calculateTotals()`

**Formula**:
```go
// calculateAverages (line 263-294)
for _, day := range dailyData {
    if day.TotalCalories > 0 {
        daysLogged++
        totalCalories += day.TotalCalories
        totalProtein += day.TotalProtein
        totalCarbs += day.TotalCarbs
        totalFat += day.TotalFat
        totalFiber += day.TotalFiber
    }
}
return Averages{
    AvgCalories: round(totalCalories / float64(daysLogged)),
    AvgProtein:  round(totalProtein / float64(daysLogged)),
    // ...
}

// calculateTotals (line 296-308)
for _, day := range dailyData {
    totals.TotalCalories += day.TotalCalories
    totals.TotalProtein += day.TotalProtein
    totals.TotalCarbs += day.TotalCarbs
    totals.TotalFat += day.TotalFat
    totals.TotalFiber += day.TotalFiber
}
```

**Naming Convention**:
- `TotalCalories`, `TotalProtein`, `TotalCarbs`, `TotalFat`, `TotalFiber`
- NO `_g` suffix on any fields

---

### Location 8: Analytics Repository - `analytics/repository.go:63-71`

**Purpose**: Aggregate nutrition from RPC function results
**Function**: `GetDailyNutrition()`

**Formula**:
```go
for rows.Next() {
    // ... scan meal data
    dailyTotals.TotalCalories += calories
    dailyTotals.TotalProtein += protein
    dailyTotals.TotalCarbs += carbs
    dailyTotals.TotalFat += fat
    dailyTotals.TotalFiber += fiber

    // Also aggregates per-meal breakdown
    meal.Calories += calories
    meal.Protein += protein
    // ...
}
```

**Naming Convention**:
- DailyTotals: `TotalCalories`, `TotalProtein`, `TotalCarbs`, `TotalFat`, `TotalFiber`
- MealBreakdown: `Calories`, `Protein`, `Carbs`, `Fat`, `Fiber`

---

## Frontend Calculations

### Location 9: `MealConfirm.tsx:37-45`

**Purpose**: Calculate totals for confirmation UI display
**Function**: `reduce()` operation on items array

**Formula**:
```typescript
const totals = items.reduce(
    (acc, item) => ({
        calories: acc.calories + item.calories,
        protein: acc.protein + item.protein_g,
        carbs: acc.carbs + item.carbs_g,
        fat: acc.fat + item.fat_g,
    }),
    { calories: 0, protein: 0, carbs: 0, fat: 0 }
)
```

**Naming Convention**:
- Item properties: `calories`, `protein_g`, `carbs_g`, `fat_g` (with `_g` suffix)
- Accumulator: `calories`, `protein`, `carbs`, `fat` (NO suffix)

**Note**: Inconsistency between item property names and accumulator names

---

### Location 10: `ManualMealForm.tsx:48-56`

**Purpose**: Calculate totals for manual meal entry form
**Function**: `reduce()` operation on items array

**Formula**:
```typescript
const totals: MealTotals = items.reduce(
    (acc, item) => ({
        calories: acc.calories + item.calories,
        protein_g: acc.protein_g + item.protein_g,
        carbs_g: acc.carbs_g + item.carbs_g,
        fat_g: acc.fat_g + item.fat_g,
    }),
    { calories: 0, protein_g: 0, carbs_g: 0, fat_g: 0 }
)
```

**Naming Convention**:
- ALL properties use `_g` suffix EXCEPT `calories`
- CONSISTENT with item properties
- Lines 150-153 send to API with same naming

---

## Database Calculations

### Location 11: SQL RPC Functions - `migrations/002_rpc_functions.up.sql`

**Purpose**: Database-side nutrition aggregation
**Functions**: Multiple RPC functions use SUM aggregates

**Formulas**:

**Line 114** (get_daily_nutrition):
```sql
'protein', COALESCE(SUM(mi.protein), 0)
```

**Line 177** (get_meal_by_id):
```sql
'protein', ROUND(SUM(mi.protein)::NUMERIC, 1)
```

**Line 222** (nested query):
```sql
'protein', COALESCE(SUM(mi2.protein), 0)
```

**Line 461** (analytics query):
```sql
ROUND(SUM(mi.protein)::NUMERIC, 1) AS protein
```

**Naming Convention**:
- Database column: `protein` (NO `_g` suffix in meal_items table)
- meals table: `total_protein_g` (WITH `_g` suffix)

---

## Macro Calorie Calculation

### Location 12: Analytics - `analytics/service.go:169-173`

**Purpose**: Calculate macro distribution percentages
**Function**: `GetMacroDistribution()`

**Formula**:
```go
// protein: 4 cal/g, carbs: 4 cal/g, fat: 9 cal/g
proteinCalories := dailyTotals.TotalProtein * 4
carbsCalories := dailyTotals.TotalCarbs * 4
fatCalories := dailyTotals.TotalFat * 9

totalMacroCalories := proteinCalories + carbsCalories + fatCalories
```

**Note**: This is the ONLY location using calorie conversion formula

---

## Naming Convention Analysis

### Go Struct Field Names

**Pattern 1**: Meal struct (database-mapped)
```go
type Meal struct {
    TotalCalories float64 `json:"total_calories" db:"total_calories"`
    TotalProteinG float64 `json:"total_protein_g" db:"total_protein_g"`
    TotalCarbsG   float64 `json:"total_carbs_g" db:"total_carbs_g"`
    TotalFatG     float64 `json:"total_fat_g" db:"total_fat_g"`
    TotalFiberG   float64 `json:"total_fiber_g" db:"total_fiber_g"`
}
```
- Field: PascalCase with `G` suffix
- JSON tag: snake_case with `_g` suffix
- DB tag: snake_case with `_g` suffix

**Pattern 2**: NutritionTotals (internal calculations)
```go
type NutritionTotals struct {
    Calories float64 `json:"calories"`
    ProteinG float64 `json:"protein_g"`
    CarbsG   float64 `json:"carbs_g"`
    FatG     float64 `json:"fat_g"`
    FiberG   float64 `json:"fiber_g"`
}
```
- Field: PascalCase with `G` suffix
- JSON tag: snake_case with `_g` suffix

**Pattern 3**: MealItem struct (database-mapped)
```go
type MealItem struct {
    Calories  float64 `json:"calories" db:"calories"`
    ProteinG  float64 `json:"protein_g" db:"protein"`
    CarbsG    float64 `json:"carbs_g" db:"carbs"`
    FatG      float64 `json:"fat_g" db:"fat"`
    FiberG    float64 `json:"fiber_g" db:"fiber"`
}
```
- Field: PascalCase with `G` suffix for macros
- JSON tag: snake_case with `_g` suffix
- DB tag: NO suffix (just `protein`, `carbs`, `fat`, `fiber`)

**Pattern 4**: AI Service NutritionData (internal AI format)
```go
type NutritionData struct {
    Calories      float64
    Protein       float64  // NO suffix
    Carbohydrates float64  // NOT "Carbs"
    Fat           float64
    Fiber         float64
    Sugar         float64
    Sodium        float64
}
```

### Database Column Names

**meals table**:
- `total_calories` (NO suffix)
- `total_protein_g` (WITH `_g` suffix)
- `total_carbs_g` (WITH `_g` suffix)
- `total_fat_g` (WITH `_g` suffix)

**meal_items table**:
- `calories` (NO suffix)
- `protein` (NO suffix)
- `carbs` (NO suffix)
- `fat` (NO suffix)
- `fiber` (NO suffix)

**Inconsistency**: meals table uses `_g` suffix, meal_items table does NOT

### TypeScript/Frontend

**Pattern 1**: MealItem interface
```typescript
export interface MealItem {
    calories: number
    protein_g: number
    carbs_g: number
    fat_g: number
}
```
- ALL have `_g` suffix except `calories`

**Pattern 2**: Meal interface
```typescript
export interface Meal {
    total_calories: number
    total_protein: number  // NO suffix!
    total_carbs: number
    total_fat: number
}
```
- Inconsistent: `types/index.ts:18` uses `total_protein` without `_g`
- But `ManualMealForm.tsx:151` sends `total_protein_g` to backend

---

## Inconsistencies Found

### 1. Database Naming Inconsistency

**Issue**: meals table uses `_g` suffix, meal_items table does NOT

**Evidence**:
- `meals.total_protein_g` (WITH suffix)
- `meal_items.protein` (WITHOUT suffix)

**Impact**: Medium - Causes confusion in mapping, requires manual field mapping in Go structs

---

### 2. Frontend Type Definition Mismatch

**Issue**: `types/index.ts` defines Meal interface WITHOUT `_g` suffix, but actual API usage includes suffix

**Evidence**:
- Type definition: `total_protein: number` (line 18)
- Actual API call: `total_protein_g: totals.protein_g` (ManualMealForm.tsx:151)
- Card display: `meal.total_protein` (MealCard.tsx:70)

**Impact**: High - Type safety violation, runtime data mismatch

---

### 3. AI Service Data Structure Different

**Issue**: AI services use different field names than domain models

**Evidence**:
- AI NutritionData: `Protein`, `Carbohydrates` (no suffix, full name)
- Domain models: `ProteinG`, `CarbsG` (suffix, abbreviated)

**Impact**: Low - Isolated to AI service layer, properly converted

---

### 4. Analytics Naming Drops Suffix

**Issue**: Analytics structs use `TotalProtein` instead of `TotalProteinG`

**Evidence**:
- `analytics/service.go` DailyTotals: `TotalProtein`, `TotalCarbs`, `TotalFat`
- Meal struct: `TotalProteinG`, `TotalCarbsG`, `TotalFatG`

**Impact**: Low - Internal analytics only, doesn't affect API

---

### 5. Duplicate Calculation Logic

**Issue**: `service_draft.go:184-192` duplicates the logic from `calculateTotals()`

**Evidence**:
```go
// service.go:403-413
func (s *service) calculateTotals(items []DraftMealItem) NutritionTotals

// service_draft.go:184-192
totals := &NutritionTotals{}
for _, item := range items {
    totals.Calories += item.Calories
    // ... SAME logic
}
```

**Impact**: Medium - Code duplication, potential for divergence

**Recommendation**: Refactor to use `calculateTotals()` helper

---

## Recommendations

### 1. Standardize Database Column Names

**Current**: Mixed usage of `_g` suffix

**Proposed**: ALL macro columns should use `_g` suffix for clarity

**Migration**:
```sql
-- Rename meal_items columns
ALTER TABLE meal_items RENAME COLUMN protein TO protein_g;
ALTER TABLE meal_items RENAME COLUMN carbs TO carbs_g;
ALTER TABLE meal_items RENAME COLUMN fat TO fat_g;
ALTER TABLE meal_items RENAME COLUMN fiber TO fiber_g;
```

**Impact**: Medium effort, requires migration and Go struct tag updates

---

### 2. Fix Frontend Type Definitions

**Current**: `types/index.ts` Meal interface missing `_g` suffix

**Proposed**:
```typescript
export interface Meal {
    total_calories: number
    total_protein_g: number  // Add _g
    total_carbs_g: number
    total_fat_g: number
}
```

**Impact**: Low effort, TypeScript-only change, verify all usage sites

---

### 3. Consolidate Calculation Logic

**Current**: Multiple places with inline totals calculation

**Proposed**: Single source of truth function

**Implementation**:
```go
// In meals/service.go
func CalculateTotalsFromItems(items []DraftMealItem) NutritionTotals {
    var totals NutritionTotals
    for _, item := range items {
        totals.Calories += item.Calories
        totals.ProteinG += item.ProteinG
        totals.CarbsG += item.CarbsG
        totals.FatG += item.FatG
        totals.FiberG += item.FiberG
    }
    return totals
}

// Update service_draft.go:184
totals := CalculateTotalsFromItems(draftItems)
```

**Impact**: Low effort, improves maintainability

---

### 4. Document Naming Conventions

**Proposed**: Add to CLAUDE.md:

```markdown
## Nutrition Data Naming Convention

### Rule: Always use `_g` suffix for macro nutrients measured in grams

**Database Columns**:
- `total_protein_g` (NOT `total_protein`)
- `protein_g` (NOT `protein`)

**Go Struct Fields**:
- `TotalProteinG` (PascalCase)
- JSON tag: `"protein_g"` (snake_case)
- DB tag: `"protein_g"` (snake_case)

**TypeScript/Frontend**:
- `protein_g: number` (snake_case)

**Exception**: `calories` never uses suffix (it's kcal, not grams)
```

---

## Summary Statistics

| Category | Count | Notes |
|----------|-------|-------|
| Backend calculation locations | 7 | Including AI services |
| Frontend calculation locations | 2 | React components |
| Database aggregations | 4 | SQL SUM operations |
| Naming inconsistencies | 5 | See section above |
| Duplicate logic instances | 1 | service_draft.go |
| Files requiring changes | 8 | For full standardization |

---

## Next Steps

1. **Immediate**: Fix frontend type definition mismatch (types/index.ts)
2. **Short-term**: Consolidate duplicate calculation logic
3. **Medium-term**: Database migration to standardize column names
4. **Long-term**: Update all code to follow naming convention document

---

**Document Owner**: Code Pattern Analyzer Agent
**Last Updated**: 2025-11-16
**Related Issues**: TBD
