-- =====================================================
-- Lumen Nutrition Tracker - Rollback Single Source of Truth
-- Version: 012
-- Description: Safely rollback migration 012 changes
-- =====================================================

BEGIN;

-- ========================================
-- PART 1: Remove Database Triggers
-- ========================================

DROP TRIGGER IF EXISTS meal_items_insert_trigger ON meal_items;
DROP TRIGGER IF EXISTS meal_items_update_trigger ON meal_items;
DROP TRIGGER IF EXISTS meal_items_delete_trigger ON meal_items;

DROP FUNCTION IF EXISTS calculate_meal_totals();

RAISE NOTICE 'Removed database triggers for meal total calculation';

-- ========================================
-- PART 2: Restore Schema (Reverse of UP migration)
-- ========================================

-- Restore name column in meals (if it was dropped)
-- NOTE: This will be NULL for all rows - manual data migration needed if rolling back
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='name'
    ) THEN
        ALTER TABLE meals ADD COLUMN name TEXT;
        RAISE NOTICE 'Restored meals.name column (values will be NULL)';
    END IF;
END $$;

-- Restore meal_time column in meals (if it was dropped)
-- NOTE: Data loss - consumed_at values won't be copied back
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meals' AND column_name='meal_time'
    ) THEN
        ALTER TABLE meals ADD COLUMN meal_time TIMESTAMPTZ;
        -- Copy consumed_at values to meal_time
        UPDATE meals SET meal_time = consumed_at WHERE consumed_at IS NOT NULL;
        -- Make NOT NULL after copying data
        ALTER TABLE meals ALTER COLUMN meal_time SET NOT NULL;
        RAISE NOTICE 'Restored meals.meal_time column from consumed_at values';
    END IF;
END $$;

-- Rename name back to food_name in meal_items (if it was renamed)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meal_items' AND column_name='name'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='meal_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE meal_items RENAME COLUMN name TO food_name;
        RAISE NOTICE 'Renamed meal_items.name back to food_name';
    END IF;
END $$;

-- Rename name back to food_name in template_items (if it was renamed)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='template_items' AND column_name='name'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='template_items' AND column_name='food_name'
    ) THEN
        ALTER TABLE template_items RENAME COLUMN name TO food_name;
        RAISE NOTICE 'Renamed template_items.name back to food_name';
    END IF;
END $$;

-- ========================================
-- PART 3: Warning About Data Loss
-- ========================================

RAISE WARNING 'Migration 012 rolled back. NOTE: Some data may need manual restoration:';
RAISE WARNING '  - meals.name column is now NULL (was dropped in UP migration)';
RAISE WARNING '  - Nutrition totals are NO LONGER auto-calculated';
RAISE WARNING '  - You may need to manually recalculate totals if data was modified';

COMMIT;
