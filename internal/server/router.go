// Package server provides HTTP routing and server configuration.
//
// This package contains the main router setup, route definitions, middleware
// configuration, and handler dependency injection. It follows a clean architecture
// pattern with clear separation between public and protected routes.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/pradord/lumen_final/backend/internal/handlers"
	serverhandlers "github.com/pradord/lumen_final/backend/internal/server/handlers"
	custommw "github.com/pradord/lumen_final/backend/internal/server/middleware"
	"github.com/pradord/lumen_final/backend/internal/supabase"
)

// RouterConfig contains configuration for the API router.
type RouterConfig struct {
	Version        string
	AllowedOrigins []string
	RequestTimeout int // in seconds
}

// HandlerDependencies contains all dependencies needed by API handlers.
// This struct is used for dependency injection into handlers.
type HandlerDependencies struct {
	Logger   *slog.Logger
	Supabase *supabase.Client
	Version  string

	// Nutrition domain handlers
	MealsHandler     interface{ RegisterMealsRoutes(chi.Router) }
	WeightHandler    interface{ RegisterRoutes(chi.Router) }
	TemplatesHandler interface{ RegisterRoutes(chi.Router) }
	AnalyticsHandler interface{ RegisterRoutes(chi.Router) }
	GoalsHandler     interface{ RegisterRoutes(chi.Router) }
	UserHandler      interface{ RegisterRoutes(chi.Router) }

	// New service handlers (optional - may be nil if services not initialized)
	MediaHandler        *handlers.MediaHandler
	VoiceHandler        *handlers.VoiceHandler
	TrajectoryHandler   *handlers.TrajectoryHandler
	SuggestionsHandler  *handlers.SuggestionsHandler
	CommonFoodsHandler  *handlers.CommonFoodsHandler
}

// Router encapsulates the HTTP router and its configuration.
type Router struct {
	chi.Router
	config *RouterConfig
	deps   *HandlerDependencies
}

// NewRouter creates and configures a new API router with all routes and middleware.
//
// Parameters:
//   - config: Router configuration (CORS, versioning, etc.)
//   - deps: Handler dependencies (logger, Supabase, etc.)
//
// Returns:
//   - *Router: Configured router ready to serve requests
func NewRouter(config *RouterConfig, deps *HandlerDependencies) *Router {
	r := &Router{
		Router: chi.NewRouter(),
		config: config,
		deps:   deps,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// setupMiddleware configures global middleware for all routes.
// Middleware is applied in order:
// 1. Request ID - assigns unique ID to each request
// 2. Real IP - extracts real client IP from proxy headers
// 3. Logger - logs all requests
// 4. Recoverer - recovers from panics
// 5. CORS - handles cross-origin requests
// 6. User Context - injects user information into context
// 7. Rate Limit - prevents API abuse
func (r *Router) setupMiddleware() {
	// Request ID middleware - adds unique ID to each request
	r.Use(middleware.RequestID)

	// Real IP middleware - extracts real client IP from headers
	r.Use(middleware.RealIP)

	// Custom structured logger middleware
	r.Use(r.loggerMiddleware)

	// Recoverer middleware - recovers from panics and returns 500
	r.Use(middleware.Recoverer)

	// CORS middleware - configure allowed origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   r.getAllowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Total-Count", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// User context injection middleware
	// TODO: Re-enable when authentication is implemented
	// r.Use(custommw.InjectUserContext(r.deps.Logger))

	// Rate limiting middleware (100 requests per second, burst of 200)
	// TODO: Re-enable when rate limiting is fully implemented
	// r.Use(custommw.RateLimit(100, 200, r.deps.Logger))

	// Compress responses if client supports it
	r.Use(middleware.Compress(5))

	// Strip trailing slashes from URLs
	r.Use(middleware.StripSlashes)
}

// setupRoutes configures all API routes organized into groups.
func (r *Router) setupRoutes() {
	healthHandler := serverhandlers.NewHealthHandler(r.deps.Logger, r.deps.Supabase, r.deps.Version)

	// Public routes (no authentication required)
	r.Group(func(r chi.Router) {
		r.Get("/health", healthHandler.Health)
		r.Get("/ready", healthHandler.Ready)
		r.Get("/ping", healthHandler.Ping)
		r.Get("/info", healthHandler.Info)
	})

	// API v1 routes
	r.Route("/api/v1", func(apiRouter chi.Router) {
		// Public routes (no authentication required)
		apiRouter.Group(func(apiRouter chi.Router) {
			apiRouter.Get("/", r.apiInfo)
			apiRouter.Get("/routes", r.listRoutes)
		})

		// Example feature: Items (will be removed later)
		itemHandler := serverhandlers.NewItemHandler(r.deps.Logger, r.deps.Supabase)
		apiRouter.Route("/items", func(apiRouter chi.Router) {
			apiRouter.Post("/", itemHandler.Create)
			apiRouter.Get("/", itemHandler.List)
		})

		// Protected routes - require authentication
		apiRouter.Group(func(protectedRouter chi.Router) {
			// Add authentication middleware for protected routes
			// Uses Supabase JWT validation
			authConfig := custommw.DefaultAuthConfig(r.deps.Supabase.GetJWTSecret())
			authConfig.Enabled = true // Enable strict authentication
			authConfig.SkipPaths = []string{} // All routes in this group require auth
			protectedRouter.Use(custommw.Auth(authConfig))

			// Nutrition domain routes - all require authentication
			if r.deps.MealsHandler != nil {
				r.deps.MealsHandler.RegisterMealsRoutes(protectedRouter)
			}
			if r.deps.WeightHandler != nil {
				r.deps.WeightHandler.RegisterRoutes(protectedRouter)
			}
			if r.deps.TemplatesHandler != nil {
				r.deps.TemplatesHandler.RegisterRoutes(protectedRouter)
			}
			if r.deps.AnalyticsHandler != nil {
				r.deps.AnalyticsHandler.RegisterRoutes(protectedRouter)
			}
			if r.deps.GoalsHandler != nil {
				r.deps.GoalsHandler.RegisterRoutes(protectedRouter)
			}
			if r.deps.UserHandler != nil {
				r.deps.UserHandler.RegisterRoutes(protectedRouter)
			}

			// New service routes
			if r.deps.MediaHandler != nil {
				protectedRouter.Post("/media/upload", r.deps.MediaHandler.Upload)
				protectedRouter.Delete("/media", r.deps.MediaHandler.Delete)
				protectedRouter.Get("/media/url", r.deps.MediaHandler.GetURL)
			}

			if r.deps.VoiceHandler != nil {
				protectedRouter.Post("/voice/transcribe", r.deps.VoiceHandler.Transcribe)
				protectedRouter.Get("/voice/formats", r.deps.VoiceHandler.GetSupportedFormats)
			}

			if r.deps.TrajectoryHandler != nil {
				protectedRouter.Post("/trajectory/predict", r.deps.TrajectoryHandler.Calculate)
			}

			if r.deps.SuggestionsHandler != nil {
				protectedRouter.Post("/suggestions/generate", r.deps.SuggestionsHandler.Generate)
				protectedRouter.Get("/suggestions/quick", r.deps.SuggestionsHandler.QuickSuggestions)
			}

			// Common foods search endpoint
			if r.deps.CommonFoodsHandler != nil {
				protectedRouter.Get("/common-foods/search", r.deps.CommonFoodsHandler.Search)
			}
		})
	})

	r.NotFound(r.notFoundHandler)
	r.MethodNotAllowed(r.methodNotAllowedHandler)
}

// loggerMiddleware is a custom structured logging middleware using slog.
func (r *Router) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)

		start := time.Now()
		defer func() {
			r.deps.Logger.Info("request completed",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.String("remote_addr", req.RemoteAddr),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", middleware.GetReqID(req.Context())),
			)
		}()

		next.ServeHTTP(ww, req)
	})
}

