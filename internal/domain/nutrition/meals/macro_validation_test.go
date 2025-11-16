package meals_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/constants"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/meals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMacroMath_Calculation verifies calories = (protein × 4) + (carbs × 4) + (fat × 9).
func TestMacroMath_Calculation(t *testing.T) {
	tests := []struct {
		name     string
		protein  float64
		carbs    float64
		fat      float64
		expected float64
	}{
		{
			name:     "standard meal",
			protein:  25.0,  // 25 × 4 = 100
			carbs:    50.0,  // 50 × 4 = 200
			fat:      15.0,  // 15 × 9 = 135
			expected: 435.0, // Total: 435 kcal
		},
		{
			name:     "high protein",
			protein:  50.0,  // 50 × 4 = 200
			carbs:    10.0,  // 10 × 4 = 40
			fat:      5.0,   // 5 × 9 = 45
			expected: 285.0, // Total: 285 kcal
		},
		{
			name:     "high fat",
			protein:  10.0,  // 10 × 4 = 40
			carbs:    5.0,   // 5 × 4 = 20
			fat:      30.0,  // 30 × 9 = 270
			expected: 330.0, // Total: 330 kcal
		},
		{
			name:     "zero fat (pure protein and carbs)",
			protein:  20.0,  // 20 × 4 = 80
			carbs:    30.0,  // 30 × 4 = 120
			fat:      0.0,   // 0 × 9 = 0
			expected: 200.0, // Total: 200 kcal
		},
		{
			name:     "fractional values",
			protein:  12.5,  // 12.5 × 4 = 50
			carbs:    18.3,  // 18.3 × 4 = 73.2
			fat:      7.2,   // 7.2 × 9 = 64.8
			expected: 188.0, // Total: 188 kcal
		},
		{
			name:     "very small values",
			protein:  0.5,  // 0.5 × 4 = 2
			carbs:    1.0,  // 1.0 × 4 = 4
			fat:      0.3,  // 0.3 × 9 = 2.7
			expected: 8.7,  // Total: 8.7 kcal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculated := constants.CalculateMacroCalories(tt.protein, tt.carbs, tt.fat)
			assert.InDelta(t, tt.expected, calculated, 0.01,
				"Calculated calories should match (P×4)+(C×4)+(F×9)")
		})
	}
}

// TestMacroMath_Tolerance verifies ±10% tolerance for validation.
func TestMacroMath_Tolerance(t *testing.T) {
	tests := []struct {
		name       string
		expected   float64
		actual     float64
		tolerance  float64
		shouldPass bool
	}{
		{
			name:       "exact match",
			expected:   100.0,
			actual:     100.0,
			tolerance:  0.10, // ±10%
			shouldPass: true,
		},
		{
			name:       "within upper bound (109 vs 100 ±10%)",
			expected:   100.0,
			actual:     109.0,
			tolerance:  0.10,
			shouldPass: true,
		},
		{
			name:       "within lower bound (91 vs 100 ±10%)",
			expected:   100.0,
			actual:     91.0,
			tolerance:  0.10,
			shouldPass: true,
		},
		{
			name:       "at upper limit (110 vs 100 ±10%)",
			expected:   100.0,
			actual:     110.0,
			tolerance:  0.10,
			shouldPass: true,
		},
		{
			name:       "at lower limit (90 vs 100 ±10%)",
			expected:   100.0,
			actual:     90.0,
			tolerance:  0.10,
			shouldPass: true,
		},
		{
			name:       "exceeds upper bound (111 vs 100 ±10%)",
			expected:   100.0,
			actual:     111.0,
			tolerance:  0.10,
			shouldPass: false,
		},
		{
			name:       "below lower bound (89 vs 100 ±10%)",
			expected:   100.0,
			actual:     89.0,
			tolerance:  0.10,
			shouldPass: false,
		},
		{
			name:       "small value skip validation (< 5 kcal)",
			expected:   3.0,
			actual:     10.0, // Way off, but should pass
			tolerance:  0.10,
			shouldPass: true, // Skipped due to MinimumCaloriesForValidation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constants.IsWithinTolerance(tt.expected, tt.actual, tt.tolerance)
			assert.Equal(t, tt.shouldPass, result,
				"Tolerance check should match expected result")
		})
	}
}

