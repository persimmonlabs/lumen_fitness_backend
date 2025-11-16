-- =====================================================
-- Lumen Nutrition Tracker - Rollback Indexes
-- Version: 003
-- Description: Drop all indexes and extensions
-- =====================================================

-- =====================================================
-- DROP INDEXES
-- =====================================================

-- Users indexes
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_created_at;

-- Meals indexes
DROP INDEX IF EXISTS idx_meals_user_time;
DROP INDEX IF EXISTS idx_meals_user_date;
DROP INDEX IF EXISTS idx_meals_created_at;

-- Meal items indexes
DROP INDEX IF EXISTS idx_meal_items_meal_id;
DROP INDEX IF EXISTS idx_meal_items_meal_nutrition;
DROP INDEX IF EXISTS idx_meal_items_food_name;

-- Meal flags indexes
DROP INDEX IF EXISTS idx_meal_flags_meal_id;
DROP INDEX IF EXISTS idx_meal_flags_user_id;
DROP INDEX IF EXISTS idx_meal_flags_type;

-- Weight entries indexes
DROP INDEX IF EXISTS idx_weight_entries_user_measured;
DROP INDEX IF EXISTS idx_weight_entries_user_date;

-- Templates indexes
DROP INDEX IF EXISTS idx_templates_user_id;
DROP INDEX IF EXISTS idx_templates_user_name;

-- Template items indexes
DROP INDEX IF EXISTS idx_template_items_template_id;
DROP INDEX IF EXISTS idx_template_items_food_name;

-- Common foods indexes
DROP INDEX IF EXISTS idx_common_foods_name_trgm;
DROP INDEX IF EXISTS idx_common_foods_name;
DROP INDEX IF EXISTS idx_common_foods_category;
DROP INDEX IF EXISTS idx_common_foods_verified;
DROP INDEX IF EXISTS idx_common_foods_category_name;

-- AI usage indexes
DROP INDEX IF EXISTS idx_ai_usage_user_time;
DROP INDEX IF EXISTS idx_ai_usage_endpoint_time;
DROP INDEX IF EXISTS idx_ai_usage_user_month;

-- Idempotency keys indexes
DROP INDEX IF EXISTS idx_idempotency_keys_created_at;
DROP INDEX IF EXISTS idx_idempotency_keys_user_created;

-- =====================================================
-- DROP EXTENSIONS
-- =====================================================
-- Note: Be careful dropping extensions that might be used elsewhere
DROP EXTENSION IF EXISTS pg_trgm;

COMMENT ON SCHEMA public IS 'Lumen Nutrition Tracker - Indexes Removed';
