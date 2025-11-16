package user

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// Handler handles HTTP requests for user management
type Handler struct {
	service Service
}

// NewHandler creates a new user handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetProfile handles GET /api/v1/user/profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/user/profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// GetSettings handles GET /api/v1/user/settings
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	settings, err := h.service.GetSettings(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// UpdateSettings handles PUT /api/v1/user/settings
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings, err := h.service.UpdateSettings(r.Context(), userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// Helper functions

func getUserIDFromContext(ctx context.Context) uuid.UUID {
	// This should extract user ID from context (set by auth middleware)
	userID := ctx.Value("user_id")
	if userID == nil {
		return uuid.Nil
	}
	if id, ok := userID.(uuid.UUID); ok {
		return id
	}
	if id, ok := userID.(string); ok {
		parsed, _ := uuid.Parse(id)
		return parsed
	}
	return uuid.Nil
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
		respondError(w, http.StatusNotFound, "user not found")
	case ErrInvalidProfile:
		respondError(w, http.StatusBadRequest, err.Error())
	case ErrInvalidSettings:
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
