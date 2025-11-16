package meals_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseTriggers_InsertItems verifies database triggers auto-calculate
// meal totals when meal_items are inserted.
func TestDatabaseTriggers_InsertItems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("single item insert triggers total calculation", func(t *testing.T) {
		// Create meal with zero totals
		mealID := uuid.New()
		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'breakfast', NOW(), '[]', 'Trigger test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert a single item (triggers should fire)
		itemID := uuid.New()
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES ($1, $2, 'Banana', 1, 'medium', 105.0, 1.3, 27.0, 0.4, 3.1)
		`, itemID, mealID)
		require.NoError(t, err)

		// Verify meal totals were auto-calculated by triggers
		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.Equal(t, 105.0, meal.TotalCalories, "Calories should match item")
		assert.Equal(t, 1.3, meal.TotalProteinG, "Protein should match item")
		assert.Equal(t, 27.0, meal.TotalCarbsG, "Carbs should match item")
		assert.Equal(t, 0.4, meal.TotalFatG, "Fat should match item")
		assert.Equal(t, 3.1, meal.TotalFiberG, "Fiber should match item")
	})

	t.Run("multiple items insert triggers sum calculation", func(t *testing.T) {
		// Create meal with zero totals
		mealID := uuid.New()
		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'lunch', NOW(), '[]', 'Multi-item test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert multiple items
		items := []struct {
			name     string
			calories float64
			protein  float64
			carbs    float64
			fat      float64
			fiber    float64
		}{
			{"Grilled Chicken", 165.0, 31.0, 0.0, 3.6, 0.0},
			{"Brown Rice", 112.0, 2.3, 23.5, 0.9, 1.8},
			{"Broccoli", 31.0, 2.5, 6.0, 0.4, 2.4},
		}

		expectedTotals := struct {
			calories float64
			protein  float64
			carbs    float64
			fat      float64
			fiber    float64
		}{
			calories: 308.0,
			protein:  35.8,
			carbs:    29.5,
			fat:      4.9,
			fiber:    4.2,
		}

		for _, item := range items {
			_, err := testDB.ExecContext(ctx, `
				INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
				VALUES ($1, $2, $3, 100, 'g', $4, $5, $6, $7, $8)
			`, uuid.New(), mealID, item.name, item.calories, item.protein, item.carbs, item.fat, item.fiber)
			require.NoError(t, err)
		}

		// Verify meal totals equal sum of all items
		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, expectedTotals.calories, meal.TotalCalories, 0.01, "Total calories should equal sum")
		assert.InDelta(t, expectedTotals.protein, meal.TotalProteinG, 0.01, "Total protein should equal sum")
		assert.InDelta(t, expectedTotals.carbs, meal.TotalCarbsG, 0.01, "Total carbs should equal sum")
		assert.InDelta(t, expectedTotals.fat, meal.TotalFatG, 0.01, "Total fat should equal sum")
		assert.InDelta(t, expectedTotals.fiber, meal.TotalFiberG, 0.01, "Total fiber should equal sum")
	})

	t.Run("empty meal has zero totals", func(t *testing.T) {
		// Create meal with no items
		mealID := uuid.New()
		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'snack', NOW(), '[]', 'Empty meal',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Verify totals remain zero (COALESCE handles NULL from SUM)
		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.Equal(t, 0.0, meal.TotalCalories)
		assert.Equal(t, 0.0, meal.TotalProteinG)
		assert.Equal(t, 0.0, meal.TotalCarbsG)
		assert.Equal(t, 0.0, meal.TotalFatG)
		assert.Equal(t, 0.0, meal.TotalFiberG)
	})
}

// TestDatabaseTriggers_UpdateItems verifies database triggers recalculate
// meal totals when meal_items are updated.
func TestDatabaseTriggers_UpdateItems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("updating item recalculates meal totals", func(t *testing.T) {
		// Create meal with initial item
		mealID := uuid.New()
		itemID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'dinner', NOW(), '[]', 'Update test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert initial item
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES ($1, $2, 'Salmon', 100, 'g', 208.0, 20.0, 0.0, 13.0, 0.0)
		`, itemID, mealID)
		require.NoError(t, err)

		// Verify initial totals
		var beforeMeal meals.Meal
		err = testDB.GetContext(ctx, &beforeMeal, `
			SELECT total_calories, total_protein_g FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)
		assert.Equal(t, 208.0, beforeMeal.TotalCalories)
		assert.Equal(t, 20.0, beforeMeal.TotalProteinG)

		// Update item (triggers should fire)
		_, err = testDB.ExecContext(ctx, `
			UPDATE meal_items
			SET calories = 312.0, protein = 30.0, quantity = 150
			WHERE id = $1
		`, itemID)
		require.NoError(t, err)

		// Verify updated totals
		var afterMeal meals.Meal
		err = testDB.GetContext(ctx, &afterMeal, `
			SELECT total_calories, total_protein_g FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.Equal(t, 312.0, afterMeal.TotalCalories, "Totals should reflect updated values")
		assert.Equal(t, 30.0, afterMeal.TotalProteinG, "Totals should reflect updated values")
	})

	t.Run("updating one item preserves other items in sum", func(t *testing.T) {
		// Create meal with two items
		mealID := uuid.New()
		item1ID := uuid.New()
		item2ID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'breakfast', NOW(), '[]', 'Multi-update test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert two items
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES
				($1, $2, 'Oatmeal', 50, 'g', 190.0, 6.8, 32.0, 3.6, 5.0),
				($3, $2, 'Milk', 200, 'ml', 100.0, 6.6, 9.4, 3.6, 0.0)
		`, item1ID, mealID, item2ID)
		require.NoError(t, err)

		// Verify initial sum: 290 calories, 13.4g protein
		var beforeMeal meals.Meal
		err = testDB.GetContext(ctx, &beforeMeal, `
			SELECT total_calories, total_protein_g FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)
		assert.InDelta(t, 290.0, beforeMeal.TotalCalories, 0.01)
		assert.InDelta(t, 13.4, beforeMeal.TotalProteinG, 0.01)

		// Update only item2 (double the milk)
		_, err = testDB.ExecContext(ctx, `
			UPDATE meal_items
			SET calories = 200.0, protein = 13.2, quantity = 400
			WHERE id = $1
		`, item2ID)
		require.NoError(t, err)

		// Verify new sum: 190 + 200 = 390 calories, 6.8 + 13.2 = 20.0g protein
		var afterMeal meals.Meal
		err = testDB.GetContext(ctx, &afterMeal, `
			SELECT total_calories, total_protein_g FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, 390.0, afterMeal.TotalCalories, 0.01, "Should include unchanged item1 + updated item2")
		assert.InDelta(t, 20.0, afterMeal.TotalProteinG, 0.01, "Should include unchanged item1 + updated item2")
	})
}

// TestDatabaseTriggers_DeleteItems verifies database triggers recalculate
// meal totals when meal_items are deleted.
func TestDatabaseTriggers_DeleteItems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("deleting item decreases meal totals", func(t *testing.T) {
		// Create meal with two items
		mealID := uuid.New()
		item1ID := uuid.New()
		item2ID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'lunch', NOW(), '[]', 'Delete test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert two items
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES
				($1, $2, 'Turkey Sandwich', 1, 'sandwich', 350.0, 28.0, 35.0, 10.0, 3.0),
				($3, $2, 'Apple', 1, 'medium', 95.0, 0.5, 25.0, 0.3, 4.4)
		`, item1ID, mealID, item2ID)
		require.NoError(t, err)

		// Verify initial sum
		var beforeMeal meals.Meal
		err = testDB.GetContext(ctx, &beforeMeal, `
			SELECT total_calories, total_protein_g, total_carbs_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)
		assert.InDelta(t, 445.0, beforeMeal.TotalCalories, 0.01)
		assert.InDelta(t, 28.5, beforeMeal.TotalProteinG, 0.01)
		assert.InDelta(t, 60.0, beforeMeal.TotalCarbsG, 0.01)

		// Delete apple (triggers should fire)
		_, err = testDB.ExecContext(ctx, `DELETE FROM meal_items WHERE id = $1`, item2ID)
		require.NoError(t, err)

		// Verify totals decreased by apple's values
		var afterMeal meals.Meal
		err = testDB.GetContext(ctx, &afterMeal, `
			SELECT total_calories, total_protein_g, total_carbs_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, 350.0, afterMeal.TotalCalories, 0.01, "Should only have sandwich calories")
		assert.InDelta(t, 28.0, afterMeal.TotalProteinG, 0.01, "Should only have sandwich protein")
		assert.InDelta(t, 35.0, afterMeal.TotalCarbsG, 0.01, "Should only have sandwich carbs")
	})

	t.Run("deleting all items sets totals to zero", func(t *testing.T) {
		// Create meal with items
		mealID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'snack', NOW(), '[]', 'Delete all test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert items
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES
				($1, $2, 'Almonds', 28, 'g', 164.0, 6.0, 6.0, 14.0, 3.5),
				($3, $2, 'Dark Chocolate', 20, 'g', 110.0, 1.5, 13.0, 7.0, 2.0)
		`, uuid.New(), mealID, uuid.New())
		require.NoError(t, err)

		// Verify items were added
		var beforeMeal meals.Meal
		err = testDB.GetContext(ctx, &beforeMeal, `
			SELECT total_calories FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)
		assert.InDelta(t, 274.0, beforeMeal.TotalCalories, 0.01)

		// Delete all items (triggers should fire)
		_, err = testDB.ExecContext(ctx, `DELETE FROM meal_items WHERE meal_id = $1`, mealID)
		require.NoError(t, err)

		// Verify totals are back to zero
		var afterMeal meals.Meal
		err = testDB.GetContext(ctx, &afterMeal, `
			SELECT total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.Equal(t, 0.0, afterMeal.TotalCalories, "All totals should be zero after deleting all items")
		assert.Equal(t, 0.0, afterMeal.TotalProteinG)
		assert.Equal(t, 0.0, afterMeal.TotalCarbsG)
		assert.Equal(t, 0.0, afterMeal.TotalFatG)
		assert.Equal(t, 0.0, afterMeal.TotalFiberG)
	})
}

// TestDatabaseTriggers_EdgeCases verifies triggers handle edge cases correctly.
func TestDatabaseTriggers_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("handles very small fractional values", func(t *testing.T) {
		mealID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'snack', NOW(), '[]', 'Fractional test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert item with tiny values (e.g., 1 cherry tomato)
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES ($1, $2, 'Cherry Tomato', 1, 'piece', 3.2, 0.2, 0.7, 0.0, 0.2)
		`, uuid.New(), mealID)
		require.NoError(t, err)

		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g, total_carbs_g, total_fiber_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, 3.2, meal.TotalCalories, 0.01)
		assert.InDelta(t, 0.2, meal.TotalProteinG, 0.01)
		assert.InDelta(t, 0.7, meal.TotalCarbsG, 0.01)
		assert.InDelta(t, 0.2, meal.TotalFiberG, 0.01)
	})

	t.Run("handles many items with precision", func(t *testing.T) {
		mealID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'dinner', NOW(), '[]', 'Many items test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert 10 items with small values
		expectedCalories := 0.0
		expectedProtein := 0.0

		for i := 0; i < 10; i++ {
			calories := float64(10 + i)
			protein := 1.0 + float64(i)*0.5

			_, err := testDB.ExecContext(ctx, `
				INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
				VALUES ($1, $2, $3, $4, 'g', $5, $6, 2.0, 1.0, 0.5)
			`, uuid.New(), mealID, fmt.Sprintf("Item %d", i+1), float64(10), calories, protein)
			require.NoError(t, err)

			expectedCalories += calories
			expectedProtein += protein
		}

		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, expectedCalories, meal.TotalCalories, 0.01, "Should accurately sum many items")
		assert.InDelta(t, expectedProtein, meal.TotalProteinG, 0.01, "Should accurately sum many items")
	})

	t.Run("handles zero-calorie items correctly", func(t *testing.T) {
		mealID := uuid.New()

		_, err := testDB.ExecContext(ctx, `
			INSERT INTO meals (id, user_id, meal_type, consumed_at, photos, notes,
				total_calories, total_protein_g, total_carbs_g, total_fat_g, total_fiber_g,
				created_at, updated_at)
			VALUES ($1, $2, 'snack', NOW(), '[]', 'Zero calorie test',
				0, 0, 0, 0, 0, NOW(), NOW())
		`, mealID, fixtures.UserID)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, mealID)

		// Insert zero-calorie items (e.g., water, black coffee)
		_, err = testDB.ExecContext(ctx, `
			INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein, carbs, fat, fiber)
			VALUES
				($1, $2, 'Water', 500, 'ml', 0.0, 0.0, 0.0, 0.0, 0.0),
				($3, $2, 'Black Coffee', 240, 'ml', 2.0, 0.3, 0.0, 0.0, 0.0)
		`, uuid.New(), mealID, uuid.New())
		require.NoError(t, err)

		var meal meals.Meal
		err = testDB.GetContext(ctx, &meal, `
			SELECT total_calories, total_protein_g
			FROM meals WHERE id = $1
		`, mealID)
		require.NoError(t, err)

		assert.InDelta(t, 2.0, meal.TotalCalories, 0.01, "Should handle zero-calorie items")
		assert.InDelta(t, 0.3, meal.TotalProteinG, 0.01, "Should handle zero-calorie items")
	})
}
