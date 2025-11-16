-- =====================================================
-- Lumen Nutrition Tracker - RPC Functions Migration
-- Version: 002
-- Description: Database functions for atomic operations
-- =====================================================

-- =====================================================
-- CREATE MEAL WITH ITEMS (Atomic Transaction)
-- =====================================================
-- Creates a meal and all its items in a single transaction
-- Returns the complete meal with all items
CREATE OR REPLACE FUNCTION create_meal_with_items(
    p_user_id UUID,
    p_meal_name TEXT,
    p_meal_time TIMESTAMPTZ,
    p_notes TEXT,
    p_items JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER -- Runs with function owner privileges
AS $$
DECLARE
    v_meal_id UUID;
    v_meal_item JSONB;
    v_result JSONB;
BEGIN
    -- Verify user is authenticated
    IF auth.uid() IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify user can only create their own meals
    IF auth.uid() != p_user_id THEN
        RAISE EXCEPTION 'Unauthorized: Cannot create meals for other users';
    END IF;

    -- Validate items array
    IF p_items IS NULL OR jsonb_array_length(p_items) = 0 THEN
        RAISE EXCEPTION 'At least one meal item is required';
    END IF;

    -- Create the meal
    INSERT INTO meals (user_id, name, meal_time, notes)
    VALUES (p_user_id, p_meal_name, p_meal_time, p_notes)
    RETURNING id INTO v_meal_id;

    -- Insert all meal items
    FOR v_meal_item IN SELECT * FROM jsonb_array_elements(p_items)
    LOOP
        INSERT INTO meal_items (
            meal_id,
            food_name,
            quantity,
            unit,
            calories,
            protein,
            carbs,
            fat,
            fiber,
            sugar,
            sodium,
            source
        )
        VALUES (
            v_meal_id,
            v_meal_item->>'food_name',
            (v_meal_item->>'quantity')::DECIMAL,
            v_meal_item->>'unit',
            (v_meal_item->>'calories')::DECIMAL,
            (v_meal_item->>'protein')::DECIMAL,
            (v_meal_item->>'carbs')::DECIMAL,
            (v_meal_item->>'fat')::DECIMAL,
            (v_meal_item->>'fiber')::DECIMAL,
            (v_meal_item->>'sugar')::DECIMAL,
            (v_meal_item->>'sodium')::DECIMAL,
            COALESCE(v_meal_item->>'source', 'ai')
        );
    END LOOP;

    -- Return complete meal with items
    SELECT jsonb_build_object(
        'id', m.id,
        'user_id', m.user_id,
        'name', m.name,
        'meal_time', m.meal_time,
        'notes', m.notes,
        'created_at', m.created_at,
        'updated_at', m.updated_at,
        'items', (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', mi.id,
                    'food_name', mi.food_name,
                    'quantity', mi.quantity,
                    'unit', mi.unit,
                    'calories', mi.calories,
                    'protein', mi.protein,
                    'carbs', mi.carbs,
                    'fat', mi.fat,
                    'fiber', mi.fiber,
                    'sugar', mi.sugar,
                    'sodium', mi.sodium,
                    'source', mi.source,
                    'created_at', mi.created_at
                )
            )
            FROM meal_items mi
            WHERE mi.meal_id = m.id
        ),
        'totals', (
            SELECT jsonb_build_object(
                'calories', COALESCE(SUM(mi.calories), 0),
                'protein', COALESCE(SUM(mi.protein), 0),
                'carbs', COALESCE(SUM(mi.carbs), 0),
                'fat', COALESCE(SUM(mi.fat), 0),
                'fiber', COALESCE(SUM(mi.fiber), 0),
                'sugar', COALESCE(SUM(mi.sugar), 0),
                'sodium', COALESCE(SUM(mi.sodium), 0)
            )
            FROM meal_items mi
            WHERE mi.meal_id = m.id
        )
    )
    INTO v_result
    FROM meals m
    WHERE m.id = v_meal_id;

    RETURN v_result;
END;
$$;

COMMENT ON FUNCTION create_meal_with_items IS 'Atomically creates a meal with all its items. Returns complete meal object with totals.';