// TestMacroMath_ValidateMacroCalories verifies the combined validation function.
func TestMacroMath_ValidateMacroCalories(t *testing.T) {
	tests := []struct {
		name            string
		statedCalories  float64
		protein         float64
		carbs           float64
		fat             float64
		shouldValidate  bool
		description     string
	}{
		{
			name:            "perfect match",
			statedCalories:  200.0,
			protein:         10.0, // 40
			carbs:           30.0, // 120
			fat:             4.44, // ~40
			shouldValidate:  true,
			description:     "Stated calories exactly match calculated",
		},
		{
			name:            "within tolerance (+5%)",
			statedCalories:  210.0,
			protein:         10.0, // 40
			carbs:           30.0, // 120
			fat:             4.44, // ~40
			// Calculated: 200, Stated: 210 → 5% over (within ±10%)
			shouldValidate: true,
			description:    "Stated calories 5% higher (acceptable)",
		},
		{
			name:            "within tolerance (-5%)",
			statedCalories:  190.0,
			protein:         10.0, // 40
			carbs:           30.0, // 120
			fat:             4.44, // ~40
			// Calculated: 200, Stated: 190 → 5% under (within ±10%)
			shouldValidate: true,
			description:    "Stated calories 5% lower (acceptable)",
		},
		{
			name:            "exceeds tolerance (+15%)",
			statedCalories:  230.0,
			protein:         10.0, // 40
			carbs:           30.0, // 120
			fat:             4.44, // ~40
			// Calculated: 200, Stated: 230 → 15% over (exceeds ±10%)
			shouldValidate: false,
			description:    "Stated calories too high (invalid)",
		},
		{
			name:            "below tolerance (-15%)",
			statedCalories:  170.0,
			protein:         10.0, // 40
			carbs:           30.0, // 120
			fat:             4.44, // ~40
			// Calculated: 200, Stated: 170 → 15% under (exceeds ±10%)
			shouldValidate: false,
			description:    "Stated calories too low (invalid)",
		},
		{
			name:            "very small portion skips validation",
			statedCalories:  2.0,
			protein:         0.1,
			carbs:           0.2,
			fat:             0.1,
			shouldValidate:  true, // < 5 kcal threshold
			description:     "Very small portions skip validation",
		},
		{
			name:            "rounding on nutrition label (acceptable)",
			statedCalories:  100.0,
			protein:         8.0,  // 32
			carbs:           12.0, // 48
			fat:             2.0,  // 18
			// Calculated: 98, Stated: 100 → ~2% (acceptable rounding)
			shouldValidate: true,
			description:    "Nutrition label rounding is acceptable",
		},
		{
			name:            "high fiber food (soluble fiber energy)",
			statedCalories:  150.0,
			protein:         5.0,  // 20
			carbs:           30.0, // 120
			fat:             1.0,  // 9
			// Calculated: 149, Stated: 150
			// Note: Fiber not counted in base formula, but real food includes it
			shouldValidate: true,
			description:    "High fiber foods account for fermentable energy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constants.ValidateMacroCalories(tt.statedCalories, tt.protein, tt.carbs, tt.fat)
			assert.Equal(t, tt.shouldValidate, result, tt.description)

			// Also log calculated vs stated for debugging
			calculated := constants.CalculateMacroCalories(tt.protein, tt.carbs, tt.fat)
			t.Logf("Stated: %.1f, Calculated: %.1f, Diff: %.1f%%, Valid: %v",
				tt.statedCalories, calculated,
				((tt.statedCalories-calculated)/calculated)*100,
				result)
		})
	}
}

