# Database Schema Ground Truth

**Document Purpose:** Definitive reference for the actual current database schema.
**Last Updated:** 2025-11-16
**Migration Version:** 011 (latest)

---

## Critical Findings Summary

### MAJOR DISCREPANCIES FOUND

1. **`meal_items.food_name` → `meal_items.name`**
   - Migration 001: Created column as `food_name`
   - Migration 010: RPC function uses `name`
   - **CONFLICT:** Column was never renamed from `food_name` to `name`

2. **`meals.meal_time` → `meals.consumed_at`**
   - Migration 001: Created column as `meal_time`
   - Migration 009: Added `consumed_at`, copied `meal_time` data to it
   - **STATUS:** Both columns exist, but `consumed_at` is the new standard

3. **`meals.name` column**
   - Migration 001: Created as `name` TEXT NOT NULL
   - Migration 009: Added `meal_type` (breakfast/lunch/dinner/snack)
   - **STATUS:** Both columns exist simultaneously

4. **Nutrition column naming**
   - Migration 001: Created without `_g` suffix (e.g., `protein`, `carbs`, `fat`)
   - Migration 009: Added with `_g` suffix to `meals` table (e.g., `total_protein_g`)
   - **STATUS:** Inconsistent naming between tables

---

## Table Schemas

### 1. `meals` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | uuid_generate_v4() | Primary key |
| `user_id` | UUID | NOT NULL | - | FK to users.id CASCADE |
| `name` | TEXT | NOT NULL | - | User-defined meal name (e.g., "Breakfast") |
| `meal_time` | TIMESTAMPTZ | NOT NULL | - | **LEGACY:** Original timestamp column |
| `consumed_at` | TIMESTAMPTZ | NULL | - | **NEW:** When meal was consumed (Migration 009) |
| `notes` | TEXT | NULL | - | Optional notes |
| `meal_type` | TEXT | NULL | - | breakfast/lunch/dinner/snack (Migration 009) |
| `total_calories` | DECIMAL(7,1) | NULL | 0 | Sum of all items (Migration 009) |
| `total_protein_g` | DECIMAL(6,1) | NULL | 0 | **NOTE: Has _g suffix** |
| `total_carbs_g` | DECIMAL(6,1) | NULL | 0 | **NOTE: Has _g suffix** |
| `total_fat_g` | DECIMAL(6,1) | NULL | 0 | **NOTE: Has _g suffix** |
| `total_fiber_g` | DECIMAL(5,1) | NULL | 0 | **NOTE: Has _g suffix** |
| `photos` | JSONB | NULL | '[]'::jsonb | Array of photo URLs (Migration 009) |
| `deleted_at` | TIMESTAMPTZ | NULL | NULL | Soft delete (Migration 009) |
| `normalized_description` | TEXT | NULL | - | AI-normalized description (Migration 004) |
| `is_draft` | BOOLEAN | NOT NULL | FALSE | Draft status (Migration 005) |
| `draft_status` | TEXT | NULL | NULL | analyzing/ready/error (Migration 005) |
| `draft_error` | TEXT | NULL | NULL | Error message if draft failed (Migration 005) |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |
| `updated_at` | TIMESTAMPTZ | NOT NULL | NOW() | Auto-updated by trigger |

**Constraints:**
- `meal_name_length`: char_length(name) >= 1 AND <= 100
- `notes_length`: notes IS NULL OR char_length(notes) <= 1000
- `future_meal_time`: meal_time <= NOW() + INTERVAL '7 days'
- `valid_meal_type`: meal_type IN ('breakfast', 'lunch', 'dinner', 'snack')
- `positive_total_calories`: total_calories >= 0 AND <= 99999
- `positive_total_protein`: total_protein_g >= 0 AND <= 9999
- `positive_total_carbs`: total_carbs_g >= 0 AND <= 9999
- `positive_total_fat`: total_fat_g >= 0 AND <= 9999
- `positive_total_fiber`: total_fiber_g >= 0 AND <= 9999
- `valid_draft_status`: draft_status IN ('analyzing', 'ready', 'error')

