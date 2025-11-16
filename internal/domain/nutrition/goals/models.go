package goals

import (
	"time"
)

// ActivityLevel represents the user's physical activity level
type ActivityLevel string

const (
	ActivitySedentary     ActivityLevel = "sedentary"      // Little or no exercise
	ActivityLightlyActive ActivityLevel = "lightly_active" // Light exercise 1-3 days/week
	ActivityActive        ActivityLevel = "active"         // Moderate exercise 3-5 days/week
	ActivityVeryActive    ActivityLevel = "very_active"    // Hard exercise 6-7 days/week
)

// ActivityMultipliers maps activity levels to TDEE calculation multipliers
var ActivityMultipliers = map[ActivityLevel]float64{
	ActivitySedentary:     1.2,
	ActivityLightlyActive: 1.375,
	ActivityActive:        1.55,
	ActivityVeryActive:    1.725,
}

// Sex represents biological sex for TDEE calculations
type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

// DayOfWeek represents days for custom daily goals
type DayOfWeek string

const (
	DayMonday    DayOfWeek = "monday"
	DayTuesday   DayOfWeek = "tuesday"
	DayWednesday DayOfWeek = "wednesday"
	DayThursday  DayOfWeek = "thursday"
	DayFriday    DayOfWeek = "friday"
	DaySaturday  DayOfWeek = "saturday"
	DaySunday    DayOfWeek = "sunday"
)

