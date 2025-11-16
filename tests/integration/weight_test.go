package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWeightFlow_CreateListTrend tests weight entry creation, listing, and trend calculation
func TestWeightFlow_CreateListTrend(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("create weight entry", func(t *testing.T) {
		createReq := map[string]interface{}{
			"weight":      75.5,
			"measured_at": time.Now().Format(time.RFC3339),
			"notes":       "Morning weight",
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		assert.NotEmpty(t, result["id"])
		assert.Equal(t, 75.5, result["weight"])
		assert.Equal(t, "Morning weight", result["notes"])
	})

	t.Run("list weight entries", func(t *testing.T) {
		// Create multiple entries
		weights := []float64{76.0, 75.8, 75.5, 75.3, 75.0}
		for i, weight := range weights {
			createWeightEntry(t, user, weight, time.Now().Add(-time.Duration(i)*24*time.Hour))
		}

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight?page=1&page_size=10", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		entries := result["entries"].([]interface{})
		assert.GreaterOrEqual(t, len(entries), 5)
		assert.NotEmpty(t, result["total"])
	})

	t.Run("get weight statistics", func(t *testing.T) {
		// Create entries for last 30 days
		startDate := time.Now().Add(-30 * 24 * time.Hour)
		for i := 0; i < 30; i++ {
			weight := 80.0 - float64(i)*0.1 // Gradual weight loss
			createWeightEntry(t, user, weight, startDate.Add(time.Duration(i)*24*time.Hour))
		}

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/stats", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.NotNil(t, result["average_7day"])
		assert.NotNil(t, result["average_30day"])
		assert.NotNil(t, result["rate_of_change"])
		assert.NotEmpty(t, result["latest_weight"])
		assert.NotEmpty(t, result["latest_date"])
	})

	t.Run("get weight trend", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/trend?days=7", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		trend := result["trend"].([]interface{})
		assert.Greater(t, len(trend), 0)
	})
}

// TestWeightFlow_DuplicateDate tests one entry per day enforcement
func TestWeightFlow_DuplicateDate(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	today := time.Now().Truncate(24 * time.Hour)

	// Create first entry for today
	createReq := map[string]interface{}{
		"weight":      75.0,
		"measured_at": today.Format(time.RFC3339),
	}

	resp1, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
	require.NoError(t, err)
	defer resp1.Body.Close()

	var result1 map[string]interface{}
	AssertResponse(t, resp1, http.StatusCreated, &result1)

	// Try to create second entry for same day
	createReq2 := map[string]interface{}{
		"weight":      75.5,
		"measured_at": today.Add(2 * time.Hour).Format(time.RFC3339), // Different time, same day
	}

	resp2, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq2, user)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// Should either reject or update existing entry
	assert.True(t, resp2.StatusCode == http.StatusConflict || resp2.StatusCode == http.StatusOK)
}

// TestWeightFlow_Statistics tests statistical calculations
func TestWeightFlow_Statistics(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("calculate 7-day average", func(t *testing.T) {
		// Create exactly 7 entries
		weights := []float64{80.0, 79.8, 79.5, 79.3, 79.0, 78.8, 78.5}
		startDate := time.Now().Add(-7 * 24 * time.Hour)

		for i, weight := range weights {
			createWeightEntry(t, user, weight, startDate.Add(time.Duration(i)*24*time.Hour))
		}

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/stats", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		avg7 := result["average_7day"].(float64)
		expectedAvg := (80.0 + 79.8 + 79.5 + 79.3 + 79.0 + 78.8 + 78.5) / 7.0

		assert.InDelta(t, expectedAvg, avg7, 0.1)
	})

	t.Run("calculate 30-day average", func(t *testing.T) {
		testServer.CleanDatabase(t)
		user := testServer.CreateTestUser(t)

		// Create 30 entries
		startWeight := 85.0
		startDate := time.Now().Add(-30 * 24 * time.Hour)

		for i := 0; i < 30; i++ {
			weight := startWeight - float64(i)*0.1
			createWeightEntry(t, user, weight, startDate.Add(time.Duration(i)*24*time.Hour))
		}

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/stats", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.NotNil(t, result["average_30day"])
		avg30 := result["average_30day"].(float64)
		assert.Greater(t, avg30, 0.0)
	})

	t.Run("calculate rate of change", func(t *testing.T) {
		testServer.CleanDatabase(t)
		user := testServer.CreateTestUser(t)

		// Create entries showing consistent weight loss
		// 2kg loss over 4 weeks = 0.5kg per week
		startDate := time.Now().Add(-28 * 24 * time.Hour)
		startWeight := 80.0
		endWeight := 78.0

		for i := 0; i < 28; i++ {
			// Linear weight loss
			weight := startWeight - (float64(i) * (startWeight - endWeight) / 27.0)
			createWeightEntry(t, user, weight, startDate.Add(time.Duration(i)*24*time.Hour))
		}

		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/stats", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		rateOfChange := result["rate_of_change"].(float64)
		// Should be approximately -0.5 kg/week
		assert.InDelta(t, -0.5, rateOfChange, 0.2)
	})
}

