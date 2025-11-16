-- =====================================================
-- Lumen Nutrition Tracker - Indexes Migration
-- Version: 003
-- Description: Performance indexes and extensions
-- =====================================================

-- =====================================================
-- EXTENSIONS
-- =====================================================
-- Enable pg_trgm for fuzzy text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

COMMENT ON EXTENSION pg_trgm IS 'Trigram similarity for fuzzy text search';

-- =====================================================
-- USERS TABLE INDEXES
-- =====================================================
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

COMMENT ON INDEX idx_users_email IS 'Fast lookup by email for authentication';
COMMENT ON INDEX idx_users_created_at IS 'User registration analytics';

-- =====================================================
-- MEALS TABLE INDEXES
-- =====================================================
-- Primary query pattern: user's meals for a date range
CREATE INDEX idx_meals_user_time ON meals(user_id, meal_time DESC);

-- Date-based queries (respecting timezone)
CREATE INDEX idx_meals_user_date ON meals(user_id, DATE(meal_time AT TIME ZONE 'UTC'));

-- Recent meals for a user
CREATE INDEX idx_meals_created_at ON meals(user_id, created_at DESC);

COMMENT ON INDEX idx_meals_user_time IS 'Primary index for user meal timeline queries';
COMMENT ON INDEX idx_meals_user_date IS 'Daily meal aggregation queries';
COMMENT ON INDEX idx_meals_created_at IS 'Recent meals for user';

-- =====================================================
-- MEAL_ITEMS TABLE INDEXES
-- =====================================================
-- Most common: get all items for a meal
CREATE INDEX idx_meal_items_meal_id ON meal_items(meal_id);

-- Nutrition aggregation by meal
CREATE INDEX idx_meal_items_meal_nutrition ON meal_items(meal_id, calories, protein, carbs, fat);

-- Search meal items by food name
CREATE INDEX idx_meal_items_food_name ON meal_items USING gin(food_name gin_trgm_ops);

COMMENT ON INDEX idx_meal_items_meal_id IS 'Lookup items for a meal';
COMMENT ON INDEX idx_meal_items_meal_nutrition IS 'Optimize nutrition totals calculation';
COMMENT ON INDEX idx_meal_items_food_name IS 'Fuzzy search meal items by food name';

-- =====================================================
-- MEAL_FLAGS TABLE INDEXES
-- =====================================================
CREATE INDEX idx_meal_flags_meal_id ON meal_flags(meal_id);
CREATE INDEX idx_meal_flags_user_id ON meal_flags(user_id, created_at DESC);
CREATE INDEX idx_meal_flags_type ON meal_flags(flag_type, created_at DESC);

COMMENT ON INDEX idx_meal_flags_meal_id IS 'Check if meal has been flagged';
COMMENT ON INDEX idx_meal_flags_user_id IS 'User flagged meals history';
COMMENT ON INDEX idx_meal_flags_type IS 'Admin analysis of flag types';

-- =====================================================
-- WEIGHT_ENTRIES TABLE INDEXES
-- =====================================================
-- Timeline queries
CREATE INDEX idx_weight_entries_user_measured ON weight_entries(user_id, measured_at DESC);

-- Date-based lookups
CREATE INDEX idx_weight_entries_user_date ON weight_entries(user_id, DATE(measured_at AT TIME ZONE 'UTC'));

COMMENT ON INDEX idx_weight_entries_user_measured IS 'Weight tracking timeline';
COMMENT ON INDEX idx_weight_entries_user_date IS 'Daily weight lookup';

-- =====================================================
-- TEMPLATES TABLE INDEXES
-- =====================================================
CREATE INDEX idx_templates_user_id ON templates(user_id, created_at DESC);
CREATE INDEX idx_templates_user_name ON templates(user_id, name);

COMMENT ON INDEX idx_templates_user_id IS 'User templates list';
COMMENT ON INDEX idx_templates_user_name IS 'Template lookup by name';

-- =====================================================
-- TEMPLATE_ITEMS TABLE INDEXES
-- =====================================================
CREATE INDEX idx_template_items_template_id ON template_items(template_id);
CREATE INDEX idx_template_items_food_name ON template_items USING gin(food_name gin_trgm_ops);

COMMENT ON INDEX idx_template_items_template_id IS 'Get all items for a template';
COMMENT ON INDEX idx_template_items_food_name IS 'Search template items by food';

-- =====================================================
-- COMMON_FOODS TABLE INDEXES
-- =====================================================
-- Full-text search on food names (primary use case)
CREATE INDEX idx_common_foods_name_trgm ON common_foods USING gin(name gin_trgm_ops);

-- Exact name lookup
CREATE INDEX idx_common_foods_name ON common_foods(name);

-- Filter by category
CREATE INDEX idx_common_foods_category ON common_foods(category) WHERE category IS NOT NULL;

-- Verified foods first
CREATE INDEX idx_common_foods_verified ON common_foods(verified, name);

-- Search within category
CREATE INDEX idx_common_foods_category_name ON common_foods(category, name) WHERE category IS NOT NULL;

COMMENT ON INDEX idx_common_foods_name_trgm IS 'Fuzzy search on food names (primary search method)';
COMMENT ON INDEX idx_common_foods_name IS 'Exact name lookup';
COMMENT ON INDEX idx_common_foods_category IS 'Filter by food category';
COMMENT ON INDEX idx_common_foods_verified IS 'Prioritize verified foods in results';
COMMENT ON INDEX idx_common_foods_category_name IS 'Search within specific category';

-- =====================================================
-- AI_USAGE TABLE INDEXES
-- =====================================================
-- Rate limiting: count user requests in time window
CREATE INDEX idx_ai_usage_user_time ON ai_usage(user_id, created_at DESC);

-- Cost analysis by endpoint
CREATE INDEX idx_ai_usage_endpoint_time ON ai_usage(endpoint, created_at DESC);

-- Monthly cost aggregation
CREATE INDEX idx_ai_usage_user_month ON ai_usage(
    user_id,
    DATE_TRUNC('month', created_at)
);

COMMENT ON INDEX idx_ai_usage_user_time IS 'Rate limiting queries';
COMMENT ON INDEX idx_ai_usage_endpoint_time IS 'Cost analysis per endpoint';
COMMENT ON INDEX idx_ai_usage_user_month IS 'Monthly usage reports';

-- =====================================================
-- IDEMPOTENCY_KEYS TABLE INDEXES
-- =====================================================
-- Primary key already creates index on 'key'
-- Add index for cleanup of old keys
CREATE INDEX idx_idempotency_keys_created_at ON idempotency_keys(created_at);
CREATE INDEX idx_idempotency_keys_user_created ON idempotency_keys(user_id, created_at DESC);

COMMENT ON INDEX idx_idempotency_keys_created_at IS 'Cleanup old idempotency keys';
COMMENT ON INDEX idx_idempotency_keys_user_created IS 'User request history';

-- =====================================================
-- USER_DAILY_GOALS TABLE INDEXES
-- =====================================================
-- Primary key on user_id already provides index
-- No additional indexes needed (one row per user)

-- =====================================================
-- ANALYZE TABLES
-- =====================================================
-- Update statistics for query planner
ANALYZE users;
ANALYZE user_daily_goals;
ANALYZE meals;
ANALYZE meal_items;
ANALYZE meal_flags;
ANALYZE weight_entries;
ANALYZE templates;
ANALYZE template_items;
ANALYZE common_foods;
ANALYZE ai_usage;
ANALYZE idempotency_keys;

COMMENT ON SCHEMA public IS 'Lumen Nutrition Tracker - With Indexes (v003)';
