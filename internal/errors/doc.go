// Package errors provides comprehensive typed domain errors for the fitness application.
//
// This package implements structured error handling with HTTP status code mapping,
// error wrapping, and contextual error information following Go 1.13+ error conventions.
//
// # Features
//
//   - Sentinel errors for common error conditions
//   - Typed AppError with HTTP status codes and context
//   - ValidationError for field-specific validation failures
//   - Error wrapping helpers (Wrap, Wrapf)
//   - Type checking functions (IsNotFound, IsValidation, etc.)
//   - HTTP status code mapping (GetHTTPStatus)
//
// # Basic Usage
//
// Create errors:
//
//	err := errors.NotFound("user", 123)
//	err := errors.Unauthorized("invalid token")
//	err := errors.Validation("invalid user data").AddField("email", "required")
//
// Check error types:
//
//	if errors.IsNotFound(err) {
//	    // Handle not found
//	}
//
// Wrap errors:
//
//	err = errors.Wrap(dbErr, "failed to fetch user")
//	err = errors.Wrapf(err, "user service error for ID %d", userID)
//
// Get HTTP status:
//
//	statusCode := errors.GetHTTPStatus(err)
//	errorType := errors.GetErrorType(err)
//	message := errors.GetErrorMessage(err)
//
// # Error Types
//
// AppError represents a structured application error:
//
//	type AppError struct {
//	    Type       string                 // Error type (e.g., "not_found")
//	    Message    string                 // Human-readable message
//	    StatusCode int                    // HTTP status code
//	    Err        error                  // Underlying error
//	    Context    map[string]interface{} // Additional context
//	}
//
// ValidationError represents field-specific validation failures:
//
//	type ValidationError struct {
//	    Fields  map[string]string // Field -> error message
//	    Message string            // Overall message
//	}
//
// # Sentinel Errors
//
// Predefined error constants that can be used directly or wrapped:
//
//	ErrNotFound, ErrUnauthorized, ErrForbidden, ErrConflict,
//	ErrInternal, ErrBadRequest, ErrValidation, ErrDuplicate,
//	ErrInvalidCredentials, ErrTokenExpired, ErrTokenInvalid,
//	ErrRateLimited, ErrServiceUnavailable
//
// # HTTP Status Mappings
//
//	NotFound           -> 404
//	Unauthorized       -> 401
//	Forbidden          -> 403
//	BadRequest         -> 400
//	Validation         -> 400
//	Conflict           -> 409
//	Duplicate          -> 409
//	Internal           -> 500
//	RateLimited        -> 429
//	ServiceUnavailable -> 503
//
// # Example: Repository Layer
//
//	func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
//	    var user User
//	    err := r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = ?", id).Scan(&user)
//
//	    if err == sql.ErrNoRows {
//	        return nil, errors.NotFound("user", id)
//	    }
//
//	    if err != nil {
//	        return nil, errors.Internal("failed to fetch user", err)
//	    }
//
//	    return &user, nil
//	}
//
// # Example: Service Layer
//
//	func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
//	    // Validate input
//	    valErr := errors.Validation("user validation failed")
//
//	    if req.Email == "" {
//	        valErr.AddField("email", "email is required")
//	    }
//	    if len(req.Password) < 8 {
//	        valErr.AddField("password", "password must be at least 8 characters")
//	    }
//
//	    if valErr.HasErrors() {
//	        return nil, valErr
//	    }
//
//	    // Check for duplicate
//	    exists, err := s.repo.ExistsByEmail(ctx, req.Email)
//	    if err != nil {
//	        return nil, errors.Wrap(err, "failed to check email existence")
//	    }
//	    if exists {
//	        return nil, errors.Duplicate("user", "email")
//	    }
//
//	    // Create user
//	    user, err := s.repo.Create(ctx, req)
//	    if err != nil {
//	        return nil, errors.Wrapf(err, "failed to create user with email %s", req.Email)
//	    }
//
//	    return user, nil
//	}
//
// # Example: HTTP Handler
//
//	func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
//	    id := chi.URLParam(r, "id")
//
//	    user, err := h.service.GetByID(r.Context(), id)
//	    if err != nil {
//	        statusCode := errors.GetHTTPStatus(err)
//	        errorType := errors.GetErrorType(err)
//	        message := errors.GetErrorMessage(err)
//
//	        response := map[string]interface{}{
//	            "error": map[string]interface{}{
//	                "type":    errorType,
//	                "message": message,
//	            },
//	        }
//
//	        w.WriteHeader(statusCode)
//	        json.NewEncoder(w).Encode(response)
//	        return
//	    }
//
//	    json.NewEncoder(w).Encode(user)
//	}
package errors