// UserGoals represents the nutrition goals for a user
type UserGoals struct {
	ID               int64      `json:"id" db:"id"`
	UserID           int64      `json:"user_id" db:"user_id"`
	DailyCalories    int        `json:"daily_calories" db:"daily_calories"`
	ProteinGrams     int        `json:"protein_grams" db:"protein_grams"`
	CarbsGrams       int        `json:"carbs_grams" db:"carbs_grams"`
	FatGrams         int        `json:"fat_grams" db:"fat_grams"`
	AutoCalculate    bool       `json:"auto_calculate" db:"auto_calculate"`
	ActivityLevel    string     `json:"activity_level" db:"activity_level"`
	TargetWeight     float64    `json:"target_weight,omitempty" db:"target_weight"`
	TargetDate       time.Time  `json:"target_date,omitempty" db:"target_date"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	LastCalculatedAt *time.Time `json:"last_calculated_at,omitempty" db:"last_calculated_at"`
}

// DailyGoal represents per-day-of-week custom goals
type DailyGoal struct {
	ID            int64     `json:"id" db:"id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	DayOfWeek     string    `json:"day_of_week" db:"day_of_week"`
	DailyCalories int       `json:"daily_calories" db:"daily_calories"`
	ProteinGrams  int       `json:"protein_grams" db:"protein_grams"`
	CarbsGrams    int       `json:"carbs_grams" db:"carbs_grams"`
	FatGrams      int       `json:"fat_grams" db:"fat_grams"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TDEEInputs represents the input data for TDEE calculation
type TDEEInputs struct {
	Age           int           `json:"age" validate:"required,min=13,max=120"`
	Sex           Sex           `json:"sex" validate:"required,oneof=male female"`
	HeightCM      float64       `json:"height_cm" validate:"required,min=100,max=250"`
	WeightKG      float64       `json:"weight_kg" validate:"required,min=30,max=500"`
	ActivityLevel ActivityLevel `json:"activity_level" validate:"required,oneof=sedentary lightly_active active very_active"`
}

// TDEEResult represents the calculated TDEE and macro breakdown
type TDEEResult struct {
	BMR           float64    `json:"bmr"`            // Basal Metabolic Rate
	TDEE          float64    `json:"tdee"`           // Total Daily Energy Expenditure
	DailyCalories int        `json:"daily_calories"` // Rounded TDEE
	ProteinGrams  int        `json:"protein_grams"`  // 2g per kg bodyweight
	FatGrams      int        `json:"fat_grams"`      // 25% of calories
	CarbsGrams    int        `json:"carbs_grams"`    // Remainder
	Inputs        TDEEInputs `json:"inputs"`         // Echo back inputs for confirmation
}

// UpdateGoalsRequest represents a request to update user goals
type UpdateGoalsRequest struct {
	DailyCalories *int    `json:"daily_calories,omitempty" validate:"omitempty,min=1000,max=5000"`
	ProteinGrams  *int    `json:"protein_grams,omitempty" validate:"omitempty,min=50,max=300"`
	CarbsGrams    *int    `json:"carbs_grams,omitempty" validate:"omitempty,min=50,max=500"`
	FatGrams      *int    `json:"fat_grams,omitempty" validate:"omitempty,min=20,max=200"`
	AutoCalculate *bool   `json:"auto_calculate,omitempty"`
	ActivityLevel *string `json:"activity_level,omitempty" validate:"omitempty,oneof=sedentary lightly_active active very_active"`
}

// GoalsResponse represents the response with current goals and daily overrides
type GoalsResponse struct {
	Goals       UserGoals               `json:"goals"`
	DailyGoals  map[DayOfWeek]DailyGoal `json:"daily_goals,omitempty"`
	CurrentDay  DayOfWeek               `json:"current_day"`
	ActiveGoals NutritionTargets        `json:"active_goals"` // The goals active for today
}

// NutritionTargets represents the active nutrition targets
type NutritionTargets struct {
	DailyCalories int    `json:"daily_calories"`
	ProteinGrams  int    `json:"protein_grams"`
	CarbsGrams    int    `json:"carbs_grams"`
	FatGrams      int    `json:"fat_grams"`
	Source        string `json:"source"` // "default", "daily_override", "auto_calculated"
}

// SetDailyGoalRequest represents a request to set a day-specific goal
type SetDailyGoalRequest struct {
	DailyCalories int `json:"daily_calories" validate:"required,min=1000,max=5000"`
	ProteinGrams  int `json:"protein_grams" validate:"required,min=50,max=300"`
	CarbsGrams    int `json:"carbs_grams" validate:"required,min=50,max=500"`
	FatGrams      int `json:"fat_grams" validate:"required,min=20,max=200"`
}

// Validate validates the TDEE inputs
func (t *TDEEInputs) Validate() error {
	if t.Age < 13 || t.Age > 120 {
		return ErrInvalidAge
	}
	if t.HeightCM < 100 || t.HeightCM > 250 {
		return ErrInvalidHeight
	}
	if t.WeightKG < 30 || t.WeightKG > 500 {
		return ErrInvalidWeight
	}
	if t.Sex != SexMale && t.Sex != SexFemale {
		return ErrInvalidSex
	}
	if _, ok := ActivityMultipliers[t.ActivityLevel]; !ok {
		return ErrInvalidActivityLevel
	}
	return nil
}

// Validate validates the update goals request
func (r *UpdateGoalsRequest) Validate() error {
	if r.DailyCalories != nil && (*r.DailyCalories < 1000 || *r.DailyCalories > 5000) {
		return ErrInvalidCalories
	}
	if r.ProteinGrams != nil && (*r.ProteinGrams < 50 || *r.ProteinGrams > 300) {
		return ErrInvalidProtein
	}
	if r.CarbsGrams != nil && (*r.CarbsGrams < 50 || *r.CarbsGrams > 500) {
		return ErrInvalidCarbs
	}
	if r.FatGrams != nil && (*r.FatGrams < 20 || *r.FatGrams > 200) {
		return ErrInvalidFat
	}
	if r.ActivityLevel != nil {
		level := ActivityLevel(*r.ActivityLevel)
		if _, ok := ActivityMultipliers[level]; !ok {
			return ErrInvalidActivityLevel
		}
	}
	return nil
}

// Validate validates the set daily goal request
func (r *SetDailyGoalRequest) Validate() error {
	if r.DailyCalories < 1000 || r.DailyCalories > 5000 {
		return ErrInvalidCalories
	}
	if r.ProteinGrams < 50 || r.ProteinGrams > 300 {
		return ErrInvalidProtein
	}
	if r.CarbsGrams < 50 || r.CarbsGrams > 500 {
		return ErrInvalidCarbs
	}
	if r.FatGrams < 20 || r.FatGrams > 200 {
		return ErrInvalidFat
	}
	return nil
}

// IsValidDayOfWeek checks if a string is a valid day of week
func IsValidDayOfWeek(day string) bool {
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	return validDays[day]
}
