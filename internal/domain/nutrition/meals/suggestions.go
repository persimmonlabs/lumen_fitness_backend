package meals

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetMealSuggestions retrieves frequently eaten meals at similar times
func (r *repository) GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) ([]MealSuggestion, error) {
	// Determine hour range from meal type
	var hourStart, hourEnd int
	switch mealType {
	case MealTypeBreakfast:
		hourStart, hourEnd = 6, 11
	case MealTypeLunch:
		hourStart, hourEnd = 11, 15
	case MealTypeDinner:
		hourStart, hourEnd = 17, 21
	case MealTypeSnack:
		// Snacks can be any time outside main meals
		hourStart, hourEnd = 0, 23
	default:
		hourStart, hourEnd = 0, 23
	}

	// Query meals from last 30 days at similar times (±1 hour)
	// NOTE: meal_items columns are 'protein', 'carbs', 'fat', 'fiber' (NO _g suffix)
	// Only meals.total_* columns have the _g suffix
	query := `
		WITH meal_descriptions AS (
			SELECT
				LOWER(TRIM(mi.name)) as normalized_description,
				AVG(mi.calories) as avg_calories,
				AVG(mi.protein) as avg_protein,
				COUNT(*) as frequency,
				MAX(m.consumed_at) as last_consumed_at
			FROM meals m
			INNER JOIN meal_items mi ON m.id = mi.meal_id
			WHERE m.user_id = $1
				AND EXTRACT(HOUR FROM m.consumed_at) BETWEEN $2 AND $3
				AND m.consumed_at >= NOW() - INTERVAL '30 days'
				AND m.deleted_at IS NULL
				AND m.is_draft = false
			GROUP BY normalized_description
		)
		SELECT
			normalized_description as description,
			avg_calories,
			avg_protein,
			frequency,
			last_consumed_at
		FROM meal_descriptions
		WHERE frequency >= 2
		ORDER BY frequency DESC, last_consumed_at DESC
		LIMIT 3
	`

	var suggestions []MealSuggestion
	err := r.db.SelectContext(ctx, &suggestions, query, userID, hourStart, hourEnd)
	if err != nil {
		return nil, fmt.Errorf("get meal suggestions: %w", err)
	}

	// Return empty slice if no suggestions found
	if suggestions == nil {
		suggestions = []MealSuggestion{}
	}

	return suggestions, nil
}
