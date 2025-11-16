package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
// This allows us to log response details after the handler completes.
type responseWriter struct {
	http.ResponseWriter
	status       int
	wroteHeader  bool
	bytesWritten int64
}

// wrapResponseWriter creates a new responseWriter wrapper.
func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default to 200 if WriteHeader not called
	}
}

// WriteHeader captures the status code and calls the underlying WriteHeader.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures bytes written and calls the underlying Write.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Status returns the HTTP status code that was written.
func (rw *responseWriter) Status() int {
	return rw.status
}

// BytesWritten returns the number of bytes written to the response.
func (rw *responseWriter) BytesWritten() int64 {
	return rw.bytesWritten
}

// Logger middleware logs HTTP requests with structured information.
// It captures method, path, status code, duration, and trace ID.
//
// The middleware:
//   - Logs request start (method and path)
//   - Wraps response writer to capture status code and bytes
//   - Measures request duration
//   - Logs request completion with all details
//   - Includes trace ID for correlation
//
// Log format:
//
//	[TRACE_ID] METHOD /path -> STATUS (DURATION) BYTES bytes
//
// Example output:
//
//	[a1b2c3d4e5f6] GET /api/users -> 200 (15ms) 1234 bytes
//	[a1b2c3d4e5f6] POST /api/workouts -> 201 (42ms) 567 bytes
//	[x7y8z9a0b1c2] GET /api/invalid -> 404 (2ms) 89 bytes
//
// Usage:
//
//	// Apply to all routes
//	router.Use(middleware.TraceID)  // Must come before Logger
//	router.Use(middleware.Logger)
//
// Note: TraceID middleware must be applied before Logger to ensure
// trace IDs are available for logging.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get trace ID from context (set by TraceID middleware)
		traceID := GetTraceID(r.Context())
		if traceID == "" {
			traceID = "no-trace-id" // Fallback if TraceID middleware not used
		}

		// Log request start
		log.Printf("[%s] --> %s %s", traceID, r.Method, r.URL.Path)

		// Wrap response writer to capture status and bytes
		wrapped := wrapResponseWriter(w)

		// Record start time
		start := time.Now()

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion with details
		log.Printf(
			"[%s] <-- %s %s -> %d (%v) %d bytes",
			traceID,
			r.Method,
			r.URL.Path,
			wrapped.Status(),
			duration.Round(time.Millisecond),
			wrapped.BytesWritten(),
		)
	})
}

// LoggerWithConfig creates a Logger middleware with custom configuration.
// This allows for more control over what gets logged and how.
//
// Config options:
//   - LogRequestBody: Whether to log request body (not recommended for production)
//   - LogResponseBody: Whether to log response body (not recommended for production)
//   - LogHeaders: Whether to log request headers
//   - SkipPaths: Paths to skip logging (e.g., health checks)
//
// Usage:
//
//	config := LoggerConfig{
//	    SkipPaths: []string{"/health", "/metrics"},
//	    LogHeaders: false,
//	}
//	router.Use(middleware.LoggerWithConfig(config))
type LoggerConfig struct {
	// LogRequestBody enables logging of request bodies.
	// WARNING: This can log sensitive data and increase memory usage.
	LogRequestBody bool

	// LogResponseBody enables logging of response bodies.
	// WARNING: This can log sensitive data and increase memory usage.
	LogResponseBody bool

	// LogHeaders enables logging of request headers.
	// WARNING: This can log sensitive data (e.g., Authorization headers).
	LogHeaders bool

	// SkipPaths is a list of URL paths to skip logging.
	// Useful for health checks, metrics endpoints, etc.
	SkipPaths []string

	// SlowRequestThreshold defines what constitutes a "slow" request.
	// Requests slower than this will be logged with a WARNING prefix.
	// Zero value disables slow request detection.
	SlowRequestThreshold time.Duration
}

// LoggerWithConfig creates a Logger middleware with custom configuration.
func LoggerWithConfig(config LoggerConfig) func(http.Handler) http.Handler {
	// Build skip path map for O(1) lookup
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for configured paths
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Get trace ID
			traceID := GetTraceID(r.Context())
			if traceID == "" {
				traceID = "no-trace-id"
			}

			// Log request start
			log.Printf("[%s] --> %s %s", traceID, r.Method, r.URL.Path)

			// Optionally log headers
			if config.LogHeaders {
				for key, values := range r.Header {
					for _, value := range values {
						// Redact sensitive headers
						if key == "Authorization" || key == "Cookie" {
							value = "[REDACTED]"
						}
						log.Printf("[%s]     %s: %s", traceID, key, value)
					}
				}
			}

			// Wrap response writer
			wrapped := wrapResponseWriter(w)
			start := time.Now()

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Determine log level based on duration
			prefix := ""
			if config.SlowRequestThreshold > 0 && duration > config.SlowRequestThreshold {
				prefix = "SLOW "
			}

			// Log completion
			log.Printf(
				"[%s] %s<-- %s %s -> %d (%v) %d bytes",
				traceID,
				prefix,
				r.Method,
				r.URL.Path,
				wrapped.Status(),
				duration.Round(time.Millisecond),
				wrapped.BytesWritten(),
			)
		})
	}
}
