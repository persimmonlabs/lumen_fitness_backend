-- =====================================================
-- Lumen Nutrition Tracker - Rollback Draft Status Tracking
-- Version: 008
-- Description: Remove draft_status and is_draft columns
-- =====================================================

-- Drop indexes
DROP INDEX IF EXISTS idx_meals_draft_created;
DROP INDEX IF EXISTS idx_meals_draft_status;

-- Drop constraint
ALTER TABLE meals DROP CONSTRAINT IF EXISTS valid_draft_status;

-- Drop columns
ALTER TABLE meals DROP COLUMN IF EXISTS draft_error;
ALTER TABLE meals DROP COLUMN IF EXISTS draft_status;
ALTER TABLE meals DROP COLUMN IF EXISTS is_draft;
