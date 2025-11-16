package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommonFood represents a common food item with nutrition information
type CommonFood struct {
	Name                string  `json:"name"`
	Category            string  `json:"category"`
	CaloriesPer100g     float64 `json:"calories_per_100g"`
	ProteinPer100g      float64 `json:"protein_per_100g"`
	CarbsPer100g        float64 `json:"carbs_per_100g"`
	FatPer100g          float64 `json:"fat_per_100g"`
	FiberPer100g        float64 `json:"fiber_per_100g"`
	CommonServingName   string  `json:"common_serving_name"`
	CommonServingGrams  float64 `json:"common_serving_grams"`
	IsVerified          bool    `json:"is_verified"`
}

// CommonFoodsData represents the JSON structure of the foods file
type CommonFoodsData struct {
	Foods []CommonFood `json:"foods"`
}

// CommonFoodsSeeder handles seeding of common foods data
type CommonFoodsSeeder struct {
	db *pgxpool.Pool
}

// NewCommonFoodsSeeder creates a new CommonFoodsSeeder instance
func NewCommonFoodsSeeder(db *pgxpool.Pool) *CommonFoodsSeeder {
	return &CommonFoodsSeeder{
		db: db,
	}
}

// LoadFromFile loads common foods from a JSON file
func (s *CommonFoodsSeeder) LoadFromFile(filename string) ([]CommonFood, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse JSON
	var foodsData CommonFoodsData
	if err := json.Unmarshal(data, &foodsData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filename, err)
	}

	if len(foodsData.Foods) == 0 {
		return nil, fmt.Errorf("no foods found in file %s", filename)
	}

	return foodsData.Foods, nil
}

// Clear removes all common foods from the database
func (s *CommonFoodsSeeder) Clear(ctx context.Context) error {
	query := "DELETE FROM common_foods"

	_, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear common_foods table: %w", err)
	}

	return nil
}

// insertFood inserts a single food item into the database
func (s *CommonFoodsSeeder) insertFood(ctx context.Context, tx pgx.Tx, food CommonFood) error {
	query := `
		INSERT INTO common_foods (
			name,
			category,
			calories_per_100g,
			protein_per_100g,
			carbs_per_100g,
			fat_per_100g,
			fiber_per_100g,
			common_serving_name,
			common_serving_grams,
			is_verified
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := tx.Exec(ctx, query,
		food.Name,
		food.Category,
		food.CaloriesPer100g,
		food.ProteinPer100g,
		food.CarbsPer100g,
		food.FatPer100g,
		food.FiberPer100g,
		food.CommonServingName,
		food.CommonServingGrams,
		food.IsVerified,
	)

	if err != nil {
		return fmt.Errorf("failed to insert food %s: %w", food.Name, err)
	}

	return nil
}

// SeedFromFile seeds the database with foods from a specific file
func (s *CommonFoodsSeeder) SeedFromFile(ctx context.Context, filename string) error {
	// Load foods from file
	foods, err := s.LoadFromFile(filename)
	if err != nil {
		return err
	}

	// Clear existing data
	if err := s.Clear(ctx); err != nil {
		return err
	}

	// Begin transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert all foods
	for _, food := range foods {
		if err := s.insertFood(ctx, tx, food); err != nil {
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Seed seeds the database with common foods from the default file
func (s *CommonFoodsSeeder) Seed(ctx context.Context) error {
	// Get the default file path
	filename := s.getDefaultFilePath()

	return s.SeedFromFile(ctx, filename)
}

// SeedWithProgress seeds the database and sends progress updates
func (s *CommonFoodsSeeder) SeedWithProgress(ctx context.Context, progress chan<- int) error {
	defer close(progress)

	// Load foods from file
	filename := s.getDefaultFilePath()
	foods, err := s.LoadFromFile(filename)
	if err != nil {
		return err
	}

	totalFoods := len(foods)

	// Clear existing data
	if err := s.Clear(ctx); err != nil {
		return err
	}

	progress <- 0

	// Begin transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert foods with progress updates
	for i, food := range foods {
		if err := s.insertFood(ctx, tx, food); err != nil {
			return err
		}

		// Send progress update every 10 items or at the end
		if (i+1)%10 == 0 || i == totalFoods-1 {
			progress <- i + 1
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStats returns statistics about seeded foods
func (s *CommonFoodsSeeder) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// Get total count
	var totalCount int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total"] = totalCount

	// Get count by category
	rows, err := s.db.Query(ctx, `
		SELECT category, COUNT(*) as count
		FROM common_foods
		GROUP BY category
		ORDER BY category
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get category counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category count: %w", err)
		}
		stats[category] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category counts: %w", err)
	}

	// Get verified count
	var verifiedCount int
	err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods WHERE is_verified = true").Scan(&verifiedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get verified count: %w", err)
	}
	stats["verified"] = verifiedCount

	return stats, nil
}

// getDefaultFilePath returns the default path to common_foods.json
func (s *CommonFoodsSeeder) getDefaultFilePath() string {
	// Try relative path first (for development)
	relativePath := "internal/seed/data/common_foods.json"
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// Try absolute path from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		absPath := filepath.Join(cwd, "internal", "seed", "data", "common_foods.json")
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Try path relative to backend directory
	backendPath := filepath.Join("backend", "internal", "seed", "data", "common_foods.json")
	if _, err := os.Stat(backendPath); err == nil {
		return backendPath
	}

	// Default fallback
	return "data/common_foods.json"
}

// ValidateFood validates a food item's data
func (s *CommonFoodsSeeder) ValidateFood(food CommonFood) error {
	if food.Name == "" {
		return fmt.Errorf("food name cannot be empty")
	}

	if food.Category == "" {
		return fmt.Errorf("food category cannot be empty for %s", food.Name)
	}

	validCategories := map[string]bool{
		"Protein":    true,
		"Carbs":      true,
		"Vegetables": true,
		"Fruits":     true,
		"Dairy":      true,
		"Fats":       true,
		"Snacks":     true,
		"Beverages":  true,
	}

	if !validCategories[food.Category] {
		return fmt.Errorf("invalid category %s for food %s", food.Category, food.Name)
	}

	if food.CaloriesPer100g <= 0 {
		return fmt.Errorf("calories must be positive for %s", food.Name)
	}

	if food.ProteinPer100g < 0 || food.CarbsPer100g < 0 || food.FatPer100g < 0 || food.FiberPer100g < 0 {
		return fmt.Errorf("macronutrients cannot be negative for %s", food.Name)
	}

	if food.CommonServingName == "" {
		return fmt.Errorf("common serving name cannot be empty for %s", food.Name)
	}

	if food.CommonServingGrams <= 0 {
		return fmt.Errorf("common serving grams must be positive for %s", food.Name)
	}

	return nil
}

// SeedWithValidation seeds the database with validation
func (s *CommonFoodsSeeder) SeedWithValidation(ctx context.Context) error {
	// Load foods from file
	filename := s.getDefaultFilePath()
	foods, err := s.LoadFromFile(filename)
	if err != nil {
		return err
	}

	// Validate all foods before seeding
	for _, food := range foods {
		if err := s.ValidateFood(food); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
	}

	// If all foods are valid, proceed with seeding
	return s.SeedFromFile(ctx, filename)
}
