package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to access the API.
	// Use ["*"] to allow all origins (not recommended for production).
	// Examples: ["https://example.com", "https://app.example.com"]
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods that are allowed.
	// Default: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
	AllowedMethods []string

	// AllowedHeaders is a list of headers that are allowed in requests.
	// Default: ["Authorization", "Content-Type", "Accept"]
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that the client can access.
	// Default: ["Content-Length", "Content-Type"]
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials (cookies, authorization headers)
	// are allowed. Default: true
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Default: 86400 (24 hours)
	MaxAge int

	// OptionsPassthrough determines whether OPTIONS requests should be passed
	// through to handlers or handled immediately by the middleware.
	// Default: false (middleware handles OPTIONS)
	OptionsPassthrough bool
}

// DefaultCORSConfig returns a CORS configuration with sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Trace-ID"},
		ExposedHeaders:     []string{"Content-Length", "Content-Type", "X-Trace-ID"},
		AllowCredentials:   true,
		MaxAge:             86400, // 24 hours
		OptionsPassthrough: false,
	}
}

// ProductionCORSConfig returns a CORS configuration suitable for production.
// It requires explicit origin configuration for security.
func ProductionCORSConfig(allowedOrigins []string) CORSConfig {
	return CORSConfig{
		AllowedOrigins:     allowedOrigins, // Must be explicitly set
		AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Trace-ID"},
		ExposedHeaders:     []string{"Content-Length", "Content-Type", "X-Trace-ID"},
		AllowCredentials:   true,
		MaxAge:             86400,
		OptionsPassthrough: false,
	}
}

// CORS creates a CORS middleware with the given configuration.
//
// The middleware:
//   - Handles preflight OPTIONS requests
//   - Sets appropriate CORS headers on all responses
//   - Validates origin against allowed origins
//   - Supports credentials (cookies, authorization)
//
// Security considerations:
//   - Do NOT use AllowedOrigins: ["*"] with AllowCredentials: true
//   - Always specify exact origins in production
//   - Use HTTPS origins in production
//   - Limit allowed methods to what's actually needed
//
// Usage:
//
//	// Development (permissive)
//	config := middleware.DefaultCORSConfig()
//	router.Use(middleware.CORS(config))
//
//	// Production (strict)
//	config := middleware.ProductionCORSConfig([]string{
//	    "https://example.com",
//	    "https://app.example.com",
//	})
//	router.Use(middleware.CORS(config))
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	// Apply defaults if not set
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Authorization", "Content-Type", "Accept"}
	}
	if len(config.ExposedHeaders) == 0 {
		config.ExposedHeaders = []string{"Content-Length", "Content-Type"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400 // 24 hours
	}

	// Pre-join headers for efficiency (done once at startup)
	allowedMethodsStr := strings.Join(config.AllowedMethods, ", ")
	allowedHeadersStr := strings.Join(config.AllowedHeaders, ", ")
	exposedHeadersStr := strings.Join(config.ExposedHeaders, ", ")
	maxAgeStr := strconv.Itoa(config.MaxAge)

	// Build origin map for O(1) lookup
	allowAllOrigins := false
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		allowedOriginsMap[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get request origin
			origin := r.Header.Get("Origin")

			// Determine if origin is allowed
			var allowedOrigin string
			if allowAllOrigins {
				// Allow all origins, but echo back the specific origin
				// when credentials are allowed (required by CORS spec)
				if config.AllowCredentials && origin != "" {
					allowedOrigin = origin
				} else {
					allowedOrigin = "*"
				}
			} else if origin != "" && allowedOriginsMap[origin] {
				// Origin is in allowed list
				allowedOrigin = origin
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersStr)
				w.Header().Set("Access-Control-Expose-Headers", exposedHeadersStr)
				w.Header().Set("Access-Control-Max-Age", maxAgeStr)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				if config.OptionsPassthrough {
					// Pass through to handler
					next.ServeHTTP(w, r)
				} else {
					// Handle immediately with 204 No Content
					w.WriteHeader(http.StatusNoContent)
				}
				return
			}

			// Continue with next handler
			next.ServeHTTP(w, r)
		})
	}
}

// CORSWithDefaults creates a CORS middleware with default configuration.
// Suitable for development, NOT for production.
//
// Usage:
//
//	router.Use(middleware.CORSWithDefaults())
func CORSWithDefaults() func(http.Handler) http.Handler {
	return CORS(DefaultCORSConfig())
}

// isOriginAllowed checks if the given origin is in the allowed list.
// Supports wildcard matching for subdomains.
//
// Examples:
//   - "https://example.com" matches "https://example.com"
//   - "https://*.example.com" matches "https://app.example.com"
//   - "*" matches any origin
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomain matching
		if strings.HasPrefix(allowed, "https://*.") {
			domain := strings.TrimPrefix(allowed, "https://*.")
			if strings.HasSuffix(origin, "."+domain) {
				return true
			}
		}
		if strings.HasPrefix(allowed, "http://*.") {
			domain := strings.TrimPrefix(allowed, "http://*.")
			if strings.HasSuffix(origin, "."+domain) {
				return true
			}
		}
	}
	return false
}

// CORSWithDynamicOrigin creates a CORS middleware that dynamically determines
// allowed origins using a custom function. Useful for multi-tenant applications.
//
// Usage:
//
//	originChecker := func(origin string) bool {
//	    // Check database, configuration, etc.
//	    return isValidTenant(origin)
//	}
//	router.Use(middleware.CORSWithDynamicOrigin(originChecker, config))
func CORSWithDynamicOrigin(originChecker func(string) bool, config CORSConfig) func(http.Handler) http.Handler {
	// Apply defaults
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Authorization", "Content-Type", "Accept"}
	}
	if len(config.ExposedHeaders) == 0 {
		config.ExposedHeaders = []string{"Content-Length", "Content-Type"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400
	}

	// Pre-join headers
	allowedMethodsStr := strings.Join(config.AllowedMethods, ", ")
	allowedHeadersStr := strings.Join(config.AllowedHeaders, ", ")
	exposedHeadersStr := strings.Join(config.ExposedHeaders, ", ")
	maxAgeStr := strconv.Itoa(config.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed using custom function
			if origin != "" && originChecker(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersStr)
				w.Header().Set("Access-Control-Expose-Headers", exposedHeadersStr)
				w.Header().Set("Access-Control-Max-Age", maxAgeStr)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				if !config.OptionsPassthrough {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