// TestMacroMath_DatabaseIntegration verifies nutrition math with real database data.
func TestMacroMath_DatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fixtures := setupTestFixtures(t)
	defer cleanupTestFixtures(t, fixtures)

	ctx := context.Background()

	t.Run("database totals validate against macro math", func(t *testing.T) {
		now := time.Now().UTC()
		// Create meal with items that have valid macro ratios
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeBreakfast,
			ConsumedAt: now,
			Photos:     []string{},
			Notes:      "Macro validation test",
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Chicken Breast",
				Quantity: 100,
				Unit:     "g",
				Calories: 165.0,
				ProteinG: 31.0, // 31 × 4 = 124
				CarbsG:   0.0,  // 0 × 4 = 0
				FatG:     3.6,  // 3.6 × 9 = 32.4
				FiberG:   0.0,
				// Calculated: 156.4, Stated: 165 → ~5% (within tolerance)
			},
			{
				ID:       uuid.New(),
				Name:     "Brown Rice",
				Quantity: 100,
				Unit:     "g",
				Calories: 112.0,
				ProteinG: 2.6,  // 2.6 × 4 = 10.4
				CarbsG:   23.5, // 23.5 × 4 = 94
				FatG:     0.9,  // 0.9 × 9 = 8.1
				FiberG:   1.8,
				// Calculated: 112.5, Stated: 112 → within tolerance
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// Verify each item's nutrition math
		for _, item := range result.Items {
			isValid := constants.ValidateMacroCalories(
				item.Calories,
				item.ProteinG,
				item.CarbsG,
				item.FatG,
			)

			assert.True(t, isValid,
				"Item '%s' should have valid macro-to-calorie ratio", item.Name)

			// Log details for transparency
			calculated := constants.CalculateMacroCalories(item.ProteinG, item.CarbsG, item.FatG)
			t.Logf("Item: %s, Stated: %.1f kcal, Calculated: %.1f kcal, Valid: %v",
				item.Name, item.Calories, calculated, isValid)
		}

		// Verify meal totals also validate
		mealValid := constants.ValidateMacroCalories(
			result.TotalCalories,
			result.TotalProteinG,
			result.TotalCarbsG,
			result.TotalFatG,
		)

		assert.True(t, mealValid, "Meal totals should validate against macro math")
	})

	t.Run("detect invalid nutrition data", func(t *testing.T) {
		now := time.Now().UTC()
		// Intentionally create item with wrong calories
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeLunch,
			ConsumedAt: now,
			Photos:     []string{},
			Notes:      "Invalid data test",
		}

		items := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Invalid Item",
				Quantity: 100,
				Unit:     "g",
				Calories: 1000.0, // WAY TOO HIGH
				ProteinG: 10.0,   // 40 kcal
				CarbsG:   10.0,   // 40 kcal
				FatG:     5.0,    // 45 kcal
				FiberG:   2.0,
				// Calculated: 125 kcal, Stated: 1000 kcal → 700% off (invalid!)
			},
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, items)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// This should fail validation
		item := result.Items[0]
		isValid := constants.ValidateMacroCalories(
			item.Calories,
			item.ProteinG,
			item.CarbsG,
			item.FatG,
		)

		assert.False(t, isValid,
			"Item with incorrect calories should fail validation")

		calculated := constants.CalculateMacroCalories(item.ProteinG, item.CarbsG, item.FatG)
		t.Logf("Invalid item detected: Stated %.1f kcal, Calculated %.1f kcal (%.0f%% error)",
			item.Calories, calculated, ((item.Calories-calculated)/calculated)*100)
	})

	t.Run("real-world food examples validate correctly", func(t *testing.T) {
		// Test with actual USDA nutrition data
		realWorldItems := []meals.MealItem{
			{
				ID:       uuid.New(),
				Name:     "Banana (USDA)",
				Quantity: 118,
				Unit:     "g",
				Calories: 105.0,
				ProteinG: 1.3,  // 5.2 kcal
				CarbsG:   27.0, // 108 kcal
				FatG:     0.4,  // 3.6 kcal
				FiberG:   3.1,
				// Calculated: 116.8, Stated: 105 → ~10% under (fiber subtraction)
			},
			{
				ID:       uuid.New(),
				Name:     "Avocado (USDA)",
				Quantity: 201,
				Unit:     "g",
				Calories: 322.0,
				ProteinG: 4.0,  // 16 kcal
				CarbsG:   17.0, // 68 kcal
				FatG:     29.5, // 265.5 kcal
				FiberG:   13.5,
				// Calculated: 349.5, Stated: 322 → ~8% under (acceptable)
			},
			{
				ID:       uuid.New(),
				Name:     "Greek Yogurt (USDA)",
				Quantity: 200,
				Unit:     "g",
				Calories: 130.0,
				ProteinG: 20.0, // 80 kcal
				CarbsG:   7.0,  // 28 kcal
				FatG:     4.0,  // 36 kcal
				FiberG:   0.0,
				// Calculated: 144, Stated: 130 → ~10% under (acceptable)
			},
		}

		now := time.Now().UTC()
		meal := &meals.Meal{
			UserID:     fixtures.UserID,
			MealType:   meals.MealTypeSnack,
			ConsumedAt: now,
			Photos:     []string{},
			Notes:      "Real-world validation",
		}

		result, err := testRepo.CreateMealWithItems(ctx, fixtures.UserID, meal, realWorldItems)
		require.NoError(t, err)
		fixtures.Meals = append(fixtures.Meals, result.ID)

		// All real-world items should validate (±10% tolerance)
		for _, item := range result.Items {
			isValid := constants.ValidateMacroCalories(
				item.Calories,
				item.ProteinG,
				item.CarbsG,
				item.FatG,
			)

			calculated := constants.CalculateMacroCalories(item.ProteinG, item.CarbsG, item.FatG)
			percentDiff := ((item.Calories - calculated) / calculated) * 100

			t.Logf("Real-world item: %s, Stated: %.0f kcal, Calculated: %.0f kcal, Diff: %.1f%%, Valid: %v",
				item.Name, item.Calories, calculated, percentDiff, isValid)

			assert.True(t, isValid,
				"Real-world item '%s' should validate with USDA data", item.Name)
		}
	})
}

