-- =====================================================
-- Lumen Nutrition Tracker - Onboarding Tracking Migration
-- Version: 009
-- Description: Add onboarding completion tracking to users
-- =====================================================

-- Add onboarding tracking fields to users table
ALTER TABLE public.users
ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMP WITH TIME ZONE;

-- Add index for efficient queries of users who haven't completed onboarding
CREATE INDEX IF NOT EXISTS idx_users_onboarding
ON users(onboarding_completed)
WHERE onboarding_completed = false;

-- Add comments for documentation
COMMENT ON COLUMN users.onboarding_completed IS 'Whether user has completed initial onboarding flow (goals, weight, etc)';
COMMENT ON COLUMN users.onboarding_completed_at IS 'Timestamp when user completed onboarding';
COMMENT ON INDEX idx_users_onboarding IS 'Find users who have not completed onboarding';

-- Update schema version comment
COMMENT ON SCHEMA public IS 'Lumen Nutrition Tracker - With Onboarding Tracking (v009)';
