// Package constants provides centralized nutrition-related constants
// to ensure consistency across all calculation logic and eliminate
// hardcoded "magic numbers" throughout the codebase.
//
// These constants follow scientific standards (Atwater factors) and
// should NEVER be modified without thorough review and testing.
package constants

// MacroCalorieCoefficients define the standard Atwater factors for
// macronutrient energy conversion. These values are scientifically
// established and used worldwide in nutrition labeling.
//
// Reference: USDA Nutrient Database, FDA Nutrition Labeling Guidelines
const (
	// CaloriesPerGramProtein is the Atwater factor for protein energy (4 kcal/g).
	// This accounts for digestibility and metabolic losses.
	CaloriesPerGramProtein = 4.0

	// CaloriesPerGramCarbs is the Atwater factor for carbohydrate energy (4 kcal/g).
	// This is for digestible carbohydrates; fiber is handled separately.
	CaloriesPerGramCarbs = 4.0

	// CaloriesPerGramFat is the Atwater factor for fat energy (9 kcal/g).
	// All dietary fats provide approximately the same energy density.
	CaloriesPerGramFat = 9.0

	// CaloriesPerGramAlcohol is the Atwater factor for alcohol energy (7 kcal/g).
	// Currently unused but defined for future alcohol tracking support.
	// Alcohol (ethanol) provides energy but is not a macronutrient.
	CaloriesPerGramAlcohol = 7.0

	// CaloriesPerGramFiberMin is the minimum energy from fiber (0 kcal/g).
	// Insoluble fiber provides no energy.
	CaloriesPerGramFiberMin = 0.0

	// CaloriesPerGramFiberMax is the maximum energy from fiber (4 kcal/g).
	// Soluble fiber can be partially fermented by gut bacteria.
	// Most nutrition labels use 2 kcal/g as a compromise.
	CaloriesPerGramFiberMax = 4.0
)

// ValidationTolerances define acceptable variance ranges for data quality
// checks and validation rules.
const (
	// MacroCalorieTolerancePercent allows for ±10% variance between
	// calculated calories (from macros × coefficients) and stated calories.
	//
	// This tolerance accounts for:
	//   - Rounding in nutrition labels (usually to nearest 5 or 10 kcal)
	//   - Fiber energy availability variation (0-4 kcal/g depending on type)
	//   - Measurement precision limitations
	//   - Resistant starch variations
	//
	// Example: A food with stated 100 kcal can have calculated range 90-110 kcal
	MacroCalorieTolerancePercent = 0.10

	// MinimumCaloriesForValidation sets the threshold below which macro
	// validation is skipped. Very small portions (< 5 kcal) have higher
	// relative error and rounding effects, making validation unreliable.
	MinimumCaloriesForValidation = 5.0
)

// DisplayPrecision defines decimal places for user-facing nutrition values.
// These ensure consistent formatting across the application.
const (
	// CalorieDisplayDecimals - show calories with 1 decimal place.
	// Example: 123.5 kcal (not 123.456)
	CalorieDisplayDecimals = 1

	// MacroDisplayDecimals - show macros with 1 decimal place.
	// Example: 12.3g protein (not 12.34567g)
	MacroDisplayDecimals = 1

	// PercentDisplayDecimals - show percentages with 0 decimal places.
	// Example: 23% of daily value (not 23.4%)
	PercentDisplayDecimals = 0
)

// DatabasePrecision defines the decimal precision used in database columns.
// These match the DECIMAL type definitions in migrations.
const (
	// CaloriesPrecision: DECIMAL(7,1) supports up to 999999.9 kcal
	CaloriesIntegerDigits  = 6
	CaloriesDecimalDigits  = 1

	// MacroPrecision: DECIMAL(6,1) supports up to 99999.9 g
	MacroIntegerDigits     = 5
	MacroDecimalDigits     = 1
)

// CalculateMacroCalories computes expected calories from macronutrient values.
//
// This function implements the standard Atwater calculation:
//   Calories = (Protein × 4) + (Carbs × 4) + (Fat × 9)
//
// Fiber is excluded from this calculation as it provides variable energy
// (0-4 kcal/g) and is typically not counted in calorie calculations.
//
// Use MacroCalorieTolerancePercent to validate if actual calories match expected.
func CalculateMacroCalories(proteinG, carbsG, fatG float64) float64 {
	return (proteinG * CaloriesPerGramProtein) +
		(carbsG * CaloriesPerGramCarbs) +
		(fatG * CaloriesPerGramFat)
}

// IsWithinTolerance checks if the actual value is within the tolerance
// percentage of the expected value.
//
// Example:
//   expected := 100.0
//   actual := 105.0
//   tolerance := 0.10 // 10%
//   IsWithinTolerance(expected, actual, tolerance) // returns true
//   // because 105 is within [90, 110]
func IsWithinTolerance(expected, actual, tolerancePercent float64) bool {
	if expected < MinimumCaloriesForValidation {
		// Skip validation for very small values
		return true
	}

	lowerBound := expected * (1.0 - tolerancePercent)
	upperBound := expected * (1.0 + tolerancePercent)

	return actual >= lowerBound && actual <= upperBound
}

// ValidateMacroCalories checks if stated calories match calculated calories
// from macros within the acceptable tolerance range.
//
// Returns true if validation passes, false if calories don't match macros.
// Returns true (skip validation) if calories < MinimumCaloriesForValidation.
func ValidateMacroCalories(statedCalories, proteinG, carbsG, fatG float64) bool {
	calculatedCalories := CalculateMacroCalories(proteinG, carbsG, fatG)
	return IsWithinTolerance(calculatedCalories, statedCalories, MacroCalorieTolerancePercent)
}
