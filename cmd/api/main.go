package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/internal/server"
	"github.com/pradord/lumen_final/backend/internal/supabase"
	"github.com/pradord/lumen_final/backend/pkg/logger"
)

func main() {
	// Run the application
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 2. Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// 3. Initialize logger
	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Pretty: cfg.Logging.Format == "console",
	})

	log.Info().
		Str("mode", string(cfg.Mode)).
		Bool("supabase_configured", cfg.Supabase.URL != "").
		Msg("Starting Go API Server")

	// 4. Initialize Supabase client
	supabaseClient, err := supabase.NewClient(cfg, log.WithComponent("supabase"))
	if err != nil {
		return fmt.Errorf("failed to initialize Supabase client: %w", err)
	}
	defer func() {
		if err := supabaseClient.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Supabase client")
		}
	}()

	// 5. Initialize database connection pool
	var db *sqlx.DB
	if cfg.Database.ConnectionString != "" {
		// Open database connection using lib/pq driver (compatible with pgx protocol)
		db, err = sqlx.Connect("postgres", cfg.Database.ConnectionString)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Configure connection pool
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
		db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

		// Verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("failed to ping database: %w", err)
		}

		log.Info().
			Str("host", "Supabase").
			Int("max_conns", cfg.Database.MaxOpenConns).
			Msg("Successfully connected to database")
	} else {
		log.Warn().Msg("No database connection string provided - running in stub mode")
	}

	// 6. Initialize nutrition dependencies
	slogLogger := slog.Default()
	nutritionDeps, err := server.NewNutritionDependencies(cfg, db, slogLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize nutrition dependencies: %w", err)
	}
	defer nutritionDeps.Close()

	log.Info().
		Bool("meals_enabled", nutritionDeps.MealsHandler != nil).
		Bool("weight_enabled", nutritionDeps.WeightHandler != nil).
		Bool("templates_enabled", nutritionDeps.TemplatesHandler != nil).
		Bool("analytics_enabled", nutritionDeps.AnalyticsHandler != nil).
		Bool("goals_enabled", nutritionDeps.GoalsHandler != nil).
		Msg("Nutrition domain initialized")

	// 7. Setup HTTP server with all handlers
	srv := server.NewServerWithHandlers(
		cfg,
		log.WithComponent("server"),
		supabaseClient,
		nutritionDeps.MealsHandler,
		nutritionDeps.WeightHandler,
		nutritionDeps.TemplatesHandler,
		nutritionDeps.AnalyticsHandler,
		nutritionDeps.GoalsHandler,
	)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      srv.Router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 7. Start HTTP server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info().
			Str("address", httpServer.Addr).
			Msg("Starting HTTP server")
		serverErrors <- httpServer.ListenAndServe()
	}()

	// 8. Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// 9. Block until shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-shutdown:
		log.Info().
			Str("signal", sig.String()).
			Msg("Shutdown signal received, starting graceful shutdown")

		// 10. Graceful shutdown with timeout (default 30 seconds)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown HTTP server
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
			if closeErr := httpServer.Close(); closeErr != nil {
				return fmt.Errorf("failed to force close HTTP server: %w", closeErr)
			}
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}

		log.Info().Msg("HTTP server stopped gracefully")
	}

	log.Info().Msg("Application shutdown complete")
	return nil
}