**Indexes:**
- `idx_meals_user_time`: (user_id, meal_time DESC)
- `idx_meals_created_at`: (user_id, created_at DESC)
- `idx_meals_normalized_desc`: (user_id, normalized_description)
- `idx_meals_meal_type`: (user_id, meal_type) WHERE deleted_at IS NULL
- `idx_meals_consumed_at`: (user_id, consumed_at DESC) WHERE deleted_at IS NULL
- `idx_meals_not_deleted`: (user_id, created_at DESC) WHERE deleted_at IS NULL
- `idx_meals_draft_status`: (user_id, draft_status) WHERE is_draft = TRUE
- `idx_meals_draft_created`: (created_at) WHERE is_draft = TRUE
- `idx_meals_user_consumed_date`: (user_id, DATE(consumed_at AT TIME ZONE 'UTC')) WHERE deleted_at IS NULL
- `idx_meals_user_consumed_type`: (user_id, consumed_at, meal_type) WHERE deleted_at IS NULL
- `idx_meals_user_consumed_desc`: (user_id, consumed_at DESC) WHERE deleted_at IS NULL

---

### 2. `meal_items` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | uuid_generate_v4() | Primary key |
| `meal_id` | UUID | NOT NULL | - | FK to meals.id CASCADE |
| `food_name` | TEXT | NOT NULL | - | **CRITICAL: Column is food_name, NOT name** |
| `quantity` | DECIMAL(8,2) | NOT NULL | - | Amount consumed |
| `unit` | TEXT | NOT NULL | - | Unit of measurement |
| `calories` | DECIMAL(7,1) | NOT NULL | - | **NOTE: No _g suffix** |
| `protein` | DECIMAL(6,1) | NOT NULL | - | **NOTE: No _g suffix** |
| `carbs` | DECIMAL(6,1) | NOT NULL | - | **NOTE: No _g suffix** |
| `fat` | DECIMAL(6,1) | NOT NULL | - | **NOTE: No _g suffix** |
| `fiber` | DECIMAL(5,1) | NULL | - | **NOTE: No _g suffix** |
| `sugar` | DECIMAL(6,1) | NULL | - | **NOTE: No _g suffix** |
| `sodium` | DECIMAL(7,1) | NULL | - | **NOTE: No _g suffix** |
| `source` | TEXT | NOT NULL | 'ai' | ai/common_foods/manual/template |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |

**Constraints:**
- `food_name_length`: char_length(food_name) >= 1 AND <= 200
- `positive_quantity`: quantity > 0 AND <= 99999
- `valid_unit`: char_length(unit) >= 1 AND <= 50
- `positive_calories`: calories >= 0 AND <= 9999
- `positive_protein`: protein >= 0 AND <= 999
- `positive_carbs`: carbs >= 0 AND <= 999
- `positive_fat`: fat >= 0 AND <= 999
- `positive_fiber`: fiber >= 0 AND <= 999
- `positive_sugar`: sugar >= 0 AND <= 999
- `positive_sodium`: sodium >= 0 AND <= 99999
- `valid_source`: source IN ('ai', 'common_foods', 'manual', 'template')

**Indexes:**
- `idx_meal_items_meal_id`: (meal_id)
- `idx_meal_items_meal_nutrition`: (meal_id, calories, protein, carbs, fat)
- `idx_meal_items_food_name`: GIN(food_name gin_trgm_ops)
- `idx_meal_items_meal`: (meal_id)
- `idx_meal_items_meal_calories`: (meal_id, calories, protein, carbs, fat)

---

### 3. `templates` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | uuid_generate_v4() | Primary key |
| `user_id` | UUID | NOT NULL | - | FK to users.id CASCADE |
| `name` | TEXT | NOT NULL | - | Template name |
| `description` | TEXT | NULL | - | Optional description |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |
| `updated_at` | TIMESTAMPTZ | NOT NULL | NOW() | Auto-updated by trigger |

**Constraints:**
- `template_name_length`: char_length(name) >= 1 AND <= 100
- `description_length`: description IS NULL OR char_length(description) <= 500
- `unique_template_name`: UNIQUE (user_id, name)

**Indexes:**
- `idx_templates_user_id`: (user_id, created_at DESC)
- `idx_templates_user_name`: (user_id, name)

---

### 4. `template_items` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | uuid_generate_v4() | Primary key |
| `template_id` | UUID | NOT NULL | - | FK to templates.id CASCADE |
| `food_name` | TEXT | NOT NULL | - | **Same as meal_items** |
| `quantity` | DECIMAL(8,2) | NOT NULL | - | Amount |
| `unit` | TEXT | NOT NULL | - | Unit of measurement |
| `calories` | DECIMAL(7,1) | NOT NULL | - | **No _g suffix** |
| `protein` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `carbs` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `fat` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `fiber` | DECIMAL(5,1) | NULL | - | **No _g suffix** |
| `sugar` | DECIMAL(6,1) | NULL | - | **No _g suffix** |
| `sodium` | DECIMAL(7,1) | NULL | - | **No _g suffix** |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |

