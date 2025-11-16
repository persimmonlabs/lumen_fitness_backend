package meals_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig holds test database configuration
type TestConfig struct {
	DatabaseURL string
}

// TestFixtures holds test data
type TestFixtures struct {
	UserID uuid.UUID
	Meals  []uuid.UUID
}

var (
	testDB   *sqlx.DB
	testRepo meals.Repository
)

// TestMain sets up and tears down the test database
func TestMain(m *testing.M) {
	var err error

	// Load test configuration from environment
	config := loadTestConfig()
	if config.DatabaseURL == "" {
		fmt.Println("Skipping integration tests: SUPABASE_TEST_DATABASE_URL not set")
		os.Exit(0)
	}

	// Connect to test database
	testDB, err = sqlx.Connect("postgres", config.DatabaseURL)
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	// Set connection pool settings
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := testDB.Ping(); err != nil {
		fmt.Printf("Failed to ping test database: %v\n", err)
		os.Exit(1)
	}

	// Initialize repository
	testRepo = meals.NewRepository(testDB)

	// Run tests
	code := m.Run()

	// Cleanup
	testDB.Close()
	os.Exit(code)
}

// loadTestConfig loads test configuration from environment
func loadTestConfig() TestConfig {
	return TestConfig{
		DatabaseURL: os.Getenv("SUPABASE_TEST_DATABASE_URL"),
	}
}

