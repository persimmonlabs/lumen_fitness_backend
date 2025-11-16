-- =====================================================
-- Lumen Nutrition Tracker - Initial Schema Migration
-- Version: 001
-- Description: Core tables with RLS policies
-- =====================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- USERS TABLE
-- =====================================================
-- Extends Supabase auth.users with profile information
CREATE TABLE users (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT valid_timezone CHECK (timezone ~ '^[A-Za-z/_]+$')
);

-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- RLS Policies for users
CREATE POLICY "Users can view own profile"
    ON users FOR SELECT
    USING (auth.uid() = id);

CREATE POLICY "Users can update own profile"
    ON users FOR UPDATE
    USING (auth.uid() = id);

CREATE POLICY "Users can insert own profile"
    ON users FOR INSERT
    WITH CHECK (auth.uid() = id);

-- Comments
COMMENT ON TABLE users IS 'User profiles extending Supabase auth.users';
COMMENT ON COLUMN users.timezone IS 'IANA timezone identifier for user local time';

-- =====================================================
-- USER DAILY GOALS TABLE
-- =====================================================
CREATE TABLE user_daily_goals (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    calories DECIMAL(7,1) NOT NULL DEFAULT 2000,
    protein DECIMAL(6,1) NOT NULL DEFAULT 150,
    carbs DECIMAL(6,1) NOT NULL DEFAULT 200,
    fat DECIMAL(6,1) NOT NULL DEFAULT 65,
    fiber DECIMAL(5,1) DEFAULT 25,
    sugar DECIMAL(6,1),
    sodium DECIMAL(7,1),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT positive_calories CHECK (calories > 0 AND calories <= 10000),
    CONSTRAINT positive_protein CHECK (protein >= 0 AND protein <= 1000),
    CONSTRAINT positive_carbs CHECK (carbs >= 0 AND carbs <= 2000),
    CONSTRAINT positive_fat CHECK (fat >= 0 AND fat <= 500),
    CONSTRAINT positive_fiber CHECK (fiber IS NULL OR (fiber >= 0 AND fiber <= 200)),
    CONSTRAINT positive_sugar CHECK (sugar IS NULL OR (sugar >= 0 AND sugar <= 500)),
    CONSTRAINT positive_sodium CHECK (sodium IS NULL OR (sodium >= 0 AND sodium <= 10000))
);

-- Enable RLS
ALTER TABLE user_daily_goals ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own goals"
    ON user_daily_goals FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can manage own goals"
    ON user_daily_goals FOR ALL
    USING (auth.uid() = user_id);

COMMENT ON TABLE user_daily_goals IS 'User daily nutrition goals';

-- =====================================================
-- MEALS TABLE
-- =====================================================
CREATE TABLE meals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    meal_time TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT meal_name_length CHECK (char_length(name) >= 1 AND char_length(name) <= 100),
    CONSTRAINT notes_length CHECK (notes IS NULL OR char_length(notes) <= 1000),
    CONSTRAINT future_meal_time CHECK (meal_time <= NOW() + INTERVAL '7 days')
);

-- Enable RLS
ALTER TABLE meals ENABLE ROW LEVEL SECURITY;

-- RLS Policies for meals
CREATE POLICY "Users can view own meals"
    ON meals FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert own meals"
    ON meals FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own meals"
    ON meals FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own meals"
    ON meals FOR DELETE
    USING (auth.uid() = user_id);

-- Comments
COMMENT ON TABLE meals IS 'User meal entries';
COMMENT ON COLUMN meals.meal_time IS 'UTC timestamp when meal was consumed';
COMMENT ON COLUMN meals.name IS 'User-defined meal name (e.g., Breakfast, Lunch)';

-- =====================================================
-- MEAL_ITEMS TABLE
-- =====================================================
-- Denormalized nutrition data for performance
CREATE TABLE meal_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    food_name TEXT NOT NULL,
    quantity DECIMAL(8,2) NOT NULL,
    unit TEXT NOT NULL,

    -- Denormalized nutrition (from AI or common_foods)
    calories DECIMAL(7,1) NOT NULL,
    protein DECIMAL(6,1) NOT NULL,
    carbs DECIMAL(6,1) NOT NULL,
    fat DECIMAL(6,1) NOT NULL,
    fiber DECIMAL(5,1),
    sugar DECIMAL(6,1),
    sodium DECIMAL(7,1),

    -- Metadata
    source TEXT NOT NULL DEFAULT 'ai',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT food_name_length CHECK (char_length(food_name) >= 1 AND char_length(food_name) <= 200),
    CONSTRAINT positive_quantity CHECK (quantity > 0 AND quantity <= 99999),
    CONSTRAINT valid_unit CHECK (char_length(unit) >= 1 AND char_length(unit) <= 50),
    CONSTRAINT positive_calories CHECK (calories >= 0 AND calories <= 9999),
    CONSTRAINT positive_protein CHECK (protein >= 0 AND protein <= 999),
    CONSTRAINT positive_carbs CHECK (carbs >= 0 AND carbs <= 999),
    CONSTRAINT positive_fat CHECK (fat >= 0 AND fat <= 999),
    CONSTRAINT positive_fiber CHECK (fiber IS NULL OR (fiber >= 0 AND fiber <= 999)),
    CONSTRAINT positive_sugar CHECK (sugar IS NULL OR (sugar >= 0 AND sugar <= 999)),
    CONSTRAINT positive_sodium CHECK (sodium IS NULL OR (sodium >= 0 AND sodium <= 99999)),
    CONSTRAINT valid_source CHECK (source IN ('ai', 'common_foods', 'manual', 'template'))
);

