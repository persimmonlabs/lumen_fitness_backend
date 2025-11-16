package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthConfig holds configuration for the authentication middleware.
type AuthConfig struct {
	// JWTSecret is the secret key used to validate JWT tokens.
	// This should be stored securely and never committed to version control.
	JWTSecret string

	// Enabled determines whether authentication is enforced.
	// When false, the middleware adds user info to context if present
	// but doesn't reject unauthenticated requests.
	// Default: true
	Enabled bool

	// SkipPaths is a list of URL paths that don't require authentication.
	// Useful for public endpoints like health checks, documentation, etc.
	// Examples: ["/health", "/api/public/*"]
	SkipPaths []string

	// TokenLookup defines where to look for the JWT token.
	// Format: "<source>:<name>"
	// Supported sources: "header", "query", "cookie"
	// Examples:
	//   - "header:Authorization" (default)
	//   - "query:token"
	//   - "cookie:auth_token"
	TokenLookup string

	// AuthScheme is the authentication scheme to expect in Authorization header.
	// Default: "Bearer"
	AuthScheme string

	// ContextKey is the context key for storing authenticated user ID.
	// Default: Uses UserIDKey from context.go
	ContextKey contextKey

	// TokenContextKey is the context key for storing the token.
	// Default: Uses AuthTokenKey from context.go
	TokenContextKey contextKey

	// ErrorHandler is called when authentication fails.
	// If nil, uses default error handler that returns 401.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// SuccessHandler is called after successful authentication (optional).
	// Can be used for logging, metrics, etc.
	SuccessHandler func(w http.ResponseWriter, r *http.Request, userID string)
}

// DefaultAuthConfig returns authentication configuration with sensible defaults.
func DefaultAuthConfig(jwtSecret string) AuthConfig {
	return AuthConfig{
		JWTSecret:       jwtSecret,
		Enabled:         true,
		SkipPaths:       []string{"/health", "/metrics"},
		TokenLookup:     "header:Authorization",
		AuthScheme:      "Bearer",
		ContextKey:      UserIDKey,
		TokenContextKey: AuthTokenKey,
	}
}

// Auth creates an authentication middleware with the given configuration.
//
// The middleware:
//   - Extracts JWT token from request (header, query, or cookie)
//   - Validates token signature and expiration
//   - Extracts user ID from token claims
//   - Adds user ID and token to request context
//   - Supports graceful degradation when auth is disabled
//   - Allows skipping authentication for specific paths
//
// Token format:
//   - Header: "Authorization: Bearer <token>"
//   - Query: "?token=<token>"
//   - Cookie: "auth_token=<token>"
//
// JWT claims expected:
//   - "sub" (subject): user ID
//   - "exp" (expiration): Unix timestamp
//
// Usage:
//
//	// Strict authentication (default)
//	config := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	router.Use(middleware.Auth(config))
//
//	// Optional authentication (graceful degradation)
//	config := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	config.Enabled = false  // Don't reject unauthenticated requests
//	router.Use(middleware.Auth(config))
//
//	// Skip authentication for specific paths
//	config := middleware.DefaultAuthConfig(os.Getenv("JWT_SECRET"))
//	config.SkipPaths = []string{"/health", "/api/public/*"}
//	router.Use(middleware.Auth(config))
func Auth(config AuthConfig) func(http.Handler) http.Handler {
	// Apply defaults
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.AuthScheme == "" {
		config.AuthScheme = "Bearer"
	}
	if config.ContextKey == "" {
		config.ContextKey = UserIDKey
	}
	if config.TokenContextKey == "" {
		config.TokenContextKey = AuthTokenKey
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultAuthErrorHandler
	}

	// Parse token lookup configuration
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("invalid TokenLookup format: %s", config.TokenLookup))
	}
	tokenSource := parts[0]
	tokenName := parts[1]

	// Build skip paths map for O(1) lookup
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip authentication
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from request
			token, err := extractToken(r, tokenSource, tokenName, config.AuthScheme)
			if err != nil {
				if config.Enabled {
					// Authentication required but token missing/invalid
					config.ErrorHandler(w, r, ErrUnauthorized.WithError(err))
					return
				}
				// Authentication optional - continue without user ID
				next.ServeHTTP(w, r)
				return
			}

			// Validate and parse JWT token
			userID, err := validateJWT(token, config.JWTSecret)
			if err != nil {
				if config.Enabled {
					// Authentication required but token invalid
					config.ErrorHandler(w, r, ErrUnauthorized.WithError(err))
					return
				}
				// Authentication optional - continue without user ID
				log.Printf("[%s] Invalid token (auth disabled): %v", GetTraceID(r.Context()), err)
				next.ServeHTTP(w, r)
				return
			}

			// Parse user ID string to UUID
			userUUID, err := uuid.Parse(userID)
			if err != nil {
				if config.Enabled {
					config.ErrorHandler(w, r, ErrUnauthorized.WithError(fmt.Errorf("invalid user ID format: %w", err)))
					return
				}
				log.Printf("[%s] Invalid user ID format (auth disabled): %v", GetTraceID(r.Context()), err)
				next.ServeHTTP(w, r)
				return
			}

			// Add user ID (as UUID) and token to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, config.ContextKey, userUUID)
			ctx = context.WithValue(ctx, config.TokenContextKey, token)

			// Call success handler if configured
			if config.SuccessHandler != nil {
				config.SuccessHandler(w, r, userID)
			}

			// Continue with authenticated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts the JWT token from the request based on configuration.
