-- =====================================================
-- Lumen Nutrition Tracker - Update RPC Functions for New Schema
-- Version: 013
-- Description: Update all RPC functions to work with migration 012 schema changes:
--              - Use 'name' instead of 'food_name' in meal_items/template_items
--              - Use 'consumed_at' instead of 'meal_time' in meals
--              - Remove manual total calculations (triggers handle it)
-- =====================================================

BEGIN;

-- =====================================================
-- UPDATE: create_meal_with_items
-- =====================================================
-- Drop old function signatures to avoid conflicts
DROP FUNCTION IF EXISTS create_meal_with_items(UUID, TEXT, TIMESTAMPTZ, TEXT, JSONB);
DROP FUNCTION IF EXISTS create_meal_with_items(UUID, TEXT, TIMESTAMPTZ, JSONB, TEXT, DECIMAL, DECIMAL, DECIMAL, DECIMAL, DECIMAL, JSONB);

-- Recreate with updated schema (no manual totals, uses 'name' column)
CREATE OR REPLACE FUNCTION create_meal_with_items(
    p_user_id UUID,
    p_meal_type TEXT,
    p_consumed_at TIMESTAMPTZ,
    p_photos JSONB,
    p_notes TEXT,
    p_items JSONB
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_meal_id UUID;
    v_meal_item JSONB;
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

    -- Validate meal type
    IF p_meal_type NOT IN ('breakfast', 'lunch', 'dinner', 'snack') THEN
        RAISE EXCEPTION 'Invalid meal type: %', p_meal_type;
    END IF;

    -- Create the meal (totals will be auto-calculated by triggers)
    INSERT INTO meals (
        user_id,
        meal_type,
        consumed_at,
        photos,
        notes
    )
    VALUES (
        p_user_id,
        p_meal_type,
        p_consumed_at,
        p_photos,
        p_notes
    )
    RETURNING id INTO v_meal_id;

    -- Insert all meal items (triggers will auto-update meal totals)
    FOR v_meal_item IN SELECT * FROM jsonb_array_elements(p_items)
    LOOP
        INSERT INTO meal_items (
            meal_id,
            name,  -- CHANGED: was 'food_name'
            quantity,
            unit,
            calories,
            protein,
            carbs,
            fat,
            fiber
        )
        VALUES (
            v_meal_id,
            v_meal_item->>'Name',
            (v_meal_item->>'Quantity')::DECIMAL,
            v_meal_item->>'Unit',
            (v_meal_item->>'Calories')::DECIMAL,
            (v_meal_item->>'ProteinG')::DECIMAL,
            (v_meal_item->>'CarbsG')::DECIMAL,
            (v_meal_item->>'FatG')::DECIMAL,
            (v_meal_item->>'FiberG')::DECIMAL
        );
    END LOOP;

    -- Return meal ID (totals are auto-calculated, no manual calculation needed)
    RETURN v_meal_id;
END;
$$;

COMMENT ON FUNCTION create_meal_with_items IS 'Atomically creates a meal with all its items. Totals auto-calculated by triggers. Updated for migration 012 schema.';

-- =====================================================
-- UPDATE: get_daily_nutrition
-- =====================================================
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
    v_start_time := (p_date || ' 00:00:00')::TIMESTAMP AT TIME ZONE p_timezone;
    v_end_time := (p_date || ' 23:59:59.999999')::TIMESTAMP AT TIME ZONE p_timezone;

    -- Aggregate nutrition data using consumed_at instead of meal_time
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
                    'meal_type', m.meal_type,  -- CHANGED: was 'name'
                    'consumed_at', m.consumed_at,  -- CHANGED: was 'meal_time'
                    'item_count', (
                        SELECT COUNT(*)
                        FROM meal_items mi2
                        WHERE mi2.meal_id = m.id
                    ),
                    'totals', (
                        -- Use meal's auto-calculated totals instead of manual sum
                        SELECT jsonb_build_object(
                            'calories', m.total_calories,
                            'protein', m.total_protein_g,
                            'carbs', m.total_carbs_g,
                            'fat', m.total_fat_g
                        )
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
        AND m.consumed_at >= v_start_time  -- CHANGED: was 'meal_time'
        AND m.consumed_at <= v_end_time    -- CHANGED: was 'meal_time'
    GROUP BY m.id, m.meal_type, m.consumed_at, m.total_calories, m.total_protein_g, m.total_carbs_g, m.total_fat_g;

    RETURN v_result;
END;
$$;

COMMENT ON FUNCTION get_daily_nutrition IS 'Returns aggregated nutrition totals for a specific date in user timezone. Uses consumed_at instead of meal_time.';

-- =====================================================
-- UPDATE: create_template_from_meal
-- =====================================================
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

    -- Copy meal items to template items (using 'name' column)
    INSERT INTO template_items (
        template_id,
        name,  -- CHANGED: was 'food_name'
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
        mi.name,  -- CHANGED: was 'mi.food_name'
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

    -- Return template with items (using 'name' column)
    SELECT jsonb_build_object(
        'id', t.id,
        'name', t.name,
        'description', t.description,
        'created_at', t.created_at,
        'items', (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', ti.id,
                    'name', ti.name,  -- CHANGED: was 'food_name'
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

COMMENT ON FUNCTION create_template_from_meal IS 'Creates a reusable template from an existing meal. Uses name instead of food_name.';

-- =====================================================
-- UPDATE: get_nutrition_trends
-- =====================================================
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

    -- Generate daily totals using consumed_at instead of meal_time
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
            DATE(m.consumed_at AT TIME ZONE p_timezone) AS day,  -- CHANGED: was 'meal_time'
            ROUND(SUM(mi.calories)::NUMERIC, 1) AS calories,
            ROUND(SUM(mi.protein)::NUMERIC, 1) AS protein,
            ROUND(SUM(mi.carbs)::NUMERIC, 1) AS carbs,
            ROUND(SUM(mi.fat)::NUMERIC, 1) AS fat,
            ROUND(SUM(mi.fiber)::NUMERIC, 1) AS fiber,
            COUNT(DISTINCT m.id) AS meal_count
        FROM meals m
        JOIN meal_items mi ON mi.meal_id = m.id
        WHERE m.user_id = p_user_id
            AND DATE(m.consumed_at AT TIME ZONE p_timezone) >= p_start_date  -- CHANGED: was 'meal_time'
            AND DATE(m.consumed_at AT TIME ZONE p_timezone) <= p_end_date    -- CHANGED: was 'meal_time'
        GROUP BY DATE(m.consumed_at AT TIME ZONE p_timezone)  -- CHANGED: was 'meal_time'
    ) day_data;

    RETURN COALESCE(v_result, '[]'::jsonb);
END;
$$;

COMMENT ON FUNCTION get_nutrition_trends IS 'Returns daily nutrition totals over a date range. Uses consumed_at instead of meal_time.';

-- =====================================================
-- NOTE: search_common_foods does not need updates
-- =====================================================
-- The search_common_foods function only queries the common_foods table,
-- which was not affected by migration 012 changes.

-- =====================================================
-- GRANT EXECUTE PERMISSIONS
-- =====================================================
GRANT EXECUTE ON FUNCTION create_meal_with_items TO authenticated;
GRANT EXECUTE ON FUNCTION get_daily_nutrition TO authenticated;
GRANT EXECUTE ON FUNCTION create_template_from_meal TO authenticated;
GRANT EXECUTE ON FUNCTION get_nutrition_trends TO authenticated;

COMMIT;

-- =====================================================
-- VALIDATION QUERIES (run manually after migration)
-- =====================================================

-- Test create_meal_with_items with new schema:
-- SELECT create_meal_with_items(
--     auth.uid(),
--     'breakfast',
--     NOW(),
--     '[]'::jsonb,
--     'Test meal',
--     '[{"Name": "Apple", "Quantity": 1, "Unit": "medium", "Calories": 95, "ProteinG": 0.5, "CarbsG": 25, "FatG": 0.3, "FiberG": 4.4}]'::jsonb
-- );

-- Verify totals are auto-calculated:
-- SELECT id, meal_type, total_calories, total_protein_g, total_carbs_g, total_fat_g
-- FROM meals
-- WHERE user_id = auth.uid()
-- ORDER BY consumed_at DESC
-- LIMIT 5;
