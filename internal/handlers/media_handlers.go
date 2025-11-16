// Package handlers provides HTTP request handlers for the fitness app API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/services/media"
)

// MediaHandler handles media upload requests.
type MediaHandler struct {
	service *media.Service
	logger  *slog.Logger
}

// NewMediaHandler creates a new media handler.
func NewMediaHandler(service *media.Service, logger *slog.Logger) *MediaHandler {
	return &MediaHandler{
		service: service,
		logger:  logger,
	}
}

// Upload handles POST /api/v1/media/upload
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to parse form", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "file is required", nil)
		return
	}
	defer file.Close()

	// Create upload request
	uploadReq := &media.UploadRequest{
		UserID:      userID,
		File:        file,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
	}

	// Upload file
	result, err := h.service.Upload(r.Context(), uploadReq)
	if err != nil {
		h.logger.Error("media upload failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// Delete handles DELETE /api/v1/media/{file_id}
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		h.errorResponse(w, http.StatusBadRequest, "file_id is required", nil)
		return
	}

	if err := h.service.Delete(r.Context(), userID, fileID); err != nil {
		h.logger.Error("media delete failed",
			slog.String("error", err.Error()),
			slog.String("file_id", fileID),
		)
		h.errorResponse(w, http.StatusInternalServerError, "failed to delete file", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetURL handles GET /api/v1/media/{file_id}/url
func (h *MediaHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		h.errorResponse(w, http.StatusBadRequest, "file_id is required", nil)
		return
	}

	url, err := h.service.GetURL(r.Context(), userID, fileID)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, "file not found", nil)
		return
	}

	h.successResponse(w, http.StatusOK, map[string]string{"url": url})
}

func (h *MediaHandler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *MediaHandler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
