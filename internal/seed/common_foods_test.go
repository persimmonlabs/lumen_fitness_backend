package seed

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Use test database URL from environment or default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/lumen_test?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err, "Failed to connect to test database")

	// Verify connection
	err = pool.Ping(context.Background())
	require.NoError(t, err, "Failed to ping test database")

	return pool
}

// cleanupTestDB drops all data from common_foods table
func cleanupTestDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, err := pool.Exec(ctx, "DELETE FROM common_foods")
	require.NoError(t, err, "Failed to clean up test database")
}

// TestCommonFoodsSeeder_New tests seeder creation
func TestCommonFoodsSeeder_New(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	seeder := NewCommonFoodsSeeder(pool)

	assert.NotNil(t, seeder)
	assert.Equal(t, pool, seeder.db)
}

// TestCommonFoodsSeeder_LoadFromFile tests loading food data from JSON file
func TestCommonFoodsSeeder_LoadFromFile(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	seeder := NewCommonFoodsSeeder(pool)

	tests := []struct {
		name        string
		filename    string
		expectError bool
		minFoods    int
	}{
		{
			name:        "Valid common_foods.json",
			filename:    "data/common_foods.json",
			expectError: false,
			minFoods:    100,
		},
		{
			name:        "Non-existent file",
			filename:    "data/non_existent.json",
			expectError: true,
			minFoods:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foods, err := seeder.LoadFromFile(tt.filename)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, foods)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, foods)
				assert.GreaterOrEqual(t, len(foods), tt.minFoods)

				// Validate first food item structure
				if len(foods) > 0 {
					food := foods[0]
					assert.NotEmpty(t, food.Name)
					assert.NotEmpty(t, food.Category)
					assert.Greater(t, food.CaloriesPer100g, float64(0))
					assert.GreaterOrEqual(t, food.ProteinPer100g, float64(0))
					assert.GreaterOrEqual(t, food.CarbsPer100g, float64(0))
					assert.GreaterOrEqual(t, food.FatPer100g, float64(0))
					assert.GreaterOrEqual(t, food.FiberPer100g, float64(0))
					assert.NotEmpty(t, food.CommonServingName)
					assert.Greater(t, food.CommonServingGrams, float64(0))
					assert.True(t, food.IsVerified)
				}
			}
		})
	}
}