**Constraints:**
- Same as meal_items (except no source column)

**Indexes:**
- `idx_template_items_template_id`: (template_id)
- `idx_template_items_food_name`: GIN(food_name gin_trgm_ops)

---

### 5. `users` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | - | PK, FK to auth.users.id CASCADE |
| `email` | TEXT | NOT NULL | - | UNIQUE |
| `full_name` | TEXT | NULL | - | User's full name |
| `timezone` | TEXT | NOT NULL | 'UTC' | IANA timezone |
| `onboarding_completed` | BOOLEAN | NOT NULL | FALSE | Migration 006 |
| `onboarding_completed_at` | TIMESTAMPTZ | NULL | - | Migration 006 |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |
| `updated_at` | TIMESTAMPTZ | NOT NULL | NOW() | Auto-updated |

**Constraints:**
- `valid_email`: email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'
- `valid_timezone`: timezone ~ '^[A-Za-z/_]+$'

**Indexes:**
- `idx_users_email`: (email)
- `idx_users_created_at`: (created_at DESC)
- `idx_users_onboarding`: (onboarding_completed) WHERE onboarding_completed = false

---

### 6. `user_daily_goals` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `user_id` | UUID | NOT NULL | - | PK, FK to users.id CASCADE |
| `calories` | DECIMAL(7,1) | NOT NULL | 2000 | Daily calorie goal |
| `protein` | DECIMAL(6,1) | NOT NULL | 150 | **No _g suffix** |
| `carbs` | DECIMAL(6,1) | NOT NULL | 200 | **No _g suffix** |
| `fat` | DECIMAL(6,1) | NOT NULL | 65 | **No _g suffix** |
| `fiber` | DECIMAL(5,1) | NULL | 25 | **No _g suffix** |
| `sugar` | DECIMAL(6,1) | NULL | - | **No _g suffix** |
| `sodium` | DECIMAL(7,1) | NULL | - | **No _g suffix** |
| `updated_at` | TIMESTAMPTZ | NOT NULL | NOW() | Auto-updated |

**Indexes:**
- `idx_user_daily_goals_user`: (user_id)

---

### 7. `common_foods` Table

| Column Name | PostgreSQL Type | Nullable | Default | Notes |
|-------------|----------------|----------|---------|-------|
| `id` | UUID | NOT NULL | uuid_generate_v4() | Primary key |
| `name` | TEXT | NOT NULL | - | Food name |
| `category` | TEXT | NULL | - | Food category |
| `serving_size` | DECIMAL(8,2) | NOT NULL | - | Serving size |
| `serving_unit` | TEXT | NOT NULL | - | Unit (g, oz, cup, etc) |
| `calories` | DECIMAL(7,1) | NOT NULL | - | **No _g suffix** |
| `protein` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `carbs` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `fat` | DECIMAL(6,1) | NOT NULL | - | **No _g suffix** |
| `fiber` | DECIMAL(5,1) | NULL | - | **No _g suffix** |
| `sugar` | DECIMAL(6,1) | NULL | - | **No _g suffix** |
| `sodium` | DECIMAL(7,1) | NULL | - | **No _g suffix** |
| `verified` | BOOLEAN | NOT NULL | FALSE | Admin verified |
| `source` | TEXT | NULL | - | Data source |
| `created_at` | TIMESTAMPTZ | NOT NULL | NOW() | - |
| `updated_at` | TIMESTAMPTZ | NOT NULL | NOW() | Auto-updated |

**Indexes:**
- `idx_common_foods_name_trgm`: GIN(name gin_trgm_ops)
- `idx_common_foods_name`: (name)
- `idx_common_foods_name_lower`: (LOWER(name))
- `idx_common_foods_category`: (category) WHERE category IS NOT NULL
- `idx_common_foods_verified`: (verified, name)
- `idx_common_foods_category_name`: (category, name) WHERE category IS NOT NULL
- `idx_common_foods_search`: GIN(to_tsvector('english', name))

---

### 8. Other Tables (Brief)

**`user_profiles`** (Migration 007):
- `user_id` (PK, FK to auth.users)
- `full_name`, `email` (UNIQUE), `avatar_url`
- `created_at`, `updated_at`

**`user_settings`** (Migration 007):
- `user_id` (PK, FK to user_profiles)
- `notifications_enabled`, `email_notifications`, `weekly_reports`
- `theme` (light/dark/auto), `language`, `timezone`
- `created_at`, `updated_at`

**`meal_flags`**:
- Track user-reported issues with meals
- `flag_type`: inaccurate/wrong_portions/missing_items/other