// TestMacroMath_EdgeCases verifies edge cases in nutrition validation.
func TestMacroMath_EdgeCases(t *testing.T) {
	t.Run("zero calories with zero macros", func(t *testing.T) {
		isValid := constants.ValidateMacroCalories(0.0, 0.0, 0.0, 0.0)
		assert.True(t, isValid, "Zero calories should validate with zero macros")
	})

	t.Run("negative values not allowed in real data", func(t *testing.T) {
		// While the math works, real nutrition data should never be negative
		calculated := constants.CalculateMacroCalories(-5.0, -10.0, -2.0)
		assert.Less(t, calculated, 0.0, "Negative macros yield negative calories")

		// Database constraints should prevent this, but validation still works
		isValid := constants.ValidateMacroCalories(-50.0, -5.0, -10.0, -2.0)
		assert.True(t, isValid, "Math still works with negatives (DB should prevent)")
	})

	t.Run("very large values", func(t *testing.T) {
		// Extreme but possible (e.g., full day totals)
		protein := 200.0  // 800 kcal
		carbs := 300.0    // 1200 kcal
		fat := 100.0      // 900 kcal
		expected := 2900.0

		calculated := constants.CalculateMacroCalories(protein, carbs, fat)
		assert.InDelta(t, expected, calculated, 0.01)

		isValid := constants.ValidateMacroCalories(2900.0, protein, carbs, fat)
		assert.True(t, isValid)
	})

	t.Run("precision at database limits", func(t *testing.T) {
		// Database stores DECIMAL(7,1) for calories, DECIMAL(6,1) for macros
		// Test boundary values
		maxMacro := 99999.9

		calculated := constants.CalculateMacroCalories(maxMacro, maxMacro, maxMacro)
		assert.Greater(t, calculated, 0.0, "Should handle max database values")
	})

	t.Run("alcohol calories not counted in base formula", func(t *testing.T) {
		// Note: Alcohol provides 7 kcal/g but is not included in macro calculation
		// This is expected - fiber and alcohol are handled separately
		protein := 0.0
		carbs := 0.0
		fat := 0.0

		calculated := constants.CalculateMacroCalories(protein, carbs, fat)
		assert.Equal(t, 0.0, calculated, "Base formula excludes alcohol calories")

		// If food has 70 kcal from 10g alcohol, validation would fail
		// This is correct - alcohol is not a macronutrient
		isValid := constants.ValidateMacroCalories(70.0, 0.0, 0.0, 0.0)
		assert.False(t, isValid, "Alcohol calories not in base macro formula")
	})
}
