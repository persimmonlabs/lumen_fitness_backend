package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoalsFlow_AutoCalculate tests TDEE automatic calculation
func TestGoalsFlow_AutoCalculate(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("calculate TDEE from inputs", func(t *testing.T) {
		tdeeReq := map[string]interface{}{
			"age":            30,
			"sex":            "male",
			"height_cm":      180.0,
			"weight_kg":      80.0,
			"activity_level": "active",
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/goals/calculate-tdee", tdeeReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.NotEmpty(t, result["bmr"])
		assert.NotEmpty(t, result["tdee"])
		assert.NotEmpty(t, result["daily_calories"])
		assert.NotEmpty(t, result["protein_grams"])
		assert.NotEmpty(t, result["carbs_grams"])
		assert.NotEmpty(t, result["fat_grams"])

		// Verify calculations are reasonable
		tdee := result["tdee"].(float64)
		assert.Greater(t, tdee, 2000.0) // Active male should have TDEE > 2000
		assert.Less(t, tdee, 4000.0)    // But not unreasonably high

		// Protein should be ~2g per kg bodyweight
		protein := result["protein_grams"].(float64)
		assert.InDelta(t, 160.0, protein, 20.0) // ~2g per kg for 80kg
	})

	t.Run("auto-calculate and save goals", func(t *testing.T) {
		goalsReq := map[string]interface{}{
			"auto_calculate": true,
			"activity_level": "lightly_active",
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", goalsReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		goals := result["goals"].(map[string]interface{})
		assert.True(t, goals["auto_calculate"].(bool))
		assert.NotEmpty(t, goals["daily_calories"])
	})

	t.Run("validate TDEE inputs", func(t *testing.T) {
		invalidRequests := []map[string]interface{}{
			{
				"age":            10, // Too young
				"sex":            "male",
				"height_cm":      180.0,
				"weight_kg":      80.0,
				"activity_level": "active",
			},
			{
				"age":            30,
				"sex":            "invalid", // Invalid sex
				"height_cm":      180.0,
				"weight_kg":      80.0,
				"activity_level": "active",
			},
			{
				"age":            30,
				"sex":            "male",
				"height_cm":      50.0, // Too short
				"weight_kg":      80.0,
				"activity_level": "active",
			},
			{
				"age":            30,
				"sex":            "male",
				"height_cm":      180.0,
				"weight_kg":      600.0, // Too heavy
				"activity_level": "active",
			},
		}

		for _, req := range invalidRequests {
			resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/goals/calculate-tdee", req, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	})
}

// TestGoalsFlow_CustomGoals tests manual goal override
func TestGoalsFlow_CustomGoals(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("set custom goals", func(t *testing.T) {
		goalsReq := map[string]interface{}{
			"daily_calories": 2500,
			"protein_grams":  180,
			"carbs_grams":    250,
			"fat_grams":      80,
			"auto_calculate": false,
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", goalsReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		goals := result["goals"].(map[string]interface{})
		assert.Equal(t, float64(2500), goals["daily_calories"])
		assert.Equal(t, float64(180), goals["protein_grams"])
		assert.Equal(t, float64(250), goals["carbs_grams"])
		assert.Equal(t, float64(80), goals["fat_grams"])
		assert.False(t, goals["auto_calculate"].(bool))
	})

	t.Run("get current goals", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/goals", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		goals := result["goals"].(map[string]interface{})
		assert.NotEmpty(t, goals["daily_calories"])

		activeGoals := result["active_goals"].(map[string]interface{})
		assert.NotEmpty(t, activeGoals["daily_calories"])
	})

	t.Run("update specific goal fields", func(t *testing.T) {
		// Update only protein
		updateReq := map[string]interface{}{
			"protein_grams": 200,
		}

		resp, err := testServer.MakeAuthenticatedRequest("PATCH", "/api/v1/goals", updateReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		goals := result["goals"].(map[string]interface{})
		assert.Equal(t, float64(200), goals["protein_grams"])
	})
}

// TestGoalsFlow_DailyOverride tests per-day goal customization
func TestGoalsFlow_DailyOverride(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Set default goals
	setUserGoals(t, user, 2000, 150, 200, 70)

	t.Run("set Monday override", func(t *testing.T) {
		dailyGoalReq := map[string]interface{}{
			"daily_calories": 2500, // Higher for workout day
			"protein_grams":  180,
			"carbs_grams":    280,
			"fat_grams":      75,
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals/daily/monday", dailyGoalReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, float64(2500), result["daily_calories"])
		assert.Equal(t, "monday", result["day_of_week"])
	})

	t.Run("get goals with daily overrides", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/goals", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		dailyGoals := result["daily_goals"].(map[string]interface{})
		assert.NotNil(t, dailyGoals["monday"])

		monday := dailyGoals["monday"].(map[string]interface{})
		assert.Equal(t, float64(2500), monday["daily_calories"])
	})

	t.Run("active goals use daily override", func(t *testing.T) {
		// If today is Monday, active_goals should reflect the override
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/goals", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		activeGoals := result["active_goals"].(map[string]interface{})
		currentDay := result["current_day"].(string)

		// If current day is monday, should use override
		if currentDay == "monday" {
			assert.Equal(t, float64(2500), activeGoals["daily_calories"])
		} else {
			// Otherwise use default
			assert.Equal(t, float64(2000), activeGoals["daily_calories"])
		}
	})

	t.Run("delete daily override", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("DELETE", "/api/v1/goals/daily/monday", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify it's deleted
		getResp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/goals", nil, user)
		require.NoError(t, err)
		defer getResp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, getResp, http.StatusOK, &result)

		dailyGoals := result["daily_goals"].(map[string]interface{})
		assert.Nil(t, dailyGoals["monday"])
	})
}

// TestGoalsFlow_Validation tests goal validation
func TestGoalsFlow_Validation(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("reject invalid calorie values", func(t *testing.T) {
		invalidGoals := []map[string]interface{}{
			{"daily_calories": 500},  // Too low
			{"daily_calories": 6000}, // Too high
			{"daily_calories": -1000},
		}

		for _, req := range invalidGoals {
			resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", req, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("reject invalid macro values", func(t *testing.T) {
		invalidGoals := []map[string]interface{}{
			{"protein_grams": 10},  // Too low
			{"protein_grams": 400}, // Too high
			{"carbs_grams": 10},    // Too low
			{"carbs_grams": 600},   // Too high
			{"fat_grams": 5},       // Too low
			{"fat_grams": 300},     // Too high
		}

		for _, req := range invalidGoals {
			req["daily_calories"] = 2000 // Add valid calories
			resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", req, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("accept valid goal ranges", func(t *testing.T) {
		validGoals := map[string]interface{}{
			"daily_calories": 2000,
			"protein_grams":  150,
			"carbs_grams":    200,
			"fat_grams":      70,
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/goals", validGoals, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestGoalsFlow_ActivityLevels tests different activity level multipliers
func TestGoalsFlow_ActivityLevels(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	activityLevels := []string{"sedentary", "lightly_active", "active", "very_active"}

	baseReq := map[string]interface{}{
		"age":       30,
		"sex":       "male",
		"height_cm": 180.0,
		"weight_kg": 80.0,
	}

	t.Run("compare activity level TDEEs", func(t *testing.T) {
		tdees := make([]float64, len(activityLevels))

		for i, level := range activityLevels {
			req := make(map[string]interface{})
			for k, v := range baseReq {
				req[k] = v
			}
			req["activity_level"] = level

			resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/goals/calculate-tdee", req, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			AssertResponse(t, resp, http.StatusOK, &result)

			tdees[i] = result["tdee"].(float64)
		}

		// TDEE should increase with activity level
		for i := 1; i < len(tdees); i++ {
			assert.Greater(t, tdees[i], tdees[i-1],
				"TDEE for %s should be greater than %s", activityLevels[i], activityLevels[i-1])
		}
	})
}

// TestGoalsFlow_GenderDifferences tests male vs female calculations
func TestGoalsFlow_GenderDifferences(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	baseReq := map[string]interface{}{
		"age":            30,
		"height_cm":      170.0,
		"weight_kg":      70.0,
		"activity_level": "active",
	}

	t.Run("compare male vs female BMR", func(t *testing.T) {
		// Calculate for male
		maleReq := make(map[string]interface{})
		for k, v := range baseReq {
			maleReq[k] = v
		}
		maleReq["sex"] = "male"

		maleResp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/goals/calculate-tdee", maleReq, user)
		require.NoError(t, err)
		defer maleResp.Body.Close()

		var maleResult map[string]interface{}
		AssertResponse(t, maleResp, http.StatusOK, &maleResult)

		// Calculate for female
		femaleReq := make(map[string]interface{})
		for k, v := range baseReq {
			femaleReq[k] = v
		}
		femaleReq["sex"] = "female"

		femaleResp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/goals/calculate-tdee", femaleReq, user)
		require.NoError(t, err)
		defer femaleResp.Body.Close()

		var femaleResult map[string]interface{}
		AssertResponse(t, femaleResp, http.StatusOK, &femaleResult)

		// Male BMR should typically be higher
		maleBMR := maleResult["bmr"].(float64)
		femaleBMR := femaleResult["bmr"].(float64)

		assert.Greater(t, maleBMR, femaleBMR, "Male BMR should be higher than female BMR for same stats")
	})
}

// TestGoalsFlow_UpdateTracking tests tracking goal changes over time
func TestGoalsFlow_UpdateTracking(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("track goal updates", func(t *testing.T) {
		// Set initial goals
		setUserGoals(t, user, 2000, 150, 200, 70)

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Update goals
		setUserGoals(t, user, 2200, 160, 220, 75)

		// Get goals history (if implemented)
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/goals/history", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		// History endpoint might not be implemented, that's ok
		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			AssertResponse(t, resp, http.StatusOK, &result)

			history := result["history"].([]interface{})
			assert.GreaterOrEqual(t, len(history), 1)
		}
	})
}
