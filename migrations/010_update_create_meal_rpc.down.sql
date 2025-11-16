-- =====================================================
-- Rollback: Restore original create_meal_with_items signature
-- =====================================================

-- Drop the updated function
DROP FUNCTION IF EXISTS create_meal_with_items(UUID, TEXT, TIMESTAMPTZ, JSONB, TEXT, DECIMAL, DECIMAL, DECIMAL, DECIMAL, DECIMAL, JSONB);

-- Restore original 5-parameter version
CREATE OR REPLACE FUNCTION create_meal_with_items(
    p_user_id UUID,
    p_meal_name TEXT,
    p_meal_time TIMESTAMPTZ,
    p_notes TEXT,
    p_items JSONB
)
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_meal_id UUID;
    v_meal_item JSONB;
    v_result JSONB;
BEGIN
    IF auth.uid() IS NULL THEN
        RAISE EXCEPTION 'Not authenticated';
    END IF;

    IF auth.uid() != p_user_id THEN
        RAISE EXCEPTION 'Unauthorized: Cannot create meals for other users';
    END IF;

    IF p_items IS NULL OR jsonb_array_length(p_items) = 0 THEN
        RAISE EXCEPTION 'At least one meal item is required';
    END IF;

    INSERT INTO meals (user_id, name, meal_time, notes)
    VALUES (p_user_id, p_meal_name, p_meal_time, p_notes)
    RETURNING id INTO v_meal_id;

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
        )
    )
    INTO v_result
    FROM meals m
    WHERE m.id = v_meal_id;

    RETURN v_result;
END;
$$;

GRANT EXECUTE ON FUNCTION create_meal_with_items TO authenticated;