// setupTestFixtures creates test data
func setupTestFixtures(t *testing.T) *TestFixtures {
	t.Helper()

	fixtures := &TestFixtures{
		UserID: uuid.New(),
		Meals:  []uuid.UUID{},
	}

	// Create test user in database
	_, err := testDB.Exec(`
		INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, fixtures.UserID, fmt.Sprintf("test-%s@example.com", fixtures.UserID.String()[:8]), "test-password")
	require.NoError(t, err)

	return fixtures
}

// cleanupTestFixtures removes test data
func cleanupTestFixtures(t *testing.T, fixtures *TestFixtures) {
	t.Helper()

	// Clean up in reverse order of dependencies
	for _, mealID := range fixtures.Meals {
		testDB.Exec("DELETE FROM meal_items WHERE meal_id = $1", mealID)
		testDB.Exec("DELETE FROM meals WHERE id = $1", mealID)
	}

	testDB.Exec("DELETE FROM auth.users WHERE id = $1", fixtures.UserID)
}

// TestCreateMealWithItems_RPC verifies the create_meal_with_items RPC function
func TestCreateMealWithItems_RPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("creates meal with items using RPC", func(t *testing.T) {
		// Prepare meal data
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeBreakfast,
			ConsumedAt:    time.Now().UTC(),
			Photos:        []string{"photo1.jpg", "photo2.jpg"},
			Notes:         "Test meal with RPC",
			TotalCalories: 500.0,
			TotalProteinG: 30.0,
			TotalCarbsG:   40.0,
			TotalFatG:     15.0,
			TotalFiberG:   5.0,
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Scrambled Eggs",
				Quantity: 2,
				Unit:     "eggs",
				Calories: 200.0,
				ProteinG: 12.0,
				CarbsG:   2.0,
				FatG:     14.0,
				FiberG:   0.0,
			},
			{
				ID:       uuid.New(),
				Name:     "Whole Wheat Toast",
				Quantity: 2,
				Unit:     "slices",
				Calories: 160.0,
				ProteinG: 8.0,
				CarbsG:   28.0,
				FatG:     2.0,
				FiberG:   4.0,
			},
			{
				ID:       uuid.New(),
				Name:     "Avocado",
				Quantity: 0.5,
				Unit:     "fruit",
				Calories: 120.0,
				ProteinG: 1.5,
				CarbsG:   6.0,
				FatG:     10.0,
				FiberG:   5.0,
			},
		}

		// Create meal using RPC
		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err, "RPC function should create meal successfully")
		require.NotNil(t, result)

		// Track for cleanup
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Verify meal data
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, fixtures.UserID, result.UserID)
		assert.Equal(t, meals.MealTypeBreakfast, result.MealType)
		assert.Equal(t, meal.Notes, result.Notes)
		assert.Equal(t, meal.TotalCalories, result.TotalCalories)
		assert.Equal(t, meal.TotalProteinG, result.TotalProteinG)
		assert.Equal(t, meal.TotalCarbsG, result.TotalCarbsG)
		assert.Equal(t, meal.TotalFatG, result.TotalFatG)
		assert.Equal(t, meal.TotalFiberG, result.TotalFiberG)

		// Verify items were created
		assert.Len(t, result.Items, 3, "All three items should be created")

		// Verify first item
		assert.Equal(t, "Scrambled Eggs", result.Items[0].Name)
		assert.Equal(t, 2.0, result.Items[0].Quantity)
		assert.Equal(t, "eggs", result.Items[0].Unit)
		assert.Equal(t, 200.0, result.Items[0].Calories)
		assert.Equal(t, 12.0, result.Items[0].ProteinG)
		assert.Equal(t, 2.0, result.Items[0].CarbsG)
		assert.Equal(t, 14.0, result.Items[0].FatG)
		assert.Equal(t, 0.0, result.Items[0].FiberG)
	})

	t.Run("verifies meal_items column names", func(t *testing.T) {
		// Create a simple meal
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeLunch,
			ConsumedAt:    time.Now().UTC(),
			Photos:        []string{},
			Notes:         "Column verification test",
			TotalCalories: 300.0,
			TotalProteinG: 20.0,
			TotalCarbsG:   30.0,
			TotalFatG:     10.0,
			TotalFiberG:   5.0,
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Test Food",
				Quantity: 100.0,
				Unit:     "g",
				Calories: 300.0,
				ProteinG: 20.0,
				CarbsG:   30.0,
				FatG:     10.0,
				FiberG:   5.0,
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Query meal_items directly to verify column names
		var dbItem struct {
			ID       uuid.UUID `db:"id"`
			MealID   uuid.UUID `db:"meal_id"`
			Name     string    `db:"name"`
			Quantity float64   `db:"quantity"`
			Unit     string    `db:"unit"`
			Calories float64   `db:"calories"`
			Protein  float64   `db:"protein"`
			Carbs    float64   `db:"carbs"`
			Fat      float64   `db:"fat"`
			Fiber    float64   `db:"fiber"`
		}

		query := `
			SELECT id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber
			FROM meal_items
			WHERE meal_id = $1
		`

		err = testDB.Get(&dbItem, query, result.ID)
		require.NoError(t, err, "Should successfully query with correct column names (protein, carbs, fat, fiber)")

		// Verify values
		assert.Equal(t, result.ID, dbItem.MealID)
		assert.Equal(t, "Test Food", dbItem.Name)
		assert.Equal(t, 100.0, dbItem.Quantity)
		assert.Equal(t, "g", dbItem.Unit)
		assert.Equal(t, 300.0, dbItem.Calories)
		assert.Equal(t, 20.0, dbItem.Protein)
		assert.Equal(t, 30.0, dbItem.Carbs)
		assert.Equal(t, 10.0, dbItem.Fat)
		assert.Equal(t, 5.0, dbItem.Fiber)
	})
}

// TestListMealsByUser_SQL verifies ListMealsByUser works with real SQL
func TestListMealsByUser_SQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	// Create multiple meals for testing
	mealTypes := []meals.MealType{
		meals.MealTypeBreakfast,
		meals.MealTypeLunch,
		meals.MealTypeDinner,
		meals.MealTypeSnack,
	}

	baseTime := time.Now().UTC().Add(-24 * time.Hour)

	for i, mealType := range mealTypes {
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      mealType,
			ConsumedAt:    baseTime.Add(time.Duration(i) * time.Hour),
			Photos:        []string{},
			Notes:         fmt.Sprintf("Test meal %d", i+1),
			TotalCalories: float64(300 + i*50),
			TotalProteinG: float64(20 + i*5),
			TotalCarbsG:   float64(30 + i*5),
			TotalFatG:     float64(10 + i*2),
			TotalFiberG:   float64(5 + i),
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     fmt.Sprintf("Food Item %d", i+1),
				Quantity: 100.0,
				Unit:     "g",
				Calories: meal.TotalCalories,
				ProteinG: meal.TotalProteinG,
				CarbsG:   meal.TotalCarbsG,
				FatG:     meal.TotalFatG,
				FiberG:   meal.TotalFiberG,
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)
	}

	t.Run("lists all meals with pagination", func(t *testing.T) {
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 10,
		}

		mealList, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err, "Should list meals without parameter binding errors")
		assert.Equal(t, 4, total, "Should return correct total count")
		assert.Len(t, mealList, 4, "Should return all 4 meals")

		// Verify meals are ordered by consumed_at DESC
		for i := 0; i < len(mealList)-1; i++ {
			assert.True(t, mealList[i].ConsumedAt.After(mealList[i+1].ConsumedAt) ||
				mealList[i].ConsumedAt.Equal(mealList[i+1].ConsumedAt),
				"Meals should be ordered by consumed_at DESC")
		}
	})

	t.Run("filters by meal type", func(t *testing.T) {
		lunchType := meals.MealTypeLunch
		filters := meals.ListMealFilters{
			Page:     1,
			Limit:    10,
			MealType: &lunchType,
		}

		mealList, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, mealList, 1)
		assert.Equal(t, meals.MealTypeLunch, mealList[0].MealType)
	})

	t.Run("filters by date", func(t *testing.T) {
		// Get date of first meal
		dateFilter := baseTime.Truncate(24 * time.Hour)
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 10,
			Date:  &dateFilter,
		}

		mealList, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err, "Should filter by date without parameter errors")
		assert.GreaterOrEqual(t, total, 1, "Should find at least one meal on that date")

		// Verify all returned meals are on the correct date
		for _, meal := range mealList {
			assert.Equal(t, dateFilter.Year(), meal.ConsumedAt.Year())
			assert.Equal(t, dateFilter.Month(), meal.ConsumedAt.Month())
			assert.Equal(t, dateFilter.Day(), meal.ConsumedAt.Day())
		}
	})

	t.Run("handles pagination correctly", func(t *testing.T) {
		// Page 1 with limit 2
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 2,
		}

		page1, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, page1, 2)

		// Page 2 with limit 2
		filters.Page = 2
		page2, total2, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)
		assert.Equal(t, 4, total2)
		assert.Len(t, page2, 2)

		// Verify no overlap between pages
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
		assert.NotEqual(t, page1[1].ID, page2[1].ID)
	})
}

// TestMealCRUD_FullCycle tests complete CRUD operations
func TestMealCRUD_FullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	var mealID uuid.UUID

	t.Run("1. Create - creates a meal successfully", func(t *testing.T) {
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeDinner,
			ConsumedAt:    time.Now().UTC(),
			Photos:        []string{"dinner.jpg"},
			Notes:         "CRUD test meal",
			TotalCalories: 650.0,
			TotalProteinG: 45.0,
			TotalCarbsG:   60.0,
			TotalFatG:     20.0,
			TotalFiberG:   8.0,
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Grilled Chicken",
				Quantity: 200.0,
				Unit:     "g",
				Calories: 330.0,
				ProteinG: 31.0,
				CarbsG:   0.0,
				FatG:     7.0,
				FiberG:   0.0,
			},
			{
				ID:       uuid.New(),
				Name:     "Brown Rice",
				Quantity: 150.0,
				Unit:     "g",
				Calories: 165.0,
				ProteinG: 3.5,
				CarbsG:   35.0,
				FatG:     1.0,
				FiberG:   2.0,
			},
			{
				ID:       uuid.New(),
				Name:     "Steamed Broccoli",
				Quantity: 100.0,
				Unit:     "g",
				Calories: 55.0,
				ProteinG: 3.7,
				CarbsG:   11.0,
				FatG:     0.6,
				FiberG:   2.6,
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		require.NotNil(t, result)

		mealID = result.ID
		fixtures.Meals = append(fixtures.Meals, mealID)

		assert.NotEqual(t, uuid.Nil, mealID)
		assert.Len(t, result.Items, 3)
	})

	t.Run("2. Read - retrieves the created meal", func(t *testing.T) {
		result, err := testRepo.GetMealByID(ctx, fixtures.UserID, mealID)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, mealID, result.ID)
		assert.Equal(t, fixtures.UserID, result.UserID)
		assert.Equal(t, meals.MealTypeDinner, result.MealType)
		assert.Equal(t, "CRUD test meal", result.Notes)
		assert.Equal(t, 650.0, result.TotalCalories)
		assert.Equal(t, 45.0, result.TotalProteinG)
		assert.Equal(t, 60.0, result.TotalCarbsG)
		assert.Equal(t, 20.0, result.TotalFatG)
		assert.Equal(t, 8.0, result.TotalFiberG)
		assert.Len(t, result.Items, 3)

		// Verify items
		itemNames := []string{"Grilled Chicken", "Brown Rice", "Steamed Broccoli"}
		for i, item := range result.Items {
			assert.Equal(t, itemNames[i], item.Name)
			assert.Equal(t, mealID, item.MealID)
		}
	})

	t.Run("3. Update - modifies the meal", func(t *testing.T) {
		updatedMeal := &meals.Meal{
			ID:            mealID,
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeLunch,
			ConsumedAt:    time.Now().UTC().Add(-1 * time.Hour),
			Photos:        []string{"updated.jpg"},
			Notes:         "Updated CRUD test meal",
			TotalCalories: 400.0,
			TotalProteinG: 30.0,
			TotalCarbsG:   40.0,
			TotalFatG:     10.0,
			TotalFiberG:   5.0,
		}

		updatedItems := []meals.MealItem{
			{
				ID:       uuid.New(),
				MealID:   mealID,
				Name:     "Updated Food",
				Quantity: 150.0,
				Unit:     "g",
				Calories: 400.0,
				ProteinG: 30.0,
				CarbsG:   40.0,
				FatG:     10.0,
				FiberG:   5.0,
			},
		}

		result, err := testRepo.UpdateMeal(ctx, fixtures.UserID, updatedMeal, updatedItems)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, mealID, result.ID)
		assert.Equal(t, meals.MealTypeLunch, result.MealType)
		assert.Equal(t, "Updated CRUD test meal", result.Notes)
		assert.Equal(t, 400.0, result.TotalCalories)
		assert.Len(t, result.Items, 1, "Old items should be replaced")
		assert.Equal(t, "Updated Food", result.Items[0].Name)
	})

	t.Run("4. List - meal appears in list", func(t *testing.T) {
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 10,
		}

		mealList, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, total, 1)

		// Find our meal in the list
		found := false
		for _, meal := range mealList {
			if meal.ID == mealID {
				found = true
				assert.Equal(t, meals.MealTypeLunch, meal.MealType)
				assert.Equal(t, 400.0, meal.TotalCalories)
				assert.Equal(t, 1, meal.ItemCount)
				break
			}
		}
		assert.True(t, found, "Updated meal should appear in list")
	})

	t.Run("5. Delete - removes the meal", func(t *testing.T) {
		err := testRepo.DeleteMeal(ctx, fixtures.UserID, mealID)
		require.NoError(t, err)

		// Verify meal is soft-deleted
		result, err := testRepo.GetMealByID(ctx, fixtures.UserID, mealID)
		assert.Error(t, err, "Should not find deleted meal")
		assert.Nil(t, result)

		// Verify it doesn't appear in list
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 100,
		}

		mealList, _, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)

		for _, meal := range mealList {
			assert.NotEqual(t, mealID, meal.ID, "Deleted meal should not appear in list")
		}
	})
}

// TestStructScanning verifies database columns map correctly to Go structs
func TestStructScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	// Create a meal with all fields populated
	meal := &meals.Meal{
		UserID:        fixtures.UserID,
		MealType:      meals.MealTypeSnack,
		ConsumedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Photos:        []string{"photo1.jpg", "photo2.jpg", "photo3.jpg"},
		Notes:         "Struct scanning test",
		TotalCalories: 250.5,
		TotalProteinG: 15.3,
		TotalCarbsG:   28.7,
		TotalFatG:     8.9,
		TotalFiberG:   4.2,
	}

	items := []meals.MealItem{
		{
			ID:       uuid.New(),
			Name:     "Test Item",
			Quantity: 125.5,
			Unit:     "g",
			Calories: 250.5,
			ProteinG: 15.3,
			CarbsG:   28.7,
			FatG:     8.9,
			FiberG:   4.2,
		},
	}

	result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
	require.NoError(t, err)
	fixtures.Meals = append(fixtures.Meals, result.ID)

	t.Run("meal struct fields scan correctly", func(t *testing.T) {
		// Retrieve and verify all fields
		retrieved, err := testRepo.GetMealByID(ctx, fixtures.UserID, result.ID)
		require.NoError(t, err)

		assert.Equal(t, result.ID, retrieved.ID)
		assert.Equal(t, fixtures.UserID, retrieved.UserID)
		assert.Equal(t, meals.MealTypeSnack, retrieved.MealType)
		assert.WithinDuration(t, meal.ConsumedAt, retrieved.ConsumedAt, time.Second)
		assert.Equal(t, meal.Photos, retrieved.Photos)
		assert.Equal(t, meal.Notes, retrieved.Notes)
		assert.InDelta(t, meal.TotalCalories, retrieved.TotalCalories, 0.01)
		assert.InDelta(t, meal.TotalProteinG, retrieved.TotalProteinG, 0.01)
		assert.InDelta(t, meal.TotalCarbsG, retrieved.TotalCarbsG, 0.01)
		assert.InDelta(t, meal.TotalFatG, retrieved.TotalFatG, 0.01)
		assert.InDelta(t, meal.TotalFiberG, retrieved.TotalFiberG, 0.01)
		assert.NotZero(t, retrieved.CreatedAt)
		assert.NotZero(t, retrieved.UpdatedAt)
		assert.Nil(t, retrieved.DeletedAt)
	})

	t.Run("meal item struct fields scan correctly", func(t *testing.T) {
		retrieved, err := testRepo.GetMealByID(ctx, fixtures.UserID, result.ID)
		require.NoError(t, err)
		require.Len(t, retrieved.Items, 1)

		item := retrieved.Items[0]
		assert.NotEqual(t, uuid.Nil, item.ID)
		assert.Equal(t, result.ID, item.MealID)
		assert.Equal(t, "Test Item", item.Name)
		assert.InDelta(t, 125.5, item.Quantity, 0.01)
		assert.Equal(t, "g", item.Unit)
		assert.InDelta(t, 250.5, item.Calories, 0.01)
		assert.InDelta(t, 15.3, item.ProteinG, 0.01)
		assert.InDelta(t, 28.7, item.CarbsG, 0.01)
		assert.InDelta(t, 8.9, item.FatG, 0.01)
		assert.InDelta(t, 4.2, item.FiberG, 0.01)
		assert.NotZero(t, item.CreatedAt)
	})

	t.Run("photos json array scans correctly", func(t *testing.T) {
		// Query directly to verify JSON handling
		var photosJSON []byte
		err := testDB.QueryRow(`
			SELECT photos FROM meals WHERE id = $1
		`, result.ID).Scan(&photosJSON)
		require.NoError(t, err)

		var photos []string
		err = json.Unmarshal(photosJSON, &photos)
		require.NoError(t, err)

		assert.Equal(t, []string{"photo1.jpg", "photo2.jpg", "photo3.jpg"}, photos)
	})
}

// TestDuplicateMeal verifies meal duplication functionality
func TestDuplicateMeal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	// Create original meal
	originalMeal := &meals.Meal{
		UserID:        fixtures.UserID,
		MealType:      meals.MealTypeBreakfast,
		ConsumedAt:    time.Now().UTC().Add(-24 * time.Hour),
		Photos:        []string{"original.jpg"},
		Notes:         "Original meal",
		TotalCalories: 400.0,
		TotalProteinG: 25.0,
		TotalCarbsG:   45.0,
		TotalFatG:     12.0,
		TotalFiberG:   6.0,
	}

	originalItems := []meals.MealItem{
		{
			ID:       uuid.New(),
			Name:     "Original Item 1",
			Quantity: 100.0,
			Unit:     "g",
			Calories: 200.0,
			ProteinG: 15.0,
			CarbsG:   20.0,
			FatG:     8.0,
			FiberG:   3.0,
		},
		{
			ID:       uuid.New(),
			Name:     "Original Item 2",
			Quantity: 150.0,
			Unit:     "g",
			Calories: 200.0,
			ProteinG: 10.0,
			CarbsG:   25.0,
			FatG:     4.0,
			FiberG:   3.0,
		},
	}

	original, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, originalMeal, originalItems)
	require.NoError(t, err)
	fixtures.Meals = append(fixtures.Meals, original.ID)

	// Duplicate the meal
	newConsumedAt := time.Now().UTC()
	newMealType := meals.MealTypeLunch

	duplicate, err := testRepo.DuplicateMeal(ctx, fixtures.UserID, original.ID, newConsumedAt, newMealType)
	require.NoError(t, err)
	require.NotNil(t, duplicate)
	fixtures.Meals = append(fixtures.Meals, duplicate.ID)

	// Verify duplicate is a new meal
	assert.NotEqual(t, original.ID, duplicate.ID, "Duplicate should have different ID")
	assert.Equal(t, newMealType, duplicate.MealType, "Duplicate should have new meal type")
	assert.WithinDuration(t, newConsumedAt, duplicate.ConsumedAt, time.Second)

	// Verify nutrition totals are copied
	assert.Equal(t, original.TotalCalories, duplicate.TotalCalories)
	assert.Equal(t, original.TotalProteinG, duplicate.TotalProteinG)
	assert.Equal(t, original.TotalCarbsG, duplicate.TotalCarbsG)
	assert.Equal(t, original.TotalFatG, duplicate.TotalFatG)
	assert.Equal(t, original.TotalFiberG, duplicate.TotalFiberG)

	// Verify items are copied
	assert.Len(t, duplicate.Items, len(original.Items))
	for i := range duplicate.Items {
		assert.NotEqual(t, original.Items[i].ID, duplicate.Items[i].ID, "Item IDs should be different")
		assert.Equal(t, duplicate.ID, duplicate.Items[i].MealID, "Items should reference new meal")
		assert.Equal(t, original.Items[i].Name, duplicate.Items[i].Name)
		assert.Equal(t, original.Items[i].Quantity, duplicate.Items[i].Quantity)
		assert.Equal(t, original.Items[i].Unit, duplicate.Items[i].Unit)
		assert.Equal(t, original.Items[i].Calories, duplicate.Items[i].Calories)
	}

	// Verify original meal is unchanged
	originalCheck, err := testRepo.GetMealByID(ctx, fixtures.UserID, original.ID)
	require.NoError(t, err)
	assert.Equal(t, meals.MealTypeBreakfast, originalCheck.MealType)
	assert.Equal(t, "Original meal", originalCheck.Notes)
}

// TestErrorCases verifies error handling
func TestErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("get non-existent meal returns error", func(t *testing.T) {
		nonExistentID := uuid.New()
		result, err := testRepo.GetMealByID(ctx, fixtures.UserID, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("delete non-existent meal returns error", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := testRepo.DeleteMeal(ctx, fixtures.UserID, nonExistentID)
		assert.Error(t, err)
	})

	t.Run("update non-existent meal returns error", func(t *testing.T) {
		nonExistentID := uuid.New()
		meal := &meals.Meal{
			ID:            nonExistentID,
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeSnack,
			ConsumedAt:    time.Now().UTC(),
			TotalCalories: 100.0,
		}
		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				MealID:   nonExistentID,
				Name:     "Item",
				Quantity: 100,
				Unit:     "g",
				Calories: 100,
			},
		}

		result, err := testRepo.UpdateMeal(ctx, fixtures.UserID, meal, items)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("cannot access other user's meal", func(t *testing.T) {
		// Create meal for user1
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeLunch,
			ConsumedAt:    time.Now().UTC(),
			TotalCalories: 300.0,
			TotalProteinG: 20.0,
			TotalCarbsG:   30.0,
			TotalFatG:     10.0,
			TotalFiberG:   5.0,
		}
		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Food",
				Quantity: 100,
				Unit:     "g",
				Calories: 300,
				ProteinG: 20,
				CarbsG:   30,
				FatG:     10,
				FiberG:   5,
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Try to access with different user ID
		otherUserID := uuid.New()
		retrieved, err := testRepo.GetMealByID(ctx, otherUserID, result.ID)
		assert.Error(t, err, "Should not access other user's meal")
		assert.Nil(t, retrieved)
	})
}

// TestConcurrentOperations verifies thread-safety
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	// Create multiple meals concurrently
	const concurrency = 5
	resultChan := make(chan uuid.UUID, concurrency)
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			meal := &meals.Meal{
				UserID:        fixtures.UserID,
				MealType:      meals.MealTypeSnack,
				ConsumedAt:    time.Now().UTC().Add(time.Duration(index) * time.Minute),
				Notes:         fmt.Sprintf("Concurrent meal %d", index),
				TotalCalories: float64(100 + index*10),
				TotalProteinG: float64(10 + index),
				TotalCarbsG:   float64(15 + index),
				TotalFatG:     float64(5 + index),
				TotalFiberG:   float64(2 + index),
			}

			items := []meals.MealItem{
				{
					ID:       uuid.New(),
					Name:     fmt.Sprintf("Item %d", index),
					Quantity: 100,
					Unit:     "g",
					Calories: meal.TotalCalories,
					ProteinG: meal.TotalProteinG,
					CarbsG:   meal.TotalCarbsG,
					FatG:     meal.TotalFatG,
					FiberG:   meal.TotalFiberG,
				},
			}

			result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
			if err != nil {
				errChan <- err
				return
			}

			resultChan <- result.ID
		}(i)
	}

	// Collect results
	createdMeals := []uuid.UUID{}
	for i := 0; i < concurrency; i++ {
		select {
		case mealID := <-resultChan:
			createdMeals = append(createdMeals, mealID)
			fixtures.Meals = append(fixtures.Meals, mealID)
		case err := <-errChan:
			t.Fatalf("Concurrent operation failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	assert.Len(t, createdMeals, concurrency, "All concurrent operations should succeed")

	// Verify all meals were created
	for _, mealID := range createdMeals {
		result, err := testRepo.GetMealByID(ctx, fixtures.UserID, mealID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}