-- Enable RLS
ALTER TABLE meal_items ENABLE ROW LEVEL SECURITY;

-- RLS Policies for meal_items (inherit from meals)
CREATE POLICY "Users can view own meal items"
    ON meal_items FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM meals
            WHERE meals.id = meal_items.meal_id
            AND meals.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can insert own meal items"
    ON meal_items FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM meals
            WHERE meals.id = meal_items.meal_id
            AND meals.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can update own meal items"
    ON meal_items FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM meals
            WHERE meals.id = meal_items.meal_id
            AND meals.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can delete own meal items"
    ON meal_items FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM meals
            WHERE meals.id = meal_items.meal_id
            AND meals.user_id = auth.uid()
        )
    );

-- Comments
COMMENT ON TABLE meal_items IS 'Individual food items within meals (denormalized nutrition)';
COMMENT ON COLUMN meal_items.source IS 'Where nutrition data came from: ai, common_foods, manual, template';

-- =====================================================
-- MEAL_FLAGS TABLE
-- =====================================================
-- Track user-reported issues with meals
CREATE TABLE meal_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flag_type TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_flag_type CHECK (flag_type IN ('inaccurate', 'wrong_portions', 'missing_items', 'other')),
    CONSTRAINT description_length CHECK (description IS NULL OR char_length(description) <= 500),
    CONSTRAINT one_flag_per_meal UNIQUE (meal_id, user_id)
);

-- Enable RLS
ALTER TABLE meal_flags ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own flags"
    ON meal_flags FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can create own flags"
    ON meal_flags FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own flags"
    ON meal_flags FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own flags"
    ON meal_flags FOR DELETE
    USING (auth.uid() = user_id);

COMMENT ON TABLE meal_flags IS 'User-reported issues with meal data accuracy';

-- =====================================================
-- WEIGHT_ENTRIES TABLE
-- =====================================================
CREATE TABLE weight_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight_kg DECIMAL(5,2) NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_weight CHECK (weight_kg > 0 AND weight_kg <= 500),
    CONSTRAINT notes_length CHECK (notes IS NULL OR char_length(notes) <= 500)
);

-- Enable RLS
ALTER TABLE weight_entries ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own weight entries"
    ON weight_entries FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can manage own weight entries"
    ON weight_entries FOR ALL
    USING (auth.uid() = user_id);

COMMENT ON TABLE weight_entries IS 'User weight tracking entries';
COMMENT ON COLUMN weight_entries.measured_at IS 'UTC timestamp when weight was measured';

-- =====================================================
-- TEMPLATES TABLE
-- =====================================================
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT template_name_length CHECK (char_length(name) >= 1 AND char_length(name) <= 100),
    CONSTRAINT description_length CHECK (description IS NULL OR char_length(description) <= 500),
    CONSTRAINT unique_template_name UNIQUE (user_id, name)
);

-- Enable RLS
ALTER TABLE templates ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own templates"
    ON templates FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can manage own templates"
    ON templates FOR ALL
    USING (auth.uid() = user_id);

COMMENT ON TABLE templates IS 'User-created meal templates for quick logging';

-- =====================================================
-- TEMPLATE_ITEMS TABLE
-- =====================================================
CREATE TABLE template_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    food_name TEXT NOT NULL,
    quantity DECIMAL(8,2) NOT NULL,
    unit TEXT NOT NULL,

    -- Denormalized nutrition
    calories DECIMAL(7,1) NOT NULL,
    protein DECIMAL(6,1) NOT NULL,
    carbs DECIMAL(6,1) NOT NULL,
    fat DECIMAL(6,1) NOT NULL,
    fiber DECIMAL(5,1),
    sugar DECIMAL(6,1),
    sodium DECIMAL(7,1),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT food_name_length CHECK (char_length(food_name) >= 1 AND char_length(food_name) <= 200),
    CONSTRAINT positive_quantity CHECK (quantity > 0 AND quantity <= 99999),
    CONSTRAINT valid_unit CHECK (char_length(unit) >= 1 AND char_length(unit) <= 50),
    CONSTRAINT positive_calories CHECK (calories >= 0 AND calories <= 9999),
    CONSTRAINT positive_protein CHECK (protein >= 0 AND protein <= 999),
    CONSTRAINT positive_carbs CHECK (carbs >= 0 AND carbs <= 999),
    CONSTRAINT positive_fat CHECK (fat >= 0 AND fat <= 999),
    CONSTRAINT positive_fiber CHECK (fiber IS NULL OR (fiber >= 0 AND fiber <= 999)),
    CONSTRAINT positive_sugar CHECK (sugar IS NULL OR (sugar >= 0 AND sugar <= 999)),
    CONSTRAINT positive_sodium CHECK (sodium IS NULL OR (sodium >= 0 AND sodium <= 99999))
);

