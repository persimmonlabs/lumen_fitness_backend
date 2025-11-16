package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pradord/lumen_final/backend/internal/seed"
)

const (
	// Default database URL
	defaultDatabaseURL = "postgres://postgres:password@localhost:5432/lumen?sslmode=disable"

	// Colors for terminal output
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// Config holds the CLI configuration
type Config struct {
	DatabaseURL string
	CustomFile  string
	Clear       bool
	Stats       bool
	Validate    bool
	Progress    bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line flags
	config := parseFlags()

	// Print banner
	printBanner()

	// Get database URL
	dbURL := config.DatabaseURL
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = defaultDatabaseURL
	}

	// Connect to database
	fmt.Printf("%sConnecting to database...%s\n", colorCyan, colorReset)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Printf("%sConnected successfully!%s\n\n", colorGreen, colorReset)

	// Create seeder
	seeder := seed.NewCommonFoodsSeeder(pool)

	// Handle --stats flag
	if config.Stats {
		return showStats(ctx, seeder)
	}

	// Handle --clear flag
	if config.Clear {
		fmt.Printf("%sClearing existing common foods...%s\n", colorYellow, colorReset)
		if err := seeder.Clear(ctx); err != nil {
			return fmt.Errorf("failed to clear data: %w", err)
		}
		fmt.Printf("%sExisting data cleared successfully!%s\n\n", colorGreen, colorReset)
	}

	// Seed database
	startTime := time.Now()

	if config.Progress {
		err = seedWithProgress(ctx, seeder, config)
	} else if config.Validate {
		fmt.Printf("%sSeeding with validation...%s\n", colorCyan, colorReset)
		err = seeder.SeedWithValidation(ctx)
	} else if config.CustomFile != "" {
		fmt.Printf("%sSeeding from custom file: %s%s\n", colorCyan, config.CustomFile, colorReset)
		err = seeder.SeedFromFile(ctx, config.CustomFile)
	} else {
		fmt.Printf("%sSeeding common foods database...%s\n", colorCyan, colorReset)
		err = seeder.Seed(ctx)
	}

	if err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	duration := time.Since(startTime)

	fmt.Printf("\n%sSeeding completed successfully in %v!%s\n\n", colorGreen, duration, colorReset)

	// Show statistics
	return showStats(ctx, seeder)
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.DatabaseURL, "db", "", "Database URL (defaults to DATABASE_URL env var)")
	flag.StringVar(&config.CustomFile, "file", "", "Custom JSON file to seed from")
	flag.BoolVar(&config.Clear, "clear", false, "Clear existing data before seeding")
	flag.BoolVar(&config.Stats, "stats", false, "Show statistics only (no seeding)")
	flag.BoolVar(&config.Validate, "validate", false, "Validate foods before seeding")
	flag.BoolVar(&config.Progress, "progress", false, "Show progress during seeding")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Lumen Nutrition Tracker - Common Foods Seeder\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                         # Seed with default data\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --clear                 # Clear existing data and seed\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --file custom.json      # Seed from custom file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --stats                 # Show statistics only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --validate --progress   # Validate and show progress\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	return config
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║         Lumen Nutrition Tracker                           ║
║         Common Foods Seeder                               ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Printf("%s%s%s\n", colorBlue, banner, colorReset)
}

func seedWithProgress(ctx context.Context, seeder *seed.CommonFoodsSeeder, config Config) error {
	progressChan := make(chan int, 200)

	// Start seeding in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if config.Validate {
			errChan <- seeder.SeedWithValidation(ctx)
		} else if config.CustomFile != "" {
			errChan <- seeder.SeedFromFile(ctx, config.CustomFile)
		} else {
			errChan <- seeder.SeedWithProgress(ctx, progressChan)
		}
	}()

	// Display progress
	fmt.Printf("%sSeeding in progress...%s\n", colorCyan, colorReset)

	var lastProgress int
	for progress := range progressChan {
		if progress > lastProgress {
			fmt.Printf("\r%sProcessed: %d foods%s", colorYellow, progress, colorReset)
			lastProgress = progress
		}
	}
	fmt.Println()

	// Check for errors
	return <-errChan
}

func showStats(ctx context.Context, seeder *seed.CommonFoodsSeeder) error {
	stats, err := seeder.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("%sDatabase Statistics:%s\n", colorBlue, colorReset)
	fmt.Printf("%s═══════════════════════════════════════════════════════════%s\n", colorBlue, colorReset)

	// Print total
	total, ok := stats["total"]
	if ok {
		fmt.Printf("Total Foods:     %s%d%s\n", colorGreen, total, colorReset)
		delete(stats, "total")
	}

	// Print verified count
	verified, ok := stats["verified"]
	if ok {
		fmt.Printf("Verified Foods:  %s%d%s\n", colorGreen, verified, colorReset)
		delete(stats, "verified")
	}

	fmt.Println()

	// Print categories
	fmt.Printf("%sBy Category:%s\n", colorBlue, colorReset)

	categories := []string{
		"Protein",
		"Carbs",
		"Vegetables",
		"Fruits",
		"Dairy",
		"Fats",
		"Snacks",
		"Beverages",
	}

	for _, category := range categories {
		count, ok := stats[category]
		if ok {
			fmt.Printf("  %-15s %s%3d%s\n", category+":", colorCyan, count, colorReset)
		}
	}

	fmt.Printf("%s═══════════════════════════════════════════════════════════%s\n", colorBlue, colorReset)
	fmt.Println()

	return nil
}
