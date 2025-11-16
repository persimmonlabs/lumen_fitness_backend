package goals

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for goals
type Handler struct {
	service Service
}

// NewHandler creates a new goals handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all goal routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/goals", func(r chi.Router) {
		r.Get("/", h.GetGoals)
		r.Put("/", h.UpdateGoals)
		r.Post("/calculate", h.CalculateTDEE)
		r.Get("/daily", h.GetDailyGoals)
		r.Put("/daily/{day}", h.SetDailyGoal)
		r.Delete("/daily/{day}", h.DeleteDailyGoal)
	})
}

// GetGoals retrieves current user goals
// GET /api/v1/goals
func (h *Handler) GetGoals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Extract user ID from auth context
	// For now using hardcoded user ID
	userID := int64(1)

	response, err := h.service.GetGoals(ctx, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}

// UpdateGoals updates user goals
// PUT /api/v1/goals
func (h *Handler) UpdateGoals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Extract user ID from auth context
	userID := int64(1)

	var req UpdateGoalsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	goals, err := h.service.UpdateGoals(ctx, userID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, goals)
}

// CalculateTDEE calculates TDEE from user data
// POST /api/v1/goals/calculate
func (h *Handler) CalculateTDEE(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var inputs TDEEInputs
	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	result, err := h.service.CalculateTDEE(ctx, &inputs)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// GetDailyGoals returns daily goals overview
// GET /api/v1/goals/daily
func (h *Handler) GetDailyGoals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Extract user ID from auth context
	userID := int64(1)

	response, err := h.service.GetGoals(ctx, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	// Return only daily goals part
	h.jsonResponse(w, http.StatusOK, response.DailyGoals)
}

// SetDailyGoal sets a day-specific goal
// PUT /api/v1/goals/daily/{day}
func (h *Handler) SetDailyGoal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Extract user ID from auth context
	userID := int64(1)

	day := chi.URLParam(r, "day")
	if day == "" {
		h.badRequest(w, "Day parameter is required")
		return
	}

	var req SetDailyGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	goal, err := h.service.SetDailyGoal(ctx, userID, day, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, goal)
}

// DeleteDailyGoal removes a day-specific goal
// DELETE /api/v1/goals/daily/{day}
func (h *Handler) DeleteDailyGoal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Extract user ID from auth context
	userID := int64(1)

	day := chi.URLParam(r, "day")
	if day == "" {
		h.badRequest(w, "Day parameter is required")
		return
	}

	err := h.service.DeleteDailyGoal(ctx, userID, day)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) badRequest(w http.ResponseWriter, message string) {
	h.errorResponse(w, http.StatusBadRequest, message)
}

func (h *Handler) errorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrGoalsNotFound, ErrDailyGoalNotFound:
		h.errorResponse(w, http.StatusNotFound, err.Error())
	case ErrInvalidAge, ErrInvalidHeight, ErrInvalidWeight, ErrInvalidSex,
		ErrInvalidActivityLevel, ErrInvalidCalories, ErrInvalidProtein,
		ErrInvalidCarbs, ErrInvalidFat, ErrInvalidDayOfWeek:
		h.errorResponse(w, http.StatusBadRequest, err.Error())
	default:
		h.errorResponse(w, http.StatusInternalServerError, "Internal server error")
	}
}
