-- Migration: Create common_foods table
-- Version: 002
-- Description: Common foods database for quick meal logging
-- Dependencies: None

-- Common Foods Table
-- Stores verified nutrition data for common foods
CREATE TABLE IF NOT EXISTS common_foods (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    calories_per_100g DECIMAL(8,2) NOT NULL CHECK (calories_per_100g >= 0),
    protein_per_100g DECIMAL(6,2) NOT NULL CHECK (protein_per_100g >= 0),
    carbs_per_100g DECIMAL(6,2) NOT NULL CHECK (carbs_per_100g >= 0),
    fat_per_100g DECIMAL(6,2) NOT NULL CHECK (fat_per_100g >= 0),
    fiber_per_100g DECIMAL(6,2) NOT NULL DEFAULT 0 CHECK (fiber_per_100g >= 0),
    common_serving_name VARCHAR(100),
    common_serving_grams DECIMAL(8,2) CHECK (common_serving_grams > 0),
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_common_foods_name ON common_foods(name);
CREATE INDEX IF NOT EXISTS idx_common_foods_category ON common_foods(category);
CREATE INDEX IF NOT EXISTS idx_common_foods_name_lower ON common_foods(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_common_foods_verified ON common_foods(is_verified) WHERE is_verified = true;

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_common_foods_search ON common_foods USING gin(to_tsvector('english', name));

-- Apply updated_at trigger
CREATE TRIGGER update_common_foods_updated_at
    BEFORE UPDATE ON common_foods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable Row Level Security (RLS) - Read-only for all authenticated users
ALTER TABLE common_foods ENABLE ROW LEVEL SECURITY;

-- RLS Policy: All authenticated users can read
CREATE POLICY "Authenticated users can view common foods"
    ON common_foods FOR SELECT
    TO authenticated
    USING (true);

-- Only service role can modify (via backend API)
CREATE POLICY "Only service role can modify common foods"
    ON common_foods FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Comments
COMMENT ON TABLE common_foods IS 'Database of common foods with verified nutrition data';
COMMENT ON COLUMN common_foods.is_verified IS 'Whether nutrition data has been verified by nutritionist';
COMMENT ON COLUMN common_foods.common_serving_name IS 'Human-readable serving size (e.g., "1 medium apple")';
COMMENT ON COLUMN common_foods.common_serving_grams IS 'Weight in grams of common serving';

-- Note: The actual food data is loaded from internal/seed/data/common_foods.json
-- at application startup into an in-memory repository for fast searching.
-- This table structure is provided for future migration to database-backed storage.