-- =====================================================
-- GET DAILY NUTRITION (Aggregated Report)
-- =====================================================
-- Returns nutrition totals for a specific date
-- Respects user timezone for date boundaries
CREATE OR REPLACE FUNCTION get_daily_nutrition(
    p_user_id UUID,
    p_date DATE,
    p_timezone TEXT DEFAULT 'UTC'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_start_time TIMESTAMPTZ;
    v_end_time TIMESTAMPTZ;
    v_result JSONB;
BEGIN
    -- Verify user is authenticated
    IF auth.uid() IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify user can only access their own data
    IF auth.uid() != p_user_id THEN
        RAISE EXCEPTION 'Unauthorized: Cannot access other users data';
    END IF;

    -- Calculate date boundaries in user timezone
    -- Start: 00:00:00 in user timezone
    -- End: 23:59:59.999999 in user timezone
    v_start_time := (p_date || ' 00:00:00')::TIMESTAMP AT TIME ZONE p_timezone;
    v_end_time := (p_date || ' 23:59:59.999999')::TIMESTAMP AT TIME ZONE p_timezone;

    -- Aggregate nutrition data for the date range
    SELECT jsonb_build_object(
        'date', p_date,
        'timezone', p_timezone,
        'totals', COALESCE(
            jsonb_build_object(
                'calories', ROUND(SUM(mi.calories)::NUMERIC, 1),
                'protein', ROUND(SUM(mi.protein)::NUMERIC, 1),
                'carbs', ROUND(SUM(mi.carbs)::NUMERIC, 1),
                'fat', ROUND(SUM(mi.fat)::NUMERIC, 1),
                'fiber', ROUND(SUM(mi.fiber)::NUMERIC, 1),
                'sugar', ROUND(SUM(mi.sugar)::NUMERIC, 1),
                'sodium', ROUND(SUM(mi.sodium)::NUMERIC, 1)
            ),
            jsonb_build_object(
                'calories', 0,
                'protein', 0,
                'carbs', 0,
                'fat', 0,
                'fiber', 0,
                'sugar', 0,
                'sodium', 0
            )
        ),
        'goals', (
            SELECT jsonb_build_object(
                'calories', g.calories,
                'protein', g.protein,
                'carbs', g.carbs,
                'fat', g.fat,
                'fiber', g.fiber,
                'sugar', g.sugar,
                'sodium', g.sodium
            )
            FROM user_daily_goals g
            WHERE g.user_id = p_user_id
        ),
        'meal_count', COUNT(DISTINCT m.id),
        'meals', COALESCE(
            jsonb_agg(
                DISTINCT jsonb_build_object(
                    'id', m.id,
                    'name', m.name,
                    'meal_time', m.meal_time,
                    'item_count', (
                        SELECT COUNT(*)
                        FROM meal_items mi2
                        WHERE mi2.meal_id = m.id
                    ),
                    'totals', (
                        SELECT jsonb_build_object(
                            'calories', COALESCE(SUM(mi2.calories), 0),
                            'protein', COALESCE(SUM(mi2.protein), 0),
                            'carbs', COALESCE(SUM(mi2.carbs), 0),
                            'fat', COALESCE(SUM(mi2.fat), 0)
                        )
                        FROM meal_items mi2
                        WHERE mi2.meal_id = m.id
                    )
                )
            ) FILTER (WHERE m.id IS NOT NULL),
            '[]'::jsonb
        )
    )
    INTO v_result
    FROM meals m
    LEFT JOIN meal_items mi ON mi.meal_id = m.id
    WHERE m.user_id = p_user_id
        AND m.meal_time >= v_start_time
        AND m.meal_time <= v_end_time;

    RETURN v_result;
END;
$$;

COMMENT ON FUNCTION get_daily_nutrition IS 'Returns aggregated nutrition totals for a specific date in user timezone. Includes goals comparison and meal breakdown.';

-- =====================================================
-- CREATE TEMPLATE FROM MEAL
-- =====================================================
-- Copies a meal and its items into a reusable template
CREATE OR REPLACE FUNCTION create_template_from_meal(
    p_user_id UUID,
    p_meal_id UUID,
    p_template_name TEXT,
    p_description TEXT DEFAULT NULL
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_template_id UUID;
    v_result JSONB;
BEGIN
    -- Verify authentication
    IF auth.uid() IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify ownership
    IF auth.uid() != p_user_id THEN
        RAISE EXCEPTION 'Unauthorized';
    END IF;

    -- Verify meal belongs to user
    IF NOT EXISTS (
        SELECT 1 FROM meals
        WHERE id = p_meal_id AND user_id = p_user_id
    ) THEN
        RAISE EXCEPTION 'Meal not found or access denied';
    END IF;

    -- Create template
    INSERT INTO templates (user_id, name, description)
    VALUES (p_user_id, p_template_name, p_description)
    RETURNING id INTO v_template_id;

    -- Copy meal items to template items
    INSERT INTO template_items (
        template_id,
        food_name,
        quantity,
        unit,
        calories,
        protein,
        carbs,
        fat,
        fiber,
        sugar,
        sodium
    )
    SELECT
        v_template_id,
        mi.food_name,
        mi.quantity,
        mi.unit,
        mi.calories,
        mi.protein,
        mi.carbs,
        mi.fat,
        mi.fiber,
        mi.sugar,
        mi.sodium
    FROM meal_items mi
    WHERE mi.meal_id = p_meal_id;

    -- Return template with items
    SELECT jsonb_build_object(
        'id', t.id,
        'name', t.name,
        'description', t.description,
        'created_at', t.created_at,
        'items', (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', ti.id,
                    'food_name', ti.food_name,
                    'quantity', ti.quantity,
                    'unit', ti.unit,
                    'calories', ti.calories,
                    'protein', ti.protein,
                    'carbs', ti.carbs,
                    'fat', ti.fat,
                    'fiber', ti.fiber,
                    'sugar', ti.sugar,
                    'sodium', ti.sodium
                )
            )
            FROM template_items ti
            WHERE ti.template_id = t.id
        )
    )
    INTO v_result
    FROM templates t
    WHERE t.id = v_template_id;

    RETURN v_result;
