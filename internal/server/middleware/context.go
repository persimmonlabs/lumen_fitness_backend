// Package middleware provides HTTP middleware for the fitness app API.
// It includes logging, error handling, CORS, authentication, and context management.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Context keys for storing values in request context.
// Using custom type prevents collisions with other packages.
type contextKey string

const (
	// TraceIDKey is the context key for storing trace/request IDs
	TraceIDKey contextKey = "trace_id"

	// UserIDKey is the context key for storing authenticated user ID
	UserIDKey contextKey = "user_id"

	// AuthTokenKey is the context key for storing the authentication token
	AuthTokenKey contextKey = "auth_token"
)

// TraceID generates a unique trace ID for request tracking and adds it to context.
// The trace ID is used for correlating logs across the request lifecycle.
//
// The middleware:
//   - Generates a cryptographically random 16-byte (32 hex chars) trace ID
//   - Stores it in the request context for access by handlers
//   - Adds it to response headers for client-side correlation
//
// Usage:
//
//	router.Use(middleware.TraceID)
//
// Retrieving trace ID in handlers:
//
//	traceID := middleware.GetTraceID(r.Context())
func TraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate trace ID
		traceID := generateTraceID()

		// Add to context
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)

		// Add to response headers for client correlation
		w.Header().Set("X-Trace-ID", traceID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTraceID retrieves the trace ID from the request context.
// Returns empty string if no trace ID is found.
//
// Usage:
//
//	traceID := middleware.GetTraceID(r.Context())
//	log.Printf("[%s] Processing request", traceID)
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetUserID retrieves the authenticated user ID from the request context.
// Returns empty string if no user is authenticated.
//
// Usage:
//
//	userID := middleware.GetUserID(r.Context())
//	if userID == "" {
//	    // User not authenticated
//	}
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	// Try UUID type (for compatibility with auth middleware)
	if userUUID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return userUUID.String()
	}
	return ""
}

// GetUserUUID retrieves the authenticated user ID as UUID from the request context.
// Returns uuid.Nil and an error if no user is authenticated or if the ID is not a valid UUID.
//
// Usage:
//
//	userID, err := middleware.GetUserUUID(r.Context())
//	if err != nil {
//	    // User not authenticated or invalid UUID
//	}
func GetUserUUID(ctx context.Context) (uuid.UUID, error) {
	// Try to get UUID directly
	if userUUID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return userUUID, nil
	}
	// Try to get string and parse to UUID
	if userIDStr, ok := ctx.Value(UserIDKey).(string); ok {
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid user ID format: %w", err)
		}
		return userUUID, nil
	}
	return uuid.Nil, fmt.Errorf("user ID not found in context")
}

// GetAuthToken retrieves the authentication token from the request context.
// Returns empty string if no token is present.
//
// Usage:
//
//	token := middleware.GetAuthToken(r.Context())
func GetAuthToken(ctx context.Context) string {
	if token, ok := ctx.Value(AuthTokenKey).(string); ok {
		return token
	}
	return ""
}

// SetUserID adds a user ID to the request context.
// This is typically called by the authentication middleware.
//
// Returns a new context with the user ID set.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// SetAuthToken adds an authentication token to the request context.
// This is typically called by the authentication middleware.
//
// Returns a new context with the token set.
func SetAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, AuthTokenKey, token)
}

// generateTraceID creates a cryptographically random trace ID.
// Returns a 32-character hexadecimal string (16 random bytes).
//
// Format: 32 hex characters (e.g., "a1b2c3d4e5f6...")
//
// In the unlikely event of random generation failure, returns a
// timestamp-based fallback ID.
func generateTraceID() string {
	// Generate 16 random bytes (128 bits)
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random fails
		// This should never happen in practice
		return generateFallbackTraceID()
	}

	// Convert to hex string (32 characters)
	return hex.EncodeToString(b)
}

// generateFallbackTraceID creates a timestamp-based trace ID as fallback.
// Only used if crypto/rand fails (extremely rare).
func generateFallbackTraceID() string {
	// Use nano timestamp as fallback
	// This maintains uniqueness within a single process
	// but may have collisions across distributed systems
	return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))
}
