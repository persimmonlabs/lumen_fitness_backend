// Package handlers provides HTTP request handlers for the fitness app API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/common_foods"
)

// CommonFoodsHandler handles common foods search requests.
type CommonFoodsHandler struct {
	service common_foods.Service
	logger  *slog.Logger
}

// NewCommonFoodsHandler creates a new common foods handler.
func NewCommonFoodsHandler(service common_foods.Service, logger *slog.Logger) *CommonFoodsHandler {
	return &CommonFoodsHandler{
		service: service,
		logger:  logger,
	}
}

// Search handles GET /api/v1/common-foods/search?q={query}&limit={limit}
func (h *CommonFoodsHandler) Search(w http.ResponseWriter, r *http.Request) {
	// Extract query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		h.errorResponse(w, http.StatusBadRequest, "query parameter 'q' is required", nil)
		return
	}

	// Extract optional limit parameter
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	h.logger.Info("common foods search",
		slog.String("query", query),
		slog.Int("limit", limit),
	)

	// Perform search
	foods, err := h.service.Search(r.Context(), query, limit)
	if err != nil {
		h.logger.Error("search failed",
			slog.String("query", query),
			slog.String("error", err.Error()),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to search common foods", nil)
		return
	}

	// Return results in the format the frontend expects
	response := map[string]interface{}{
		"foods": foods,
	}

	h.successResponse(w, http.StatusOK, response)
}

func (h *CommonFoodsHandler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *CommonFoodsHandler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
