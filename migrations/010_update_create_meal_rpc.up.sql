-- =====================================================
-- Lumen Nutrition Tracker - Update create_meal_with_items RPC
-- Version: 010
-- Description: Update RPC function to match new meal schema with nutrition totals
-- =====================================================

-- Drop the old function first
DROP FUNCTION IF EXISTS create_meal_with_items(UUID, TEXT, TIMESTAMPTZ, TEXT, JSONB);

-- Recreate with updated signature matching migration 009 schema
CREATE OR REPLACE FUNCTION create_meal_with_items(
    p_user_id UUID,
    p_meal_type TEXT,
    p_consumed_at TIMESTAMPTZ,
    p_photos JSONB,
    p_notes TEXT,
    p_total_calories DECIMAL,
    p_total_protein_g DECIMAL,
    p_total_carbs_g DECIMAL,
    p_total_fat_g DECIMAL,
    p_total_fiber_g DECIMAL,
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

    -- Create the meal with new schema
    INSERT INTO meals (
        user_id,
        meal_type,
        consumed_at,
        photos,
        notes,
        total_calories,
        total_protein_g,
        total_carbs_g,
        total_fat_g,
        total_fiber_g
    )
    VALUES (
        p_user_id,
        p_meal_type,
        p_consumed_at,
        p_photos,
        p_notes,
        p_total_calories,
        p_total_protein_g,
        p_total_carbs_g,
        p_total_fat_g,
        p_total_fiber_g
    )
    RETURNING id INTO v_meal_id;

    -- Insert all meal items
    FOR v_meal_item IN SELECT * FROM jsonb_array_elements(p_items)
    LOOP
        INSERT INTO meal_items (
            meal_id,
            name,
            quantity,
            unit,
            calories,
            protein_g,
            carbs_g,
            fat_g,
            fiber_g
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

    RETURN v_meal_id;
END;
$$;

COMMENT ON FUNCTION create_meal_with_items IS 'Atomically creates a meal with nutrition totals and all its items. Returns meal ID. Updated to match migration 009 schema.';

-- Grant execute permissions
GRANT EXECUTE ON FUNCTION create_meal_with_items TO authenticated;
