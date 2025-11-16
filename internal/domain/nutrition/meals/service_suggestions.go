package meals

import (
	"context"

	"github.com/google/uuid"
)

// GetMealSuggestions retrieves meal suggestions based on meal type
func (s *service) GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) (*MealSuggestionsResponse, error) {
	suggestions, err := s.repo.GetMealSuggestions(ctx, userID, mealType)
	if err != nil {
		return nil, err
	}

	// Determine time window from meal type
	var timeWindow string
	switch mealType {
	case MealTypeBreakfast:
		timeWindow = "06:00-11:00"
	case MealTypeLunch:
		timeWindow = "11:00-15:00"
	case MealTypeDinner:
		timeWindow = "17:00-21:00"
	case MealTypeSnack:
		timeWindow = "all day"
	default:
		timeWindow = "all day"
	}

	return &MealSuggestionsResponse{
		Suggestions: suggestions,
		QueryTime:   string(mealType),
		TimeWindow:  timeWindow,
	}, nil
}
