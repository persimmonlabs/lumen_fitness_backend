package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// ErrorResponse represents a standardized error response structure.
// All API errors should follow this format for consistency.
type ErrorResponse struct {
	// Error contains the error message
	Error string `json:"error"`

	// Code is an application-specific error code (optional)
	Code string `json:"code,omitempty"`

	// TraceID is the request trace ID for debugging
	TraceID string `json:"trace_id,omitempty"`

	// Details contains additional error context (optional, for debugging)
	Details interface{} `json:"details,omitempty"`
}

// AppError represents an application error with HTTP status code.
// Use this type in handlers to return structured errors.
type AppError struct {
	// StatusCode is the HTTP status code to return
	StatusCode int

	// Message is the error message shown to the user
	Message string

	// Code is an optional application-specific error code
	Code string

	// Err is the underlying error (for logging, not exposed to client)
	Err error

	// Details contains additional context (for debugging)
	Details interface{}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
// This allows errors.Is and errors.As to work with AppError.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Common application errors for convenience.
var (
	// ErrUnauthorized is returned when authentication is required but missing
	ErrUnauthorized = &AppError{
		StatusCode: http.StatusUnauthorized,
		Message:    "Authentication required",
		Code:       "AUTH_REQUIRED",
	}

	// ErrForbidden is returned when the user lacks permission
	ErrForbidden = &AppError{
		StatusCode: http.StatusForbidden,
		Message:    "Insufficient permissions",
		Code:       "FORBIDDEN",
	}

	// ErrNotFound is returned when a resource is not found
	ErrNotFound = &AppError{
		StatusCode: http.StatusNotFound,
		Message:    "Resource not found",
		Code:       "NOT_FOUND",
	}

	// ErrBadRequest is returned for invalid request data
	ErrBadRequest = &AppError{
		StatusCode: http.StatusBadRequest,
		Message:    "Invalid request data",
		Code:       "BAD_REQUEST",
	}

	// ErrConflict is returned when a resource already exists
	ErrConflict = &AppError{
		StatusCode: http.StatusConflict,
		Message:    "Resource already exists",
		Code:       "CONFLICT",
	}

	// ErrInternal is returned for internal server errors
	ErrInternal = &AppError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal server error",
		Code:       "INTERNAL_ERROR",
	}
)

// NewAppError creates a new AppError with the given status code and message.
func NewAppError(statusCode int, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// WithError adds an underlying error to the AppError.
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		StatusCode: e.StatusCode,
		Message:    e.Message,
		Code:       e.Code,
		Err:        err,
		Details:    e.Details,
	}
}

// WithDetails adds additional details to the AppError.
func (e *AppError) WithDetails(details interface{}) *AppError {
	return &AppError{
		StatusCode: e.StatusCode,
		Message:    e.Message,
		Code:       e.Code,
		Err:        e.Err,
		Details:    details,
	}
}

// Recovery middleware recovers from panics and converts them to 500 errors.
// This prevents the server from crashing on panics in handlers.
//
// The middleware:
//   - Catches panics in downstream handlers
//   - Logs the panic with stack trace
//   - Returns a 500 error to the client
//   - Includes trace ID for debugging
//
// Stack traces are logged for debugging but NOT sent to clients
// to avoid exposing internal implementation details.
//
// Usage:
//
//	// Apply early in middleware chain to catch all panics
//	router.Use(middleware.TraceID)
//	router.Use(middleware.Recovery)
//	router.Use(middleware.Logger)
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get trace ID for correlation
				traceID := GetTraceID(r.Context())

				// Log panic with stack trace
				log.Printf("[%s] PANIC: %v\n%s", traceID, err, debug.Stack())

				// Return 500 error to client
				WriteError(w, r, &AppError{
					StatusCode: http.StatusInternalServerError,
					Message:    "Internal server error",
					Code:       "PANIC_RECOVERED",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// WriteError writes an error response to the client.
// It handles both AppError and standard errors.
//
// For AppError: Uses the status code and message from the error.
// For other errors: Returns 500 with a generic message.
//
// The response includes:
//   - Error message (user-facing)
//   - Error code (if available)
//   - Trace ID (for debugging)
//   - Details (if available and not in production)
//
// Usage in handlers:
//
//	func Handler(w http.ResponseWriter, r *http.Request) {
//	    user, err := getUser(id)
//	    if err != nil {
//	        middleware.WriteError(w, r, middleware.ErrNotFound.WithError(err))
//	        return
//	    }
//	    // ... continue processing
//	}
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	// Get trace ID
	traceID := GetTraceID(r.Context())

	// Determine status code and message
	var statusCode int
	var message string
	var code string
	var details interface{}

	// Check if it's an AppError
	if e, ok := err.(*AppError); ok {
		statusCode = e.StatusCode
		message = e.Message
		code = e.Code
		details = e.Details

		// Log underlying error if present
		if e.Err != nil {
			log.Printf("[%s] Error: %v", traceID, e.Err)
		}
	} else {
		// Generic error - return 500
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
		code = "INTERNAL_ERROR"

		// Log the actual error
		log.Printf("[%s] Unexpected error: %v", traceID, err)
	}

	// Build response
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		TraceID: traceID,
	}

	// Include details for debugging (only for 4xx errors or in dev mode)
	// In production, omit details for 5xx errors to avoid leaking internals
	if statusCode < 500 && details != nil {
		response.Details = details
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[%s] Failed to encode error response: %v", traceID, err)
	}
}

// ErrorHandler is a middleware that provides centralized error handling.
// It allows handlers to return errors instead of manually writing responses.
//
// This is an alternative approach where handlers return errors:
//
//	type HandlerFunc func(http.ResponseWriter, *http.Request) error
//
// Usage:
//
//	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
//	    user, err := getUser(id)
//	    if err != nil {
//	        return middleware.ErrNotFound.WithError(err)
//	    }
//	    return writeJSON(w, user)
//	})
//	router.Handle("/users/{id}", handler)
func ErrorHandler(handler func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			WriteError(w, r, err)
		}
	})
}

// HTTPErrorToAppError converts standard HTTP errors to AppError.
// This is useful when integrating with libraries that return standard errors.
func HTTPErrorToAppError(statusCode int, err error) *AppError {
	message := http.StatusText(statusCode)
	if message == "" {
		message = "Unknown error"
	}

	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// WrapError wraps a standard error as an AppError with the given status code.
// This is a convenience function for quick error wrapping.
func WrapError(statusCode int, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}
