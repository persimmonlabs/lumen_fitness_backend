-- =====================================================
-- Lumen Nutrition Tracker - Onboarding Tracking Rollback
-- Version: 009
-- Description: Remove onboarding tracking fields
-- =====================================================

-- Drop index
DROP INDEX IF EXISTS idx_users_onboarding;

-- Remove columns
ALTER TABLE public.users
DROP COLUMN IF EXISTS onboarding_completed,
DROP COLUMN IF EXISTS onboarding_completed_at;

-- Restore previous schema version comment
COMMENT ON SCHEMA public IS 'Lumen Nutrition Tracker - With Draft Status (v008)';
