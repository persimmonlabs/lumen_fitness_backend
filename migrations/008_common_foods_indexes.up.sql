-- Migration: Add indexes to existing common_foods table
-- Version: 002
-- Description: Performance indexes for common_foods table created in 001_initial_schema
-- Dependencies: 001_initial_schema.up.sql
--
-- NOTE: The common_foods table was already created in 001_initial_schema.up.sql
-- This migration just adds additional indexes for the in-memory repository search functionality

-- Additional indexes for performance (table already exists from 001_initial_schema)
CREATE INDEX IF NOT EXISTS idx_common_foods_name_lower ON common_foods(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_common_foods_verified ON common_foods(verified) WHERE verified = true;

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_common_foods_search ON common_foods USING gin(to_tsvector('english', name));

-- Comments for clarity
COMMENT ON TABLE common_foods IS 'Database of common foods with verified nutrition data (loaded from JSON at startup)';
COMMENT ON COLUMN common_foods.verified IS 'Whether nutrition data has been verified by nutritionist';

-- Note: The actual food data is loaded from internal/seed/data/common_foods.json
-- at application startup into an in-memory repository for fast searching.
-- The database table structure is available for future database-backed storage if needed.
