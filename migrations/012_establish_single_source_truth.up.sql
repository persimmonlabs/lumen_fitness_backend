-- =====================================================
-- Lumen Nutrition Tracker - Establish Single Source of Truth
-- Version: 012
-- Description: Fix schema inconsistencies and establish database triggers
--              as the SINGLE authoritative source for nutrition totals
-- =====================================================

BEGIN;

-- ========================================
-- PART 1: Schema Alignment
-- ========================================

-- Fix #1: Rename food_name to name in meal_items (if exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meal_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE meal_items RENAME COLUMN food_name TO name;
        RAISE NOTICE 'Renamed meal_items.food_name to name';
    ELSE
        RAISE NOTICE 'meal_items.name already exists, skipping rename';
    END IF;
END $$;

-- Fix #2: Rename food_name to name in template_items (if exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='template_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE template_items RENAME COLUMN food_name TO name;
        RAISE NOTICE 'Renamed template_items.food_name to name';
    ELSE
        RAISE NOTICE 'template_items.name already exists, skipping rename';
    END IF;
END $$;

-- Fix #3: Drop deprecated meal_time column (use consumed_at)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='meal_time'
    ) THEN
        ALTER TABLE meals DROP COLUMN meal_time;
        RAISE NOTICE 'Dropped deprecated meals.meal_time column';
    ELSE
        RAISE NOTICE 'meals.meal_time already dropped, skipping';
    END IF;
END $$;

-- Fix #4: Drop deprecated name column in meals (use meal_type)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='name'
    ) THEN
        ALTER TABLE meals DROP COLUMN name;
        RAISE NOTICE 'Dropped deprecated meals.name column';
    ELSE
        RAISE NOTICE 'meals.name already dropped, skipping';
    END IF;
END $$;

-- ========================================
-- PART 2: Database Triggers for Auto-Calculation
-- ========================================

-- Create trigger function to automatically calculate meal totals
-- This becomes the SINGLE source of truth for nutrition totals
CREATE OR REPLACE FUNCTION calculate_meal_totals()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Determine which meal_id to update
    DECLARE
        v_meal_id UUID;
    BEGIN
        -- On DELETE, use OLD.meal_id; otherwise use NEW.meal_id
        IF TG_OP = 'DELETE' THEN
            v_meal_id := OLD.meal_id;
        ELSE
            v_meal_id := NEW.meal_id;
        END IF;

        -- Update meal totals from sum of items
        UPDATE meals
        SET
            total_calories = COALESCE((
                SELECT SUM(calories)
                FROM meal_items
                WHERE meal_id = v_meal_id
            ), 0),

            total_protein_g = COALESCE((
                SELECT SUM(protein)  -- DB column is 'protein' not 'protein_g'
                FROM meal_items
                WHERE meal_id = v_meal_id
            ), 0),

            total_carbs_g = COALESCE((
                SELECT SUM(carbs)    -- DB column is 'carbs' not 'carbs_g'
                FROM meal_items
                WHERE meal_id = v_meal_id
            ), 0),

            total_fat_g = COALESCE((
                SELECT SUM(fat)      -- DB column is 'fat' not 'fat_g'
                FROM meal_items
                WHERE meal_id = v_meal_id
            ), 0),

            total_fiber_g = COALESCE((
                SELECT SUM(fiber)    -- DB column is 'fiber' not 'fiber_g'
                FROM meal_items
                WHERE meal_id = v_meal_id
            ), 0),

            updated_at = NOW()
        WHERE id = v_meal_id;

        -- Return appropriate value based on operation
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        ELSE
            RETURN NEW;
        END IF;
    END;
END;
$$;

COMMENT ON FUNCTION calculate_meal_totals IS
    'Automatically recalculate meal nutrition totals when meal_items change. '
    'This is the SINGLE source of truth for meal.total_* columns.';

-- Attach triggers to meal_items table
DROP TRIGGER IF EXISTS meal_items_insert_trigger ON meal_items;
CREATE TRIGGER meal_items_insert_trigger
    AFTER INSERT ON meal_items
    FOR EACH ROW
    EXECUTE FUNCTION calculate_meal_totals();

DROP TRIGGER IF EXISTS meal_items_update_trigger ON meal_items;
CREATE TRIGGER meal_items_update_trigger
    AFTER UPDATE ON meal_items
    FOR EACH ROW
    EXECUTE FUNCTION calculate_meal_totals();

DROP TRIGGER IF EXISTS meal_items_delete_trigger ON meal_items;
CREATE TRIGGER meal_items_delete_trigger
    AFTER DELETE ON meal_items
    FOR EACH ROW
    EXECUTE FUNCTION calculate_meal_totals();

-- ========================================
-- PART 3: Recalculate All Existing Totals
-- ========================================

-- Ensure all existing meals have correct totals calculated from their items
UPDATE meals m
SET
    total_calories = COALESCE((
        SELECT SUM(mi.calories)
        FROM meal_items mi
        WHERE mi.meal_id = m.id
    ), 0),

    total_protein_g = COALESCE((
        SELECT SUM(mi.protein)
        FROM meal_items mi
        WHERE mi.meal_id = m.id
    ), 0),

    total_carbs_g = COALESCE((
        SELECT SUM(mi.carbs)
        FROM meal_items mi
        WHERE mi.meal_id = m.id
    ), 0),

    total_fat_g = COALESCE((
        SELECT SUM(mi.fat)
        FROM meal_items mi
        WHERE mi.meal_id = m.id
    ), 0),

    total_fiber_g = COALESCE((
        SELECT SUM(mi.fiber)
        FROM meal_items mi
        WHERE mi.meal_id = m.id
    ), 0),

    updated_at = NOW()
WHERE id IN (
    -- Only update meals that have items (avoids unnecessary updates)
    SELECT DISTINCT meal_id FROM meal_items
);

-- Log recalculation stats
DO $$
DECLARE
    v_meals_updated INTEGER;
BEGIN
    GET DIAGNOSTICS v_meals_updated = ROW_COUNT;
    RAISE NOTICE 'Recalculated nutrition totals for % meals', v_meals_updated;
END $$;

COMMIT;

-- ========================================
-- PART 4: Validation Queries (for manual verification)
-- ========================================

-- Run these manually after migration to verify everything is correct:

-- Check for any meals where totals don't match items (should return 0 rows)
-- SELECT
--     m.id,
--     m.total_calories as stored_calories,
--     COALESCE(SUM(mi.calories), 0) as calculated_calories,
--     ABS(m.total_calories - COALESCE(SUM(mi.calories), 0)) as diff
-- FROM meals m
-- LEFT JOIN meal_items mi ON m.id = mi.meal_id
-- WHERE m.deleted_at IS NULL
-- GROUP BY m.id
-- HAVING ABS(m.total_calories - COALESCE(SUM(mi.calories), 0)) > 0.01
-- ORDER BY diff DESC;

-- Check for orphaned meal_items (should return 0 rows)
-- SELECT * FROM meal_items
-- WHERE meal_id NOT IN (SELECT id FROM meals);