// getAllowedOrigins returns the list of allowed CORS origins.
func (r *Router) getAllowedOrigins() []string {
	if len(r.config.AllowedOrigins) > 0 {
		return r.config.AllowedOrigins
	}

	return []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
	}
}

// apiInfo returns basic API information for the root endpoint.
func (r *Router) apiInfo(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"name":    "Lumen Fitness API Server",
		"version": r.config.Version,
		"endpoints": map[string]string{
			"health":       "/health",
			"ready":        "/ready",
			"ping":         "/ping",
			"info":         "/info",
			"api_v1":       "/api/v1",
			"route_list":   "/api/v1/routes",
			"meals":        "/api/v1/meals",
			"weight":       "/api/v1/weight",
			"templates":    "/api/v1/templates",
			"analytics":    "/api/v1/analytics",
			"goals":        "/api/v1/goals",
			"user":         "/api/v1/user",
			"media":        "/api/v1/media",
			"voice":        "/api/v1/voice",
			"trajectory":   "/api/v1/trajectory",
			"suggestions":  "/api/v1/suggestions",
			"common-foods": "/api/v1/common-foods",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.deps.Logger.Error("failed to encode API info response",
			slog.String("error", err.Error()),
		)
	}
}

// listRoutes returns a detailed list of all available API endpoints.
func (r *Router) listRoutes(w http.ResponseWriter, req *http.Request) {
	routes := map[string]interface{}{
		"version": r.config.Version,
		"routes": map[string]interface{}{
			"meals": []map[string]string{
				{"method": "POST", "path": "/api/v1/meals/parse", "description": "Parse meal from description or photo"},
				{"method": "POST", "path": "/api/v1/meals/confirm", "description": "Confirm and save parsed meal"},
				{"method": "GET", "path": "/api/v1/meals", "description": "List meals with optional filters"},
				{"method": "GET", "path": "/api/v1/meals/{id}", "description": "Get specific meal by ID"},
				{"method": "PUT", "path": "/api/v1/meals/{id}", "description": "Update meal"},
				{"method": "DELETE", "path": "/api/v1/meals/{id}", "description": "Delete meal"},
				{"method": "POST", "path": "/api/v1/meals/{id}/copy", "description": "Copy meal to another date/time"},
			},
			"weight": []map[string]string{
				{"method": "POST", "path": "/api/v1/weight", "description": "Record weight entry"},
				{"method": "GET", "path": "/api/v1/weight", "description": "List weight entries"},
				{"method": "GET", "path": "/api/v1/weight/{id}", "description": "Get specific weight entry"},
				{"method": "GET", "path": "/api/v1/weight/latest", "description": "Get latest weight entry"},
				{"method": "GET", "path": "/api/v1/weight/trend", "description": "Get weight trend statistics"},
				{"method": "PUT", "path": "/api/v1/weight/{id}", "description": "Update weight entry"},
				{"method": "DELETE", "path": "/api/v1/weight/{id}", "description": "Delete weight entry"},
			},
			"templates": []map[string]string{
				{"method": "POST", "path": "/api/v1/templates", "description": "Create meal template"},
				{"method": "POST", "path": "/api/v1/templates/from-meal", "description": "Create template from existing meal"},
				{"method": "POST", "path": "/api/v1/templates/{id}/use", "description": "Use template to create meal"},
				{"method": "GET", "path": "/api/v1/templates", "description": "List templates"},
				{"method": "GET", "path": "/api/v1/templates/{id}", "description": "Get specific template"},
				{"method": "PUT", "path": "/api/v1/templates/{id}", "description": "Update template"},
				{"method": "DELETE", "path": "/api/v1/templates/{id}", "description": "Delete template"},
			},
			"analytics": []map[string]string{
				{"method": "GET", "path": "/api/v1/analytics/daily", "description": "Get daily nutrition analytics"},
				{"method": "GET", "path": "/api/v1/analytics/weekly", "description": "Get weekly nutrition trends"},
				{"method": "GET", "path": "/api/v1/analytics/trends", "description": "Get custom date range trends"},
				{"method": "GET", "path": "/api/v1/analytics/distribution", "description": "Get macro distribution"},
				{"method": "GET", "path": "/api/v1/analytics/progress", "description": "Get goal progress"},
			},
			"goals": []map[string]string{
				{"method": "GET", "path": "/api/v1/goals", "description": "Get current goals"},
				{"method": "PUT", "path": "/api/v1/goals", "description": "Update goals"},
				{"method": "POST", "path": "/api/v1/goals/calculate", "description": "Calculate TDEE"},
				{"method": "GET", "path": "/api/v1/goals/daily", "description": "Get daily goals"},
				{"method": "PUT", "path": "/api/v1/goals/daily/{day}", "description": "Set day-specific goal"},
				{"method": "DELETE", "path": "/api/v1/goals/daily/{day}", "description": "Delete day-specific goal"},
			},
			"user": []map[string]string{
				{"method": "GET", "path": "/api/v1/user/profile", "description": "Get current user profile"},
				{"method": "PUT", "path": "/api/v1/user/profile", "description": "Update user profile"},
				{"method": "GET", "path": "/api/v1/user/settings", "description": "Get user settings"},
				{"method": "PUT", "path": "/api/v1/user/settings", "description": "Update user settings"},
			},
			"media": []map[string]string{
				{"method": "POST", "path": "/api/v1/media/upload", "description": "Upload media file"},
				{"method": "DELETE", "path": "/api/v1/media", "description": "Delete media file"},
				{"method": "GET", "path": "/api/v1/media/url", "description": "Get media file URL"},
			},
			"voice": []map[string]string{
				{"method": "POST", "path": "/api/v1/voice/transcribe", "description": "Transcribe voice to text"},
				{"method": "GET", "path": "/api/v1/voice/formats", "description": "Get supported audio formats"},
			},
			"trajectory": []map[string]string{
				{"method": "POST", "path": "/api/v1/trajectory/predict", "description": "Calculate weight trajectory prediction"},
			},
			"suggestions": []map[string]string{
				{"method": "POST", "path": "/api/v1/suggestions/generate", "description": "Generate meal suggestions"},
				{"method": "GET", "path": "/api/v1/suggestions/quick", "description": "Get quick meal suggestions"},
			},
			"common-foods": []map[string]string{
				{"method": "GET", "path": "/api/v1/common-foods/search", "description": "Search common foods database"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(routes); err != nil {
		r.deps.Logger.Error("failed to encode routes list",
			slog.String("error", err.Error()),
		)
	}
}

// notFoundHandler handles 404 Not Found responses.
func (r *Router) notFoundHandler(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "NOT_FOUND",
			"message": "The requested resource was not found",
			"path":    req.URL.Path,
		},
		"meta": map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.deps.Logger.Error("failed to encode 404 response",
			slog.String("error", err.Error()),
		)
	}
}

// methodNotAllowedHandler handles 405 Method Not Allowed responses.
func (r *Router) methodNotAllowedHandler(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "METHOD_NOT_ALLOWED",
			"message": "The requested HTTP method is not allowed for this resource",
			"method":  req.Method,
			"path":    req.URL.Path,
		},
		"meta": map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		r.deps.Logger.Error("failed to encode 405 response",
			slog.String("error", err.Error()),
		)
	}
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(w, req)
}
