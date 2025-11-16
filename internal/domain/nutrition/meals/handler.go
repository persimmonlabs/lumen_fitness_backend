package meals

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for meal endpoints
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler creates a new meal handler
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// ParseMeal handles POST /api/v1/meals/parse
func (h *Handler) ParseMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req ParseMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	// Validate request
	if req.Description == "" {
		h.errorResponse(w, http.StatusBadRequest, "description is required", nil)
		return
	}
	if !req.MealType.IsValid() {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal type", nil)
		return
	}
	if req.IdempotencyKey == "" {
		h.errorResponse(w, http.StatusBadRequest, "idempotency key is required", nil)
		return
	}

	result, err := h.service.ParseMeal(r.Context(), userID, &req)
	if err != nil {
		h.logger.Error("parse meal failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to parse meal", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// ConfirmMeal handles POST /api/v1/meals/confirm
func (h *Handler) ConfirmMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req ConfirmMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	// Validate request
	if !req.MealType.IsValid() {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal type", nil)
		return
	}
	if len(req.Items) == 0 {
		h.errorResponse(w, http.StatusBadRequest, "at least one item is required", nil)
		return
	}

	result, err := h.service.ConfirmMeal(r.Context(), userID, &req)
	if err != nil {
		h.logger.Error("confirm meal failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to confirm meal", nil)
		return
	}

	h.successResponse(w, http.StatusCreated, result)
}

// GetMeal handles GET /api/v1/meals/{id}
func (h *Handler) GetMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	mealIDStr := chi.URLParam(r, "id")
	mealID, err := uuid.Parse(mealIDStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal ID", nil)
		return
	}

	result, err := h.service.GetMeal(r.Context(), userID, mealID)
	if err != nil {
		h.logger.Error("get meal failed",
			slog.String("error", err.Error()),
			slog.String("meal_id", mealID.String()),
		)
		h.errorResponse(w, http.StatusNotFound, "meal not found", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// ListMeals handles GET /api/v1/meals
func (h *Handler) ListMeals(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
	}

	// Parse query parameters
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > 100 {
				limit = 100
			}
			filters.Limit = limit
		}
	}

	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			filters.Date = &date
		}
	}

	if mealTypeStr := r.URL.Query().Get("meal_type"); mealTypeStr != "" {
		mealType := MealType(mealTypeStr)
		if mealType.IsValid() {
			filters.MealType = &mealType
		}
	}

	result, err := h.service.ListMeals(r.Context(), userID, filters)
	if err != nil {
		h.logger.Error("list meals failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to list meals", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// UpdateMeal handles PUT /api/v1/meals/{id}
func (h *Handler) UpdateMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	mealIDStr := chi.URLParam(r, "id")
	mealID, err := uuid.Parse(mealIDStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal ID", nil)
		return
	}

	var req UpdateMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	result, err := h.service.UpdateMeal(r.Context(), userID, mealID, &req)
	if err != nil {
		h.logger.Error("update meal failed",
			slog.String("error", err.Error()),
			slog.String("meal_id", mealID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to update meal", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// DeleteMeal handles DELETE /api/v1/meals/{id}
func (h *Handler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	mealIDStr := chi.URLParam(r, "id")
	mealID, err := uuid.Parse(mealIDStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal ID", nil)
		return
	}

	if err := h.service.DeleteMeal(r.Context(), userID, mealID); err != nil {
		h.logger.Error("delete meal failed",
			slog.String("error", err.Error()),
			slog.String("meal_id", mealID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to delete meal", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CopyMeal handles POST /api/v1/meals/{id}/copy
func (h *Handler) CopyMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	mealIDStr := chi.URLParam(r, "id")
	mealID, err := uuid.Parse(mealIDStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal ID", nil)
		return
	}

	var req CopyMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	result, err := h.service.CopyMeal(r.Context(), userID, mealID, &req)
	if err != nil {
		h.logger.Error("copy meal failed",
			slog.String("error", err.Error()),
			slog.String("meal_id", mealID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to copy meal", nil)
		return
	}

	h.successResponse(w, http.StatusCreated, result)
}

// EstimateMeal handles POST /api/v1/meals/estimate
func (h *Handler) EstimateMeal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req EstimateMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	// Validate request
	if req.Description == "" {
		h.errorResponse(w, http.StatusBadRequest, "description is required", nil)
		return
	}
	if len(req.Description) > 200 {
		h.errorResponse(w, http.StatusBadRequest, "description too long (max 200 chars)", nil)
		return
	}

	result, err := h.service.EstimateMeal(r.Context(), userID, &req)
	if err != nil {
		h.logger.Error("estimate meal failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to estimate meal", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// ParseVoice handles POST /api/v1/meals/parse-voice
func (h *Handler) ParseVoice(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Parse multipart form (max 5MB)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to parse form", nil)
		return
	}

	// Get audio file
	file, header, err := r.FormFile("audio")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "audio file is required", nil)
		return
	}
	defer file.Close()

	// Validate file size (max 5MB)
	if header.Size > 5<<20 {
		h.errorResponse(w, http.StatusBadRequest, "audio file too large (max 5MB)", nil)
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	validTypes := map[string]bool{
		"audio/mpeg":     true, // mp3
		"audio/mp4":      true, // m4a
		"audio/webm":     true, // webm
		"audio/ogg":      true, // ogg
		"audio/wav":      true, // wav
	}
	if !validTypes[contentType] {
		h.errorResponse(w, http.StatusBadRequest, "invalid audio format (supported: mp3, m4a, webm, ogg, wav)", nil)
		return
	}

	// Read audio data
	audioData := make([]byte, header.Size)
	if _, err := file.Read(audioData); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "failed to read audio file", nil)
		return
	}

	// Get other form fields
	mealTypeStr := r.FormValue("meal_type")
	mealType := MealType(mealTypeStr)
	if !mealType.IsValid() {
		h.errorResponse(w, http.StatusBadRequest, "invalid meal type", nil)
		return
	}

	consumedAtStr := r.FormValue("consumed_at")
	consumedAt, err := time.Parse(time.RFC3339, consumedAtStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid consumed_at format (use RFC3339)", nil)
		return
	}

	idempotencyKey := r.FormValue("idempotency_key")
	if idempotencyKey == "" {
		h.errorResponse(w, http.StatusBadRequest, "idempotency_key is required", nil)
		return
	}

	// Call service to parse voice
	result, err := h.service.ParseVoice(r.Context(), userID, audioData, contentType, mealType, consumedAt, idempotencyKey)
	if err != nil {
		h.logger.Error("parse voice failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to parse voice", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// Helper methods

func (h *Handler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
