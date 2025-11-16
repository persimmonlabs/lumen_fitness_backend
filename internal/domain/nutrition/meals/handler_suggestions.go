package meals

import (
	"net/http"
	"time"

	custommw "github.com/pradord/lumen_final/backend/internal/server/middleware"
)

// GetMealSuggestions handles GET /api/v1/meals/suggestions?time=12:30
// @Summary Get meal suggestions based on time
// @Description Returns frequently eaten meals at similar times
// @Tags meals
// @Accept json
// @Produce json
// @Param time query string true "Time in HH:MM format (e.g., 12:30)"
// @Success 200 {object} MealSuggestionsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /meals/suggestions [get]
func (h *Handler) GetMealSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, err := custommw.GetUserUUID(r.Context())
	if err != nil {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Get time query parameter
	timeStr := r.URL.Query().Get("time")
	if timeStr == "" {
		h.errorResponse(w, http.StatusBadRequest, "time parameter is required (format: HH:MM)", nil)
		return
	}

	// Parse time (format: HH:MM)
	queryTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid time format, use HH:MM (e.g., 12:30)", nil)
		return
	}

	// Determine meal type from time
	hour := queryTime.Hour()
	var mealType MealType
	if hour >= 6 && hour < 11 {
		mealType = MealTypeBreakfast
	} else if hour >= 11 && hour < 15 {
		mealType = MealTypeLunch
	} else if hour >= 17 && hour < 21 {
		mealType = MealTypeDinner
	} else {
		mealType = MealTypeSnack
	}

	// Get suggestions from service
	response, err := h.service.GetMealSuggestions(r.Context(), userID, mealType)
	if err != nil {
		h.logger.Error("failed to get meal suggestions",
			"user_id", userID,
			"time", timeStr,
			"error", err,
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to get meal suggestions", nil)
		return
	}

	h.successResponse(w, http.StatusOK, response)
}