**`weight_entries`**:
- `weight_kg`, `measured_at`, `notes`
- Indexes: `idx_weight_entries_user_measured`, `idx_weight_entries_date_range`

**`ai_usage`**:
- Track AI API usage for rate limiting
- `endpoint`, `tokens_used`, `cost_usd`

**`idempotency_keys`**:
- Prevent duplicate API requests
- `key` (PK), `response` (JSONB)

---

## RPC Functions

### 1. `create_meal_with_items` (Migration 010 - CURRENT)

**CRITICAL ISSUE:** This function uses `name` but the column is actually `food_name`!

```sql
CREATE OR REPLACE FUNCTION create_meal_with_items(
    p_user_id UUID,
    p_meal_type TEXT,
    p_consumed_at TIMESTAMPTZ,
    p_photos JSONB,
    p_notes TEXT,
    p_total_calories DECIMAL,
    p_total_protein_g DECIMAL,
    p_total_carbs_g DECIMAL,
    p_total_fat_g DECIMAL,
    p_total_fiber_g DECIMAL,
    p_items JSONB
) RETURNS UUID
```

**Parameters:**
- Uses `total_protein_g` (with suffix) for `meals` table
- Expects JSONB items with: `Name`, `Quantity`, `Unit`, `Calories`, `ProteinG`, `CarbsG`, `FatG`, `FiberG`

**BUG ON LINE 84:**
```sql
INSERT INTO meal_items (meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
```
Should be:
```sql
INSERT INTO meal_items (meal_id, food_name, quantity, unit, calories, protein, carbs, fat, fiber)
```

### 2. Other RPC Functions (Migration 002)

- `get_daily_nutrition(p_user_id, p_date, p_timezone)` - Uses `meal_time` (legacy)
- `create_template_from_meal(...)` - Uses `food_name` correctly
- `search_common_foods(p_query, p_limit)` - Fuzzy search
- `get_nutrition_trends(...)` - Date range analytics

---

## Detailed Discrepancy Analysis

### Discrepancy 1: `meal_items.food_name` vs `meal_items.name`

**Migration 001 (Line 135):**
```sql
CREATE TABLE meal_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    food_name TEXT NOT NULL,  -- ✅ CREATED AS food_name
    ...
)
```

**Migration 010 (Line 84):**
```sql
INSERT INTO meal_items (
    meal_id,
    name,  -- ❌ USES name (WRONG!)
    quantity,
    unit,
    ...
)
```

**Reality:**
- Column is **`food_name`** (never renamed)
- Migration 010 RPC function will **FAIL** when executed
- No migration renamed `food_name` → `name`

**Impact:** CRITICAL - RPC function is broken!

---

### Discrepancy 2: `meals.meal_time` vs `meals.consumed_at`

**Migration 001 (Line 92):**
```sql
CREATE TABLE meals (
    ...
    meal_time TIMESTAMPTZ NOT NULL,  -- ✅ Original column
    ...
)
```

**Migration 009 (Lines 10-14):**
```sql
ALTER TABLE meals ADD COLUMN IF NOT EXISTS consumed_at TIMESTAMPTZ;
UPDATE meals SET consumed_at = meal_time WHERE consumed_at IS NULL;
```

**Reality:**
- **Both columns exist**
- `meal_time` is still NOT NULL (original constraint)
- `consumed_at` is NULL (new column)
- Data was copied from `meal_time` → `consumed_at`
- New code should use `consumed_at`
- Legacy code still references `meal_time`

**Impact:** MEDIUM - Causes confusion, both columns have same data

---

### Discrepancy 3: `meals.name` and `meals.meal_type` Coexistence

**Migration 001 (Line 91):**
```sql
name TEXT NOT NULL,  -- User-defined meal name (e.g., "Breakfast", "Lunch")
```

**Migration 009 (Line 8):**
```sql
ALTER TABLE meals ADD COLUMN IF NOT EXISTS meal_type TEXT;
-- Valid values: breakfast, lunch, dinner, snack
```

**Reality:**
- `name`: Free-text field, user-defined (e.g., "Post-Workout Meal")
- `meal_type`: Enum-like constraint (breakfast/lunch/dinner/snack)
- Both serve different purposes, both exist
- Migration 010 RPC only sets `meal_type`, not `name`

**Impact:** LOW - This is intentional, but confusing

---

### Discrepancy 4: Nutrition Column Naming Inconsistency

**`meal_items` table (Migration 001):**
- `protein` (no suffix)
- `carbs` (no suffix)
- `fat` (no suffix)
- `fiber` (no suffix)

