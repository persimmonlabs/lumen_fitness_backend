package meals

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	custommw "github.com/pradord/lumen_final/backend/internal/server/middleware"
)

// GetDraftStatus handles GET /api/v1/meals/draft/{id}/status
// Returns the current processing status of a draft meal
func (h *Handler) GetDraftStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := custommw.GetUserUUID(r.Context())
	if err != nil {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	draftIDStr := chi.URLParam(r, "id")
	draftID, err := uuid.Parse(draftIDStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid draft ID", nil)
		return
	}

	result, err := h.service.GetDraftStatus(r.Context(), userID, draftID)
	if err != nil {
		h.logger.Error("get draft status failed",
			slog.String("error", err.Error()),
			slog.String("draft_id", draftID.String()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusNotFound, "draft not found", nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}
