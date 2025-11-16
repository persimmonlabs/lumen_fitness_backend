package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrForbidden", ErrForbidden},
		{"ErrConflict", ErrConflict},
		{"ErrInternal", ErrInternal},
		{"ErrBadRequest", ErrBadRequest},
		{"ErrValidation", ErrValidation},
		{"ErrInvalidCredentials", ErrInvalidCredentials},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrTokenInvalid", ErrTokenInvalid},
		{"ErrDuplicate", ErrDuplicate},
		{"ErrRateLimited", ErrRateLimited},
		{"ErrServiceUnavailable", ErrServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("sentinel error %s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("sentinel error %s should have a message", tt.name)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name: "error with underlying error",
			appError: &AppError{
				Type:    "test",
				Message: "test message",
				Err:     errors.New("underlying"),
			},
			want: "test: test message: underlying",
		},
		{
			name: "error without underlying error",
			appError: &AppError{
				Type:    "test",
				Message: "test message",
			},
			want: "test: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appError.Error()
			if got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	appErr := &AppError{
		Type:    "test",
		Message: "test",
		Err:     underlying,
	}

	unwrapped := appErr.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestAppError_WithContext(t *testing.T) {
	appErr := &AppError{
		Type:    "test",
		Message: "test",
	}

	result := appErr.WithContext("key", "value")

	if result != appErr {
		t.Error("WithContext should return the same error instance")
	}

	if appErr.Context == nil {
		t.Error("Context should be initialized")
	}

	if appErr.Context["key"] != "value" {
		t.Errorf("Context[key] = %v, want %v", appErr.Context["key"], "value")
	}
}

func TestValidationError(t *testing.T) {
	t.Run("Error message", func(t *testing.T) {
		valErr := &ValidationError{Message: "custom message"}
		expected := "validation failed: custom message"
		if valErr.Error() != expected {
			t.Errorf("Error() = %v, want %v", valErr.Error(), expected)
		}
	})

	t.Run("Default message", func(t *testing.T) {
		valErr := &ValidationError{}
		expected := "validation failed"
		if valErr.Error() != expected {
			t.Errorf("Error() = %v, want %v", valErr.Error(), expected)
		}
	})

	t.Run("AddField", func(t *testing.T) {
		valErr := &ValidationError{}
		result := valErr.AddField("email", "invalid email")

		if result != valErr {
			t.Error("AddField should return the same error instance")
		}

		if valErr.Fields["email"] != "invalid email" {
			t.Errorf("Fields[email] = %v, want %v", valErr.Fields["email"], "invalid email")
		}
	})

	t.Run("HasErrors", func(t *testing.T) {
		valErr := &ValidationError{}
		if valErr.HasErrors() {
			t.Error("HasErrors() should return false for empty fields")
		}

		valErr.AddField("test", "error")
		if !valErr.HasErrors() {
			t.Error("HasErrors() should return true when fields exist")
		}
	})
}

func TestNotFound(t *testing.T) {
	err := NotFound("user", 123)

	if err.Type != "not_found" {
		t.Errorf("Type = %v, want not_found", err.Type)
	}

	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusNotFound)
	}

	if err.Context["resource"] != "user" {
		t.Errorf("Context[resource] = %v, want user", err.Context["resource"])
	}

	if err.Context["identifier"] != 123 {
		t.Errorf("Context[identifier] = %v, want 123", err.Context["identifier"])
	}

	if !errors.Is(err.Err, ErrNotFound) {
		t.Error("Err should wrap ErrNotFound")
	}
}

func TestUnauthorized(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{"with message", "custom message", "custom message"},
		{"empty message", "", "authentication required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unauthorized(tt.message)

			if err.Message != tt.want {
				t.Errorf("Message = %v, want %v", err.Message, tt.want)
			}

			if err.StatusCode != http.StatusUnauthorized {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("custom reason")

	if err.Message != "custom reason" {
		t.Errorf("Message = %v, want custom reason", err.Message)
	}

	if err.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusForbidden)
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("workout", "already exists")

	if err.Type != "conflict" {
		t.Errorf("Type = %v, want conflict", err.Type)
	}

	if err.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusConflict)
	}

	if err.Context["resource"] != "workout" {
		t.Errorf("Context[resource] = %v, want workout", err.Context["resource"])
	}
}

func TestInternal(t *testing.T) {
	underlying := errors.New("database error")
	err := Internal("database operation failed", underlying)

	if err.Type != "internal_error" {
		t.Errorf("Type = %v, want internal_error", err.Type)
	}

	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusInternalServerError)
	}

	if err.Err != underlying {
		t.Error("Should wrap the underlying error")
	}
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid format")

	if err.Type != "bad_request" {
		t.Errorf("Type = %v, want bad_request", err.Type)
	}

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusBadRequest)
	}
}

func TestValidation(t *testing.T) {
	valErr := Validation("custom validation message")

	if valErr.Message != "custom validation message" {
		t.Errorf("Message = %v, want custom validation message", valErr.Message)
	}

	if valErr.Fields == nil {
		t.Error("Fields should be initialized")
	}
}

func TestInvalidCredentials(t *testing.T) {
	err := InvalidCredentials()

	if err.Type != "invalid_credentials" {
		t.Errorf("Type = %v, want invalid_credentials", err.Type)
	}

	if err.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusUnauthorized)
	}
}