func extractToken(r *http.Request, source, name, authScheme string) (string, error) {
	var token string

	switch source {
	case "header":
		// Extract from header (e.g., "Authorization: Bearer <token>")
		authHeader := r.Header.Get(name)
		if authHeader == "" {
			return "", fmt.Errorf("missing %s header", name)
		}

		// Parse auth scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid authorization header format")
		}

		if parts[0] != authScheme {
			return "", fmt.Errorf("unsupported auth scheme: %s (expected %s)", parts[0], authScheme)
		}

		token = parts[1]

	case "query":
		// Extract from query parameter (e.g., "?token=<token>")
		token = r.URL.Query().Get(name)
		if token == "" {
			return "", fmt.Errorf("missing %s query parameter", name)
		}

	case "cookie":
		// Extract from cookie
		cookie, err := r.Cookie(name)
		if err != nil {
			return "", fmt.Errorf("missing %s cookie", name)
		}
		token = cookie.Value

	default:
		return "", fmt.Errorf("unsupported token source: %s", source)
	}

	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	return token, nil
}

// validateJWT validates a JWT token and returns the user ID.
func validateJWT(tokenString, secret string) (string, error) {
	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Extract user ID from "sub" claim
	sub, ok := claims["sub"]
	if !ok {
		return "", fmt.Errorf("missing sub claim")
	}

	userID, ok := sub.(string)
	if !ok {
		return "", fmt.Errorf("invalid sub claim type")
	}

	if userID == "" {
		return "", fmt.Errorf("empty user ID")
	}

	return userID, nil
}

// defaultAuthErrorHandler is the default error handler for authentication failures.
func defaultAuthErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	WriteError(w, r, err)
}

// RequireAuth creates a middleware that requires authentication for specific routes.
// This is useful when you have a mix of public and private endpoints.
//
// Usage:
//
//	// Apply auth to specific routes
//	router.Handle("/api/users", middleware.RequireAuth(config, usersHandler))
//	router.Handle("/api/public", publicHandler)  // No auth required
func RequireAuth(config AuthConfig) func(http.Handler) http.Handler {
	// Force authentication to be enabled
	config.Enabled = true
	config.SkipPaths = nil

	return Auth(config)
}

// OptionalAuth creates a middleware that adds user info to context if present
// but doesn't reject unauthenticated requests.
//
// Usage:
//
//	// Try to authenticate, but allow anonymous access
//	router.Use(middleware.OptionalAuth(config))
//
//	// In handlers, check if user is authenticated:
//	userID := middleware.GetUserID(r.Context())
//	if userID != "" {
//	    // Authenticated user
//	} else {
//	    // Anonymous user
//	}
func OptionalAuth(config AuthConfig) func(http.Handler) http.Handler {
	// Disable strict authentication
	config.Enabled = false

	return Auth(config)
}

// AuthWithCustomValidator creates an auth middleware with custom token validation.
// This is useful when using a different token format or validation service.
//
// Usage:
//
//	validator := func(token string) (string, error) {
//	    // Custom validation logic
//	    return userID, nil
//	}
//	router.Use(middleware.AuthWithCustomValidator(validator, config))
func AuthWithCustomValidator(validator func(string) (string, error), config AuthConfig) func(http.Handler) http.Handler {
	// Apply defaults
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.AuthScheme == "" {
		config.AuthScheme = "Bearer"
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultAuthErrorHandler
	}

	// Parse token lookup
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("invalid TokenLookup format: %s", config.TokenLookup))
	}
	tokenSource := parts[0]
	tokenName := parts[1]

	// Build skip paths map
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip paths
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token
			token, err := extractToken(r, tokenSource, tokenName, config.AuthScheme)
			if err != nil {
				if config.Enabled {
					config.ErrorHandler(w, r, ErrUnauthorized.WithError(err))
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Use custom validator
			userID, err := validator(token)
			if err != nil {
				if config.Enabled {
					config.ErrorHandler(w, r, ErrUnauthorized.WithError(err))
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Add to context
			ctx := SetUserID(r.Context(), userID)
			ctx = SetAuthToken(ctx, token)

			// Call success handler
			if config.SuccessHandler != nil {
				config.SuccessHandler(w, r, userID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
