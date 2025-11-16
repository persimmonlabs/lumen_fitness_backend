-- =====================================================
-- Lumen Nutrition Tracker - Analytics Performance Indices
-- Version: 011
-- Description: Add indices to optimize analytics queries
-- =====================================================

-- =====================================================
-- MEALS TABLE INDICES
-- =====================================================

-- Index for date-based analytics queries
-- Supports: GetDailyNutrition, GetLoggingStreak, date range filters
CREATE INDEX IF NOT EXISTS idx_meals_user_consumed_date
    ON meals(user_id, DATE(consumed_at AT TIME ZONE 'UTC'))
    WHERE deleted_at IS NULL;

-- Index for date range queries with meal type filtering
-- Supports: ListMealsByUser with filters
CREATE INDEX IF NOT EXISTS idx_meals_user_consumed_type
    ON meals(user_id, consumed_at, meal_type)
    WHERE deleted_at IS NULL;

-- Index for timezone-aware date queries
-- Supports: GetDailyAnalytics with timezone conversion
CREATE INDEX IF NOT EXISTS idx_meals_user_consumed_desc
    ON meals(user_id, consumed_at DESC)
    WHERE deleted_at IS NULL;

-- =====================================================
-- WEIGHT ENTRIES TABLE INDICES
-- =====================================================

-- Index for weight trajectory calculations
-- Supports: GetWeightTrajectory with date range lookups
CREATE INDEX IF NOT EXISTS idx_weight_entries_user_measured
    ON weight_entries(user_id, measured_at DESC);

-- Index for date range queries on weight data
-- Supports: GetByDateRange in weight repository
CREATE INDEX IF NOT EXISTS idx_weight_entries_date_range
    ON weight_entries(user_id, measured_at);

-- =====================================================
-- USER DAILY GOALS TABLE INDEX
-- =====================================================

-- Index for user goals lookups in analytics
-- Supports: GetDailyAnalytics, GetWeeklyAnalytics goal comparisons
CREATE INDEX IF NOT EXISTS idx_user_daily_goals_user
    ON user_daily_goals(user_id);

-- =====================================================
-- MEAL ITEMS TABLE INDEX
-- =====================================================

-- Index for aggregating nutrition by meal
-- Supports: JOIN operations in analytics queries
CREATE INDEX IF NOT EXISTS idx_meal_items_meal
    ON meal_items(meal_id);

-- Index for nutrition totals calculations
-- Supports: SUM aggregations in GetDailyNutrition
CREATE INDEX IF NOT EXISTS idx_meal_items_meal_calories
    ON meal_items(meal_id, calories, protein, carbs, fat);

-- =====================================================
-- COMMENTS
-- =====================================================

COMMENT ON INDEX idx_meals_user_consumed_date IS 'Optimizes date-based analytics queries on meals table';
COMMENT ON INDEX idx_meals_user_consumed_type IS 'Optimizes filtered meal listing by user, date, and type';
COMMENT ON INDEX idx_meals_user_consumed_desc IS 'Optimizes descending date queries for recent meals';
COMMENT ON INDEX idx_weight_entries_user_measured IS 'Optimizes weight trajectory queries with date ordering';
COMMENT ON INDEX idx_weight_entries_date_range IS 'Optimizes weight date range queries';
COMMENT ON INDEX idx_user_daily_goals_user IS 'Optimizes user goals lookup in analytics';
COMMENT ON INDEX idx_meal_items_meal IS 'Optimizes meal items JOIN operations';
COMMENT ON INDEX idx_meal_items_meal_calories IS 'Optimizes nutrition aggregation queries';
