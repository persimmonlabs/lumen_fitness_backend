// Package middleware provides HTTP middleware for the fitness app API.
//
// # Overview
//
// This package implements a comprehensive middleware stack for Go HTTP servers,
// providing essential features like logging, error handling, CORS, authentication,
// and request context management.
//
// # Middleware Chain Order
//
// Middleware should be applied in the following order for optimal functionality:
//
//	router.Use(middleware.Recovery)      // 1. Catch panics first
//	router.Use(middleware.TraceID)       // 2. Generate trace IDs early
//	router.Use(middleware.Logger)        // 3. Log all requests
//	router.Use(middleware.CORS(config))  // 4. Handle CORS
//	router.Use(middleware.Auth(config))  // 5. Authenticate users
//
// # Trace ID and Context
//
// The TraceID middleware generates a unique identifier for each request,
// which is then available throughout the request lifecycle:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    traceID := middleware.GetTraceID(r.Context())
//	    log.Printf("[%s] Processing request", traceID)
//	}
//
// Trace IDs are:
//   - 32-character hexadecimal strings (128 bits of randomness)
//   - Added to response headers as "X-Trace-ID"
//   - Included in all log messages for correlation
//   - Available in error responses for debugging
//
// # Logging
//
// The Logger middleware provides structured request/response logging:
//
//	[a1b2c3d4] --> GET /api/users
//	[a1b2c3d4] <-- GET /api/users -> 200 (15ms) 1234 bytes
//
// For custom logging configuration:
//
//	config := middleware.LoggerConfig{
//	    SkipPaths: []string{"/health", "/metrics"},
//	    SlowRequestThreshold: 1 * time.Second,
//	}
//	router.Use(middleware.LoggerWithConfig(config))
//
// # Error Handling
//
// The package provides structured error handling with type-safe HTTP status mapping:
//
//	// In handlers
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    user, err := getUserByID(id)
//	    if err != nil {
//	        middleware.WriteError(w, r, middleware.ErrNotFound.WithError(err))
//	        return
//	    }
//	    // ... continue processing
//	}
//
// Or use the ErrorHandler wrapper for cleaner code:
//
//	handler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request) error {
//	    user, err := getUserByID(id)
//	    if err != nil {
//	        return middleware.ErrNotFound.WithError(err)
//	    }
//	    return writeJSON(w, user)
//	})
//
// Predefined errors:
//   - ErrUnauthorized (401)
//   - ErrForbidden (403)
//   - ErrNotFound (404)
//   - ErrBadRequest (400)
//   - ErrConflict (409)
//   - ErrInternal (500)
//
// Custom errors:
//
//	err := middleware.NewAppError(422, "Validation failed").
//	    WithDetails(map[string]string{"email": "invalid format"})
//
// # CORS Configuration
//
// Development (permissive):
//
//	config := middleware.DefaultCORSConfig()
//	router.Use(middleware.CORS(config))
//
// Production (strict):
//
//	config := middleware.ProductionCORSConfig([]string{
//	    "https://example.com",
//	    "https://app.example.com",
//	})
//	router.Use(middleware.CORS(config))
//
// Dynamic origin validation:
//
//	originChecker := func(origin string) bool {
//	    return isValidTenant(origin)
//	}
//	router.Use(middleware.CORSWithDynamicOrigin(originChecker, config))
//
// # Authentication
//
// Basic usage with JWT:
//
//	config := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	router.Use(middleware.Auth(config))
//
// Optional authentication (graceful degradation):
//
//	config := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	config.Enabled = false  // Don't reject unauthenticated requests
//	router.Use(middleware.Auth(config))
//
// Mixed public/private routes:
//
//	// Public routes
//	router.Handle("/api/public", publicHandler)
//
//	// Private routes
//	privateRouter := router.PathPrefix("/api/private").Subrouter()
//	privateRouter.Use(middleware.RequireAuth(config))
//
// Custom token validation:
//
//	validator := func(token string) (string, error) {
//	    // Call auth service, validate token, etc.
//	    return userID, nil
//	}
//	router.Use(middleware.AuthWithCustomValidator(validator, config))
//
// Accessing authenticated user:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    userID := middleware.GetUserID(r.Context())
//	    if userID == "" {
//	        // Not authenticated
//	    }
//	    // ... use userID
//	}
//
// # Panic Recovery
//
// The Recovery middleware catches panics and converts them to 500 errors:
//
//	router.Use(middleware.Recovery)
//
// Features:
//   - Logs panic with full stack trace
//   - Returns structured error response to client
//   - Prevents server crashes
//   - Includes trace ID for debugging
//
// # Performance Considerations
//
// All middleware is designed for minimal overhead:
//   - String operations pre-computed at startup (CORS headers, etc.)
//   - Map lookups for O(1) path/origin checking
//   - No reflection in hot paths
//   - Minimal allocations (reused buffers where possible)
//
// # Security Best Practices
//
// CORS:
//   - Never use AllowedOrigins: ["*"] with AllowCredentials: true
//   - Always specify exact origins in production
//   - Use HTTPS origins in production
//
// Authentication:
//   - Store JWT secret in environment variables, not code
//   - Use strong secrets (32+ random bytes)
//   - Set appropriate token expiration times
//   - Use HTTPS to protect tokens in transit
//
// Error Handling:
//   - Never expose internal errors to clients (5xx errors)
//   - Use trace IDs for debugging instead of detailed errors
//   - Log sensitive errors server-side only
//
// # Example: Complete Server Setup
//
//	func main() {
//	    router := mux.NewRouter()
//
//	    // Apply middleware in correct order
//	    router.Use(middleware.Recovery)
//	    router.Use(middleware.TraceID)
//	    router.Use(middleware.Logger)
//
//	    // CORS
//	    corsConfig := middleware.ProductionCORSConfig([]string{
//	        "https://example.com",
//	    })
//	    router.Use(middleware.CORS(corsConfig))
//
//	    // Optional auth (adds user to context if present)
//	    authConfig := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	    authConfig.Enabled = false
//	    router.Use(middleware.Auth(authConfig))
//
//	    // Public routes
//	    router.HandleFunc("/health", healthHandler).Methods("GET")
//	    router.HandleFunc("/api/public/info", publicInfoHandler).Methods("GET")
//
//	    // Private routes
//	    privateRouter := router.PathPrefix("/api/private").Subrouter()
//	    privateConfig := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	    privateRouter.Use(middleware.RequireAuth(privateConfig))
//	    privateRouter.HandleFunc("/profile", profileHandler).Methods("GET")
//
//	    log.Fatal(http.ListenAndServe(":8080", router))
//	}
//
// # Testing
//
// The middleware can be tested using standard Go testing tools:
//
//	func TestTraceID(t *testing.T) {
//	    handler := middleware.TraceID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        traceID := middleware.GetTraceID(r.Context())
//	        if traceID == "" {
//	            t.Error("expected trace ID in context")
//	        }
//	    }))
//
//	    req := httptest.NewRequest("GET", "/test", nil)
//	    rec := httptest.NewRecorder()
//	    handler.ServeHTTP(rec, req)
//
//	    if rec.Header().Get("X-Trace-ID") == "" {
//	        t.Error("expected X-Trace-ID header")
//	    }
//	}
package middleware
