package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyticsFlow_Daily tests daily nutrition totals with goals
func TestAnalyticsFlow_Daily(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Set user goals
	setUserGoals(t, user, 2000, 150, 200, 70)

	// Create meals for today
	today := time.Now().Truncate(24 * time.Hour)
	createTestMealWithMacros(t, user, "breakfast", today.Add(8*time.Hour), 500, 30, 50, 15)
	createTestMealWithMacros(t, user, "lunch", today.Add(12*time.Hour), 700, 50, 70, 20)
	createTestMealWithMacros(t, user, "dinner", today.Add(18*time.Hour), 600, 40, 60, 25)

	t.Run("get daily totals", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/daily", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		// Check totals
		totals := result["totals"].(map[string]interface{})
		assert.InDelta(t, 1800.0, totals["calories"], 10.0)
		assert.InDelta(t, 120.0, totals["protein_g"], 5.0)
		assert.InDelta(t, 180.0, totals["carbs_g"], 5.0)
		assert.InDelta(t, 60.0, totals["fat_g"], 5.0)

		// Check goals
		goals := result["goals"].(map[string]interface{})
		assert.Equal(t, float64(2000), goals["daily_calories"])
		assert.Equal(t, float64(150), goals["protein_grams"])

		// Check progress
		progress := result["progress"].(map[string]interface{})
		assert.NotNil(t, progress["calories_percentage"])
		assert.NotNil(t, progress["protein_percentage"])
	})

	t.Run("get daily by date", func(t *testing.T) {
		yesterday := today.Add(-24 * time.Hour)
		createTestMealWithMacros(t, user, "breakfast", yesterday.Add(8*time.Hour), 400, 25, 45, 12)

		dateStr := yesterday.Format("2006-01-02")
		resp, err := testServer.MakeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/analytics/daily?date=%s", dateStr), nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		totals := result["totals"].(map[string]interface{})
		assert.InDelta(t, 400.0, totals["calories"], 10.0)
	})
}

// TestAnalyticsFlow_Weekly tests 7-day trends
func TestAnalyticsFlow_Weekly(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Set goals
	setUserGoals(t, user, 2000, 150, 200, 70)

	// Create meals for last 7 days
	today := time.Now().Truncate(24 * time.Hour)
	for i := 0; i < 7; i++ {
		date := today.Add(-time.Duration(i) * 24 * time.Hour)
		// Breakfast
		createTestMealWithMacros(t, user, "breakfast", date.Add(8*time.Hour), 500, 30, 50, 15)
		// Lunch
		createTestMealWithMacros(t, user, "lunch", date.Add(12*time.Hour), 700, 50, 70, 20)
		// Dinner
		createTestMealWithMacros(t, user, "dinner", date.Add(18*time.Hour), 600, 40, 60, 25)
	}

	t.Run("get weekly summary", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/weekly", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		// Check daily breakdown
		days := result["days"].([]interface{})
		assert.Len(t, days, 7)

		// Check averages
		averages := result["averages"].(map[string]interface{})
		assert.InDelta(t, 1800.0, averages["calories"], 50.0)
		assert.InDelta(t, 120.0, averages["protein_g"], 10.0)

		// Check adherence to goals
		adherence := result["adherence"].(map[string]interface{})
		assert.NotNil(t, adherence["calories_percentage"])
	})

	t.Run("get weekly trends", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/trends?period=7", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		trends := result["trends"].([]interface{})
		assert.GreaterOrEqual(t, len(trends), 7)
	})
}

// TestAnalyticsFlow_Progress tests goal tracking
func TestAnalyticsFlow_Progress(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Set goals
	setUserGoals(t, user, 2000, 150, 200, 70)

	t.Run("get progress toward goals", func(t *testing.T) {
		// Create meals totaling 75% of daily goal
		today := time.Now().Truncate(24 * time.Hour)
		createTestMealWithMacros(t, user, "breakfast", today.Add(8*time.Hour), 500, 40, 50, 15)
		createTestMealWithMacros(t, user, "lunch", today.Add(12*time.Hour), 700, 60, 70, 25)
		// Total: 1200 calories (60% of 2000)

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/progress", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		progress := result["progress"].(map[string]interface{})
		caloriesProgress := progress["calories"].(map[string]interface{})

		consumed := caloriesProgress["consumed"].(float64)
		goal := caloriesProgress["goal"].(float64)
		percentage := caloriesProgress["percentage"].(float64)

		assert.InDelta(t, 1200.0, consumed, 10.0)
		assert.Equal(t, 2000.0, goal)
		assert.InDelta(t, 60.0, percentage, 2.0)
	})

	t.Run("get macro breakdown", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/macros", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		macros := result["macros"].(map[string]interface{})
		assert.NotNil(t, macros["protein"])
		assert.NotNil(t, macros["carbs"])
		assert.NotNil(t, macros["fat"])

		// Check protein breakdown
		protein := macros["protein"].(map[string]interface{})
		assert.NotNil(t, protein["grams"])
		assert.NotNil(t, protein["calories"])
		assert.NotNil(t, protein["percentage_of_total"])
	})
}