func TestTokenExpired(t *testing.T) {
	err := TokenExpired()

	if err.Type != "token_expired" {
		t.Errorf("Type = %v, want token_expired", err.Type)
	}

	if !errors.Is(err.Err, ErrTokenExpired) {
		t.Error("Should wrap ErrTokenExpired")
	}
}

func TestTokenInvalid(t *testing.T) {
	err := TokenInvalid("malformed")

	if err.Type != "token_invalid" {
		t.Errorf("Type = %v, want token_invalid", err.Type)
	}

	if err.Message != "invalid authentication token: malformed" {
		t.Errorf("Message = %v, want message with reason", err.Message)
	}
}

func TestDuplicate(t *testing.T) {
	err := Duplicate("user", "email")

	if err.Type != "duplicate" {
		t.Errorf("Type = %v, want duplicate", err.Type)
	}

	if err.Context["field"] != "email" {
		t.Errorf("Context[field] = %v, want email", err.Context["field"])
	}
}

func TestRateLimited(t *testing.T) {
	err := RateLimited(60)

	if err.Type != "rate_limited" {
		t.Errorf("Type = %v, want rate_limited", err.Type)
	}

	if err.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusTooManyRequests)
	}

	if err.Context["retry_after_seconds"] != 60 {
		t.Errorf("Context[retry_after_seconds] = %v, want 60", err.Context["retry_after_seconds"])
	}
}

func TestServiceUnavailable(t *testing.T) {
	err := ServiceUnavailable("database")

	if err.Type != "service_unavailable" {
		t.Errorf("Type = %v, want service_unavailable", err.Type)
	}

	if err.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"NotFound AppError", NotFound("user", 1), true},
		{"Sentinel ErrNotFound", ErrNotFound, true},
		{"Wrapped ErrNotFound", Wrap(ErrNotFound, "context"), true},
		{"Other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"Unauthorized AppError", Unauthorized("test"), true},
		{"Sentinel ErrUnauthorized", ErrUnauthorized, true},
		{"Other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnauthorized(tt.err)
			if got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"ValidationError", Validation("test"), true},
		{"Sentinel ErrValidation", ErrValidation, true},
		{"Other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidation(tt.err)
			if got != tt.want {
				t.Errorf("IsValidation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"Duplicate AppError", Duplicate("user", "email"), true},
		{"Sentinel ErrDuplicate", ErrDuplicate, true},
		{"Other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDuplicate(tt.err)
			if got != tt.want {
				t.Errorf("IsDuplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		want    string
	}{
		{"nil error", nil, "context", ""},
		{"wrap error", errors.New("original"), "context", "context: original"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Wrap(tt.err, tt.message)
			if tt.err == nil {
				if wrapped != nil {
					t.Errorf("Wrap() should return nil for nil error")
				}
			} else {
				if wrapped.Error() != tt.want {
					t.Errorf("Wrap() = %v, want %v", wrapped.Error(), tt.want)
				}
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	err := errors.New("original")
	wrapped := Wrapf(err, "user %d not found", 123)

	expected := "user 123 not found: original"
	if wrapped.Error() != expected {
		t.Errorf("Wrapf() = %v, want %v", wrapped.Error(), expected)
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil error", nil, http.StatusOK},
		{"NotFound", NotFound("user", 1), http.StatusNotFound},
		{"Unauthorized", Unauthorized("test"), http.StatusUnauthorized},
		{"Forbidden", Forbidden("test"), http.StatusForbidden},
		{"Conflict", Conflict("test", "reason"), http.StatusConflict},
		{"Internal", Internal("test", nil), http.StatusInternalServerError},
		{"BadRequest", BadRequest("test"), http.StatusBadRequest},
		{"ValidationError", Validation("test"), http.StatusBadRequest},
		{"Duplicate", Duplicate("user", "email"), http.StatusConflict},
		{"RateLimited", RateLimited(60), http.StatusTooManyRequests},
		{"ServiceUnavailable", ServiceUnavailable("db"), http.StatusServiceUnavailable},
		{"Sentinel ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"Unknown error", errors.New("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("GetHTTPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"NotFound", NotFound("user", 1), "not_found"},
		{"Unauthorized", Unauthorized("test"), "unauthorized"},
		{"Validation", Validation("test"), "validation"},
		{"Duplicate", Duplicate("user", "email"), "duplicate"},
		{"Unknown error", errors.New("unknown"), "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetErrorType(tt.err)
			if got != tt.want {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"NotFound", NotFound("user", 1), "user not found"},
		{"ValidationError", Validation("invalid data"), "invalid data"},
		{"Generic error", errors.New("generic"), "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetErrorMessage(tt.err)
			if got != tt.want {
				t.Errorf("GetErrorMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that errors can be properly unwrapped through the chain
	original := errors.New("database connection failed")
	wrapped1 := Wrap(original, "failed to save user")
	wrapped2 := Wrapf(wrapped1, "user registration failed for user %s", "john@example.com")

	if !errors.Is(wrapped2, original) {
		t.Error("errors.Is should find original error through chain")
	}
}

func TestAppErrorChaining(t *testing.T) {
	// Test AppError with error chain
	original := errors.New("connection timeout")
	appErr := Internal("database error", original)

	if !errors.Is(appErr, original) {
		t.Error("errors.Is should find original error in AppError")
	}

	if appErr.Unwrap() != original {
		t.Error("Unwrap should return original error")
	}
}