**`meals` table (Migration 009):**
- `total_protein_g` (with `_g` suffix)
- `total_carbs_g` (with `_g` suffix)
- `total_fat_g` (with `_g` suffix)
- `total_fiber_g` (with `_g` suffix)

**`user_daily_goals` table (Migration 001):**
- `protein` (no suffix)
- `carbs` (no suffix)
- `fat` (no suffix)
- `fiber` (no suffix)

**`common_foods` table (Migration 001):**
- `protein` (no suffix)
- `carbs` (no suffix)
- `fat` (no suffix)
- `fiber` (no suffix)

**Reality:**
- **Only `meals` table uses `_g` suffix**
- All other tables omit the suffix
- Inconsistent naming convention

**Impact:** MEDIUM - Causes mapping confusion in code

---

## Recommendations

### CRITICAL FIX REQUIRED

**Migration 012: Fix `create_meal_with_items` RPC Function**

```sql
-- Migration: 012_fix_create_meal_rpc_column_name.up.sql
DROP FUNCTION IF EXISTS create_meal_with_items(...);

CREATE OR REPLACE FUNCTION create_meal_with_items(...)
RETURNS UUID AS $$
BEGIN
    -- ...

    -- FIX: Change 'name' to 'food_name'
    INSERT INTO meal_items (
        meal_id,
        food_name,  -- ✅ CORRECT COLUMN NAME
        quantity,
        unit,
        calories,
        protein,
        carbs,
        fat,
        fiber
    )
    VALUES (
        v_meal_id,
        v_meal_item->>'Name',
        (v_meal_item->>'Quantity')::DECIMAL,
        v_meal_item->>'Unit',
        (v_meal_item->>'Calories')::DECIMAL,
        (v_meal_item->>'ProteinG')::DECIMAL,
        (v_meal_item->>'CarbsG')::DECIMAL,
        (v_meal_item->>'FatG')::DECIMAL,
        (v_meal_item->>'FiberG')::DECIMAL
    );

    -- ...
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

### OPTIONAL IMPROVEMENTS

**Option 1: Standardize to `consumed_at`**
- Remove `meal_time` column (breaking change)
- Update all queries to use `consumed_at`

**Option 2: Rename to `food_name` everywhere**
- Keep `food_name` as standard
- Update all documentation

**Option 3: Standardize nutrition column naming**
- Either remove `_g` suffix from `meals` table
- Or add `_g` suffix to all tables
- Choose one convention

---

## Foreign Key Cascade Rules

**All FKs use `ON DELETE CASCADE`:**

- `meal_items.meal_id` → `meals.id` (CASCADE)
- `meals.user_id` → `users.id` (CASCADE)
- `templates.user_id` → `users.id` (CASCADE)
- `template_items.template_id` → `templates.id` (CASCADE)
- `user_daily_goals.user_id` → `users.id` (CASCADE)
- `weight_entries.user_id` → `users.id` (CASCADE)
- `meal_flags.meal_id` → `meals.id` (CASCADE)
- `meal_flags.user_id` → `users.id` (CASCADE)
- `ai_usage.user_id` → `users.id` (CASCADE)
- `idempotency_keys.user_id` → `users.id` (CASCADE)
- `users.id` → `auth.users.id` (CASCADE)
- `user_profiles.user_id` → `auth.users.id` (CASCADE)
- `user_settings.user_id` → `user_profiles.user_id` (CASCADE)

**Impact:** Deleting a user cascades to ALL related data.

---

## Summary of Current State

✅ **Working correctly:**
- All tables exist with proper structure
- RLS policies are in place
- Indexes are optimized for queries
- Triggers update `updated_at` timestamps

❌ **Broken:**
- Migration 010 RPC function uses wrong column name (`name` instead of `food_name`)
- Function will fail on execution

⚠️ **Confusing but functional:**
- Both `meal_time` and `consumed_at` exist (use `consumed_at` going forward)
- Both `meals.name` and `meals.meal_type` exist (different purposes)
- Nutrition columns have inconsistent naming (`_g` suffix only on `meals` table)

---

## Next Steps for Development

1. **Create Migration 012** to fix the RPC function bug
2. **Update Go code** to use:
   - `food_name` (not `name`) for meal_items
   - `consumed_at` (not `meal_time`) for meals
   - `total_protein_g` (with suffix) for meals totals
   - `protein` (no suffix) for meal_items/goals/foods
3. **Document the dual-column situation** in API docs
4. **Consider cleanup migration** to remove `meal_time` after transition period

---

**END OF GROUND TRUTH DOCUMENT**
