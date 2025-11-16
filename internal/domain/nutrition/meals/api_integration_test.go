package meals_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPI_CreateMeal verifies POST /api/v1/meals returns correct calculated totals.
func TestAPI_CreateMeal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("create meal returns auto-calculated totals", func(t *testing.T) {
		now := time.Now().UTC()
		// Prepare meal data (API would receive this via CreateMealWithItems)
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeBreakfast,
			ConsumedAt: now,
			Photos:     []string{"photo1.jpg"},
			Notes:      "API integration test",
		}

		items := []meals.MealItem{
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
				Name:     "Peanut Butter",
				Quantity: 2,
				Unit:     "tbsp",
				Calories: 190.0,
				ProteinG: 8.0,
				CarbsG:   7.0,
				FatG:     16.0,
				FiberG:   2.0,
			},
		}

		// Expected totals (sum of items)
		expectedCalories := 350.0
		expectedProtein := 16.0
		expectedCarbs := 35.0
		expectedFat := 18.0
		expectedFiber := 6.0

		// Call repository (simulates what API handler would do)
		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err, "API call should succeed")
		require.NotNil(t, result)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Verify response contains correct totals (NOT what client sent, but what DB calculated)
		assert.InDelta(t, expectedCalories, result.TotalCalories, 0.01, "Response should include DB-calculated calories")
		assert.InDelta(t, expectedProtein, result.TotalProteinG, 0.01, "Response should include DB-calculated protein")
		assert.InDelta(t, expectedCarbs, result.TotalCarbsG, 0.01, "Response should include DB-calculated carbs")
		assert.InDelta(t, expectedFat, result.TotalFatG, 0.01, "Response should include DB-calculated fat")
		assert.InDelta(t, expectedFiber, result.TotalFiberG, 0.01, "Response should include DB-calculated fiber")

		// Verify items are included in response
		assert.Len(t, result.Items, 2, "Response should include all items")
		assert.Equal(t, "Whole Wheat Toast", result.Items[0].Name)
		assert.Equal(t, "Peanut Butter", result.Items[1].Name)
	})

	t.Run("create meal ignores client-provided totals", func(t *testing.T) {
		// Client might send wrong totals (intentionally or by mistake)
		meal := &meals.Meal{
			UserID:        fixtures.UserID,
			MealType:      meals.MealTypeLunch,
			ConsumedAt:    time.Now().UTC(),
			Photos:        []string{},
			Notes:         "Client sent wrong totals",
			TotalCalories: 999.0, // Wrong value (should be ignored)
			TotalProteinG: 999.0, // Wrong value (should be ignored)
			TotalCarbsG:   999.0, // Wrong value (should be ignored)
			TotalFatG:     999.0, // Wrong value (should be ignored)
			TotalFiberG:   999.0, // Wrong value (should be ignored)
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Caesar Salad",
				Quantity: 1,
				Unit:     "bowl",
				Calories: 350.0,
				ProteinG: 25.0,
				CarbsG:   20.0,
				FatG:     18.0,
				FiberG:   3.0,
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Verify database triggers overrode client-provided totals
		assert.InDelta(t, 350.0, result.TotalCalories, 0.01, "Database should override wrong client totals")
		assert.InDelta(t, 25.0, result.TotalProteinG, 0.01, "Database should override wrong client totals")
		assert.InDelta(t, 20.0, result.TotalCarbsG, 0.01, "Database should override wrong client totals")
		assert.InDelta(t, 18.0, result.TotalFatG, 0.01, "Database should override wrong client totals")
		assert.InDelta(t, 3.0, result.TotalFiberG, 0.01, "Database should override wrong client totals")
	})
}

