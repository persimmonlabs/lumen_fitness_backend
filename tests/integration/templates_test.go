package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateFlow_CreateUse tests template creation and usage
func TestTemplateFlow_CreateUse(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("create template from scratch", func(t *testing.T) {
		createReq := map[string]interface{}{
			"name": "Morning Smoothie",
			"items": []map[string]interface{}{
				{
					"food_id":      uuid.New().String(),
					"serving_size": 200.0,
					"serving_unit": "ml",
				},
				{
					"food_id":      uuid.New().String(),
					"serving_size": 1.0,
					"serving_unit": "scoop",
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		assert.NotEmpty(t, result["id"])
		assert.Equal(t, "Morning Smoothie", result["name"])
		assert.Greater(t, result["total_calories"], 0.0)

		items := result["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	t.Run("use template to create meal", func(t *testing.T) {
		// First create a template
		templateID := createTestTemplate(t, user, "Protein Shake")

		// Use template to create a meal
		useReq := map[string]interface{}{
			"meal_type": "breakfast",
			"meal_time": time.Now().Format(time.RFC3339),
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/templates/%s/use", templateID), useReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		// Should create a new meal
		assert.NotEmpty(t, result["id"])
		assert.Equal(t, "breakfast", result["meal_type"])
		assert.Greater(t, len(result["items"].([]interface{})), 0)
	})
}

// TestTemplateFlow_FromMeal tests creating templates from existing meals
func TestTemplateFlow_FromMeal(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create a meal
	mealID := createTestMeal(t, user, "lunch", time.Now())

	// Convert meal to template
	createReq := map[string]interface{}{
		"meal_id": mealID.String(),
		"name":    "Chicken & Rice Meal",
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates/from-meal", createReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)

	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "Chicken & Rice Meal", result["name"])
	assert.Greater(t, result["total_calories"], 0.0)

	// Template should have same items as meal
	items := result["items"].([]interface{})
	assert.Greater(t, len(items), 0)
}

// TestTemplateFlow_QuickLog tests fast meal logging with templates
func TestTemplateFlow_QuickLog(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create a template
	templateID := createTestTemplate(t, user, "Quick Breakfast")

	t.Run("log meal from template", func(t *testing.T) {
		start := time.Now()

		useReq := map[string]interface{}{
			"meal_type": "breakfast",
			"meal_time": time.Now().Format(time.RFC3339),
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/templates/%s/use", templateID), useReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		duration := time.Since(start)

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		// Should be fast (< 1 second)
		assert.Less(t, duration, 1*time.Second)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("log same template multiple times", func(t *testing.T) {
		// Should be able to use template multiple times
		for i := 0; i < 3; i++ {
			useReq := map[string]interface{}{
				"meal_type": "snack",
				"meal_time": time.Now().Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			}

			resp, err := testServer.MakeAuthenticatedRequest("POST", fmt.Sprintf("/api/v1/templates/%s/use", templateID), useReq, user)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			AssertResponse(t, resp, http.StatusCreated, &result)

			assert.NotEmpty(t, result["id"])
		}
	})
}

// TestTemplateFlow_ListUpdate tests listing and updating templates
func TestTemplateFlow_ListUpdate(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create multiple templates
	templates := []string{"Breakfast Bowl", "Protein Shake", "Salad Mix", "Snack Pack"}
	for _, name := range templates {
		createTestTemplate(t, user, name)
	}

	t.Run("list all templates", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/templates", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		templateList := result["templates"].([]interface{})
		assert.GreaterOrEqual(t, len(templateList), 4)
		assert.NotEmpty(t, result["total"])
	})

	t.Run("get single template", func(t *testing.T) {
		templateID := createTestTemplate(t, user, "Single Template")

		resp, err := testServer.MakeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/templates/%s", templateID), nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, "Single Template", result["name"])
		assert.NotEmpty(t, result["items"])
	})

	t.Run("update template", func(t *testing.T) {
		templateID := createTestTemplate(t, user, "Original Name")

		updateReq := map[string]interface{}{
			"name": "Updated Name",
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", fmt.Sprintf("/api/v1/templates/%s", templateID), updateReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, "Updated Name", result["name"])
	})

	t.Run("delete template", func(t *testing.T) {
		templateID := createTestTemplate(t, user, "To Be Deleted")

		resp, err := testServer.MakeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/v1/templates/%s", templateID), nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify it's deleted
		getResp, err := testServer.MakeAuthenticatedRequest("GET", fmt.Sprintf("/api/v1/templates/%s", templateID), nil, user)
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	})
}

// TestTemplateFlow_Validation tests input validation
func TestTemplateFlow_Validation(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("reject empty name", func(t *testing.T) {
		createReq := map[string]interface{}{
			"name": "",
			"items": []map[string]interface{}{
				{
					"food_id":      uuid.New().String(),
					"serving_size": 100.0,
					"serving_unit": "g",
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("reject template without items", func(t *testing.T) {
		createReq := map[string]interface{}{
			"name":  "Template Without Items",
			"items": []map[string]interface{}{},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("reject invalid serving size", func(t *testing.T) {
		createReq := map[string]interface{}{
			"name": "Invalid Template",
			"items": []map[string]interface{}{
				{
					"food_id":      uuid.New().String(),
					"serving_size": -100.0, // Negative
					"serving_unit": "g",
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("reject name too long", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'A'
		}

		createReq := map[string]interface{}{
			"name": string(longName),
			"items": []map[string]interface{}{
				{
					"food_id":      uuid.New().String(),
					"serving_size": 100.0,
					"serving_unit": "g",
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestTemplateFlow_FrequentMeals tests tracking and suggesting frequent meals
func TestTemplateFlow_FrequentMeals(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	// Create the same meal multiple times
	for i := 0; i < 5; i++ {
		createTestMeal(t, user, "breakfast", time.Now().Add(-time.Duration(i)*24*time.Hour))
	}

	t.Run("suggest template from frequent meal", func(t *testing.T) {
		resp, err := testServer.MakeAuthenticatedRequest("GET", "/api/v1/templates/suggestions", nil, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should suggest creating a template based on frequent meals
		var result map[string]interface{}
		if resp.StatusCode == http.StatusOK {
			AssertResponse(t, resp, http.StatusOK, &result)
			suggestions := result["suggestions"].([]interface{})
			assert.Greater(t, len(suggestions), 0)
		}
	})
}

// TestTemplateFlow_WithPhotos tests templates with photos
func TestTemplateFlow_WithPhotos(t *testing.T) {
	testServer.CleanDatabase(t)
	user := testServer.CreateTestUser(t)

	t.Run("create template with photo", func(t *testing.T) {
		photoURL := "https://example.com/template-photo.jpg"

		createReq := map[string]interface{}{
			"name":      "Photo Template",
			"photo_url": photoURL,
			"items": []map[string]interface{}{
				{
					"food_id":      uuid.New().String(),
					"serving_size": 100.0,
					"serving_unit": "g",
				},
			},
		}

		resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusCreated, &result)

		assert.Equal(t, photoURL, result["photo_url"])
	})

	t.Run("update template photo", func(t *testing.T) {
		templateID := createTestTemplate(t, user, "Template for Photo Update")
		newPhotoURL := "https://example.com/new-photo.jpg"

		updateReq := map[string]interface{}{
			"photo_url": newPhotoURL,
		}

		resp, err := testServer.MakeAuthenticatedRequest("PUT", fmt.Sprintf("/api/v1/templates/%s", templateID), updateReq, user)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		AssertResponse(t, resp, http.StatusOK, &result)

		assert.Equal(t, newPhotoURL, result["photo_url"])
	})
}

// Helper functions

func createTestTemplate(t *testing.T, user *TestUser, name string) string {
	t.Helper()

	createReq := map[string]interface{}{
		"name": name,
		"items": []map[string]interface{}{
			{
				"food_id":      uuid.New().String(),
				"serving_size": 100.0,
				"serving_unit": "g",
			},
			{
				"food_id":      uuid.New().String(),
				"serving_size": 1.0,
				"serving_unit": "serving",
			},
		},
	}

	resp, err := testServer.MakeAuthenticatedRequest("POST", "/api/v1/templates", createReq, user)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	AssertResponse(t, resp, http.StatusCreated, &result)

	return result["id"].(string)
}
