-- =====================================================
-- Lumen Nutrition Tracker - Add Draft Status Tracking
-- Version: 008
-- Description: Add draft_status and is_draft columns for asynchronous AI meal parsing
-- =====================================================

-- Add draft-related columns to meals table
ALTER TABLE meals ADD COLUMN IF NOT EXISTS is_draft BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS draft_status TEXT DEFAULT NULL;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS draft_error TEXT DEFAULT NULL;

-- Add constraint for valid draft statuses
ALTER TABLE meals ADD CONSTRAINT valid_draft_status
    CHECK (draft_status IS NULL OR draft_status IN ('analyzing', 'ready', 'error'));

-- Add partial index for querying draft meals efficiently
CREATE INDEX IF NOT EXISTS idx_meals_draft_status
    ON meals(user_id, draft_status)
    WHERE is_draft = TRUE;

-- Add index for draft meals by creation time (for cleanup/monitoring)
CREATE INDEX IF NOT EXISTS idx_meals_draft_created
    ON meals(created_at)
    WHERE is_draft = TRUE;

-- Comments
COMMENT ON COLUMN meals.is_draft IS 'Whether this meal is a draft (not yet confirmed by user)';
COMMENT ON COLUMN meals.draft_status IS 'Status of AI processing: analyzing, ready, or error';
COMMENT ON COLUMN meals.draft_error IS 'Error message if draft_status is error';

-- Update RLS policies to include draft meals
-- Users can view their own draft meals
-- (Existing policies already cover this via user_id check)
