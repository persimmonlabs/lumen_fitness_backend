-- Migration: Create meal_flags table for quality checking system
-- Created: 2025-01-15
-- Description: Adds automated meal quality flagging system

-- Create enum for flag types
CREATE TYPE flag_type AS ENUM (
    'unusual_portion',
    'macro_mismatch',
    'duplicate',
    'low_confidence'
);

-- Create enum for severity levels
CREATE TYPE flag_severity AS ENUM (
    'low',
    'medium',
    'high'
);

-- Create meal_flags table
CREATE TABLE meal_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flag_type flag_type NOT NULL,
    severity flag_severity NOT NULL,
    description TEXT NOT NULL,
    details JSONB,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX idx_meal_flags_meal_id ON meal_flags(meal_id);
CREATE INDEX idx_meal_flags_user_id ON meal_flags(user_id);
CREATE INDEX idx_meal_flags_type ON meal_flags(flag_type);
CREATE INDEX idx_meal_flags_severity ON meal_flags(severity);
CREATE INDEX idx_meal_flags_resolved ON meal_flags(resolved) WHERE resolved = FALSE;
CREATE INDEX idx_meal_flags_created_at ON meal_flags(created_at DESC);
CREATE INDEX idx_meal_flags_user_created ON meal_flags(user_id, created_at DESC);

-- Create composite index for common queries
CREATE INDEX idx_meal_flags_unresolved ON meal_flags(user_id, resolved, severity, created_at DESC)
    WHERE resolved = FALSE;

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_meal_flags_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_meal_flags_updated_at
    BEFORE UPDATE ON meal_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_meal_flags_updated_at();

-- Add constraint to ensure resolved_at is set when resolved
ALTER TABLE meal_flags
    ADD CONSTRAINT check_resolved_at CHECK (
        (resolved = TRUE AND resolved_at IS NOT NULL) OR
        (resolved = FALSE AND resolved_at IS NULL)
    );

-- Add comment to table
COMMENT ON TABLE meal_flags IS 'Automated quality flags for meal entries detected by nightly job';

-- Add comments to columns
COMMENT ON COLUMN meal_flags.flag_type IS 'Type of quality issue detected';
COMMENT ON COLUMN meal_flags.severity IS 'Severity level of the flag (low, medium, high)';
COMMENT ON COLUMN meal_flags.description IS 'Human-readable description of the issue';
COMMENT ON COLUMN meal_flags.details IS 'JSON object with additional context and metadata';
COMMENT ON COLUMN meal_flags.resolved IS 'Whether the flag has been reviewed and resolved';
COMMENT ON COLUMN meal_flags.resolved_at IS 'Timestamp when flag was resolved';
COMMENT ON COLUMN meal_flags.resolved_by IS 'User who resolved the flag';

-- Create view for unresolved high priority flags
CREATE OR REPLACE VIEW high_priority_meal_flags AS
SELECT
    f.id,
    f.meal_id,
    f.user_id,
    u.email as user_email,
    u.name as user_name,
    m.name as meal_name,
    m.logged_at as meal_logged_at,
    f.flag_type,
    f.severity,
    f.description,
    f.details,
    f.created_at
FROM meal_flags f
JOIN users u ON f.user_id = u.id
JOIN meals m ON f.meal_id = m.id
WHERE f.resolved = FALSE
  AND f.severity IN ('high', 'medium')
  AND m.deleted_at IS NULL
ORDER BY
    CASE f.severity
        WHEN 'high' THEN 1
        WHEN 'medium' THEN 2
        ELSE 3
    END,
    f.created_at DESC;

COMMENT ON VIEW high_priority_meal_flags IS 'Unresolved medium and high severity meal flags requiring review';

-- Create function to get flag summary for a user
CREATE OR REPLACE FUNCTION get_user_flag_summary(p_user_id UUID, p_days INTEGER DEFAULT 30)
RETURNS TABLE(
    flag_type flag_type,
    severity flag_severity,
    count BIGINT,
    latest_flag TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        f.flag_type,
        f.severity,
        COUNT(*) as count,
        MAX(f.created_at) as latest_flag
    FROM meal_flags f
    WHERE f.user_id = p_user_id
      AND f.created_at > NOW() - (p_days || ' days')::INTERVAL
      AND f.resolved = FALSE
    GROUP BY f.flag_type, f.severity
    ORDER BY count DESC;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_user_flag_summary IS 'Get summary of unresolved flags for a user over specified days';

-- Create function to resolve a flag
CREATE OR REPLACE FUNCTION resolve_meal_flag(
    p_flag_id UUID,
    p_resolved_by UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE meal_flags
    SET
        resolved = TRUE,
        resolved_at = NOW(),
        resolved_by = p_resolved_by,
        updated_at = NOW()
    WHERE id = p_flag_id
      AND resolved = FALSE;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION resolve_meal_flag IS 'Mark a flag as resolved with timestamp and resolver';

-- Insert sample thresholds configuration (optional reference data)
CREATE TABLE meal_flagging_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE meal_flagging_config IS 'Configuration for meal flagging job thresholds';

-- Insert default configuration
INSERT INTO meal_flagging_config (config_key, config_value, description) VALUES
(
    'portion_thresholds',
    '{
        "chicken": {"max_quantity": 450, "unit": "g", "max_calories": 1000},
        "beef": {"max_quantity": 450, "unit": "g", "max_calories": 1000},
        "pork": {"max_quantity": 450, "unit": "g", "max_calories": 1000},
        "fish": {"max_quantity": 500, "unit": "g", "max_calories": 1000},
        "rice": {"max_quantity": 300, "unit": "g", "max_calories": 1000},
        "pasta": {"max_quantity": 300, "unit": "g", "max_calories": 1000},
        "bread": {"max_quantity": 400, "unit": "g", "max_calories": 1000},
        "oil": {"max_quantity": 30, "unit": "ml", "max_calories": 300},
        "butter": {"max_quantity": 50, "unit": "g", "max_calories": 400},
        "cheese": {"max_quantity": 200, "unit": "g", "max_calories": 800},
        "nuts": {"max_quantity": 100, "unit": "g", "max_calories": 600}
    }',
    'Maximum portion sizes for common food categories'
),
(
    'job_config',
    '{
        "enabled": true,
        "schedule_time": "02:00",
        "batch_size": 1000,
        "lookback_days": 1,
        "macro_tolerance_percent": 10.0,
        "confidence_threshold": 0.7,
        "duplicate_window_minutes": 30,
        "duplicate_similarity_percent": 80.0
    }',
    'Default job configuration for meal flagging'
);

-- Grant permissions (adjust roles as needed)
-- GRANT SELECT ON meal_flags TO app_user;
-- GRANT INSERT, UPDATE ON meal_flags TO app_user;
-- GRANT SELECT ON high_priority_meal_flags TO app_user;
-- GRANT EXECUTE ON FUNCTION get_user_flag_summary TO app_user;
-- GRANT EXECUTE ON FUNCTION resolve_meal_flag TO app_user;

-- Rollback script (commented out, for reference)
/*
DROP VIEW IF EXISTS high_priority_meal_flags;
DROP FUNCTION IF EXISTS get_user_flag_summary(UUID, INTEGER);
DROP FUNCTION IF EXISTS resolve_meal_flag(UUID, UUID);
DROP FUNCTION IF EXISTS update_meal_flags_updated_at();
DROP TABLE IF EXISTS meal_flagging_config;
DROP TABLE IF EXISTS meal_flags;
DROP TYPE IF EXISTS flag_severity;
DROP TYPE IF EXISTS flag_type;
*/