-- Enable RLS
ALTER TABLE template_items ENABLE ROW LEVEL SECURITY;

-- RLS Policies (inherit from templates)
CREATE POLICY "Users can view own template items"
    ON template_items FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM templates
            WHERE templates.id = template_items.template_id
            AND templates.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can manage own template items"
    ON template_items FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM templates
            WHERE templates.id = template_items.template_id
            AND templates.user_id = auth.uid()
        )
    );

COMMENT ON TABLE template_items IS 'Food items within meal templates';

-- =====================================================
-- COMMON_FOODS TABLE
-- =====================================================
-- Global database of common foods (admin managed)
CREATE TABLE common_foods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    category TEXT,
    serving_size DECIMAL(8,2) NOT NULL,
    serving_unit TEXT NOT NULL,

    -- Nutrition per serving
    calories DECIMAL(7,1) NOT NULL,
    protein DECIMAL(6,1) NOT NULL,
    carbs DECIMAL(6,1) NOT NULL,
    fat DECIMAL(6,1) NOT NULL,
    fiber DECIMAL(5,1),
    sugar DECIMAL(6,1),
    sodium DECIMAL(7,1),

    -- Metadata
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    source TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT food_name_length CHECK (char_length(name) >= 1 AND char_length(name) <= 200),
    CONSTRAINT category_length CHECK (category IS NULL OR char_length(category) <= 50),
    CONSTRAINT positive_serving CHECK (serving_size > 0 AND serving_size <= 99999),
    CONSTRAINT valid_unit CHECK (char_length(serving_unit) >= 1 AND char_length(serving_unit) <= 50),
    CONSTRAINT positive_calories CHECK (calories >= 0 AND calories <= 9999),
    CONSTRAINT positive_protein CHECK (protein >= 0 AND protein <= 999),
    CONSTRAINT positive_carbs CHECK (carbs >= 0 AND carbs <= 999),
    CONSTRAINT positive_fat CHECK (fat >= 0 AND fat <= 999),
    CONSTRAINT positive_fiber CHECK (fiber IS NULL OR (fiber >= 0 AND fiber <= 999)),
    CONSTRAINT positive_sugar CHECK (sugar IS NULL OR (sugar >= 0 AND sugar <= 999)),
    CONSTRAINT positive_sodium CHECK (sodium IS NULL OR (sodium >= 0 AND sodium <= 99999))
);

-- Enable RLS (read-only for users)
ALTER TABLE common_foods ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Anyone can view common foods"
    ON common_foods FOR SELECT
    USING (true);

COMMENT ON TABLE common_foods IS 'Global database of common foods with verified nutrition data';
COMMENT ON COLUMN common_foods.verified IS 'Whether nutrition data has been verified by admin';

-- =====================================================
-- AI_USAGE TABLE
-- =====================================================
-- Track AI API usage for rate limiting and cost monitoring
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    tokens_used INTEGER NOT NULL,
    cost_usd DECIMAL(10,4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_endpoint CHECK (endpoint IN ('analyze_meal', 'parse_food')),
    CONSTRAINT positive_tokens CHECK (tokens_used > 0),
    CONSTRAINT positive_cost CHECK (cost_usd IS NULL OR cost_usd >= 0)
);

-- Enable RLS
ALTER TABLE ai_usage ENABLE ROW LEVEL SECURITY;

-- RLS Policies (users cannot view this - admin only via direct SQL)
CREATE POLICY "No user access to ai_usage"
    ON ai_usage FOR SELECT
    USING (false);

COMMENT ON TABLE ai_usage IS 'AI API usage tracking for rate limiting and cost monitoring';

-- =====================================================
-- IDEMPOTENCY_KEYS TABLE
-- =====================================================
-- Prevent duplicate API requests
CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT key_length CHECK (char_length(key) >= 1 AND char_length(key) <= 255)
);

-- Enable RLS
ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY "Users can view own idempotency keys"
    ON idempotency_keys FOR SELECT
    USING (auth.uid() = user_id);

COMMENT ON TABLE idempotency_keys IS 'Idempotency keys to prevent duplicate API requests';

-- =====================================================
-- TRIGGERS FOR UPDATED_AT
-- =====================================================
-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_meals_updated_at
    BEFORE UPDATE ON meals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_common_foods_updated_at
    BEFORE UPDATE ON common_foods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_daily_goals_updated_at
    BEFORE UPDATE ON user_daily_goals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- INITIAL DATA
-- =====================================================
-- Note: Common foods should be populated via separate data migration
-- or admin interface

COMMENT ON SCHEMA public IS 'Lumen Nutrition Tracker - Initial Schema (v001)';
