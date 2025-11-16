package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMealFlow_ParseConfirmEdit tests the complete meal lifecycle
func TestMealFlow_ParseConfirmEdit(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("parse meal from text", func(t *testing.T) {
		parsedMeal := parseMealDescription(t, user, "2 eggs, 1 slice toast, coffee")
		assert.NotNil(t, parsedMeal)
		assert.Greater(t, parsedMeal.DraftMeal.Total.Calories, 0.0)
		assert.Greater(t, len(parsedMeal.DraftMeal.Items), 0)
	})

	t.Run("confirm and save meal", func(t *testing.T) {
		// First parse
		parsedMeal := parseMealDescription(t, user, "chicken breast 200g, rice 150g, broccoli 100g")

		// Then confirm
		confirmReq := map[string]interface{}{
			"meal_type":   "lunch",
			"consumed_at": time.Now().Format(time.RFC3339),
			"photos":      []string{},
			"notes":       "Post-workout meal",
			"items":       parsedMeal.DraftMeal.Items,
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		assert.NotEmpty(t, result["id"])
		assert.Equal(t, "lunch", result["meal_type"])
		assert.Greater(t, result["total_calories"], 0.0)
	})

	t.Run("update meal", func(t *testing.T) {
		// Create a meal
		meal := createTestMeal(t, user, "breakfast", time.Now())

		// Update it
		updateReq := map[string]interface{}{
			"meal_type":   "brunch",
			"consumed_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			"notes":       "Updated notes",
			"items": []map[string]interface{}{
				{
					"name":      "Updated Food",
					"quantity":  150.0,
					"unit":      "g",
					"calories":  300.0,
					"protein_g": 20.0,
					"carbs_g":   30.0,
					"fat_g":     10.0,
					"fiber_g":   5.0,
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", "/api/v1/meals/"+meal.String(), updateReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, "brunch", result["meal_type"])
		assert.Equal(t, "Updated notes", result["notes"])
	})

	t.Run("delete meal", func(t *testing.T) {
		// Create a meal
		meal := createTestMeal(t, user, "snack", time.Now())

		// Delete it
		resp, err := testServer.MakeAuthenticatedRequest("DELETE", "/api/v1/meals/"+meal.String(), nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify it's deleted
		getResp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals/"+meal.String(), nil, user)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})
}

// TestMealFlow_WithPhotos tests meal creation with photo uploads
func TestMealFlow_WithPhotos(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("parse meal with photos", func(t *testing.T) {
		parseReq := map[string]interface{}{
			"description":     "steak with vegetables",
			"meal_type":       "dinner",
			"consumed_at":     time.Now().Format(time.RFC3339),
			"photos":          []string{"photo1.jpg", "photo2.jpg"},
			"idempotency_key": uuid.New().String(),
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/parse", parseReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.NotEmpty(t, result["draft_meal"])
	})

	t.Run("confirm meal with photos", func(t *testing.T) {
		confirmReq := map[string]interface{}{
			"meal_type":   "dinner",
			"consumed_at": time.Now().Format(time.RFC3339),
			"photos":      []string{"photo1.jpg", "photo2.jpg"},
			"notes":       "Delicious meal",
			"items": []map[string]interface{}{
				{
					"name":      "Steak",
					"quantity":  200.0,
					"unit":      "g",
					"calories":  400.0,
					"protein_g": 50.0,
					"carbs_g":   0.0,
					"fat_g":     20.0,
					"fiber_g":   0.0,
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		photos := result["photos"].([]interface{})
		assert.Len(t, photos, 2)
	})
}

// TestMealFlow_DuplicateDetection tests duplicate meal prevention
func TestMealFlow_DuplicateDetection(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	idempotencyKey := uuid.New().String()
	parseReq := map[string]interface{}{
		"description":     "oatmeal with banana",
		"meal_type":       "breakfast",
		"consumed_at":     time.Now().Format(time.RFC3339),
		"photos":          []string{},
		"idempotency_key": idempotencyKey,
	}

	// First request
	resp1, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/parse", parseReq, user)
	require.NoError(t, err)
	defer resp1.Body.Close()

	var result1 map[string]interface{}
	AssertResponse(t, resp1, http.StatusOK, &result1)

	// Second request with same idempotency key (should return cached result)
	resp2, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/parse", parseReq, user)
	require.NoError(t, err)
	defer resp2.Body.Close()

	var result2 map[string]interface{}
	AssertResponse(t, resp2, http.StatusOK, &result2)

	// Results should be identical (from cache)
	assert.Equal(t, result1, result2)
}

// TestMealFlow_Validation tests input validation
func TestMealFlow_Validation(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("reject future date", func(t *testing.T) {
		confirmReq := map[string]interface{}{
			"meal_type":   "breakfast",
			"consumed_at": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			"notes":       "",
			"items": []map[string]interface{}{
				{
					"name":      "Food",
					"quantity":  100.0,
					"unit":      "g",
					"calories":  200.0,
					"protein_g": 10.0,
					"carbs_g":   20.0,
					"fat_g":     5.0,
					"fiber_g":   2.0,
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject future dates
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("reject invalid macros", func(t *testing.T) {
		confirmReq := map[string]interface{}{
			"meal_type":   "lunch",
			"consumed_at": time.Now().Format(time.RFC3339),
			"notes":       "",
			"items": []map[string]interface{}{
				{
					"name":      "Invalid Food",
					"quantity":  -100.0, // Negative quantity
					"unit":      "g",
					"calories":  -200.0, // Negative calories
					"protein_g": -10.0,
					"carbs_g":   -20.0,
					"fat_g":     -5.0,
					"fiber_g":   -2.0,
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("reject too many photos", func(t *testing.T) {
		confirmReq := map[string]interface{}{
			"meal_type":   "dinner",
			"consumed_at": time.Now().Format(time.RFC3339),
			"photos":      []string{"p1.jpg", "p2.jpg", "p3.jpg", "p4.jpg"}, // Max 3 allowed
			"items": []map[string]interface{}{
				{
					"name":      "Food",
					"quantity":  100.0,
					"unit":      "g",
					"calories":  200.0,
					"protein_g": 10.0,
					"carbs_g":   20.0,
					"fat_g":     5.0,
					"fiber_g":   2.0,
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestMealFlow_Caching tests caching behavior
func TestMealFlow_Caching(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create meals to test list caching
	for i := 0; i < 5; i++ {
		createTestMeal(t, user, "lunch", time.Now().Add(-time.Duration(i)*24*time.Hour))
	}

	t.Run("cache hit on repeated list requests", func(t *testing.T) {
		// First request
		start1 := time.Now()
		resp1, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals?page=1&limit=10", nil, user)
		require.NoError(t, err)
		defer resp1.Body.Close()
		duration1 := time.Since(start1)

		var result1 map[string]interface{}
		AssertResponse(t, resp1, http.StatusOK, &result1)

		// Second request (should be faster due to cache)
		start2 := time.Now()
		resp2, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals?page=1&limit=10", nil, user)
		require.NoError(t, err)
		defer resp2.Body.Close()
		duration2 := time.Since(start2)

		var result2 map[string]interface{}
		AssertResponse(t, resp2, http.StatusOK, &result2)

		// Second request should be faster (cached)
		assert.LessOrEqual(t, duration2, duration1)
	})
}

// Helper functions

func parseMealDescription(t *testing.T, user *TestUser, description string) *ParsedMealResponse {
	t.Helper()

	parseReq := map[string]interface{}{
		"description":     description,
		"meal_type":       "breakfast",
		"consumed_at":     time.Now().Format(time.RFC3339),
		"photos":          []string{},
		"idempotency_key": uuid.New().String(),
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/parse", parseReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result ParsedMealResponse
	AssertResponse(t, resp, http.StatusOK, &result)

	return &result
}

func createTestMeal(t *testing.T, user *TestUser, mealType string, consumedAt time.Time) uuid.UUID {
	t.Helper()

	confirmReq := map[string]interface{}{
		"meal_type":   mealType,
		"consumed_at": consumedAt.Format(time.RFC3339),
		"notes":       "Test meal",
		"items": []map[string]interface{}{
			{
				"name":      "Test Food",
				"quantity":  100.0,
				"unit":      "g",
				"calories":  200.0,
				"protein_g": 15.0,
				"carbs_g":   25.0,
				"fat_g":     8.0,
				"fiber_g":   3.0,
			},
		},
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/meals/confirm", confirmReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)

	mealID, err := uuid.Parse(result["id"].(string))
	require.NoError(t, err)

	return mealID
}

// Response types

type ParsedMealResponse struct {
	DraftMeal struct {
		Items []struct {
			Name     string  `json:"name"`
			Quantity float64 `json:"quantity"`
			Unit     string  `json:"unit"`
			Calories float64 `json:"calories"`
			ProteinG float64 `json:"protein_g"`
			CarbsG   float64 `json:"carbs_g"`
			FatG     float64 `json:"fat_g"`
			FiberG   float64 `json:"fiber_g"`
		} `json:"items"`
		Total struct {
			Calories float64 `json:"calories"`
			ProteinG float64 `json:"protein_g"`
			CarbsG   float64 `json:"carbs_g"`
			FatG     float64 `json:"fat_g"`
			FiberG   float64 `json:"fiber_g"`
		} `json:"total"`
		Confidence float64 `json:"confidence"`
		CostUSD    float64 `json:"cost_usd"`
	} `json:"draft_meal"`
}

// TestMealFlow_ListAndPagination tests meal listing and pagination
func TestMealFlow_ListAndPagination(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create 15 meals
	for i := 0; i < 15; i++ {
		createTestMeal(t, user, "lunch", time.Now().Add(-time.Duration(i)*time.Hour))
	}

	t.Run("list first page", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals?page=1&limit=10", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		meals := result["meals"].([]interface{})
		pagination := result["pagination"].(map[string]interface{})

		assert.Len(t, meals, 10)
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
		assert.Equal(t, float64(15), pagination["total"])
		assert.Equal(t, float64(2), pagination["total_pages"])
	})

	t.Run("list second page", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/meals?page=2&limit=10", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		meals := result["meals"].([]interface{})
		assert.Len(t, meals, 5) // Remaining 5 meals
	})
}

// TestMealFlow_CopyMeal tests meal duplication
func TestMealFlow_CopyMeal(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create original meal
	originalMeal := createTestMeal(t, user, "breakfast", time.Now().Add(-24*time.Hour))

	// Copy the meal
	copyReq := map[string]interface{}{
		"consumed_at": time.Now().Format(time.RFC3339),
		"meal_type":   "breakfast",
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/meals/%s/copy", originalMeal.String()), copyReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)

	// Verify it's a new meal with same items
	assert.NotEqual(t, originalMeal.String(), result["id"])
	assert.Equal(t, "breakfast", result["meal_type"])

	// Check items match
	items := result["items"].([]interface{})
	assert.Greater(t, len(items), 0)
}
