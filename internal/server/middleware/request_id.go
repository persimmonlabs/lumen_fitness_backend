package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// RequestIDKey is the context key for request ID.
const RequestIDKey = "request_id"

// RequestID is a middleware that injects a unique request ID into each request.
// It uses chi's built-in RequestID middleware as a base.
func RequestID(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use chi's RequestID middleware first
		chiMiddleware := middleware.RequestID(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get request ID from chi middleware
			requestID := middleware.GetReqID(r.Context())

			// If no request ID from chi, generate one
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Add to context with our key
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			// Add to response headers for tracing
			w.Header().Set("X-Request-ID", requestID)

			logger.Debug("request started",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			chiMiddleware.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID extracts request ID from context.
func GetRequestID(ctx context.Context) string {
	// Try our key first
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}

	// Fall back to chi's request ID
	return middleware.GetReqID(ctx)
}
