# API Nutrition Contracts - Field Naming Reference

**Version:** 1.0
**Date:** 2025-11-16
**Status:** ✅ DEFINITIVE - All API responses MUST follow these contracts

---

## Overview

This document defines the exact JSON field names for all nutrition-related API endpoints. These contracts are the authoritative source for frontend TypeScript interfaces.

**Key Principle:** JSON field names ALWAYS use `_g` suffix for macro nutrients (protein, carbs, fat, fiber) to eliminate ambiguity about units.

---

## Core Data Structures

### Meal Object

**Endpoint:** `GET /meals/{id}`, `GET /meals?date={date}`, `POST /meals/confirm`, `PUT /meals/{id}`

**JSON Structure:**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "meal_type": "breakfast",
  "consumed_at": "2025-11-16T08:30:00Z",
  "total_calories": 523.5,
  "total_protein_g": 28.3,
  "total_carbs_g": 45.2,
  "total_fat_g": 18.7,
  "total_fiber_g": 8.1,
  "is_draft": false,
  "draft_status": null,
  "photos": ["https://..."],
  "notes": "Post-workout meal",
  "created_at": "2025-11-16T08:25:00Z",
  "updated_at": "2025-11-16T08:30:00Z"
}
```

**Field Definitions:**

| Field | Type | Unit | Required | Description |
|-------|------|------|----------|-------------|
| `id` | string (UUID) | - | Yes | Unique meal identifier |
| `user_id` | string (UUID) | - | Yes | User who logged the meal |
| `meal_type` | enum | - | Yes | One of: `breakfast`, `lunch`, `dinner`, `snack` |
| `consumed_at` | string (ISO 8601) | - | Yes | When the meal was eaten |
| `total_calories` | number | kcal | Yes | Sum of all item calories (auto-calculated) |
| `total_protein_g` | number | grams | Yes | Sum of all item protein (auto-calculated) |
| `total_carbs_g` | number | grams | Yes | Sum of all item carbs (auto-calculated) |
| `total_fat_g` | number | grams | Yes | Sum of all item fat (auto-calculated) |
| `total_fiber_g` | number | grams | No | Sum of all item fiber (auto-calculated, nullable) |
| `is_draft` | boolean | - | Yes | Whether meal is in draft state |
| `draft_status` | enum | - | No | One of: `analyzing`, `ready`, `error` (null if not draft) |
| `photos` | array[string] | - | No | Photo URLs |
| `notes` | string | - | No | User notes |
| `created_at` | string (ISO 8601) | - | Yes | Timestamp when created |
| `updated_at` | string (ISO 8601) | - | Yes | Timestamp when last modified |

**Critical Notes:**
- All `total_*` fields are **auto-calculated by database triggers**
- Backend NEVER manually calculates these values
- Frontend MUST NOT recalculate; always display backend values

---

### MealItem Object

**Endpoint:** Nested in `GET /meals/{id}` response as `items` array

**JSON Structure:**
```json
{
  "id": "uuid",
  "meal_id": "uuid",
  "name": "Grilled chicken breast",
  "quantity": 150.0,
  "unit": "g",
  "calories": 165.0,
  "protein_g": 31.0,
  "carbs_g": 0.0,
  "fat_g": 3.6,
  "fiber_g": 0.0,
  "created_at": "2025-11-16T08:30:00Z"
}
```

**Field Definitions:**

| Field | Type | Unit | Required | Description |
|-------|------|------|----------|-------------|
| `id` | string (UUID) | - | Yes | Unique item identifier |
| `meal_id` | string (UUID) | - | Yes | Parent meal reference |
| `name` | string | - | Yes | Food name (NOT `food_name`) |
| `quantity` | number | (see unit) | Yes | Amount consumed |
| `unit` | string | - | Yes | Measurement unit (`g`, `oz`, `serving`, `cup`, etc.) |
| `calories` | number | kcal | Yes | Calories in this serving |
| `protein_g` | number | grams | Yes | Protein in this serving |
| `carbs_g` | number | grams | Yes | Carbohydrates in this serving |
| `fat_g` | number | grams | Yes | Fat in this serving |
| `fiber_g` | number | grams | No | Fiber in this serving (nullable) |
| `created_at` | string (ISO 8601) | - | Yes | Timestamp when created |

**Critical Notes:**
- Field name is `name`, NOT `food_name` or `description`
- JSON uses `protein_g` even though database column is `protein` (no suffix)
- All macro fields use `_g` suffix for API clarity

---

### DraftMealItem Object

**Endpoint:** `POST /meals/draft`, `PUT /meals/draft/{id}`, `GET /meals/draft/{id}/status`

**JSON Structure:** (Same as MealItem but used in draft context)

```json
{
  "name": "Oatmeal with banana",
  "quantity": 1.0,
  "unit": "serving",
  "calories": 210.0,
  "protein_g": 7.0,
  "carbs_g": 38.0,
  "fat_g": 4.5,
  "fiber_g": 5.0
}
```

**Note:** No `id` or `meal_id` when creating draft items.

---

### NutritionTotals Object

**Endpoint:** `GET /meals/draft/{id}/status` (embedded in response)

**JSON Structure:**
```json
{
  "calories": 523.5,
  "protein_g": 28.3,
  "carbs_g": 45.2,
  "fat_g": 18.7,
  "fiber_g": 8.1
}
```

**Usage:** Internal representation for aggregated nutrition. Frontend can use this for draft meal previews.

---

## API Endpoint Examples

### 1. Get Meals for Date

**Request:**
```bash
curl -X GET "https://api.example.com/meals?date=2025-11-16" \
  -H "Authorization: Bearer {token}"
