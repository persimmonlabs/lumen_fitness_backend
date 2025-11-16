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
	storage_go "github.com/supabase-community/storage-go"

	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/analytics"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/common_foods"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/goals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/templates"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/weight"
	"github.com/pradord/lumen_final/backend/internal/domain/user"
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
	MealsRepo       meals.Repository
	WeightRepo      weight.Repository
	TemplatesRepo   templates.Repository
	AnalyticsRepo   analytics.Repository
	GoalsRepo       goals.Repository
	UserRepo        user.Repository
	CommonFoodsRepo common_foods.Repository

	// Services (business logic)
	MealsService       meals.Service
	WeightService      weight.Service
	TemplatesService   templates.Service
	AnalyticsService   analytics.Service
	GoalsService       goals.Service
	UserService        user.Service
	CommonFoodsService common_foods.Service

	// Handlers (HTTP layer)
	MealsHandler       *meals.Handler
	WeightHandler      *weight.Handler
	TemplatesHandler   *templates.Handler
	AnalyticsHandler   *analytics.Handler
	GoalsHandler       *goals.Handler
	UserHandler        *user.Handler
	CommonFoodsHandler *handlers.CommonFoodsHandler

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

	// Supabase storage client for media uploads
	StorageClient interface{}
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
//   - storageClient: Supabase storage client for media uploads (can be nil)
//
// Returns:
//   - *NutritionDependencies: Fully initialized dependencies
//   - error: Any initialization error
func NewNutritionDependencies(cfg *config.Config, db *sqlx.DB, log *slog.Logger, storageClient *storage_go.Client) (*NutritionDependencies, error) {
	deps := &NutritionDependencies{
		DB:            db,
		Logger:        log,
		StorageClient: storageClient,
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

	// Initialize new services (pass config for Voice service initialization)
	if err := deps.initNewServices(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize new services: %w", err)
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
		d.TemplatesRepo = templates.NewRepository(db)
		d.AnalyticsRepo = analytics.NewRepository(db.DB)
		d.GoalsRepo = goals.NewRepository(db)
		d.UserRepo = user.NewPostgresRepository(db)

		d.Logger.Info("initialized repositories",
			slog.Bool("meals", d.MealsRepo != nil),
			slog.Bool("weight", d.WeightRepo != nil),
			slog.Bool("templates", d.TemplatesRepo != nil),
			slog.Bool("analytics", d.AnalyticsRepo != nil),
			slog.Bool("goals", d.GoalsRepo != nil),
			slog.Bool("user", d.UserRepo != nil),
		)
	} else {
		d.Logger.Warn("no database connection - running in stub mode")
	}

	// Initialize in-memory common foods repository (always available)
	d.CommonFoodsRepo = common_foods.NewRepository()

	// Load common foods data from JSON file
	dataPath := common_foods.GetDefaultFilePath()
	if err := d.CommonFoodsRepo.LoadFromFile(dataPath); err != nil {
		d.Logger.Warn("failed to load common foods data",
			slog.String("path", dataPath),
			slog.String("error", err.Error()),
		)
	} else {
		d.Logger.Info("loaded common foods data",
			slog.String("path", dataPath),
		)
	}

	return nil
}

// initServices initializes all domain services with their dependencies.
func (d *NutritionDependencies) initServices() error {
	// Initialize domain services if repositories are available
	if d.MealsRepo != nil {
		// Create adapters to bridge AI Coordinator to meals-specific interfaces
		var aiCoordinator meals.AICoordinator
		var aiEstimator meals.AIEstimator
		var aiTranscriber meals.AITranscriber
		var aiNormalizer meals.AINormalizer
		var cache meals.Cache

		// Connect AI Coordinator if available
		if d.AICoordinator != nil {
			aiCoordinator = &aiCoordinatorAdapter{coordinator: d.AICoordinator}
			aiEstimator = &aiEstimatorAdapter{coordinator: d.AICoordinator}
			aiTranscriber = &aiTranscriberAdapter{coordinator: d.AICoordinator}
			aiNormalizer = &aiNormalizerAdapter{coordinator: d.AICoordinator}
			d.Logger.Info("connected AI coordinator to meals service",
				slog.Bool("vision_supported", d.AICoordinator.SupportsVision()),
				slog.Bool("audio_supported", d.AICoordinator.SupportsAudio()),
			)
		}

		// Connect cache if available
		if d.Cache != nil {
			cache = &cacheAdapter{cache: d.Cache}
		}

		d.MealsService = meals.NewService(
			d.MealsRepo,
			aiCoordinator,
			aiEstimator,
			aiTranscriber,
			aiNormalizer,
			cache,
			nil, // CostTracker - TODO: implement when needed
			nil, // PhotoStorage - TODO: implement when Supabase storage is ready
			d.Logger,
		)

		if d.AICoordinator != nil {
			d.Logger.Info("initialized meals service with AI features enabled")
		} else {
			d.Logger.Info("initialized meals service without AI features")
		}
	}

	if d.WeightRepo != nil {
		d.WeightService = weight.NewService(d.WeightRepo)
		d.Logger.Info("initialized weight service")
	}

	if d.TemplatesRepo != nil {
		d.TemplatesService = templates.NewService(d.TemplatesRepo, d.DB)
		d.Logger.Info("initialized templates service")
	}

	if d.GoalsRepo != nil {
		d.GoalsService = goals.NewService(d.GoalsRepo)
		d.Logger.Info("initialized goals service")
	}

	if d.AnalyticsRepo != nil {
		// Analytics service requires weight and goals repositories
		d.AnalyticsService = analytics.NewService(d.AnalyticsRepo, d.WeightRepo, d.GoalsRepo)
		d.Logger.Info("initialized analytics service")
	}

	if d.UserRepo != nil {
		d.UserService = user.NewService(d.UserRepo)
		d.Logger.Info("initialized user service")
	}

	if d.CommonFoodsRepo != nil {
		d.CommonFoodsService = common_foods.NewService(d.CommonFoodsRepo, d.Logger)
		d.Logger.Info("initialized common foods service")
	}

	return nil
}

// initNewServices initializes media, voice, trajectory, and suggestions services.
func (d *NutritionDependencies) initNewServices(cfg *config.Config) error {
	// Initialize media service with Supabase Storage integration
	var storageClient *storage_go.Client
	if d.StorageClient != nil {
		var ok bool
		storageClient, ok = d.StorageClient.(*storage_go.Client)
		if !ok {
			d.Logger.Warn("storage client type assertion failed - using mock mode")
			storageClient = nil
		}
	}

	// Configure media service with storage bucket from config
	mediaConfig := media.DefaultConfig()
	if cfg.Supabase.StorageBucket != "" {
		mediaConfig.StorageBucket = cfg.Supabase.StorageBucket
	}

	d.MediaService = media.NewService(storageClient, d.Logger, mediaConfig)

	// Log initialization status
	if storageClient != nil && cfg.Features.EnableStorage {
		d.Logger.Info("initialized media service with Supabase Storage",
			slog.String("mode", "production"),
			slog.String("bucket", mediaConfig.StorageBucket),
			slog.Bool("storage_connected", true),
		)
	} else if cfg.Features.EnableStorage && cfg.Supabase.URL != "" {
		d.Logger.Warn("initialized media service in mock mode",
			slog.String("mode", "mock"),
			slog.String("reason", "storage client not available"),
			slog.String("bucket", mediaConfig.StorageBucket),
		)
	} else {
		d.Logger.Info("initialized media service in mock mode",
			slog.String("mode", "mock"),
			slog.Bool("storage_enabled", false),
		)
	}

	// Initialize voice service (requires Groq API key)
	// Voice service is optional and only initialized if Groq API key is available
	if cfg.Features.EnableAIAnalysis && cfg.AI.GroqAPIKey != "" {
		d.VoiceService = voice.NewService(cfg.AI.GroqAPIKey, d.Logger, voice.DefaultConfig())
		d.Logger.Info("initialized voice service",
			slog.String("provider", "groq"),
			slog.String("model", "whisper-large-v3"),
			slog.Int("max_file_size_mb", 25),
		)
	} else {
		d.Logger.Warn("voice service not initialized",
			slog.Bool("ai_enabled", cfg.Features.EnableAIAnalysis),
			slog.Bool("groq_key_present", cfg.AI.GroqAPIKey != ""),
			slog.String("reason", "Groq API key required for voice transcription"),
		)
	}

	// Initialize trajectory service (no external dependencies)
	d.TrajectoryService = trajectory.NewService(d.Logger, trajectory.DefaultConfig())
	d.Logger.Info("initialized trajectory service")

	// Initialize suggestions service (requires AI coordinator)
	if d.AICoordinator != nil {
		d.SuggestionsService = suggestions.NewService(d.AICoordinator, d.Logger, suggestions.DefaultConfig())
		d.Logger.Info("initialized suggestions service")
	} else {
		d.Logger.Warn("suggestions service not initialized - AI coordinator not available")
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

	if d.UserService != nil {
		d.UserHandler = user.NewHandler(d.UserService)
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

	if d.CommonFoodsService != nil {
		d.CommonFoodsHandler = handlers.NewCommonFoodsHandler(d.CommonFoodsService, d.Logger)
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
