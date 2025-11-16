package goals

import (
	"context"
	"testing"
	"time"
)

// Mock repository for testing
type mockRepository struct {
	goals      map[int64]*UserGoals
	dailyGoals map[int64]map[string]*DailyGoal
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		goals:      make(map[int64]*UserGoals),
		dailyGoals: make(map[int64]map[string]*DailyGoal),
	}
}

func (m *mockRepository) GetUserGoals(ctx context.Context, userID int64) (*UserGoals, error) {
	goals, ok := m.goals[userID]
	if !ok {
		return nil, ErrGoalsNotFound
	}
	return goals, nil
}

func (m *mockRepository) UpdateUserGoals(ctx context.Context, goals *UserGoals) error {
	if _, ok := m.goals[goals.UserID]; !ok {
		return ErrGoalsNotFound
	}
	goals.UpdatedAt = time.Now()
	m.goals[goals.UserID] = goals
	return nil
}

func (m *mockRepository) CreateUserGoals(ctx context.Context, goals *UserGoals) error {
	goals.ID = int64(len(m.goals) + 1)
	goals.CreatedAt = time.Now()
	goals.UpdatedAt = time.Now()
	m.goals[goals.UserID] = goals
	return nil
}

func (m *mockRepository) GetDailyGoals(ctx context.Context, userID int64) ([]DailyGoal, error) {
	userDaily, ok := m.dailyGoals[userID]
	if !ok {
		return []DailyGoal{}, nil
	}

	result := make([]DailyGoal, 0, len(userDaily))
	for _, goal := range userDaily {
		result = append(result, *goal)
	}
	return result, nil
}

func (m *mockRepository) GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error) {
	userDaily, ok := m.dailyGoals[userID]
	if !ok {
		return nil, ErrDailyGoalNotFound
	}

	goal, ok := userDaily[day]
	if !ok {
		return nil, ErrDailyGoalNotFound
	}
	return goal, nil
}

func (m *mockRepository) SetDailyGoal(ctx context.Context, goal *DailyGoal) error {
	if !IsValidDayOfWeek(goal.DayOfWeek) {
		return ErrInvalidDayOfWeek
	}

	if m.dailyGoals[goal.UserID] == nil {
		m.dailyGoals[goal.UserID] = make(map[string]*DailyGoal)
	}

	goal.ID = int64(len(m.dailyGoals[goal.UserID]) + 1)
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	m.dailyGoals[goal.UserID][goal.DayOfWeek] = goal
	return nil
}

func (m *mockRepository) DeleteDailyGoal(ctx context.Context, userID int64, day string) error {
	userDaily, ok := m.dailyGoals[userID]
	if !ok {
		return ErrDailyGoalNotFound
	}

	if _, ok := userDaily[day]; !ok {
		return ErrDailyGoalNotFound
	}

	delete(userDaily, day)
	return nil
}

func TestMockRepository_GetUserGoals(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Test not found
	_, err := repo.GetUserGoals(ctx, 1)
	if err != ErrGoalsNotFound {
		t.Errorf("Expected ErrGoalsNotFound, got %v", err)
	}

	// Create and retrieve
	goals := &UserGoals{
		UserID:        1,
		DailyCalories: 2000,
		ProteinGrams:  150,
		CarbsGrams:    200,
		FatGrams:      67,
	}
	_ = repo.CreateUserGoals(ctx, goals)

	retrieved, err := repo.GetUserGoals(ctx, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if retrieved.DailyCalories != 2000 {
		t.Errorf("Expected 2000 calories, got %d", retrieved.DailyCalories)
	}
}

func TestMockRepository_UpdateUserGoals(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Test update non-existent
	goals := &UserGoals{UserID: 1}
	err := repo.UpdateUserGoals(ctx, goals)
	if err != ErrGoalsNotFound {
		t.Errorf("Expected ErrGoalsNotFound, got %v", err)
	}

	// Create and update
	_ = repo.CreateUserGoals(ctx, goals)
	goals.DailyCalories = 2500
	err = repo.UpdateUserGoals(ctx, goals)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	retrieved, _ := repo.GetUserGoals(ctx, 1)
	if retrieved.DailyCalories != 2500 {
		t.Errorf("Expected 2500 calories, got %d", retrieved.DailyCalories)
	}
}

func TestMockRepository_DailyGoals(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Test get non-existent
	_, err := repo.GetDailyGoal(ctx, 1, "monday")
	if err != ErrDailyGoalNotFound {
		t.Errorf("Expected ErrDailyGoalNotFound, got %v", err)
	}

	// Set daily goal
	goal := &DailyGoal{
		UserID:        1,
		DayOfWeek:     "monday",
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}
	err = repo.SetDailyGoal(ctx, goal)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Retrieve daily goal
	retrieved, err := repo.GetDailyGoal(ctx, 1, "monday")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if retrieved.DailyCalories != 2500 {
		t.Errorf("Expected 2500 calories, got %d", retrieved.DailyCalories)
	}

	// Get all daily goals
	allGoals, err := repo.GetDailyGoals(ctx, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(allGoals) != 1 {
		t.Errorf("Expected 1 daily goal, got %d", len(allGoals))
	}

	// Delete daily goal
	err = repo.DeleteDailyGoal(ctx, 1, "monday")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = repo.GetDailyGoal(ctx, 1, "monday")
	if err != ErrDailyGoalNotFound {
		t.Errorf("Expected ErrDailyGoalNotFound after delete, got %v", err)
	}
}

func TestMockRepository_InvalidDayOfWeek(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	goal := &DailyGoal{
		UserID:    1,
		DayOfWeek: "invalid_day",
	}
	err := repo.SetDailyGoal(ctx, goal)
	if err != ErrInvalidDayOfWeek {
		t.Errorf("Expected ErrInvalidDayOfWeek, got %v", err)
	}
}