// TestAPI_UpdateMeal verifies PUT /api/v1/meals/:id recalculates totals.
func TestAPI_UpdateMeal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("update meal recalculates totals from new items", func(t *testing.T) {
		// Create initial meal
		originalMeal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeDinner,
			ConsumedAt: time.Now().UTC(),
			Photos:     []string{},
			Notes:      "Original meal",
		}

		originalItems := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Original Item",
				Quantity: 100,
				Unit:     "g",
				Calories: 200.0,
				ProteinG: 15.0,
				CarbsG:   20.0,
				FatG:     5.0,
				FiberG:   3.0,
			},
		}

		created, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, originalMeal, originalItems)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, created.ID)

		// Verify initial totals
		assert.InDelta(t, 200.0, created.TotalCalories, 0.01)
		assert.InDelta(t, 15.0, created.TotalProteinG, 0.01)

		// Update meal with different items (API PUT request)
		updatedMeal := &meals.Meal{
			ID:         created.ID,
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeDinner,
			ConsumedAt: created.ConsumedAt,
			Photos:     []string{"updated.jpg"},
			Notes:      "Updated meal",
		}

		updatedItems := []meals.MealItem{
			{
				ID:       uuid.New(),
				MealID:   created.ID,
				Name:     "New Item 1",
				Quantity: 150,
				Unit:     "g",
				Calories: 180.0,
				ProteinG: 20.0,
				CarbsG:   15.0,
				FatG:     6.0,
				FiberG:   2.0,
			},
			{
				ID:       uuid.New(),
				MealID:   created.ID,
				Name:     "New Item 2",
				Quantity: 100,
				Unit:     "g",
				Calories: 120.0,
				ProteinG: 10.0,
				CarbsG:   12.0,
				FatG:     4.0,
				FiberG:   1.0,
			},
		}

		expectedCalories := 300.0 // 180 + 120
		expectedProtein := 30.0   // 20 + 10

		result, err := testRepo.UpdateMeal(ctx, fixtures.UserID, updatedMeal, updatedItems)
		require.NoError(t, err)

		// Verify totals reflect NEW items, not original
		assert.InDelta(t, expectedCalories, result.TotalCalories, 0.01, "Totals should be recalculated from new items")
		assert.InDelta(t, expectedProtein, result.TotalProteinG, 0.01, "Totals should be recalculated from new items")
		assert.Len(t, result.Items, 2, "Should have new items count")
		assert.Equal(t, "New Item 1", result.Items[0].Name)
		assert.Equal(t, "New Item 2", result.Items[1].Name)
	})

	t.Run("update meal with fewer items adjusts totals down", func(t *testing.T) {
		// Create meal with 3 items
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeSnack,
			ConsumedAt: time.Now().UTC(),
			Photos:     []string{},
			Notes:      "Three items",
		}

		items := []meals.MealItem{
			{ID: uuid.New(), Name: "Item 1", Quantity: 100, Unit: "g", Calories: 100.0, ProteinG: 5.0, CarbsG: 10.0, FatG: 3.0, FiberG: 1.0},
			{ID: uuid.New(), Name: "Item 2", Quantity: 100, Unit: "g", Calories: 100.0, ProteinG: 5.0, CarbsG: 10.0, FatG: 3.0, FiberG: 1.0},
			{ID: uuid.New(), Name: "Item 3", Quantity: 100, Unit: "g", Calories: 100.0, ProteinG: 5.0, CarbsG: 10.0, FatG: 3.0, FiberG: 1.0},
		}

		created, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, created.ID)

		// Verify initial: 300 calories, 15g protein
		assert.InDelta(t, 300.0, created.TotalCalories, 0.01)
		assert.InDelta(t, 15.0, created.TotalProteinG, 0.01)

		// Update to only 1 item
		updatedMeal := &meals.Meal{
			ID:         created.ID,
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeSnack,
			ConsumedAt: created.ConsumedAt,
			Photos:     []string{},
			Notes:      "Reduced to one item",
		}

		updatedItems := []meals.MealItem{
			{ID: uuid.New(), MealID: created.ID, Name: "Single Item", Quantity: 100, Unit: "g", Calories: 150.0, ProteinG: 8.0, CarbsG: 15.0, FatG: 5.0, FiberG: 2.0},
		}

		result, err := testRepo.UpdateMeal(ctx, fixtures.UserID, updatedMeal, updatedItems)
		require.NoError(t, err)

		// Verify totals decreased to match single item
		assert.InDelta(t, 150.0, result.TotalCalories, 0.01, "Totals should decrease with fewer items")
		assert.InDelta(t, 8.0, result.TotalProteinG, 0.01, "Totals should decrease with fewer items")
		assert.Len(t, result.Items, 1)
	})
}