// TestWeightFlow_UpdateDelete tests updating and deleting entries
func TestWeightFlow_UpdateDelete(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("update weight entry", func(t *testing.T) {
		// Create entry
		entryID := createWeightEntry(t, user, 75.0, time.Now())

		// Update it
		updateReq := map[string]interface{}{
			"weight": 75.5,
			"notes":  "Updated weight",
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/weight/"+entryID, updateReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, 75.5, result["weight"])
		assert.Equal(t, "Updated weight", result["notes"])
	})

	t.Run("delete weight entry", func(t *testing.T) {
		// Create entry
		entryID := createWeightEntry(t, user, 76.0, time.Now().Add(-24*time.Hour))

		// Delete it
		resp, err := testServer.MakeAuthenticatedRequest("DELETE", "/api/v1/weight/"+entryID, nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify it's deleted
		getResp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/weight/"+entryID, nil, user)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})
}

// TestWeightFlow_Validation tests input validation
func TestWeightFlow_Validation(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("reject invalid weight values", func(t *testing.T) {
		invalidWeights := []float64{-1.0, 0, 600.0} // Negative, zero, too high

		for _, weight := range invalidWeights {
			createReq := map[string]interface{}{
				"weight":      weight,
				"measured_at": time.Now().Format(time.RFC3339),
			}

			resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("reject future dates", func(t *testing.T) {
		createReq := map[string]interface{}{
			"weight":      75.0,
			"measured_at": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("accept valid weight range", func(t *testing.T) {
		validWeights := []float64{30.0, 75.5, 150.0, 250.0}

		for _, weight := range validWeights {
			testServer.CleanDatabase(t)
			user := testServer.CreateTestUser(t)

			createReq := map[string]interface{}{
				"weight":      weight,
				"measured_at": time.Now().Format(time.RFC3339),
			}

			resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}
	})
}

// TestWeightFlow_DateRangeFilter tests filtering by date range
func TestWeightFlow_DateRangeFilter(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create entries across 60 days
	startDate := time.Now().Add(-60 * 24 * time.Hour)
	for i := 0; i < 60; i++ {
		weight := 80.0 - float64(i)*0.05
		createWeightEntry(t, user, weight, startDate.Add(time.Duration(i)*24*time.Hour))
	}

	t.Run("filter last 30 days", func(t *testing.T) {
		startDateFilter := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
		endDateFilter := time.Now().Format(time.RFC3339)

		url := fmt.Sprintf("/api/v1/weight?start_date=%s&end_date=%s", startDateFilter, endDateFilter)
		resp, err := testServer.MakeAuthenticatedRequest("GET", url, nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		entries := result["entries"].([]interface{})
		// Should return approximately 30 entries
		assert.GreaterOrEqual(t, len(entries), 28)
		assert.LessOrEqual(t, len(entries), 31)
	})
}

// Helper functions

func createWeightEntry(t *testing.T, user *TestUser, weight float64, measuredAt time.Time) string {
	t.Helper()

	createReq := map[string]interface{}{
		"weight":      weight,
		"measured_at": measuredAt.Format(time.RFC3339),
		"notes":       "Test entry",
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/weight", createReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)

	return result["id"].(string)
}
