package analytics

import (
	"context"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/goals"
)

// mockGoalsRepositoryAnalytics implements goals.Repository for analytics tests
type mockGoalsRepositoryAnalytics struct{}

// GetByUserID - not used in regular analytics tests
func (m *mockGoalsRepositoryAnalytics) GetByUserID(ctx context.Context, userID uuid.UUID) (*goals.UserGoals, error) {
	return nil, nil
}

// GetUserGoals - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) GetUserGoals(ctx context.Context, userID int64) (*goals.UserGoals, error) {
	return nil, nil
}

// UpdateUserGoals - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) UpdateUserGoals(ctx context.Context, g *goals.UserGoals) error {
	return nil
}

// CreateUserGoals - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) CreateUserGoals(ctx context.Context, g *goals.UserGoals) error {
	return nil
}

// GetDailyGoals - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) GetDailyGoals(ctx context.Context, userID int64) ([]goals.DailyGoal, error) {
	return nil, nil
}

// GetDailyGoal - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) GetDailyGoal(ctx context.Context, userID int64, day string) (*goals.DailyGoal, error) {
	return nil, nil
}

// SetDailyGoal - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) SetDailyGoal(ctx context.Context, goal *goals.DailyGoal) error {
	return nil
}

// DeleteDailyGoal - not used in analytics tests
func (m *mockGoalsRepositoryAnalytics) DeleteDailyGoal(ctx context.Context, userID int64, day string) error {
	return nil
}
