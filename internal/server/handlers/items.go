package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/pradord/lumen_final/backend/internal/supabase"
)

// ItemHandler handles item endpoints
type ItemHandler struct {
	*BaseHandler
	supabase *supabase.Client
}

// NewItemHandler creates a new item handler
func NewItemHandler(logger *slog.Logger, sb *supabase.Client) *ItemHandler {
	return &ItemHandler{
		BaseHandler: NewBaseHandler(logger),
		supabase:    sb,
	}
}

// CreateItemRequest validates item creation
type CreateItemRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"max=500"`
}

// ItemResponse is the API response
type ItemResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Create handles POST /api/v1/items
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.BadRequestError(w, err.Error(), nil)
		return
	}

	// For now, just return mock data since we don't have Supabase table yet
	mockItem := ItemResponse{
		ID:          "mock-id-123",
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	h.SuccessResponse(w, http.StatusCreated, mockItem, nil)
}

// List handles GET /api/v1/items
func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	// Return empty list for now
	items := []ItemResponse{}
	h.SuccessResponse(w, http.StatusOK, items, nil)
}
