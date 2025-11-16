package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/internal/supabase"
	"github.com/pradord/lumen_final/backend/pkg/logger"
)

// Import chi.Router for interface constraints
var _ chi.Router

// Server holds the server dependencies
type Server struct {
	Config   *config.Config
	Logger   *logger.Logger
	Supabase *supabase.Client
	Router   chi.Router
}

// NewServer creates a new server instance with clean dependency injection
func NewServer(cfg *config.Config, log *logger.Logger, sb *supabase.Client) *Server {
	return NewServerWithHandlers(cfg, log, sb, nil, nil, nil, nil, nil)
}

// NewServerWithHandlers creates a new server instance with nutrition domain handlers
func NewServerWithHandlers(
	cfg *config.Config,
	log *logger.Logger,
	sb *supabase.Client,
	mealsHandler interface{ RegisterMealsRoutes(chi.Router) },
	weightHandler interface{ RegisterRoutes(chi.Router) },
	templatesHandler interface{ RegisterRoutes(chi.Router) },
	analyticsHandler interface{ RegisterRoutes(chi.Router) },
	goalsHandler interface{ RegisterRoutes(chi.Router) },
) *Server {
	// Convert logger.Logger to *slog.Logger for handlers
	slogLogger := slog.Default()

	// Create router config
	routerConfig := &RouterConfig{
		Version:        "1.0.0",
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		RequestTimeout: int(cfg.Server.ReadTimeout.Seconds()),
	}

	// Create handler dependencies
	deps := &HandlerDependencies{
		Logger:           slogLogger,
		Supabase:         sb,
		Version:          "1.0.0",
		MealsHandler:     mealsHandler,
		WeightHandler:    weightHandler,
		TemplatesHandler: templatesHandler,
		AnalyticsHandler: analyticsHandler,
		GoalsHandler:     goalsHandler,
	}

	// Create router
	router := NewRouter(routerConfig, deps)

	return &Server{
		Config:   cfg,
		Logger:   log,
		Supabase: sb,
		Router:   router,
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}
