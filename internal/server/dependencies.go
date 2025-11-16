// Package server provides dependency injection for nutrition domain.
//
// This package contains the NutritionDependencies struct which holds all
// dependencies needed by the nutrition tracking system including services,
// repositories, and handlers.
package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/analytics"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/goals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/templates"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/weight"
	"github.com/pradord/lumen_final/backend/internal/handlers"
	"github.com/pradord/lumen_final/backend/internal/services/ai"
	"github.com/pradord/lumen_final/backend/internal/services/cache"
	"github.com/pradord/lumen_final/backend/internal/services/media"
	"github.com/pradord/lumen_final/backend/internal/services/suggestions"
	"github.com/pradord/lumen_final/backend/internal/services/trajectory"
	"github.com/pradord/lumen_final/backend/internal/services/voice"
)

// NutritionDependencies contains all dependencies for the nutrition domain.
// This struct is used for dependency injection into the server and handlers.
type NutritionDependencies struct {
	// Core services
	AICoordinator *ai.Coordinator
	Cache         cache.Cache
	Logger        *slog.Logger

	// Repositories (interfaces from each domain)
	MealsRepo     meals.Repository
	WeightRepo    weight.Repository
	TemplatesRepo templates.Repository
	AnalyticsRepo analytics.Repository
	GoalsRepo     goals.Repository

	// Services (business logic)
	MealsService     meals.Service
	WeightService    weight.Service
	TemplatesService templates.Service
	AnalyticsService analytics.Service
	GoalsService     goals.Service

	// Handlers (HTTP layer)
	MealsHandler     *meals.Handler
	WeightHandler    *weight.Handler
	TemplatesHandler *templates.Handler
	AnalyticsHandler *analytics.Handler
	GoalsHandler     *goals.Handler

	// New service handlers
	MediaHandler       *handlers.MediaHandler
	VoiceHandler       *handlers.VoiceHandler
	TrajectoryHandler  *handlers.TrajectoryHandler
	SuggestionsHandler *handlers.SuggestionsHandler

	// New services
	MediaService       *media.Service
	VoiceService       *voice.Service
	TrajectoryService  *trajectory.Service
	SuggestionsService *suggestions.Service

	// Database connection pool
	DB *sqlx.DB
}

// NewNutritionDependencies initializes all nutrition domain dependencies.
//
// This function creates and wires up all repositories, services, and handlers
// needed for the nutrition tracking system. It returns an error if any
// initialization fails.
//
// Parameters:
//   - cfg: Application configuration
//   - db: Database connection pool (can be nil for stub mode)
//   - log: Structured logger
//
// Returns:
//   - *NutritionDependencies: Fully initialized dependencies
//   - error: Any initialization error
func NewNutritionDependencies(cfg *config.Config, db *sqlx.DB, log *slog.Logger) (*NutritionDependencies, error) {
	deps := &NutritionDependencies{
		DB:     db,
		Logger: log,
	}

	// Initialize core services
	if err := deps.initCoreServices(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize core services: %w", err)
	}

	// Initialize repositories
	if err := deps.initRepositories(db); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Initialize domain services
	if err := deps.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize HTTP handlers
	if err := deps.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	return deps, nil
}

// initCoreServices initializes shared services like AI coordinator and cache.
func (d *NutritionDependencies) initCoreServices(cfg *config.Config) error {
	// Initialize cache (in-memory for now)
	d.Cache = cache.NewMemoryCache(nil)

	// Initialize AI coordinator if enabled
	if cfg.Features.EnableAIAnalysis {
		coordinatorConfig := ai.CoordinatorConfig{
			GroqAPIKey:       cfg.AI.GroqAPIKey,
			OpenRouterAPIKey: cfg.AI.OpenRouterAPIKey,
			OpenRouterModel:  cfg.AI.OpenRouterNutritionPreset,
			AppName:          "Lumen Nutrition Tracker",
			SiteURL:          "https://lumen.app",
			CostLimit:        cfg.RateLimit.GlobalCostPerMonth,
			PreferGroq:       true,
		}
		coordinator, err := ai.NewCoordinator(coordinatorConfig)
		if err != nil {
			// Log error but don't fail - AI is optional
			d.Logger.Warn("failed to initialize AI coordinator, running without AI",
				slog.String("error", err.Error()),
			)
		} else {
			d.AICoordinator = coordinator
		}
	}

	return nil
}

// initRepositories initializes all domain repositories.
// If db is nil, repositories will be stub implementations.
func (d *NutritionDependencies) initRepositories(db *sqlx.DB) error {
	if db != nil {
		// Initialize PostgreSQL repositories
		d.MealsRepo = meals.NewRepository(db)
		d.WeightRepo = weight.NewPostgresRepository(db)
		// TODO: Implement PostgreSQL repositories for these domains
		// d.TemplatesRepo = templates.NewPostgresRepository(db)
		// d.AnalyticsRepo = analytics.NewPostgresRepository(db)
		// d.GoalsRepo = goals.NewPostgresRepository(db)

		d.Logger.Info("initialized repositories",
			slog.Bool("meals", d.MealsRepo != nil),
			slog.Bool("weight", d.WeightRepo != nil),
			slog.Bool("templates", d.TemplatesRepo != nil),
			slog.Bool("analytics", d.AnalyticsRepo != nil),
			slog.Bool("goals", d.GoalsRepo != nil),
		)
	} else {
		d.Logger.Warn("no database connection - running in stub mode")
	}

	return nil
}

