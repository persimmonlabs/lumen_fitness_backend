// Package analytics provides HTTP handlers for nutrition analytics endpoints.
//
// This package contains handlers for retrieving daily analytics, weekly trends,
// macro distribution, goal progress, and custom date range nutrition trends.
package analytics

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for nutrition analytics
type Handler struct {
	service Service
}

// NewHandler creates a new analytics handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers analytics routes with the router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/analytics", func(r chi.Router) {
		r.Get("/daily", h.GetDailyAnalytics)
		r.Get("/weekly", h.GetWeeklyAnalytics)
		r.Get("/trends", h.GetTrends)
		r.Get("/distribution", h.GetMacroDistribution)
		r.Get("/progress", h.GetGoalProgress)
		r.Get("/trajectory", h.GetTrajectory)
	})
}

// GetDailyAnalytics handles GET /api/v1/analytics/daily
// @Summary Get daily nutrition analytics
// @Description Returns today's nutrition totals with goal comparison
// @Tags analytics
// @Accept json
// @Produce json
// @Param date query string false "Date (YYYY-MM-DD, defaults to today)"
// @Param timezone query string false "Timezone (defaults to UTC)"
// @Success 200 {object} DailyAnalyticsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/daily [get]
func (h *Handler) GetDailyAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	dateStr := r.URL.Query().Get("date")
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	var date time.Time
	var err error
	if dateStr == "" {
		date = time.Now()
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
	}

	// Get daily analytics
	analytics, err := h.service.GetDailyAnalytics(r.Context(), userID, date, timezone)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to get daily analytics")
		return
	}

	h.successResponse(w, http.StatusOK, analytics)
}

// GetWeeklyAnalytics handles GET /api/v1/analytics/weekly
// @Summary Get weekly nutrition trends
// @Description Returns 7-day nutrition trends with averages
// @Tags analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (YYYY-MM-DD, defaults to 7 days ago)"
// @Param timezone query string false "Timezone (defaults to UTC)"
// @Success 200 {object} WeeklyAnalyticsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/weekly [get]
func (h *Handler) GetWeeklyAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	var startDate time.Time
	var err error
	if startDateStr == "" {
		startDate = time.Now().AddDate(0, 0, -6)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD")
			return
		}
	}

	// Get weekly analytics
	analytics, err := h.service.GetWeeklyTrends(r.Context(), userID, startDate, timezone)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to get weekly analytics")
		return
	}

	h.successResponse(w, http.StatusOK, analytics)
}

// GetTrends handles GET /api/v1/analytics/trends
// @Summary Get nutrition trends for custom date range
// @Description Returns daily nutrition data for a custom date range
// @Tags analytics
// @Accept json
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Param timezone query string false "Timezone (defaults to UTC)"
// @Success 200 {object} TrendsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/trends [get]
func (h *Handler) GetTrends(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	if startDateStr == "" || endDateStr == "" {
		h.errorResponse(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD")
		return
	}

	if endDate.Before(startDate) {
		h.errorResponse(w, http.StatusBadRequest, "end_date must be after start_date")
		return
	}

	// Get date range stats
	stats, err := h.service.GetDateRangeStats(r.Context(), userID, startDate, endDate, timezone)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to get nutrition trends")
		return
	}

	h.successResponse(w, http.StatusOK, stats)
}

// GetMacroDistribution handles GET /api/v1/analytics/distribution
// @Summary Get macro distribution
// @Description Returns macronutrient percentage breakdown
// @Tags analytics
// @Accept json
// @Produce json
// @Param date query string false "Date (YYYY-MM-DD, defaults to today)"
// @Param timezone query string false "Timezone (defaults to UTC)"
// @Success 200 {object} DistributionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/distribution [get]
func (h *Handler) GetMacroDistribution(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	dateStr := r.URL.Query().Get("date")
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	var date time.Time
	var err error
	if dateStr == "" {
		date = time.Now()
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
	}

	// Get macro distribution
	distribution, err := h.service.GetMacroDistribution(r.Context(), userID, date, timezone)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to get macro distribution")
		return
	}

	h.successResponse(w, http.StatusOK, distribution)
}

// GetGoalProgress handles GET /api/v1/analytics/progress
// @Summary Get goal progress
// @Description Returns nutrition goals progress comparison
// @Tags analytics
// @Accept json
// @Produce json
// @Param date query string false "Date (YYYY-MM-DD, defaults to today)"
// @Param timezone query string false "Timezone (defaults to UTC)"
// @Success 200 {object} ProgressResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/progress [get]
func (h *Handler) GetGoalProgress(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	dateStr := r.URL.Query().Get("date")
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	var date time.Time
	var err error
	if dateStr == "" {
		date = time.Now()
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.errorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
	}

	// Get daily analytics which includes goal progress
	analytics, err := h.service.GetDailyAnalytics(r.Context(), userID, date, timezone)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to get goal progress")
		return
	}

	// Build progress response
	var progress GoalComparison
	if analytics.Progress != nil {
		progress = *analytics.Progress
	}

	response := ProgressResponse{
		Date:     date,
		Timezone: timezone,
		Progress: progress,
		Goals:    analytics.Goals,
		Totals:   analytics.Totals,
	}

	h.successResponse(w, http.StatusOK, response)
}

// GetTrajectory handles GET /api/v1/analytics/trajectory
// @Summary Get weight trajectory prediction
// @Description Predicts future weight based on historical data using linear regression
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {object} Trajectory
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /analytics/trajectory [get]
func (h *Handler) GetTrajectory(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		h.errorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Calculate trajectory
	trajectory, err := h.service.CalculateTrajectory(r.Context(), userID)
	if err != nil {
		// Check if it's insufficient data error
		if err.Error() == "insufficient data: need at least 7 weight entries" {
			h.errorResponse(w, http.StatusBadRequest, "Insufficient data: need at least 7 weight entries in the last 60 days")
			return
		}
		h.errorResponse(w, http.StatusInternalServerError, "Failed to calculate trajectory")
		return
	}

	h.successResponse(w, http.StatusOK, trajectory)
}

// Helper functions

// getUserID extracts user ID from request context
// This should be set by authentication middleware
func (h *Handler) getUserID(r *http.Request) string {
	// Try UUID type first (current auth middleware stores UUID)
	if userUUID, ok := r.Context().Value("user_id").(uuid.UUID); ok {
		return userUUID.String()
	}
	// Fall back to string type for compatibility
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// errorResponse writes an error response
func (h *Handler) errorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

// successResponse writes a success response
func (h *Handler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(SuccessResponse{
		Status: "success",
		Data:   data,
	})
}
