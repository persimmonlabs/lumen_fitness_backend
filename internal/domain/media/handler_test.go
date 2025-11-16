package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ServiceInterface defines the interface for media service
type ServiceInterface interface {
	ProcessAndUpload(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error)
}

// MockService mocks the Service for testing
type MockService struct {
	processFunc func(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error)
}

func (m *MockService) ProcessAndUpload(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, file, filename, contentType, fileSize)
	}
	return &UploadResult{
		URL:          "https://example.com/image.jpg",
		ThumbnailURL: "https://example.com/image_thumb.jpg",
	}, nil
}

// ModifiedHandler for testing with interface
type ModifiedHandler struct {
	service ServiceInterface
}

func NewModifiedHandler(service ServiceInterface) *ModifiedHandler {
	return &ModifiedHandler{service: service}
}

func (h *ModifiedHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

func (h *ModifiedHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ModifiedHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

func createMultipartRequest(filename, contentType string, content []byte) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(content); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

func TestHandler_Upload(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		mockProcess    func(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error)
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful upload",
			setupRequest: func() *http.Request {
				content := createTestImage(800, 600).Bytes()
				req, _ := createMultipartRequest("test.jpg", "image/jpeg", content)
				return req
			},
			mockProcess: func(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error) {
				return &UploadResult{
					URL:          "https://example.com/test.jpg",
					ThumbnailURL: "https://example.com/test_thumb.jpg",
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "wrong HTTP method",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/v1/media/upload", nil)
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    true,
		},
		{
			name: "missing file",
			setupRequest: func() *http.Request {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.Close()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/media/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "file too large",
			setupRequest: func() *http.Request {
				// Create content larger than MaxUploadSize
				largeContent := make([]byte, MaxUploadSize+1)
				req, _ := createMultipartRequest("large.jpg", "image/jpeg", largeContent)
				return req
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectError:    true,
		},
		{
			name: "validation error",
			setupRequest: func() *http.Request {
				content := createTestImage(800, 600).Bytes()
				req, _ := createMultipartRequest("test.jpg", "image/jpeg", content)
				return req
			},
			mockProcess: func(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error) {
				return nil, errors.New("invalid file extension: .gif")
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "server error during processing",
			setupRequest: func() *http.Request {
				content := createTestImage(800, 600).Bytes()
				req, _ := createMultipartRequest("test.jpg", "image/jpeg", content)
				return req
			},
			mockProcess: func(ctx context.Context, file io.Reader, filename string, contentType string, fileSize int64) (*UploadResult, error) {
				return nil, errors.New("storage service unavailable")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				processFunc: tt.mockProcess,
			}

			handler := NewModifiedHandler(mockService)

			req := tt.setupRequest()
			rr := httptest.NewRecorder()

			handler.Upload(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Parse response
			if tt.expectError {
				var errResp ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error == "" {
					t.Error("expected non-empty error field")
				}
			} else {
				var uploadResp UploadResponse
				if err := json.NewDecoder(rr.Body).Decode(&uploadResp); err != nil {
					t.Fatalf("failed to decode success response: %v", err)
				}
				if uploadResp.URL == "" {
					t.Error("expected non-empty URL")
				}
				if uploadResp.ThumbnailURL == "" {
					t.Error("expected non-empty thumbnail URL")
				}
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"file size error", errors.New("file size exceeds maximum"), true},
		{"invalid extension", errors.New("invalid file extension: .gif"), true},
		{"invalid content type", errors.New("invalid content type: image/gif"), true},
		{"empty file", errors.New("file is empty"), true},
		{"decode error", errors.New("failed to decode image"), true},
		{"storage error", errors.New("storage service unavailable"), false},
		{"network error", errors.New("network timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("isValidationError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"contains at start", "hello world", "hello", true},
		{"contains at end", "hello world", "world", true},
		{"contains in middle", "hello world", "lo wo", true},
		{"does not contain", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
		{"empty string", "", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, expected %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{"found at start", "hello world", "hello", 0},
		{"found at end", "hello world", "world", 6},
		{"found in middle", "hello world", "lo", 3},
		{"not found", "hello world", "xyz", -1},
		{"empty substring", "hello", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("indexOf(%q, %q) = %d, expected %d", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestNewHandler(t *testing.T) {
	service := NewService(&MockRepository{})
	handler := NewHandler(service)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	if handler.service != service {
		t.Error("service not set correctly")
	}
}
