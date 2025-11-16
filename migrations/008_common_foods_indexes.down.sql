-- Rollback: Remove additional common_foods indexes
-- Version: 011

DROP INDEX IF EXISTS idx_common_foods_search;
DROP INDEX IF EXISTS idx_common_foods_verified;
DROP INDEX IF EXISTS idx_common_foods_name_lower;