// initServices initializes all domain services with their dependencies.
func (d *NutritionDependencies) initServices() error {
	// Initialize domain services if repositories are available
	if d.MealsRepo != nil {
		// TODO: Create adapters to bridge AI Coordinator to meals-specific interfaces
		// For now, pass nil for AI services - they are optional
		d.MealsService = meals.NewService(
			d.MealsRepo,
			nil, // AICoordinator - TODO: implement adapter
			nil, // AIEstimator - TODO: implement adapter
			nil, // AITranscriber - TODO: implement adapter
			nil, // AINormalizer - TODO: implement adapter
			nil, // Cache - TODO: implement adapter
			nil, // CostTracker - TODO: implement
			nil, // PhotoStorage - TODO: implement
			d.Logger,
		)
		d.Logger.Info("initialized meals service (without AI features)")
	}

	if d.WeightRepo != nil {
		d.WeightService = weight.NewService(d.WeightRepo)
		d.Logger.Info("initialized weight service")
	}

	// TODO: Initialize these services when repositories are implemented
	// if d.TemplatesRepo != nil {
	// 	d.TemplatesService = templates.NewService(d.TemplatesRepo, nil)
	// }
	//
	// if d.AnalyticsRepo != nil {
	// 	d.AnalyticsService = analytics.NewService(d.AnalyticsRepo)
	// }
	//
	// if d.GoalsRepo != nil {
	// 	d.GoalsService = goals.NewService(d.GoalsRepo)
	// }

	// Initialize new services (these don't require database)
	if err := d.initNewServices(); err != nil {
		return fmt.Errorf("failed to initialize new services: %w", err)
	}

	return nil
}

// initNewServices initializes media, voice, trajectory, and suggestions services.
func (d *NutritionDependencies) initNewServices() error {
	// Initialize media service
	// TODO: Initialize when Supabase storage client is available
	// For now, pass nil which will be handled gracefully by the service
	d.MediaService = media.NewService(nil, d.Logger, media.DefaultConfig())
	d.Logger.Info("initialized media service")

	// Initialize voice service (requires Groq API key)
	// Voice service is optional, only initialize if API key is available
	// Note: Groq API key should be available from AICoordinator config
	if d.AICoordinator != nil {
		// Voice service is integrated with AI coordinator
		// TODO: Extract Groq API key from config and initialize voice service
		d.Logger.Info("voice service available through AI coordinator")
	}

	// Initialize trajectory service (no external dependencies)
	d.TrajectoryService = trajectory.NewService(d.Logger, trajectory.DefaultConfig())
	d.Logger.Info("initialized trajectory service")

	// Initialize suggestions service (requires AI coordinator)
	if d.AICoordinator != nil {
		d.SuggestionsService = suggestions.NewService(d.AICoordinator, d.Logger, suggestions.DefaultConfig())
		d.Logger.Info("initialized suggestions service")
	}

	return nil
}

// initHandlers initializes all HTTP handlers.
func (d *NutritionDependencies) initHandlers() error {
	// Initialize handlers (only if services are available)
	if d.MealsService != nil {
		d.MealsHandler = meals.NewHandler(d.MealsService, d.Logger)
	}

	if d.WeightService != nil {
		d.WeightHandler = weight.NewHandler(d.WeightService)
	}

	if d.TemplatesService != nil {
		d.TemplatesHandler = templates.NewHandler(d.TemplatesService)
	}

	if d.AnalyticsService != nil {
		d.AnalyticsHandler = analytics.NewHandler(d.AnalyticsService)
	}

	if d.GoalsService != nil {
		d.GoalsHandler = goals.NewHandler(d.GoalsService)
	}

	// Initialize new service handlers
	if d.MediaService != nil {
		d.MediaHandler = handlers.NewMediaHandler(d.MediaService, d.Logger)
	}

	if d.VoiceService != nil {
		d.VoiceHandler = handlers.NewVoiceHandler(d.VoiceService, d.Logger)
	}

	if d.TrajectoryService != nil {
		d.TrajectoryHandler = handlers.NewTrajectoryHandler(d.TrajectoryService, d.Logger)
	}

	if d.SuggestionsService != nil {
		d.SuggestionsHandler = handlers.NewSuggestionsHandler(d.SuggestionsService, d.Logger)
	}

	return nil
}

// Close cleans up all resources held by the dependencies.
func (d *NutritionDependencies) Close() error {
	// Close database connection if it exists
	if d.DB != nil {
		d.DB.Close()
	}

	// Close cache if it exists
	if d.Cache != nil {
		if err := d.Cache.Close(); err != nil {
			d.Logger.Error("failed to close cache",
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

// HealthCheck performs a health check on all dependencies.
func (d *NutritionDependencies) HealthCheck(ctx context.Context) error {
	// Check database connection
	if d.DB != nil {
		if err := d.DB.PingContext(ctx); err != nil {
			return fmt.Errorf("database health check failed: %w", err)
		}
	}

	// Check cache
	if d.Cache != nil {
		// TODO: Add cache health check
	}

	return nil
}

// Adapter types will be added here when repositories are implemented
// TODO: Add adapter types to bridge interface incompatibilities between
// domain-specific interfaces and shared services
