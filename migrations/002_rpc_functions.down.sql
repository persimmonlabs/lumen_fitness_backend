-- =====================================================
-- Lumen Nutrition Tracker - Rollback RPC Functions
-- Version: 002
-- Description: Drop all database functions
-- =====================================================

-- Revoke permissions first
REVOKE EXECUTE ON FUNCTION create_meal_with_items FROM authenticated;
REVOKE EXECUTE ON FUNCTION get_daily_nutrition FROM authenticated;
REVOKE EXECUTE ON FUNCTION create_template_from_meal FROM authenticated;
REVOKE EXECUTE ON FUNCTION search_common_foods FROM authenticated;
REVOKE EXECUTE ON FUNCTION get_nutrition_trends FROM authenticated;

-- Drop functions
DROP FUNCTION IF EXISTS create_meal_with_items(UUID, TEXT, TIMESTAMPTZ, TEXT, JSONB);
DROP FUNCTION IF EXISTS get_daily_nutrition(UUID, DATE, TEXT);
DROP FUNCTION IF EXISTS create_template_from_meal(UUID, UUID, TEXT, TEXT);
DROP FUNCTION IF EXISTS search_common_foods(TEXT, INTEGER);
DROP FUNCTION IF EXISTS get_nutrition_trends(UUID, DATE, DATE, TEXT);
