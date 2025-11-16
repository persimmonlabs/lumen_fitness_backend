package templates

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	custommw "github.com/pradord/lumen_final/backend/internal/server/middleware"
)

// Handler handles HTTP requests for templates
type Handler struct {
	service Service
}

// NewHandler creates a new template handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers template routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/templates", func(r chi.Router) {
		r.Post("/", h.CreateTemplate)
		r.Post("/from-meal", h.CreateFromMeal)
		r.Get("/", h.ListTemplates)
		r.Get("/{id}", h.GetTemplate)
		r.Put("/{id}", h.UpdateTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
		r.Post("/{id}/use", h.UseTemplate)
	})
}

// CreateTemplate handles POST /api/v1/templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	template, err := h.service.CreateTemplate(r.Context(), userID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, template)
}

// CreateFromMeal handles POST /api/v1/templates/from-meal
func (h *Handler) CreateFromMeal(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTemplateFromMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	template, err := h.service.CreateFromMeal(r.Context(), userID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, template)
}

// UseTemplate handles POST /api/v1/templates/{id}/use
func (h *Handler) UseTemplate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req UseTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	mealID, err := h.service.UseTemplate(r.Context(), userID, templateID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"meal_id": mealID,
		"message": "meal created from template",
	})
}

// ListTemplates handles GET /api/v1/templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	templates, err := h.service.ListTemplates(r.Context(), userID, limit, offset)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, templates)
}

// GetTemplate handles GET /api/v1/templates/{id}
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	template, err := h.service.GetTemplate(r.Context(), userID, templateID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, template)
}

// UpdateTemplate handles PUT /api/v1/templates/{id}
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	template, err := h.service.UpdateTemplate(r.Context(), userID, templateID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, template)
}

// DeleteTemplate handles DELETE /api/v1/templates/{id}
func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	err = h.service.DeleteTemplate(r.Context(), userID, templateID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "template deleted"})
}

// handleServiceError maps service errors to HTTP responses
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case ErrTemplateNotFound:
		respondWithError(w, http.StatusNotFound, err.Error())
	case ErrUnauthorized:
		respondWithError(w, http.StatusForbidden, err.Error())
	case ErrTemplateNameRequired, ErrTemplateItemsRequired, ErrInvalidServingSize:
		respondWithError(w, http.StatusBadRequest, err.Error())
	case ErrMealNotFound:
		respondWithError(w, http.StatusNotFound, err.Error())
	case ErrInvalidMealType, ErrInvalidMealTime:
		respondWithError(w, http.StatusBadRequest, err.Error())
	default:
		respondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	userID, err := custommw.GetUserUUID(r.Context())
	if err != nil {
		return uuid.Nil, http.ErrNotSupported
	}
	return userID, nil
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
