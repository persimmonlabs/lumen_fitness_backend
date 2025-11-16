package weight

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	custommw "github.com/pradord/lumen_final/backend/internal/server/middleware"
)

// Handler handles HTTP requests for weight tracking
type Handler struct {
	service Service
}

// NewHandler creates a new weight handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers weight routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/weight", func(r chi.Router) {
		r.Post("/", h.RecordWeight)
		r.Get("/", h.ListWeights)
		r.Get("/trend", h.GetTrend)
		r.Get("/latest", h.GetLatest)
		r.Get("/{id}", h.GetWeight)
		r.Put("/{id}", h.UpdateWeight)
		r.Delete("/{id}", h.DeleteWeight)
	})
}

// RecordWeight handles POST /api/v1/weight
func (h *Handler) RecordWeight(w http.ResponseWriter, r *http.Request) {
	var req CreateWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	entry, err := h.service.CreateEntry(r.Context(), userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, entry)
}

// GetWeight handles GET /api/v1/weight/{id}
func (h *Handler) GetWeight(w http.ResponseWriter, r *http.Request) {
	weightIDStr := chi.URLParam(r, "id")
	weightID, err := uuid.Parse(weightIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid weight ID")
		return
	}

	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	entry, err := h.service.GetEntry(r.Context(), userID, weightID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, entry)
}

// GetLatest handles GET /api/v1/weight/latest
func (h *Handler) GetLatest(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	entry, err := h.service.GetLatestEntry(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, entry)
}

// ListWeights handles GET /api/v1/weight
func (h *Handler) ListWeights(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	// Parse date filters
	var startDate, endDate *time.Time
	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &t
		}
	}

	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = &t
		}
	}

	filter := WeightListFilter{
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      page,
		PageSize:  pageSize,
	}

	response, err := h.service.ListEntries(r.Context(), filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateWeight handles PUT /api/v1/weight/{id}
func (h *Handler) UpdateWeight(w http.ResponseWriter, r *http.Request) {
	weightIDStr := chi.URLParam(r, "id")
	weightID, err := uuid.Parse(weightIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid weight ID")
		return
	}

	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req UpdateWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry, err := h.service.UpdateEntry(r.Context(), userID, weightID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, entry)
}

// DeleteWeight handles DELETE /api/v1/weight/{id}
func (h *Handler) DeleteWeight(w http.ResponseWriter, r *http.Request) {
	weightIDStr := chi.URLParam(r, "id")
	weightID, err := uuid.Parse(weightIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid weight ID")
		return
	}

	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.service.DeleteEntry(r.Context(), userID, weightID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTrend handles GET /api/v1/weight/trend
func (h *Handler) GetTrend(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	stats, err := h.service.GetStats(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper functions

func getUserIDFromContext(ctx context.Context) uuid.UUID {
	// Use middleware helper function which handles typed context keys correctly
	userID, err := custommw.GetUserUUID(ctx)
	if err != nil {
		return uuid.Nil
	}
	return userID
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case ErrNotFound:
		respondError(w, http.StatusNotFound, "weight entry not found")
	case ErrInvalidWeight:
		respondError(w, http.StatusBadRequest, err.Error())
	case ErrFutureDate:
		respondError(w, http.StatusBadRequest, err.Error())
	case ErrDuplicateEntry:
		respondError(w, http.StatusConflict, err.Error())
	case ErrUnauthorized:
		respondError(w, http.StatusForbidden, "unauthorized access")
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
