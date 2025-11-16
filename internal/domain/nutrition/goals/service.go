package goals

import (
	"context"
	"math"
	"strings"
	"time"
)

// Service handles nutrition goals business logic
type Service interface {
	// Goals management
	GetGoals(ctx context.Context, userID int64) (*GoalsResponse, error)
	UpdateGoals(ctx context.Context, userID int64, req *UpdateGoalsRequest) (*UserGoals, error)

	// TDEE calculation
	CalculateTDEE(ctx context.Context, inputs *TDEEInputs) (*TDEEResult, error)

	// Daily goals management
	SetDailyGoal(ctx context.Context, userID int64, day string, req *SetDailyGoalRequest) (*DailyGoal, error)
	GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error)
	DeleteDailyGoal(ctx context.Context, userID int64, day string) error
}

type service struct {
	repo Repository
}

// NewService creates a new goals service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetGoals retrieves current goals with auto-calculation if enabled
func (s *service) GetGoals(ctx context.Context, userID int64) (*GoalsResponse, error) {
	// Get user goals (create default if not found)
	goals, err := s.repo.GetUserGoals(ctx, userID)
	if err != nil {
		if err == ErrGoalsNotFound {
			// Create default goals
			goals = &UserGoals{
				UserID:        userID,
				DailyCalories: 2000,
				ProteinGrams:  150,
				CarbsGrams:    200,
				FatGrams:      67,
				AutoCalculate: false,
				ActivityLevel: string(ActivitySedentary),
			}
			err = s.repo.CreateUserGoals(ctx, goals)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Get daily goals
	dailyGoalsList, err := s.repo.GetDailyGoals(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert to map
	dailyGoalsMap := make(map[DayOfWeek]DailyGoal)
	for _, dg := range dailyGoalsList {
		dailyGoalsMap[DayOfWeek(dg.DayOfWeek)] = dg
	}

	// Determine current day and active goals
	currentDay := getCurrentDayOfWeek()
	activeGoals := s.getActiveGoalsForDay(goals, dailyGoalsMap, currentDay)

	return &GoalsResponse{
		Goals:       *goals,
		DailyGoals:  dailyGoalsMap,
		CurrentDay:  currentDay,
		ActiveGoals: activeGoals,
	}, nil
}

// UpdateGoals updates user goals
func (s *service) UpdateGoals(ctx context.Context, userID int64, req *UpdateGoalsRequest) (*UserGoals, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get existing goals or create new
	goals, err := s.repo.GetUserGoals(ctx, userID)
	if err != nil {
		if err == ErrGoalsNotFound {
			// Create new goals
			goals = &UserGoals{
				UserID:        userID,
				DailyCalories: 2000,
				ProteinGrams:  150,
				CarbsGrams:    200,
				FatGrams:      67,
				AutoCalculate: false,
				ActivityLevel: string(ActivitySedentary),
			}
		} else {
			return nil, err
		}
	}

	// Update fields if provided
	if req.DailyCalories != nil {
		goals.DailyCalories = *req.DailyCalories
	}
	if req.ProteinGrams != nil {
		goals.ProteinGrams = *req.ProteinGrams
	}
	if req.CarbsGrams != nil {
		goals.CarbsGrams = *req.CarbsGrams
	}
	if req.FatGrams != nil {
		goals.FatGrams = *req.FatGrams
	}
	if req.AutoCalculate != nil {
		goals.AutoCalculate = *req.AutoCalculate
	}
	if req.ActivityLevel != nil {
		goals.ActivityLevel = *req.ActivityLevel
	}

	// Save or update
	if goals.ID == 0 {
		err = s.repo.CreateUserGoals(ctx, goals)
	} else {
		err = s.repo.UpdateUserGoals(ctx, goals)
	}

	if err != nil {
		return nil, err
	}

	return goals, nil
}

// CalculateTDEE calculates TDEE using Mifflin-St Jeor equation
func (s *service) CalculateTDEE(ctx context.Context, inputs *TDEEInputs) (*TDEEResult, error) {
	if err := inputs.Validate(); err != nil {
		return nil, err
	}

	// Calculate BMR using Mifflin-St Jeor equation
	var bmr float64
	if inputs.Sex == SexMale {
		// BMR (men) = 10*weight + 6.25*height - 5*age + 5
		bmr = (10 * inputs.WeightKG) + (6.25 * inputs.HeightCM) - (5 * float64(inputs.Age)) + 5
	} else {
		// BMR (women) = 10*weight + 6.25*height - 5*age - 161
		bmr = (10 * inputs.WeightKG) + (6.25 * inputs.HeightCM) - (5 * float64(inputs.Age)) - 161
	}

	// Get activity multiplier
	activityMultiplier, ok := ActivityMultipliers[inputs.ActivityLevel]
	if !ok {
		return nil, ErrInvalidActivityLevel
	}

	// Calculate TDEE
	tdee := bmr * activityMultiplier

	// Calculate macros
	// Protein: 2g per kg bodyweight
	proteinGrams := inputs.WeightKG * 2

	// Fat: 25% of calories (9 cal/g)
	fatCalories := tdee * 0.25
	fatGrams := fatCalories / 9

	// Carbs: Remainder (4 cal/g)
	proteinCalories := proteinGrams * 4
	carbsCalories := tdee - proteinCalories - fatCalories
	carbsGrams := carbsCalories / 4

	return &TDEEResult{
		BMR:           math.Round(bmr),
		TDEE:          math.Round(tdee),
		DailyCalories: int(math.Round(tdee)),
		ProteinGrams:  int(math.Round(proteinGrams)),
		FatGrams:      int(math.Round(fatGrams)),
		CarbsGrams:    int(math.Round(carbsGrams)),
		Inputs:        *inputs,
	}, nil
}

// SetDailyGoal sets a day-specific goal
func (s *service) SetDailyGoal(ctx context.Context, userID int64, day string, req *SetDailyGoalRequest) (*DailyGoal, error) {
	day = strings.ToLower(day)
	if !IsValidDayOfWeek(day) {
		return nil, ErrInvalidDayOfWeek
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	goal := &DailyGoal{
		UserID:        userID,
		DayOfWeek:     day,
		DailyCalories: req.DailyCalories,
		ProteinGrams:  req.ProteinGrams,
		CarbsGrams:    req.CarbsGrams,
		FatGrams:      req.FatGrams,
	}

	err := s.repo.SetDailyGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

// GetDailyGoal retrieves a day-specific goal
func (s *service) GetDailyGoal(ctx context.Context, userID int64, day string) (*DailyGoal, error) {
	day = strings.ToLower(day)
	if !IsValidDayOfWeek(day) {
		return nil, ErrInvalidDayOfWeek
	}

	return s.repo.GetDailyGoal(ctx, userID, day)
}

// DeleteDailyGoal removes a day-specific goal
func (s *service) DeleteDailyGoal(ctx context.Context, userID int64, day string) error {
	day = strings.ToLower(day)
	if !IsValidDayOfWeek(day) {
		return ErrInvalidDayOfWeek
	}

	return s.repo.DeleteDailyGoal(ctx, userID, day)
}

// getActiveGoalsForDay determines the active goals for a specific day
func (s *service) getActiveGoalsForDay(userGoals *UserGoals, dailyGoals map[DayOfWeek]DailyGoal, day DayOfWeek) NutritionTargets {
	// Check for daily override
	if dailyGoal, ok := dailyGoals[day]; ok {
		return NutritionTargets{
			DailyCalories: dailyGoal.DailyCalories,
			ProteinGrams:  dailyGoal.ProteinGrams,
			CarbsGrams:    dailyGoal.CarbsGrams,
			FatGrams:      dailyGoal.FatGrams,
			Source:        "daily_override",
		}
	}

	// Use user goals
	source := "default"
	if userGoals.AutoCalculate {
		source = "auto_calculated"
	}

	return NutritionTargets{
		DailyCalories: userGoals.DailyCalories,
		ProteinGrams:  userGoals.ProteinGrams,
		CarbsGrams:    userGoals.CarbsGrams,
		FatGrams:      userGoals.FatGrams,
		Source:        source,
	}
}

// getCurrentDayOfWeek returns the current day of week as string
func getCurrentDayOfWeek() DayOfWeek {
	now := time.Now()
	weekday := now.Weekday()

	switch weekday {
	case time.Monday:
		return DayMonday
	case time.Tuesday:
		return DayTuesday
	case time.Wednesday:
		return DayWednesday
	case time.Thursday:
		return DayThursday
	case time.Friday:
		return DayFriday
	case time.Saturday:
		return DaySaturday
	case time.Sunday:
		return DaySunday
	default:
		return DayMonday
	}
}
