-- =====================================================
-- Lumen Nutrition Tracker - Rollback Initial Schema
-- Version: 001
-- Description: Drop all tables and extensions
-- =====================================================

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS idempotency_keys CASCADE;
DROP TABLE IF EXISTS ai_usage CASCADE;
DROP TABLE IF EXISTS meal_flags CASCADE;
DROP TABLE IF EXISTS template_items CASCADE;
DROP TABLE IF EXISTS templates CASCADE;
DROP TABLE IF EXISTS weight_entries CASCADE;
DROP TABLE IF EXISTS meal_items CASCADE;
DROP TABLE IF EXISTS meals CASCADE;
DROP TABLE IF EXISTS common_foods CASCADE;
DROP TABLE IF EXISTS user_daily_goals CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Note: We don't drop uuid-ossp extension as it might be used by other schemas
-- DROP EXTENSION IF EXISTS "uuid-ossp";

COMMENT ON SCHEMA public IS NULL;
