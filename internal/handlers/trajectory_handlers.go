// Package handlers provides HTTP request handlers for the fitness app API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/services/trajectory"
)

// TrajectoryHandler handles weight trajectory prediction requests.
type TrajectoryHandler struct {
	service *trajectory.Service
	logger  *slog.Logger
}

// NewTrajectoryHandler creates a new trajectory handler.
func NewTrajectoryHandler(service *trajectory.Service, logger *slog.Logger) *TrajectoryHandler {
	return &TrajectoryHandler{
		service: service,
		logger:  logger,
	}
}

// PredictionRequest represents the JSON request body.
type PredictionRequestBody struct {
	CurrentWeight     float64                       `json:"current_weight"`
	GoalWeight        float64                       `json:"goal_weight"`
	TargetDate        *time.Time                    `json:"target_date"`
	WeightHistory     []trajectory.WeightEntry      `json:"weight_history"`
	CalorieHistory    []trajectory.CalorieEntry     `json:"calorie_history"`
	DailyGoalCalories float64                       `json:"daily_goal_calories"`
	TDEE              float64                       `json:"tdee"`
	PredictionDays    int                           `json:"prediction_days"`
}

// Calculate handles POST /api/v1/trajectory/predict
func (h *TrajectoryHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var reqBody PredictionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	// Validate input
	if reqBody.CurrentWeight <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "current_weight must be positive", nil)
		return
	}
	if reqBody.TDEE <= 0 {
		h.errorResponse(w, http.StatusBadRequest, "tdee must be positive", nil)
		return
	}

	// Create prediction request
	predReq := &trajectory.PredictionRequest{
		UserID:            userID,
		CurrentWeight:     reqBody.CurrentWeight,
		GoalWeight:        reqBody.GoalWeight,
		TargetDate:        reqBody.TargetDate,
		WeightHistory:     reqBody.WeightHistory,
		CalorieHistory:    reqBody.CalorieHistory,
		DailyGoalCalories: reqBody.DailyGoalCalories,
		TDEE:              reqBody.TDEE,
		PredictionDays:    reqBody.PredictionDays,
	}

	// Calculate trajectory
	result, err := h.service.Calculate(r.Context(), predReq)
	if err != nil {
		h.logger.Error("trajectory calculation failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

func (h *TrajectoryHandler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *TrajectoryHandler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
