// Package handlers provides HTTP request handlers for the fitness app API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/services/suggestions"
)

// SuggestionsHandler handles meal suggestion requests.
type SuggestionsHandler struct {
	service *suggestions.Service
	logger  *slog.Logger
}

// NewSuggestionsHandler creates a new suggestions handler.
func NewSuggestionsHandler(service *suggestions.Service, logger *slog.Logger) *SuggestionsHandler {
	return &SuggestionsHandler{
		service: service,
		logger:  logger,
	}
}

// SuggestionRequestBody represents the JSON request body.
type SuggestionRequestBody struct {
	MealContext      suggestions.MealContext      `json:"meal_context"`
	NutritionContext suggestions.NutritionContext `json:"nutrition_context"`
	UserPreferences  suggestions.UserPreferences  `json:"user_preferences"`
	RecentMeals      []string                     `json:"recent_meals"`
	MaxSuggestions   int                          `json:"max_suggestions"`
}

// Generate handles POST /api/v1/suggestions/generate
func (h *SuggestionsHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var reqBody SuggestionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	// Set defaults
	if reqBody.MaxSuggestions == 0 {
		reqBody.MaxSuggestions = 5
	}

	// Create suggestion request
	suggestReq := &suggestions.SuggestionRequest{
		UserID:           userID,
		MealContext:      reqBody.MealContext,
		NutritionContext: reqBody.NutritionContext,
		UserPreferences:  reqBody.UserPreferences,
		RecentMeals:      reqBody.RecentMeals,
		MaxSuggestions:   reqBody.MaxSuggestions,
	}

	// Generate suggestions
	result, err := h.service.Generate(r.Context(), suggestReq)
	if err != nil {
		h.logger.Error("suggestion generation failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// QuickSuggestions handles GET /api/v1/suggestions/quick
func (h *SuggestionsHandler) QuickSuggestions(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	mealType := r.URL.Query().Get("meal_type")
	if mealType == "" {
		mealType = "lunch"
	}

	// Get quick suggestions
	result := h.service.GetQuickSuggestions(r.Context(), mealType, 500)
	h.successResponse(w, http.StatusOK, result)
}

func (h *SuggestionsHandler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *SuggestionsHandler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
