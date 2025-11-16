// Package errors provides typed domain errors for the fitness application.
// It implements error types with HTTP status code mapping, error wrapping,
// and contextual error handling following Go 1.13+ error conventions.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors represent common error conditions across the application.
// These can be used directly or wrapped with additional context.
var (
	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrUnauthorized indicates the request lacks valid authentication credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the authenticated user lacks permission for the requested operation.
	ErrForbidden = errors.New("forbidden")

	// ErrConflict indicates a conflict with the current state of the resource.
	ErrConflict = errors.New("resource conflict")

	// ErrInternal indicates an internal server error.
	ErrInternal = errors.New("internal server error")

	// ErrBadRequest indicates invalid request parameters or payload.
	ErrBadRequest = errors.New("bad request")

	// ErrValidation indicates validation failed for the provided data.
	ErrValidation = errors.New("validation failed")

	// ErrInvalidCredentials indicates authentication failed due to invalid credentials.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenExpired indicates the authentication token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenInvalid indicates the authentication token is invalid or malformed.
	ErrTokenInvalid = errors.New("token invalid")

	// ErrDuplicate indicates an attempt to create a resource that already exists.
	ErrDuplicate = errors.New("duplicate resource")

	// ErrRateLimited indicates too many requests from the client.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrServiceUnavailable indicates the service is temporarily unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)

// AppError represents a structured application error with type information,
// HTTP status code mapping, and optional context data.
type AppError struct {
	// Type categorizes the error (e.g., "validation", "not_found")
	Type string `json:"type"`

	// Message provides a human-readable error description
	Message string `json:"message"`

	// StatusCode is the HTTP status code associated with this error
	StatusCode int `json:"-"`

	// Err is the underlying error being wrapped
	Err error `json:"-"`

	// Context provides additional structured data about the error
	Context map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error chain inspection.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext adds contextual information to the error.
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// ValidationError represents validation failure with field-specific details.
type ValidationError struct {
	// Fields maps field names to their validation error messages
	Fields map[string]string `json:"fields"`

	// Message provides an overall validation error description
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("validation failed: %s", e.Message)
	}
	return "validation failed"
}

// AddField adds a field validation error.
func (e *ValidationError) AddField(field, message string) *ValidationError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[field] = message
	return e
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}

// Error constructors

// NotFound creates a new not found error with optional context.
func NotFound(resource string, identifier interface{}) *AppError {
	return &AppError{
		Type:       "not_found",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
		Err:        ErrNotFound,
		Context: map[string]interface{}{
			"resource":   resource,
			"identifier": identifier,
		},
	}
}

// Unauthorized creates a new unauthorized error.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return &AppError{
		Type:       "unauthorized",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Err:        ErrUnauthorized,
	}
}

// Forbidden creates a new forbidden error.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "access denied"
	}
	return &AppError{
		Type:       "forbidden",
		Message:    message,
		StatusCode: http.StatusForbidden,
		Err:        ErrForbidden,
	}
}

// Conflict creates a new conflict error.
func Conflict(resource string, reason string) *AppError {
	return &AppError{
		Type:       "conflict",
		Message:    fmt.Sprintf("%s conflict: %s", resource, reason),
		StatusCode: http.StatusConflict,
		Err:        ErrConflict,
		Context: map[string]interface{}{
			"resource": resource,
			"reason":   reason,
		},
	}
}

