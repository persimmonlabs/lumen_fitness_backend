-- =====================================================
-- Lumen Nutrition Tracker - Add Nutrition Columns to Meals
-- Version: 009
-- Description: Add meal_type, nutrition totals, photos, consumed_at, and soft delete support
-- =====================================================

-- Add meal_type column
ALTER TABLE meals ADD COLUMN IF NOT EXISTS meal_type TEXT;

-- Add consumed_at column (replaces meal_time conceptually)
ALTER TABLE meals ADD COLUMN IF NOT EXISTS consumed_at TIMESTAMPTZ;

-- Copy meal_time to consumed_at for existing records
UPDATE meals SET consumed_at = meal_time WHERE consumed_at IS NULL;

-- Add nutrition total columns
ALTER TABLE meals ADD COLUMN IF NOT EXISTS total_calories DECIMAL(7,1) DEFAULT 0;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS total_protein_g DECIMAL(6,1) DEFAULT 0;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS total_carbs_g DECIMAL(6,1) DEFAULT 0;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS total_fat_g DECIMAL(6,1) DEFAULT 0;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS total_fiber_g DECIMAL(5,1) DEFAULT 0;

-- Add photos column (JSONB array)
ALTER TABLE meals ADD COLUMN IF NOT EXISTS photos JSONB DEFAULT '[]'::jsonb;

-- Add soft delete support
ALTER TABLE meals ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Add constraints
ALTER TABLE meals ADD CONSTRAINT valid_meal_type
    CHECK (meal_type IS NULL OR meal_type IN ('breakfast', 'lunch', 'dinner', 'snack'));

ALTER TABLE meals ADD CONSTRAINT positive_total_calories
    CHECK (total_calories >= 0 AND total_calories <= 99999);

ALTER TABLE meals ADD CONSTRAINT positive_total_protein
    CHECK (total_protein_g >= 0 AND total_protein_g <= 9999);

ALTER TABLE meals ADD CONSTRAINT positive_total_carbs
    CHECK (total_carbs_g >= 0 AND total_carbs_g <= 9999);

ALTER TABLE meals ADD CONSTRAINT positive_total_fat
    CHECK (total_fat_g >= 0 AND total_fat_g <= 9999);

ALTER TABLE meals ADD CONSTRAINT positive_total_fiber
    CHECK (total_fiber_g >= 0 AND total_fiber_g <= 9999);

-- Add indexes for common queries
CREATE INDEX IF NOT EXISTS idx_meals_meal_type
    ON meals(user_id, meal_type)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meals_consumed_at
    ON meals(user_id, consumed_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meals_not_deleted
    ON meals(user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Comments
COMMENT ON COLUMN meals.meal_type IS 'Type of meal: breakfast, lunch, dinner, or snack';
COMMENT ON COLUMN meals.consumed_at IS 'UTC timestamp when meal was consumed';
COMMENT ON COLUMN meals.total_calories IS 'Total calories for the meal (sum of all items)';
COMMENT ON COLUMN meals.total_protein_g IS 'Total protein in grams (sum of all items)';
COMMENT ON COLUMN meals.total_carbs_g IS 'Total carbohydrates in grams (sum of all items)';
COMMENT ON COLUMN meals.total_fat_g IS 'Total fat in grams (sum of all items)';
COMMENT ON COLUMN meals.total_fiber_g IS 'Total fiber in grams (sum of all items)';
COMMENT ON COLUMN meals.photos IS 'Array of photo URLs for the meal';
COMMENT ON COLUMN meals.deleted_at IS 'Soft delete timestamp (NULL if not deleted)';
