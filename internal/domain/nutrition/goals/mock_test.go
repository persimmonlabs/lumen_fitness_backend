package goals

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// mockRepository implements Repository interface for testing
type mockRepository struct {
	mu          sync.RWMutex
	userGoals   map[int64]*UserGoals
	dailyGoals  map[int64]map[string]*DailyGoal
	userGoalsV2 map[uuid.UUID]*UserGoals
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		userGoals:   make(map[int64]*UserGoals),
		dailyGoals:  make(map[int64]map[string]*DailyGoal),
		userGoalsV2: make(map[uuid.UUID]*UserGoals),
	}
}

// UserGoals operations (int64 userID)
func (m *mockRepository) GetUserGoals(ctx context.Context, userID int64) (*UserGoals, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if goals, exists := m.userGoals[userID]; exists {
		return goals, nil
	}
	return nil, ErrGoalsNotFound
}

func (m *mockRepository) UpdateUserGoals(ctx context.Context, goals *UserGoals) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.userGoals[goals.UserID] = goals
	return nil
}

func (m *mockRepository) CreateUserGoals(ctx context.Context, goals *UserGoals) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.userGoals[goals.UserID] = goals
	return nil
}

// GetByUserID implements the UUID-based version
func (m *mockRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserGoals, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if goals, exists := m.userGoalsV2[userID]; exists {
		return goals, nil
	}
	return nil, ErrGoalsNotFound
}

// DailyGoal operations
func (m *mockRepository) GetDailyGoals(ctx context.Context, userID int64) ([]DailyGoal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if dayGoals, exists := m.dailyGoals[userID]; exists {
		goals := make([]DailyGoal, 0, len(dayGoals))
		for _, goal := range dayGoals {
			goals = append(goals, *goal)
		}
		return goals, nil
	}
	return []DailyGoal{}, nil
}

func (m *mockRepository) GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if dayGoals, exists := m.dailyGoals[userID]; exists {
		if goal, exists := dayGoals[day]; exists {
			return goal, nil
		}
	}
	return nil, ErrDailyGoalNotFound
}

func (m *mockRepository) SetDailyGoal(ctx context.Context, goal *DailyGoal) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.dailyGoals[goal.UserID]; !exists {
		m.dailyGoals[goal.UserID] = make(map[string]*DailyGoal)
	}
	m.dailyGoals[goal.UserID][goal.DayOfWeek] = goal
	return nil
}

func (m *mockRepository) DeleteDailyGoal(ctx context.Context, userID int64, day string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dayGoals, exists := m.dailyGoals[userID]; exists {
		delete(dayGoals, day)
		return nil
	}
	return ErrDailyGoalNotFound
}
