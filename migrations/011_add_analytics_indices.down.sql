-- =====================================================
-- Rollback: Remove analytics performance indices
-- =====================================================

DROP INDEX IF EXISTS idx_meals_user_consumed_date;
DROP INDEX IF EXISTS idx_meals_user_consumed_type;
DROP INDEX IF EXISTS idx_meals_user_consumed_desc;
DROP INDEX IF EXISTS idx_weight_entries_user_measured;
DROP INDEX IF EXISTS idx_weight_entries_date_range;
DROP INDEX IF EXISTS idx_user_daily_goals_user;
DROP INDEX IF EXISTS idx_meal_items_meal;
DROP INDEX IF EXISTS idx_meal_items_meal_calories;