END;
$$;

COMMENT ON FUNCTION create_template_from_meal IS 'Creates a reusable template from an existing meal';

-- =====================================================
-- SEARCH COMMON FOODS (Fuzzy Search)
-- =====================================================
-- Full-text search on common foods with similarity ranking
-- Requires pg_trgm extension (created in 003_indexes.up.sql)
CREATE OR REPLACE FUNCTION search_common_foods(
    p_query TEXT,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    category TEXT,
    serving_size DECIMAL,
    serving_unit TEXT,
    calories DECIMAL,
    protein DECIMAL,
    carbs DECIMAL,
    fat DECIMAL,
    fiber DECIMAL,
    sugar DECIMAL,
    sodium DECIMAL,
    verified BOOLEAN,
    similarity REAL
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT
        cf.id,
        cf.name,
        cf.category,
        cf.serving_size,
        cf.serving_unit,
        cf.calories,
        cf.protein,
        cf.carbs,
        cf.fat,
        cf.fiber,
        cf.sugar,
        cf.sodium,
        cf.verified,
        similarity(cf.name, p_query) AS sim
    FROM common_foods cf
    WHERE cf.name % p_query  -- Uses trigram similarity operator
    ORDER BY sim DESC, cf.verified DESC, cf.name
    LIMIT p_limit;
END;
$$;

COMMENT ON FUNCTION search_common_foods IS 'Fuzzy search for common foods using trigram similarity';

-- =====================================================
-- GET NUTRITION TRENDS (Analytics)
-- =====================================================
-- Returns daily nutrition trends over a date range
CREATE OR REPLACE FUNCTION get_nutrition_trends(
    p_user_id UUID,
    p_start_date DATE,
    p_end_date DATE,
    p_timezone TEXT DEFAULT 'UTC'
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_result JSONB;
BEGIN
    -- Verify authentication
    IF auth.uid() IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    -- Verify ownership
    IF auth.uid() != p_user_id THEN
        RAISE EXCEPTION 'Unauthorized';
    END IF;

    -- Validate date range
    IF p_end_date < p_start_date THEN
        RAISE EXCEPTION 'End date must be after start date';
    END IF;

    IF p_end_date - p_start_date > 365 THEN
        RAISE EXCEPTION 'Date range cannot exceed 365 days';
    END IF;

    -- Generate daily totals
    SELECT jsonb_agg(
        jsonb_build_object(
            'date', day_data.day,
            'calories', COALESCE(day_data.calories, 0),
            'protein', COALESCE(day_data.protein, 0),
            'carbs', COALESCE(day_data.carbs, 0),
            'fat', COALESCE(day_data.fat, 0),
            'fiber', COALESCE(day_data.fiber, 0),
            'meal_count', COALESCE(day_data.meal_count, 0)
        )
        ORDER BY day_data.day
    )
    INTO v_result
    FROM (
        SELECT
            DATE(m.meal_time AT TIME ZONE p_timezone) AS day,
            ROUND(SUM(mi.calories)::NUMERIC, 1) AS calories,
            ROUND(SUM(mi.protein)::NUMERIC, 1) AS protein,
            ROUND(SUM(mi.carbs)::NUMERIC, 1) AS carbs,
            ROUND(SUM(mi.fat)::NUMERIC, 1) AS fat,
            ROUND(SUM(mi.fiber)::NUMERIC, 1) AS fiber,
            COUNT(DISTINCT m.id) AS meal_count
        FROM meals m
        JOIN meal_items mi ON mi.meal_id = m.id
        WHERE m.user_id = p_user_id
            AND DATE(m.meal_time AT TIME ZONE p_timezone) >= p_start_date
            AND DATE(m.meal_time AT TIME ZONE p_timezone) <= p_end_date
        GROUP BY DATE(m.meal_time AT TIME ZONE p_timezone)
    ) day_data;

    RETURN COALESCE(v_result, '[]'::jsonb);
END;
$$;

COMMENT ON FUNCTION get_nutrition_trends IS 'Returns daily nutrition totals over a date range for trend analysis';

-- =====================================================
-- GRANT EXECUTE PERMISSIONS
-- =====================================================
-- Allow authenticated users to execute these functions
GRANT EXECUTE ON FUNCTION create_meal_with_items TO authenticated;
GRANT EXECUTE ON FUNCTION get_daily_nutrition TO authenticated;
GRANT EXECUTE ON FUNCTION create_template_from_meal TO authenticated;
GRANT EXECUTE ON FUNCTION search_common_foods TO authenticated;
GRANT EXECUTE ON FUNCTION get_nutrition_trends TO authenticated;
