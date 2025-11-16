// Package handlers provides HTTP request handlers for the fitness app API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/services/voice"
)

// VoiceHandler handles voice transcription requests.
type VoiceHandler struct {
	service *voice.Service
	logger  *slog.Logger
}

// NewVoiceHandler creates a new voice handler.
func NewVoiceHandler(service *voice.Service, logger *slog.Logger) *VoiceHandler {
	return &VoiceHandler{
		service: service,
		logger:  logger,
	}
}

// Transcribe handles POST /api/v1/voice/transcribe
func (h *VoiceHandler) Transcribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		h.errorResponse(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Parse multipart form (max 25MB for audio)
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to parse form", nil)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "audio file is required", nil)
		return
	}
	defer file.Close()

	// Get optional language parameter
	language := r.FormValue("language")
	if language == "" {
		language = "en"
	}

	// Create transcription request
	transcribeReq := &voice.TranscribeRequest{
		UserID:      userID,
		AudioFile:   file,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Language:    language,
	}

	// Transcribe audio
	result, err := h.service.Transcribe(r.Context(), transcribeReq)
	if err != nil {
		h.logger.Error("voice transcription failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		h.errorResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	h.successResponse(w, http.StatusOK, result)
}

// GetSupportedFormats handles GET /api/v1/voice/formats
func (h *VoiceHandler) GetSupportedFormats(w http.ResponseWriter, r *http.Request) {
	formats := h.service.GetSupportedFormats()
	h.successResponse(w, http.StatusOK, map[string]interface{}{
		"formats": formats,
	})
}

func (h *VoiceHandler) successResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *VoiceHandler) errorResponse(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"details": details,
	})
}