// TestAPI_GetMeal verifies GET /api/v1/meals/:id returns database totals.
func TestAPI_GetMeal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("get meal returns database-calculated totals", func(t *testing.T) {
		// Create meal
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeBreakfast,
			ConsumedAt: time.Now().UTC(),
			Photos:     []string{"test.jpg"},
			Notes:      "Get test",
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Greek Yogurt",
				Quantity: 200,
				Unit:     "g",
				Calories: 130.0,
				ProteinG: 20.0,
				CarbsG:   7.0,
				FatG:     4.0,
				FiberG:   0.0,
			},
			{
				ID:       uuid.New(),
				Name:     "Blueberries",
				Quantity: 100,
				Unit:     "g",
				Calories: 57.0,
				ProteinG: 0.7,
				CarbsG:   14.5,
				FatG:     0.3,
				FiberG:   2.4,
			},
		}

		created, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, created.ID)

		// Simulate API GET request
		retrieved, err := testRepo.GetMealByID(ctx, fixtures.UserID, created.ID)
		require.NoError(t, err)

		// Verify response has exact database values
		assert.Equal(t, created.ID, retrieved.ID)
		assert.InDelta(t, 187.0, retrieved.TotalCalories, 0.01, "GET should return exact DB totals")
		assert.InDelta(t, 20.7, retrieved.TotalProteinG, 0.01, "GET should return exact DB totals")
		assert.InDelta(t, 21.5, retrieved.TotalCarbsG, 0.01, "GET should return exact DB totals")
		assert.InDelta(t, 4.3, retrieved.TotalFatG, 0.01, "GET should return exact DB totals")
		assert.InDelta(t, 2.4, retrieved.TotalFiberG, 0.01, "GET should return exact DB totals")
		assert.Len(t, retrieved.Items, 2)
	})

	t.Run("get meal after manual item insertion still shows correct totals", func(t *testing.T) {
		// Create meal via repository
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeLunch,
			ConsumedAt: time.Now().UTC(),
			Photos:     []string{},
			Notes:      "Manual insert test",
		}

		items := []meals.MealItem{
			{ID: uuid.New(), Name: "Initial Item", Quantity: 100, Unit: "g", Calories: 150.0, ProteinG: 10.0, CarbsG: 15.0, FatG: 5.0, FiberG: 2.0},
		}

		created, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, created.ID)

		// Manually insert another item directly to database (simulates data migration or admin action)
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES ($1, $2, 'Manually Added', 50, 'g', 80.0, 5.0, 8.0, 2.0, 1.0)
		`, uuid.New(), created.ID)
		require.NoError(t, err)

		// GET meal via API
		retrieved, err := testRepo.GetMealByID(ctx, fixtures.UserID, created.ID)
		require.NoError(t, err)

		// Verify totals include manually inserted item
		assert.InDelta(t, 230.0, retrieved.TotalCalories, 0.01, "Totals should include manually added item")
		assert.InDelta(t, 15.0, retrieved.TotalProteinG, 0.01, "Totals should include manually added item")
		assert.Len(t, retrieved.Items, 2, "Items should include manually added item")
	})
}

// TestAPI_ListMeals verifies GET /api/v1/meals returns correct totals.
func TestAPI_ListMeals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("list meals shows correct totals for all meals", func(t *testing.T) {
		// Create multiple meals with different totals
		mealsData := []struct {
			mealType meals.MealType
			items    []meals.MealItem
			expected float64 // expected total calories
		}{
			{
				mealType: meals.MealTypeBreakfast,
				items: []meals.MealItem{
					{ID: uuid.New(), Name: "Eggs", Quantity: 2, Unit: "large", Calories: 140.0, ProteinG: 12.0, CarbsG: 1.0, FatG: 10.0, FiberG: 0.0},
				},
				expected: 140.0,
			},
			{
				mealType: meals.MealTypeLunch,
				items: []meals.MealItem{
					{ID: uuid.New(), Name: "Chicken", Quantity: 150, Unit: "g", Calories: 248.0, ProteinG: 46.5, CarbsG: 0.0, FatG: 5.5, FiberG: 0.0},
					{ID: uuid.New(), Name: "Rice", Quantity: 100, Unit: "g", Calories: 130.0, ProteinG: 2.7, CarbsG: 28.0, FatG: 0.3, FiberG: 0.4},
				},
				expected: 378.0,
			},
			{
				mealType: meals.MealTypeSnack,
				items: []meals.MealItem{
					{ID: uuid.New(), Name: "Apple", Quantity: 1, Unit: "medium", Calories: 95.0, ProteinG: 0.5, CarbsG: 25.0, FatG: 0.3, FiberG: 4.4},
				},
				expected: 95.0,
			},
		}

		createdIDs := []uuid.UUID{}

		for _, data := range mealsData {
			meal := &meals.Meal{
				UserID:     fixtures.UserID,
				MealType:   data.mealType,
				ConsumedAt: time.Now().UTC(),
				Photos:     []string{},
				Notes:      "List test",
			}

			result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, data.items)
			require.NoError(t, err)
			createdIDs = append(createdIDs, result.ID)
			fixtures.Meals = append(fixtures.Meals, result.ID)
		}

		// Simulate API GET /api/v1/meals
		filters := meals.ListMealFilters{
			Page:  1,
			Limit: 10,
		}

		mealList, total, err := testRepo.ListMealsByUser(ctx, fixtures.UserID, filters)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3, "Should find at least 3 meals")

		// Verify each meal in list has correct totals
		for i, expectedData := range mealsData {
			found := false
			for _, listedMeal := range mealList {
				if listedMeal.ID == createdIDs[i] {
					found = true
					assert.InDelta(t, expectedData.expected, listedMeal.TotalCalories, 0.01,
						"Meal %s should have correct total in list", expectedData.mealType)
					assert.Equal(t, len(expectedData.items), listedMeal.ItemCount,
						"Meal should show correct item count")
					break
				}
			}
			assert.True(t, found, "Meal %s should appear in list", expectedData.mealType)
		}
	})
}

// TestAPI_DailyAnalytics verifies GET /api/v1/analytics/daily aggregates correctly.
//
// Note: This test verifies the meals layer provides correct data for analytics.
// The analytics service itself is tested separately.
func TestAPI_DailyAnalytics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("daily totals aggregate from all meals", func(t *testing.T) {
		today := time.Now().UTC().Truncate(24 * time.Hour)

		// Create 3 meals for today
		mealTestData := []struct {
			mealType meals.MealType
			calories float64
			protein  float64
		}{
			{meals.MealTypeBreakfast, 400.0, 25.0},
			{meals.MealTypeLunch, 600.0, 45.0},
			{meals.MealTypeDinner, 700.0, 50.0},
		}

		for _, data := range mealTestData {
			mealData := &meals.Meal{
				UserID:     fixtures.UserID,
				MealType:   data.mealType,
				ConsumedAt: today.Add(8 * time.Hour), // 8 AM
				Photos:     []string{},
				Notes:      "Analytics test",
			}

			items := []meals.MealItem{
				{
					ID:       uuid.New(),
					Name:     "Test Item",
					Quantity: 100,
					Unit:     "g",
					Calories: data.calories,
					ProteinG: data.protein,
					CarbsG:   50.0,
					FatG:     20.0,
					FiberG:   5.0,
				},
			}

			result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, mealData, items)
			require.NoError(t, err)
			fixtures.Meals = append(fixtures.Meals, result.ID)
		}

		// Query daily totals (simulates what analytics endpoint would do)
		var dailyTotals struct {
			TotalCalories float64 `db:"total_calories"`
			TotalProtein  float64 `db:"total_protein"`
		}

		err := testDB.GetContext(ctx, &dailyTotals, `
			SELECT
				COALESCE(SUM(total_calories), 0) as total_calories,
				COALESCE(SUM(total_protein_g), 0) as total_protein
			FROM meals
			WHERE user_id = $1
				AND DATE(consumed_at) = DATE($2)
				AND deleted_at IS NULL
		`, fixtures.UserID, today)
		require.NoError(t, err)

		// Verify aggregated totals
		expectedCalories := 1700.0 // 400 + 600 + 700
		expectedProtein := 120.0   // 25 + 45 + 50

		assert.InDelta(t, expectedCalories, dailyTotals.TotalCalories, 0.01,
			"Daily totals should sum all meal totals")
		assert.InDelta(t, expectedProtein, dailyTotals.TotalProtein, 0.01,
			"Daily totals should sum all meal totals")
	})

	t.Run("analytics excludes deleted meals", func(t *testing.T) {
		today := time.Now().UTC().Truncate(24 * time.Hour)

		// Create meal
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeSnack,
			ConsumedAt: today,
			Photos:     []string{},
			Notes:      "To be deleted",
		}

		items := []meals.MealItem{
			{ID: uuid.New(), Name: "Snack", Quantity: 50, Unit: "g", Calories: 200.0, ProteinG: 10.0, CarbsG: 20.0, FatG: 8.0, FiberG: 2.0},
		}

		created, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, created.ID)

		// Get initial count
		var beforeCount int
		err = testDB.GetContext(ctx, &beforeCount, `
			SELECT COUNT(*) FROM meals
			WHERE user_id = $1 AND DATE(consumed_at) = DATE($2) AND deleted_at IS NULL
		`, fixtures.UserID, today)
		require.NoError(t, err)

		// Delete meal
		err = testRepo.DeleteMeal(ctx, fixtures.UserID, created.ID)
		require.NoError(t, err)

		// Get after count
		var afterCount int
		err = testDB.GetContext(ctx, &afterCount, `
			SELECT COUNT(*) FROM meals
			WHERE user_id = $1 AND DATE(consumed_at) = DATE($2) AND deleted_at IS NULL
		`, fixtures.UserID, today)
		require.NoError(t, err)

		assert.Equal(t, beforeCount-1, afterCount, "Deleted meals should not appear in analytics")
	})
}