```

**Response:**
```json
{
  "meals": [
    {
      "id": "a1b2c3d4-...",
      "meal_type": "breakfast",
      "consumed_at": "2025-11-16T08:30:00Z",
      "total_calories": 523.5,
      "total_protein_g": 28.3,
      "total_carbs_g": 45.2,
      "total_fat_g": 18.7,
      "total_fiber_g": 8.1,
      "items": [
        {
          "id": "item1-...",
          "name": "Oatmeal",
          "quantity": 1.0,
          "unit": "serving",
          "calories": 150.0,
          "protein_g": 5.0,
          "carbs_g": 27.0,
          "fat_g": 3.0,
          "fiber_g": 4.0
        }
      ]
    }
  ]
}
```

---

### 2. Confirm Draft Meal

**Request:**
```bash
curl -X POST "https://api.example.com/meals/confirm" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "meal_type": "lunch",
    "consumed_at": "2025-11-16T12:30:00Z",
    "items": [
      {
        "name": "Grilled chicken breast",
        "quantity": 150.0,
        "unit": "g",
        "calories": 165.0,
        "protein_g": 31.0,
        "carbs_g": 0.0,
        "fat_g": 3.6
      },
      {
        "name": "Brown rice",
        "quantity": 200.0,
        "unit": "g",
        "calories": 220.0,
        "protein_g": 5.0,
        "carbs_g": 46.0,
        "fat_g": 1.6,
        "fiber_g": 3.5
      }
    ],
    "notes": "Post-workout meal"
  }'
```

**Response:**
```json
{
  "id": "meal-uuid-...",
  "meal_type": "lunch",
  "consumed_at": "2025-11-16T12:30:00Z",
  "total_calories": 385.0,
  "total_protein_g": 36.0,
  "total_carbs_g": 46.0,
  "total_fat_g": 5.2,
  "total_fiber_g": 3.5,
  "is_draft": false,
  "notes": "Post-workout meal"
}
```

**Note:** `total_*` fields are calculated by database trigger, NOT by backend service.

---

### 3. Update Meal

**Request:**
```bash
curl -X PUT "https://api.example.com/meals/{id}" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "meal_type": "dinner",
    "consumed_at": "2025-11-16T19:00:00Z",
    "items": [
      {
        "name": "Salmon fillet",
        "quantity": 180.0,
        "unit": "g",
        "calories": 330.0,
        "protein_g": 36.0,
        "carbs_g": 0.0,
        "fat_g": 20.0
      }
    ]
  }'
```

**Response:** (Same structure as Meal object with updated values)

---

### 4. Get Draft Status

**Request:**
```bash
curl -X GET "https://api.example.com/meals/draft/{id}/status" \
  -H "Authorization: Bearer {token}"
```

**Response:**
```json
{
  "meal_id": "draft-uuid-...",
  "status": "ready",
  "items": [
    {
      "name": "Greek yogurt",
      "quantity": 150.0,
      "unit": "g",
      "calories": 100.0,
      "protein_g": 10.0,
      "carbs_g": 6.0,
      "fat_g": 4.0
    }
  ],
  "total": {
    "calories": 100.0,
    "protein_g": 10.0,
    "carbs_g": 6.0,
    "fat_g": 4.0,
    "fiber_g": 0.0
  }
}
```

---

## TypeScript Interface Generation

**IMPORTANT:** TypeScript interfaces MUST match these JSON contracts exactly.

### Recommended TypeScript Definitions

**File:** `frontend/types/nutrition.ts`

```typescript
/**
 * Auto-generated from backend API contracts
 * DO NOT modify without updating backend
 *
 * Source: backend/docs/API-NUTRITION-CONTRACTS.md
 * Last synced: 2025-11-16
 */

export type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack'
export type DraftStatus = 'analyzing' | 'ready' | 'error'

export interface Meal {
  id: string
  user_id: string
  meal_type: MealType
  consumed_at: string  // ISO 8601

  // Auto-calculated by database triggers
  total_calories: number
  total_protein_g: number
  total_carbs_g: number
  total_fat_g: number
  total_fiber_g?: number