// TestAnalyticsFlow_Caching tests cache performance
func TestAnalyticsFlow_Caching(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create data
	today := time.Now().Truncate(24 * time.Hour)
	for i := 0; i < 5; i++ {
		createTestMealWithMacros(t, user, "lunch", today.Add(-time.Duration(i)*24*time.Hour), 600, 40, 60, 20)
	}

	t.Run("cache daily analytics", func(t *testing.T) {
		// First request
		start1 := time.Now()
		resp1, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/daily", nil, user)
		require.NoError(t, err)
		defer resp1.Body.Close()
		duration1 := time.Since(start1)

		var result1 map[string]interface{}
		AssertResponse(t, resp1, http.StatusOK, &result1)

		// Second request (should be cached)
		start2 := time.Now()
		resp2, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/daily", nil, user)
		require.NoError(t, err)
		defer resp2.Body.Close()
		duration2 := time.Since(start2)

		var result2 map[string]interface{}
		AssertResponse(t, resp2, http.StatusOK, &result2)

		// Second request should be faster
		assert.LessOrEqual(t, duration2, duration1)
	})

	t.Run("cache invalidation on new meal", func(t *testing.T) {
		// Get cached analytics
		resp1, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/daily", nil, user)
		require.NoError(t, err)
		defer resp1.Body.Close()

		var result1 map[string]interface{}
		AssertResponse(t, resp1, http.StatusOK, &result1)

		totals1 := result1["totals"].(map[string]interface{})
		calories1 := totals1["calories"].(float64)

		// Add new meal (should invalidate cache)
		createTestMealWithMacros(t, user, "snack", time.Now(), 200, 10, 20, 5)

		// Get analytics again
		resp2, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/daily", nil, user)
		require.NoError(t, err)
		defer resp2.Body.Close()

		var result2 map[string]interface{}
		AssertResponse(t, resp2, http.StatusOK, &result2)

		totals2 := result2["totals"].(map[string]interface{})
		calories2 := totals2["calories"].(float64)

		// Should have more calories now
		assert.Greater(t, calories2, calories1)
	})
}

// TestAnalyticsFlow_Monthly tests monthly statistics
func TestAnalyticsFlow_Monthly(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create meals for last 30 days
	today := time.Now().Truncate(24 * time.Hour)
	for i := 0; i < 30; i++ {
		date := today.Add(-time.Duration(i) * 24 * time.Hour)
		calories := 1800.0 + float64(i%100) // Varying calories
		createTestMealWithMacros(t, user, "lunch", date.Add(12*time.Hour), int(calories), 120, 180, 60)
	}

	t.Run("get monthly summary", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/monthly", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		// Check averages
		averages := result["averages"].(map[string]interface{})
		assert.Greater(t, averages["calories"], 1800.0)
		assert.Less(t, averages["calories"], 1900.0)

		// Check totals
		totals := result["totals"].(map[string]interface{})
		assert.Greater(t, totals["calories"], 50000.0)
	})
}

// TestAnalyticsFlow_MealTypeBreakdown tests analytics by meal type
func TestAnalyticsFlow_MealTypeBreakdown(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	today := time.Now().Truncate(24 * time.Hour)

	// Create different meal types
	createTestMealWithMacros(t, user, "breakfast", today.Add(8*time.Hour), 500, 30, 50, 15)
	createTestMealWithMacros(t, user, "breakfast", today.Add(8*time.Hour+1*time.Minute), 300, 20, 30, 10)
	createTestMealWithMacros(t, user, "lunch", today.Add(12*time.Hour), 700, 50, 70, 20)
	createTestMealWithMacros(t, user, "dinner", today.Add(18*time.Hour), 600, 40, 60, 25)
	createTestMealWithMacros(t, user, "snack", today.Add(15*time.Hour), 200, 10, 20, 5)

	t.Run("get meal type breakdown", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/analytics/by-meal-type", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		breakdown := result["breakdown"].(map[string]interface{})

		breakfast := breakdown["breakfast"].(map[string]interface{})
		lunch := breakdown["lunch"].(map[string]interface{})
		dinner := breakdown["dinner"].(map[string]interface{})
		snack := breakdown["snack"].(map[string]interface{})

		// Breakfast should have ~800 calories (500 + 300)
		assert.InDelta(t, 800.0, breakfast["calories"], 10.0)
		assert.InDelta(t, 700.0, lunch["calories"], 10.0)
		assert.InDelta(t, 600.0, dinner["calories"], 10.0)
		assert.InDelta(t, 200.0, snack["calories"], 10.0)
	})
}

// Helper functions

func setUserGoals(t *testing.T, user *TestUser, calories, protein, carbs, fat int) {
	t.Helper()

	goalsReq := map[string]interface{}{
		"daily_calories": calories,
		"protein_grams":  protein,
		"carbs_grams":    carbs,
		"fat_grams":      fat,
	}

	resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", goalsReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusOK, &result)
}

func createTestMealWithMacros(t *testing.T, user *TestUser, mealType string, consumedAt time.Time, calories, protein, carbs, fat int) {
	t.Helper()

	confirmReq := map[string]interface{}{
		"meal_type":   mealType,
		"consumed_at": consumedAt.Format(time.RFC3339),
		"notes":       "Test meal for analytics",
		"items": []map[string]interface{}{
			{
				"name":      "Test Food",
				"quantity":  100.0,
				"unit":      "g",
				"calories":  float64(calories),
				"protein_g": float64(protein),
				"carbs_g":   float64(carbs),
				"fat_g":     float64(fat),
				"fiber_g":   5.0,
			},
		},
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)
}
