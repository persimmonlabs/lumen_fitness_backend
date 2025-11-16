package goals

import (
	"context"
	"testing"
)

func TestService_CalculateTDEE(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name          string
		inputs        TDEEInputs
		expectError   bool
		expectedTDEE  int // Approximate
	}{
		{
			name: "male sedentary",
			inputs: TDEEInputs{
				Age:           30,
				Sex:           SexMale,
				HeightCM:      180,
				WeightKG:      80,
				ActivityLevel: ActivitySedentary,
			},
			expectError:  false,
			expectedTDEE: 2100, // Approximate
		},
		{
			name: "female active",
			inputs: TDEEInputs{
				Age:           25,
				Sex:           SexFemale,
				HeightCM:      165,
				WeightKG:      60,
				ActivityLevel: ActivityActive,
			},
			expectError:  false,
			expectedTDEE: 2000, // Approximate
		},
		{
			name: "invalid age",
			inputs: TDEEInputs{
				Age:           10,
				Sex:           SexMale,
				HeightCM:      180,
				WeightKG:      80,
				ActivityLevel: ActivitySedentary,
			},
			expectError: true,
		},
		{
			name: "invalid weight",
			inputs: TDEEInputs{
				Age:           30,
				Sex:           SexMale,
				HeightCM:      180,
				WeightKG:      600,
				ActivityLevel: ActivitySedentary,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CalculateTDEE(ctx, &tt.inputs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			// Check that TDEE is reasonable (within 500 cal of expected)
			if result.DailyCalories < tt.expectedTDEE-500 || result.DailyCalories > tt.expectedTDEE+500 {
				t.Errorf("Expected TDEE around %d, got %d", tt.expectedTDEE, result.DailyCalories)
			}

			// Check that macros are calculated
			if result.ProteinGrams == 0 || result.CarbsGrams == 0 || result.FatGrams == 0 {
				t.Error("Expected non-zero macros")
			}
		})
	}
}

func TestService_GetGoals(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// First call should create default goals
	response, err := svc.GetGoals(ctx, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Goals.DailyCalories != 2000 {
		t.Errorf("Expected default 2000 calories, got %d", response.Goals.DailyCalories)
	}

	// Update goals
	updateReq := &UpdateGoalsRequest{
		DailyCalories: intPtr(2500),
		ProteinGrams:  intPtr(180),
	}

	updated, err := svc.UpdateGoals(ctx, 1, updateReq)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if updated.DailyCalories != 2500 {
		t.Errorf("Expected 2500 calories, got %d", updated.DailyCalories)
	}
	if updated.ProteinGrams != 180 {
		t.Errorf("Expected 180g protein, got %d", updated.ProteinGrams)
	}
}

func TestService_DailyGoals(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// Set Monday goal
	req := &SetDailyGoalRequest{
		DailyCalories: 2500,
		ProteinGrams:  180,
		CarbsGrams:    250,
		FatGrams:      83,
	}

	goal, err := svc.SetDailyGoal(ctx, 1, "monday", req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if goal.DayOfWeek != "monday" {
		t.Errorf("Expected monday, got %s", goal.DayOfWeek)
	}

	// Retrieve Monday goal
	retrieved, err := svc.GetDailyGoal(ctx, 1, "monday")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved.DailyCalories != 2500 {
		t.Errorf("Expected 2500 calories, got %d", retrieved.DailyCalories)
	}

	// Delete Monday goal
	err = svc.DeleteDailyGoal(ctx, 1, "monday")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify deleted
	_, err = svc.GetDailyGoal(ctx, 1, "monday")
	if err != ErrDailyGoalNotFound {
		t.Errorf("Expected ErrDailyGoalNotFound, got %v", err)
	}
}

func TestService_UpdateGoals_Validation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         UpdateGoalsRequest
		expectError bool
	}{
		{
			name: "valid update",
			req: UpdateGoalsRequest{
				DailyCalories: intPtr(2000),
				ProteinGrams:  intPtr(150),
			},
			expectError: false,
		},
		{
			name: "invalid calories too low",
			req: UpdateGoalsRequest{
				DailyCalories: intPtr(500),
			},
			expectError: true,
		},
		{
			name: "invalid calories too high",
			req: UpdateGoalsRequest{
				DailyCalories: intPtr(6000),
			},
			expectError: true,
		},
		{
			name: "invalid protein too low",
			req: UpdateGoalsRequest{
				ProteinGrams: intPtr(30),
			},
			expectError: true,
		},
		{
			name: "invalid protein too high",
			req: UpdateGoalsRequest{
				ProteinGrams: intPtr(400),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateGoals(ctx, 1, &tt.req)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestService_SetDailyGoal_Validation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name        string
		day         string
		req         SetDailyGoalRequest
		expectError bool
	}{
		{
			name: "valid monday goal",
			day:  "monday",
			req: SetDailyGoalRequest{
				DailyCalories: 2000,
				ProteinGrams:  150,
				CarbsGrams:    200,
				FatGrams:      67,
			},
			expectError: false,
		},
		{
			name: "invalid day",
			day:  "notaday",
			req: SetDailyGoalRequest{
				DailyCalories: 2000,
				ProteinGrams:  150,
				CarbsGrams:    200,
				FatGrams:      67,
			},
			expectError: true,
		},
		{
			name: "invalid calories",
			day:  "tuesday",
			req: SetDailyGoalRequest{
				DailyCalories: 500,
				ProteinGrams:  150,
				CarbsGrams:    200,
				FatGrams:      67,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.SetDailyGoal(ctx, 1, tt.day, &tt.req)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}
