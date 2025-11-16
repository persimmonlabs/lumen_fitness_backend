package goals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupTestHandler() (*Handler, *mockRepository) {
	repo := newMockRepository()
	svc := NewService(repo)
	handler := NewHandler(svc)
	return handler, repo
}

func TestHandler_GetGoals(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/goals", nil)
	w := httptest.NewRecorder()

	handler.GetGoals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response GoalsResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have default goals
	if response.Goals.DailyCalories != 2000 {
		t.Errorf("Expected default 2000 calories, got %d", response.Goals.DailyCalories)
	}
}

func TestHandler_UpdateGoals(t *testing.T) {
	handler, _ := setupTestHandler()

	updateReq := UpdateGoalsRequest{
		DailyCalories: intPtr(2500),
		ProteinGrams:  intPtr(180),
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/goals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateGoals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var goals UserGoals
	err := json.NewDecoder(w.Body).Decode(&goals)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if goals.DailyCalories != 2500 {
		t.Errorf("Expected 2500 calories, got %d", goals.DailyCalories)
	}
}

func TestHandler_UpdateGoals_InvalidRequest(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodPut, "/goals", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateGoals(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_CalculateTDEE(t *testing.T) {
	handler, _ := setupTestHandler()

	tdeeReq := TDEEInputs{
		Age:           30,
		Sex:           SexMale,
		HeightCM:      180,
		WeightKG:      80,
		ActivityLevel: ActivitySedentary,
	}
	body, _ := json.Marshal(tdeeReq)

	req := httptest.NewRequest(http.MethodPost, "/goals/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CalculateTDEE(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result TDEEResult
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.DailyCalories == 0 {
		t.Error("Expected non-zero TDEE")
	}
	if result.BMR == 0 {
		t.Error("Expected non-zero BMR")
	}
}

func TestHandler_SetDailyGoal(t *testing.T) {
	handler, _ := setupTestHandler()

	dailyReq := SetDailyGoalRequest{
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}
	body, _ := json.Marshal(dailyReq)

	req := httptest.NewRequest(http.MethodPut, "/goals/daily/monday", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Setup chi router to handle URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("day", "monday")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.SetDailyGoal(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_SetDailyGoal_InvalidDay(t *testing.T) {
	handler, _ := setupTestHandler()

	dailyReq := SetDailyGoalRequest{
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}
	body, _ := json.Marshal(dailyReq)

	req := httptest.NewRequest(http.MethodPut, "/goals/daily/notaday", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("day", "notaday")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.SetDailyGoal(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_DeleteDailyGoal(t *testing.T) {
	handler, repo := setupTestHandler()

	// First create a daily goal
	goal := &DailyGoal{
		UserID:        1,
		DayOfWeek:     "monday",
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}
	_ = repo.SetDailyGoal(context.Background(), goal)

	req := httptest.NewRequest(http.MethodDelete, "/goals/daily/monday", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("day", "monday")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.DeleteDailyGoal(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestHandler_GetDailyGoals(t *testing.T) {
	handler, repo := setupTestHandler()

	// Create a daily goal
	goal := &DailyGoal{
		UserID:        1,
		DayOfWeek:     "monday",
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}
	_ = repo.SetDailyGoal(context.Background(), goal)

	req := httptest.NewRequest(http.MethodGet, "/goals/daily", nil)
	w := httptest.NewRecorder()

	handler.GetDailyGoals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var dailyGoals map[DayOfWeek]DailyGoal
	err := json.NewDecoder(w.Body).Decode(&dailyGoals)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(dailyGoals) != 1 {
		t.Errorf("Expected 1 daily goal, got %d", len(dailyGoals))
	}
}