  is_draft: boolean
  draft_status?: DraftStatus | null
  photos?: string[]
  notes?: string

  created_at: string  // ISO 8601
  updated_at: string  // ISO 8601

  // Nested items (optional, included in some endpoints)
  items?: MealItem[]
}

export interface MealItem {
  id: string
  meal_id: string
  name: string  // NOT food_name
  quantity: number
  unit: string

  calories: number
  protein_g: number  // Note: _g suffix in JSON
  carbs_g: number
  fat_g: number
  fiber_g?: number

  created_at: string  // ISO 8601
}

export interface DraftMealItem {
  name: string
  quantity: number
  unit: string

  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  fiber_g?: number
}

export interface NutritionTotals {
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  fiber_g?: number
}
```

---

## Field Naming Pattern Summary

### The `_g` Suffix Pattern

**Rule:** All macro nutrients (protein, carbs, fat, fiber) use `_g` suffix in JSON to indicate grams.

**Rationale:**
- **Clarity:** Eliminates ambiguity about units
- **Consistency:** Same pattern across all API responses
- **Type Safety:** Frontend can validate units at compile time

**Exception:** `calories` never uses suffix (it's always kcal, not grams)

### Database vs JSON Mapping

**meals table → JSON:**
```
total_calories  → total_calories   (no change)
total_protein_g → total_protein_g  (no change)
total_carbs_g   → total_carbs_g    (no change)
total_fat_g     → total_fat_g      (no change)
```

**meal_items table → JSON:**
```
calories → calories   (no change)
protein  → protein_g  (adds _g suffix)
carbs    → carbs_g    (adds _g suffix)
fat      → fat_g      (adds _g suffix)
fiber    → fiber_g    (adds _g suffix)
```

**Why the difference?**
- `meals` table stores aggregated totals → suffix in DB for clarity
- `meal_items` table has `unit` column → suffix not needed in DB
- JSON ALWAYS uses suffix for API consistency

---

## Validation Rules

### Client-Side Validation

Frontend MAY validate:
- Required fields are present
- Field types match interface
- Enum values are valid (`meal_type`, `draft_status`)
- Numeric values are non-negative

Frontend MUST NOT:
- Recalculate nutrition totals
- Validate total_* fields match sum of items (backend responsibility)
- Apply Atwater factors or other macro calculations

### Server-Side Validation

Backend MUST validate:
- All required fields present
- Field types correct
- Enum values valid
- Nutrition values reasonable (use `constants.ValidateMacroCalories`)

Backend MUST NOT:
- Manually calculate `total_*` fields (database triggers handle this)

---

## Migration Notes

### Breaking Changes (Migration 012)

**Field Renames:**
- `food_name` → `name` (in meal_items and template_items)
- `meal_time` → `consumed_at` (in meals) - DROPPED, not renamed

**Dropped Fields:**
- `meals.name` - replaced by `meal_type` enum

**New Fields:**
- `meals.total_fiber_g` - now tracked and aggregated

**Frontend Migration:**
```typescript
// OLD (before migration 012)
interface MealItem {
  food_name: string  // ❌ Removed
}

interface Meal {
  meal_time: string  // ❌ Removed
  name: string       // ❌ Removed
}

// NEW (after migration 012)
interface MealItem {
  name: string  // ✅ Simplified
}

interface Meal {
  consumed_at: string  // ✅ Clearer
  meal_type: MealType  // ✅ Type-safe
}
```

---

## Common Mistakes

### ❌ Wrong Field Names
```typescript
// WRONG
interface MealItem {
  food_name: string  // Should be "name"
  grams: number      // Should be "quantity"
}
```

### ❌ Missing `_g` Suffix
```typescript
// WRONG
interface Meal {
  total_protein: number  // Should be "total_protein_g"
}
```

### ❌ Client-Side Calculation
```typescript
// WRONG
const totalCalories = meal.items.reduce((sum, item) => sum + item.calories, 0)

// CORRECT
const totalCalories = meal.total_calories  // Use backend value
```

### ❌ Inconsistent Types
```typescript
// WRONG
interface Meal {
  consumed_at: Date  // Should be string (ISO 8601)
}

// CORRECT
interface Meal {
  consumed_at: string  // ISO 8601 string, parse on client if needed
}
```

---

## Related Documentation

- **Naming Standards:** `backend/docs/naming-standards.md` - Complete naming rules
- **Architecture:** `backend/docs/ARCHITECTURE-SINGLE-SOURCE-TRUTH.md` - System design
- **Backend Guide:** `backend/CLAUDE.md` - Development rules
- **Calculation History:** `backend/docs/calculation-inventory.md` - Migration analysis

---

**Questions?** This document is the single source of truth for API contracts. If you find inconsistencies between this doc and actual API responses, file a bug - the API is wrong.

**Last Updated:** 2025-11-16
**Schema Version:** 1.0 (post-migration 012/013)
