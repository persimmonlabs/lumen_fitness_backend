-- Rollback: Remove nutrition columns from meals table

-- Drop indexes
DROP INDEX IF EXISTS idx_meals_not_deleted;
DROP INDEX IF EXISTS idx_meals_consumed_at;
DROP INDEX IF EXISTS idx_meals_meal_type;

-- Drop constraints
ALTER TABLE meals DROP CONSTRAINT IF EXISTS positive_total_fiber;
ALTER TABLE meals DROP CONSTRAINT IF EXISTS positive_total_fat;
ALTER TABLE meals DROP CONSTRAINT IF EXISTS positive_total_carbs;
ALTER TABLE meals DROP CONSTRAINT IF EXISTS positive_total_protein;
ALTER TABLE meals DROP CONSTRAINT IF EXISTS positive_total_calories;
ALTER TABLE meals DROP CONSTRAINT IF EXISTS valid_meal_type;

-- Drop columns
ALTER TABLE meals DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE meals DROP COLUMN IF EXISTS photos;
ALTER TABLE meals DROP COLUMN IF EXISTS total_fiber_g;
ALTER TABLE meals DROP COLUMN IF EXISTS total_fat_g;
ALTER TABLE meals DROP COLUMN IF EXISTS total_carbs_g;
ALTER TABLE meals DROP COLUMN IF EXISTS total_protein_g;
ALTER TABLE meals DROP COLUMN IF EXISTS total_calories;
ALTER TABLE meals DROP COLUMN IF EXISTS consumed_at;
ALTER TABLE meals DROP COLUMN IF EXISTS meal_type;