// TestCommonFoodsSeeder_LoadFromFile_InvalidJSON tests handling of invalid JSON
func TestCommonFoodsSeeder_LoadFromFile_InvalidJSON(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	// Create temporary invalid JSON file
	tmpFile, err := os.CreateTemp("", "invalid_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(`{"foods": [{"invalid json`)
	require.NoError(t, err)
	tmpFile.Close()

	seeder := NewCommonFoodsSeeder(pool)
	foods, err := seeder.LoadFromFile(tmpFile.Name())

	assert.Error(t, err)
	assert.Nil(t, foods)
}

// TestCommonFoodsSeeder_Seed tests seeding database with common foods
func TestCommonFoodsSeeder_Seed(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)

	ctx := context.Background()
	err := seeder.Seed(ctx)

	assert.NoError(t, err)

	// Verify foods were inserted
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 100, "Should have at least 100 foods")
}

// TestCommonFoodsSeeder_SeedWithCustomFile tests seeding with custom file
func TestCommonFoodsSeeder_SeedWithCustomFile(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	// Create temporary custom foods file
	tmpFile, err := os.CreateTemp("", "custom_foods_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	customData := CommonFoodsData{
		Foods: []CommonFood{
			{
				Name:                "Test Food 1",
				Category:            "Protein",
				CaloriesPer100g:     200,
				ProteinPer100g:      25.0,
				CarbsPer100g:        5.0,
				FatPer100g:          10.0,
				FiberPer100g:        2.0,
				CommonServingName:   "serving",
				CommonServingGrams:  100,
				IsVerified:          true,
			},
			{
				Name:                "Test Food 2",
				Category:            "Carbs",
				CaloriesPer100g:     150,
				ProteinPer100g:      3.0,
				CarbsPer100g:        30.0,
				FatPer100g:          2.0,
				FiberPer100g:        5.0,
				CommonServingName:   "cup",
				CommonServingGrams:  150,
				IsVerified:          true,
			},
		},
	}

	jsonData, err := json.Marshal(customData)
	require.NoError(t, err)

	_, err = tmpFile.Write(jsonData)
	require.NoError(t, err)
	tmpFile.Close()

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()
	err = seeder.SeedFromFile(ctx, tmpFile.Name())

	assert.NoError(t, err)

	// Verify custom foods were inserted
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should have exactly 2 custom foods")

	// Verify food details
	var name, category string
	err = pool.QueryRow(ctx,
		"SELECT name, category FROM common_foods WHERE name = $1",
		"Test Food 1",
	).Scan(&name, &category)

	require.NoError(t, err)
	assert.Equal(t, "Test Food 1", name)
	assert.Equal(t, "Protein", category)
}

// TestCommonFoodsSeeder_Clear tests clearing existing foods
func TestCommonFoodsSeeder_Clear(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	// First seed some data
	err := seeder.Seed(ctx)
	require.NoError(t, err)

	// Verify data exists
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "Should have foods before clearing")

	// Clear the data
	err = seeder.Clear(ctx)
	require.NoError(t, err)

	// Verify data is cleared
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should have no foods after clearing")
}

// TestCommonFoodsSeeder_Seed_Categories tests all food categories
func TestCommonFoodsSeeder_Seed_Categories(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	err := seeder.Seed(ctx)
	require.NoError(t, err)

	expectedCategories := []string{
		"Protein",
		"Carbs",
		"Vegetables",
		"Fruits",
		"Dairy",
		"Fats",
		"Snacks",
		"Beverages",
	}

	for _, category := range expectedCategories {
		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM common_foods WHERE category = $1",
			category,
		).Scan(&count)

		require.NoError(t, err)
		assert.Greater(t, count, 0, "Category %s should have at least one food", category)
	}
}

// TestCommonFoodsSeeder_Seed_Idempotent tests that seeding is idempotent
func TestCommonFoodsSeeder_Seed_Idempotent(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	// Seed first time
	err := seeder.Seed(ctx)
	require.NoError(t, err)

	var countFirst int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&countFirst)
	require.NoError(t, err)

	// Seed second time (should clear and re-seed)
	err = seeder.Seed(ctx)
	require.NoError(t, err)

	var countSecond int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM common_foods").Scan(&countSecond)
	require.NoError(t, err)

	assert.Equal(t, countFirst, countSecond, "Seeding should be idempotent")
}

// TestCommonFoodsSeeder_Seed_NutritionData tests nutrition data accuracy
func TestCommonFoodsSeeder_Seed_NutritionData(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	err := seeder.Seed(ctx)
	require.NoError(t, err)

	// Test specific known food
	var calories, protein, carbs, fat, fiber float64
	err = pool.QueryRow(ctx, `
		SELECT calories_per_100g, protein_per_100g, carbs_per_100g, fat_per_100g, fiber_per_100g
		FROM common_foods
		WHERE name = $1
	`, "Chicken breast, grilled").Scan(&calories, &protein, &carbs, &fat, &fiber)

	require.NoError(t, err)
	assert.Equal(t, 165.0, calories)
	assert.Equal(t, 31.0, protein)
	assert.Equal(t, 0.0, carbs)
	assert.Equal(t, 3.6, fat)
	assert.Equal(t, 0.0, fiber)
}

// TestCommonFoodsSeeder_Seed_ServingSizes tests serving size data
func TestCommonFoodsSeeder_Seed_ServingSizes(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	err := seeder.Seed(ctx)
	require.NoError(t, err)

	// Verify all foods have valid serving sizes
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM common_foods
		WHERE common_serving_grams <= 0 OR common_serving_name = ''
	`).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "All foods should have valid serving sizes")
}

// TestCommonFoodsSeeder_Seed_VerificationStatus tests is_verified flag
func TestCommonFoodsSeeder_Seed_VerificationStatus(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	err := seeder.Seed(ctx)
	require.NoError(t, err)

	// All seeded foods should be verified
	var unverifiedCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM common_foods
		WHERE is_verified = false
	`).Scan(&unverifiedCount)

	require.NoError(t, err)
	assert.Equal(t, 0, unverifiedCount, "All seeded foods should be verified")
}

// TestCommonFoodsSeeder_SeedProgress tests progress reporting during seeding
func TestCommonFoodsSeeder_SeedProgress(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)

	seeder := NewCommonFoodsSeeder(pool)

	// Create a channel to receive progress updates
	progressChan := make(chan int, 200)

	ctx := context.Background()
	err := seeder.SeedWithProgress(ctx, progressChan)

	assert.NoError(t, err)

	// Check that we received progress updates
	progressUpdates := 0
	for range progressChan {
		progressUpdates++
	}

	assert.Greater(t, progressUpdates, 0, "Should receive progress updates")
}

// BenchmarkCommonFoodsSeeder_Seed benchmarks seeding performance
func BenchmarkCommonFoodsSeeder_Seed(b *testing.B) {
	pool := setupTestDB(&testing.T{})
	defer pool.Close()

	seeder := NewCommonFoodsSeeder(pool)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = seeder.Clear(ctx)
		_ = seeder.Seed(ctx)
	}
}

// BenchmarkCommonFoodsSeeder_LoadFromFile benchmarks file loading
func BenchmarkCommonFoodsSeeder_LoadFromFile(b *testing.B) {
	pool := setupTestDB(&testing.T{})
	defer pool.Close()

	seeder := NewCommonFoodsSeeder(pool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = seeder.LoadFromFile("data/common_foods.json")
	}
}
