package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/pradord/lumen_final/backend/internal/jobs/meal_flagging"
)

const (
	version = "1.0.0"
)

var (
	// Flags
	mode         = flag.String("mode", "run-once", "Execution mode: run-once, schedule, user")
	userID       = flag.String("user", "", "User ID (required for user mode)")
	configPath   = flag.String("config", "", "Path to config file (optional)")
	showVersion  = flag.Bool("version", false, "Show version")
	showHelp     = flag.Bool("help", false, "Show help")
	verbose      = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Meal Flagging Job v%s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Setup logger
	logger := setupLogger(*verbose)
	logger.Printf("Starting Meal Flagging Job v%s", version)

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := connectDB()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create job instance
	job := meal_flagging.NewMealFlaggingJob(db, config, logger)

	// Execute based on mode
	ctx := context.Background()

	switch *mode {
	case "run-once":
		runOnce(ctx, job, logger)
	case "schedule":
		runScheduled(job, logger)
	case "user":
		runForUser(ctx, job, logger, *userID)
	default:
		logger.Fatalf("Invalid mode: %s. Use run-once, schedule, or user", *mode)
	}
}

func runOnce(ctx context.Context, job *meal_flagging.MealFlaggingJob, logger *log.Logger) {
	logger.Println("Running job once")

	startTime := time.Now()
	if err := job.Run(ctx); err != nil {
		logger.Fatalf("Job execution failed: %v", err)
	}

	logger.Printf("Job completed in %v", time.Since(startTime))
}

func runScheduled(job *meal_flagging.MealFlaggingJob, logger *log.Logger) {
	logger.Println("Starting scheduled job")

	if err := job.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Println("Job scheduler started. Press Ctrl+C to stop.")
	<-sigChan

	logger.Println("Shutting down...")
	job.Stop()
	logger.Println("Shutdown complete")
}

func runForUser(ctx context.Context, job *meal_flagging.MealFlaggingJob, logger *log.Logger, userIDStr string) {
	if userIDStr == "" {
		logger.Fatal("User ID is required for user mode. Use -user=<uuid>")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Fatalf("Invalid user ID: %v", err)
	}

	logger.Printf("Running job for user %s", userUUID)

	startTime := time.Now()
	summary, err := job.RunForUser(ctx, userUUID)
	if err != nil {
		logger.Fatalf("Job execution failed: %v", err)
	}

	// Print summary
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println("\nJob Summary:")
	fmt.Println(string(summaryJSON))

	logger.Printf("Job completed in %v", time.Since(startTime))
}

func loadConfig(configPath string) (*meal_flagging.Config, error) {
	// If no config path provided, use defaults
	if configPath == "" {
		return meal_flagging.DefaultConfig(), nil
	}

	// Load config from file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config meal_flagging.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return &config, nil
}

func connectDB() (*sql.DB, error) {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func setupLogger(verbose bool) *log.Logger {
	prefix := "[MEAL-FLAGGING] "
	flags := log.LstdFlags

	if verbose {
		flags |= log.Lshortfile
	}

	return log.New(os.Stdout, prefix, flags)
}

func printHelp() {
	fmt.Printf(`Meal Flagging Job v%s

USAGE:
    meal_flagging [OPTIONS]

MODES:
    run-once    Run the job once and exit (default)
    schedule    Run as a scheduled daemon
    user        Run for a specific user only

OPTIONS:
    -mode=MODE          Execution mode (run-once, schedule, user)
    -user=UUID          User ID (required for user mode)
    -config=PATH        Path to JSON config file
    -verbose            Enable verbose logging
    -version            Show version
    -help               Show this help

EXAMPLES:
    # Run once with default settings
    meal_flagging

    # Run once with custom config
    meal_flagging -config=config.json

    # Run as scheduled daemon
    meal_flagging -mode=schedule

    # Run for specific user
    meal_flagging -mode=user -user=550e8400-e29b-41d4-a716-446655440000

    # Enable verbose logging
    meal_flagging -verbose

ENVIRONMENT VARIABLES:
    DATABASE_URL        PostgreSQL connection string (required)

CONFIG FILE FORMAT (JSON):
    {
      "enabled": true,
      "schedule_time": "02:00",
      "batch_size": 1000,
      "lookback_days": 1,
      "macro_tolerance_percent": 10.0,
      "confidence_threshold": 0.7,
      "duplicate_window_minutes": 30,
      "duplicate_similarity_percent": 80.0
    }

FLAGGING RULES:
    1. Unusual Portion    - Items exceeding typical serving sizes
    2. Macro Mismatch     - Calories don't match macro calculation
    3. Duplicate          - Similar meals within 30 minutes
    4. Low Confidence     - AI confidence below threshold

EXIT CODES:
    0    Success
    1    Error

`, version)
}
