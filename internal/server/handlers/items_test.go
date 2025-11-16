package handlers_test

import (
	"log/slog"
	"testing"

	"github.com/pradord/lumen_final/backend/internal/server/handlers"
	"github.com/pradord/lumen_final/backend/internal/testutil"
)

func TestItemHandler_Create_Valid(t *testing.T) {
	// Setup
	logger := slog.Default()
	handler := handlers.NewItemHandler(logger, nil) // nil supabase for now

	// Create request
	req := testutil.NewTestRequest(t, "POST", "/api/v1/items", map[string]interface{}{
		"name":        "Test Item",
		"description": "A valid test item",
	})

	w := testutil.NewResponseRecorder()

	// Execute
	handler.Create(w, req)

	// Assert
	testutil.AssertStatusCode(t, w, 201)

	// Parse response
	type Response struct {
		Success bool `json:"success"`
		Data    struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}

	resp := testutil.ParseJSONResponse[Response](t, w)

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	if resp.Data.Name != "Test Item" {
		t.Errorf("Expected name 'Test Item', got '%s'", resp.Data.Name)
	}
}

func TestItemHandler_Create_ValidationError(t *testing.T) {
	logger := slog.Default()
	handler := handlers.NewItemHandler(logger, nil)

	// Invalid request - name too short (min 3 chars)
	req := testutil.NewTestRequest(t, "POST", "/api/v1/items", map[string]interface{}{
		"name":        "ab", // Too short!
		"description": "Test",
	})

	w := testutil.NewResponseRecorder()
	handler.Create(w, req)

	// Should return 400 Bad Request
	testutil.AssertStatusCode(t, w, 400)
}

func TestItemHandler_List(t *testing.T) {
	logger := slog.Default()
	handler := handlers.NewItemHandler(logger, nil)

	req := testutil.NewTestRequest(t, "GET", "/api/v1/items", nil)
	w := testutil.NewResponseRecorder()

	handler.List(w, req)

	testutil.AssertStatusCode(t, w, 200)
}
