package media

import (
	"encoding/json"
	"net/http"
)

const (
	MaxUploadSize = 10 * 1024 * 1024 // 10MB
)

// Handler handles HTTP requests for media operations
type Handler struct {
	service *Service
}

// NewHandler creates a new media handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// UploadResponse represents a successful upload response
type UploadResponse struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// Upload handles POST /api/v1/media/upload
// @Summary Upload an image file
// @Description Upload and optimize an image file (max 10MB, jpg/png/heic only). Images are resized to max 1920px width and converted to JPEG format. Thumbnails (400px) are generated automatically.
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file to upload"
// @Success 200 {object} UploadResponse "Returns the URL of the uploaded image and its thumbnail"
// @Failure 400 {object} ErrorResponse "Invalid file type, size, or format"
// @Failure 413 {object} ErrorResponse "File size exceeds 10MB limit"
// @Failure 500 {object} ErrorResponse "Server error during processing"
// @Router /api/v1/media/upload [post]
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse multipart form with max memory
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "missing or invalid file parameter: "+err.Error())
		return
	}
	defer file.Close()

	// Check file size
	if fileHeader.Size > MaxUploadSize {
		h.writeError(w, http.StatusRequestEntityTooLarge, "file size exceeds maximum of 10MB")
		return
	}

	// Get content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Process and upload
	result, err := h.service.ProcessAndUpload(
		r.Context(),
		file,
		fileHeader.Filename,
		contentType,
		fileHeader.Size,
	)
	if err != nil {
		// Check for validation errors vs server errors
		if isValidationError(err) {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "failed to process upload: "+err.Error())
		}
		return
	}

	// Return success response
	h.writeJSON(w, http.StatusOK, UploadResponse{
		URL:          result.URL,
		ThumbnailURL: result.ThumbnailURL,
	})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change response at this point
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

// isValidationError checks if an error is a validation error
func isValidationError(err error) bool {
	errMsg := err.Error()
	validationErrors := []string{
		"file size exceeds",
		"invalid file extension",
		"invalid content type",
		"file is empty",
		"failed to decode image",
	}

	for _, validErr := range validationErrors {
		if contains(errMsg, validErr) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