// Internal creates a new internal server error, optionally wrapping an existing error.
func Internal(message string, err error) *AppError {
	if message == "" {
		message = "an internal error occurred"
	}
	return &AppError{
		Type:       "internal_error",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// BadRequest creates a new bad request error.
func BadRequest(message string) *AppError {
	if message == "" {
		message = "invalid request"
	}
	return &AppError{
		Type:       "bad_request",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        ErrBadRequest,
	}
}

// Validation creates a new validation error.
func Validation(message string) *ValidationError {
	if message == "" {
		message = "validation failed"
	}
	return &ValidationError{
		Message: message,
		Fields:  make(map[string]string),
	}
}

// InvalidCredentials creates a new invalid credentials error.
func InvalidCredentials() *AppError {
	return &AppError{
		Type:       "invalid_credentials",
		Message:    "invalid email or password",
		StatusCode: http.StatusUnauthorized,
		Err:        ErrInvalidCredentials,
	}
}

// TokenExpired creates a new token expired error.
func TokenExpired() *AppError {
	return &AppError{
		Type:       "token_expired",
		Message:    "authentication token has expired",
		StatusCode: http.StatusUnauthorized,
		Err:        ErrTokenExpired,
	}
}

// TokenInvalid creates a new token invalid error.
func TokenInvalid(reason string) *AppError {
	message := "invalid authentication token"
	if reason != "" {
		message = fmt.Sprintf("%s: %s", message, reason)
	}
	return &AppError{
		Type:       "token_invalid",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Err:        ErrTokenInvalid,
	}
}

// Duplicate creates a new duplicate resource error.
func Duplicate(resource string, field string) *AppError {
	return &AppError{
		Type:       "duplicate",
		Message:    fmt.Sprintf("%s already exists", resource),
		StatusCode: http.StatusConflict,
		Err:        ErrDuplicate,
		Context: map[string]interface{}{
			"resource": resource,
			"field":    field,
		},
	}
}

// RateLimited creates a new rate limit exceeded error.
func RateLimited(retryAfter int) *AppError {
	return &AppError{
		Type:       "rate_limited",
		Message:    "too many requests, please try again later",
		StatusCode: http.StatusTooManyRequests,
		Err:        ErrRateLimited,
		Context: map[string]interface{}{
			"retry_after_seconds": retryAfter,
		},
	}
}

// ServiceUnavailable creates a new service unavailable error.
func ServiceUnavailable(service string) *AppError {
	return &AppError{
		Type:       "service_unavailable",
		Message:    fmt.Sprintf("%s is temporarily unavailable", service),
		StatusCode: http.StatusServiceUnavailable,
		Err:        ErrServiceUnavailable,
		Context: map[string]interface{}{
			"service": service,
		},
	}
}

// Error type checking functions

// IsNotFound checks if the error is or wraps a not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "not_found" || errors.Is(appErr.Err, ErrNotFound)
	}
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if the error is or wraps an unauthorized error.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "unauthorized" || errors.Is(appErr.Err, ErrUnauthorized)
	}
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if the error is or wraps a forbidden error.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "forbidden" || errors.Is(appErr.Err, ErrForbidden)
	}
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if the error is or wraps a conflict error.
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "conflict" || errors.Is(appErr.Err, ErrConflict)
	}
	return errors.Is(err, ErrConflict)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	if err == nil {
		return false
	}
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return true
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Err, ErrValidation)
	}
	return errors.Is(err, ErrValidation)
}

// IsInternal checks if the error is or wraps an internal error.
func IsInternal(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "internal_error" || errors.Is(appErr.Err, ErrInternal)
	}
	return errors.Is(err, ErrInternal)
}

// IsBadRequest checks if the error is or wraps a bad request error.
func IsBadRequest(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "bad_request" || errors.Is(appErr.Err, ErrBadRequest)
	}
	return errors.Is(err, ErrBadRequest)
}

// IsDuplicate checks if the error is or wraps a duplicate error.
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "duplicate" || errors.Is(appErr.Err, ErrDuplicate)
	}
	return errors.Is(err, ErrDuplicate)
}

// IsRateLimited checks if the error is or wraps a rate limit error.
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == "rate_limited" || errors.Is(appErr.Err, ErrRateLimited)
	}
	return errors.Is(err, ErrRateLimited)
}

// Error wrapping helpers

// Wrap wraps an error with additional context while preserving the error chain.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// HTTP status code mapping

// GetHTTPStatus returns the appropriate HTTP status code for an error.
// If the error is not an AppError, it defaults to 500 Internal Server Error.
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	// Check for ValidationError
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return http.StatusBadRequest
	}

	// Check sentinel errors
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// GetErrorType returns a string representation of the error type.
func GetErrorType(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type
	}

	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return "validation"
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrBadRequest):
		return "bad_request"
	case errors.Is(err, ErrValidation):
		return "validation"
	case errors.Is(err, ErrDuplicate):
		return "duplicate"
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrServiceUnavailable):
		return "service_unavailable"
	default:
		return "internal_error"
	}
}

// GetErrorMessage returns a user-friendly error message.
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}

	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return valErr.Message
	}

	return err.Error()
}
