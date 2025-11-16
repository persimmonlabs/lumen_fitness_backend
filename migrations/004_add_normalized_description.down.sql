-- Drop index
DROP INDEX IF EXISTS idx_meals_normalized_desc;

-- Remove normalized_description column from meals table
ALTER TABLE meals DROP COLUMN IF EXISTS normalized_description;
