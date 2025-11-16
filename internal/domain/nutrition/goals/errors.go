package goals

import "errors"

var (
	// Repository errors
	ErrGoalsNotFound      = errors.New("user goals not found")
	ErrDailyGoalNotFound  = errors.New("daily goal not found")
	ErrDuplicateDailyGoal = errors.New("daily goal already exists for this day")

	// Validation errors
	ErrInvalidAge           = errors.New("age must be between 13 and 120 years")
	ErrInvalidHeight        = errors.New("height must be between 100 and 250 cm")
	ErrInvalidWeight        = errors.New("weight must be between 30 and 500 kg")
	ErrInvalidSex           = errors.New("sex must be 'male' or 'female'")
	ErrInvalidActivityLevel = errors.New("invalid activity level")
	ErrInvalidCalories      = errors.New("calories must be between 1000 and 5000")
	ErrInvalidProtein       = errors.New("protein must be between 50 and 300 grams")
	ErrInvalidCarbs         = errors.New("carbs must be between 50 and 500 grams")
	ErrInvalidFat           = errors.New("fat must be between 20 and 200 grams")
	ErrInvalidDayOfWeek     = errors.New("invalid day of week")

	// Service errors
	ErrUserNotFound    = errors.New("user not found")
	ErrMissingUserData = errors.New("missing required user data for TDEE calculation")
)
