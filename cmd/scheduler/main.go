package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/pradord/lumen_final/backend/internal/jobs/meal_flagging"
)

const (
	defaultConfigPath = "config/meal_flagging.json"
)

func main() {
	logger := log.New(os.Stdout, "[SCHEDULER] ", log.LstdFlags)

	// Load database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}

	// Load job configuration
	configPath := os.Getenv("JOB_CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	config, err := loadConfig(configPath)
	if err != nil {
		logger.Printf("Warning: failed to load config from %s: %v", configPath, err)
		logger.Println("Using default configuration")
		config = meal_flagging.DefaultConfig()
	}

	// Connect to database
	logger.Println("Connecting to database...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Println("Database connection established")

	// Create and start the meal flagging job
	job := meal_flagging.NewMealFlaggingJob(db, config, logger)

	if err := job.Start(); err != nil {
		logger.Fatalf("Failed to start job: %v", err)
	}

	// Print configuration
	logger.Println("=== Meal Flagging Job Configuration ===")
	logger.Printf("Enabled: %v", config.Enabled)
	logger.Printf("Schedule: %s daily", config.ScheduleTime)
	logger.Printf("Batch size: %d", config.BatchSize)
	logger.Printf("Lookback days: %d", config.LookbackDays)
	logger.Printf("Macro tolerance: %.1f%%", config.MacroTolerancePercent)
	logger.Printf("Confidence threshold: %.1f%%", config.ConfidenceThreshold*100)
	logger.Printf("Duplicate window: %d minutes", config.DuplicateWindowMinutes)
	logger.Printf("Duplicate similarity: %.1f%%", config.DuplicateSimilarityPercent)
	logger.Println("=======================================")

	// Run immediately on startup if enabled
	if shouldRunOnStartup() {
		logger.Println("Running job immediately on startup...")
		if err := job.Run(context.Background()); err != nil {
			logger.Printf("Initial job run failed: %v", err)
		}
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("Scheduler is running. Press Ctrl+C to exit.")
	<-quit

	logger.Println("Shutting down scheduler...")
	job.Stop()

	// Give the job some time to finish current execution
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	logger.Println("Scheduler stopped")
}

// loadConfig loads the job configuration from a JSON file
func loadConfig(path string) (*meal_flagging.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	var config meal_flagging.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

// shouldRunOnStartup checks if the job should run immediately on startup
func shouldRunOnStartup() bool {
	runOnStartup := os.Getenv("RUN_ON_STARTUP")
	return runOnStartup == "true" || runOnStartup == "1"
}
