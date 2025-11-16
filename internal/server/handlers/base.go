// Package handlers provides HTTP request handlers for the fitness app API.
//
// This package contains the base handler functionality and common utilities
// used across all API endpoints including JSON response helpers, error
// formatting, and request validation utilities.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// BaseHandler provides common functionality for all API handlers.
// It includes utilities for JSON responses, error handling, and logging.
type BaseHandler struct {
	logger *slog.Logger
}

// NewBaseHandler creates a new base handler instance with the provided logger.
func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

// DecodeJSON decodes JSON request body into the provided struct and validates it
func (h *BaseHandler) DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate the struct
	if err := ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// Response represents a standard API response structure.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo contains detailed error information for API responses.
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Meta contains metadata for API responses (pagination, timestamps, etc).
type Meta struct {
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	Pagination *PaginationMeta        `json:"pagination,omitempty"`
	Additional map[string]interface{} `json:"additional,omitempty"`
}

// PaginationMeta contains pagination information.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

// JSONResponse sends a JSON response with the given status code and data.
// It automatically sets the Content-Type header and handles marshaling errors.
func (h *BaseHandler) JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response",
			slog.String("error", err.Error()),
		)
	}
}

// SuccessResponse sends a successful JSON response with optional metadata.
func (h *BaseHandler) SuccessResponse(w http.ResponseWriter, statusCode int, data interface{}, meta *Meta) {
	if meta == nil {
		meta = &Meta{
			Timestamp: time.Now().UTC(),
		}
	} else if meta.Timestamp.IsZero() {
		meta.Timestamp = time.Now().UTC()
	}

	response := Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	h.JSONResponse(w, statusCode, response)
}

// ErrorResponse sends an error JSON response with the given status code and error information.
func (h *BaseHandler) ErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &Meta{
			Timestamp: time.Now().UTC(),
		},
	}

	h.logger.Warn("API error response",
		slog.Int("status_code", statusCode),
		slog.String("error_code", code),
		slog.String("message", message),
	)

	h.JSONResponse(w, statusCode, response)
}

// BadRequestError sends a 400 Bad Request error response.
func (h *BaseHandler) BadRequestError(w http.ResponseWriter, message string, details map[string]interface{}) {
	h.ErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", message, details)
}

// UnauthorizedError sends a 401 Unauthorized error response.
func (h *BaseHandler) UnauthorizedError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required"
	}
	h.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// ForbiddenError sends a 403 Forbidden error response.
func (h *BaseHandler) ForbiddenError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	h.ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", message, nil)
}

// NotFoundError sends a 404 Not Found error response.
func (h *BaseHandler) NotFoundError(w http.ResponseWriter, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	h.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", message, nil)
}

// ConflictError sends a 409 Conflict error response.
func (h *BaseHandler) ConflictError(w http.ResponseWriter, message string, details map[string]interface{}) {
	if message == "" {
		message = "Resource conflict"
	}
	h.ErrorResponse(w, http.StatusConflict, "CONFLICT", message, details)
}

// ValidationError sends a 422 Unprocessable Entity error response for validation failures.
func (h *BaseHandler) ValidationError(w http.ResponseWriter, details map[string]interface{}) {
	h.ErrorResponse(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", details)
}

// InternalServerError sends a 500 Internal Server Error response.
func (h *BaseHandler) InternalServerError(w http.ResponseWriter, err error) {
	message := "Internal server error"
	h.logger.Error("internal server error",
		slog.String("error", err.Error()),
	)
	h.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

// ServiceUnavailableError sends a 503 Service Unavailable error response.
func (h *BaseHandler) ServiceUnavailableError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	h.ErrorResponse(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, nil)
}

// GetRequestID extracts the request ID from the context.
// Returns an empty string if no request ID is found.
func (h *BaseHandler) GetRequestID(r *http.Request) string {
	if id := r.Context().Value("request_id"); id != nil {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}

// LogRequest logs the incoming HTTP request with relevant details.
func (h *BaseHandler) LogRequest(r *http.Request, message string) {
	h.logger.Info(message,
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
		slog.String("request_id", h.GetRequestID(r)),
	)
}

// LogError logs an error with request context.
func (h *BaseHandler) LogError(r *http.Request, message string, err error) {
	h.logger.Error(message,
		slog.String("error", err.Error()),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("request_id", h.GetRequestID(r)),
	)
}
