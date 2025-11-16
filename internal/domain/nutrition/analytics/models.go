package analytics

import (
	"time"
)

// DailyTotals represents aggregated nutrition data for a single day
type DailyTotals struct {
	Date         time.Time          `json:"date"`
	Timezone     string             `json:"timezone"`
	TotalCalories float64           `json:"total_calories"`
	TotalProtein  float64           `json:"total_protein"`
	TotalCarbs    float64           `json:"total_carbs"`
	TotalFat      float64           `json:"total_fat"`
	TotalFiber    float64           `json:"total_fiber"`
	MealBreakdown []MealBreakdown   `json:"meal_breakdown"`
	GoalComparison *GoalComparison  `json:"goal_comparison,omitempty"`
}

// MealBreakdown represents nutrition totals for a specific meal type
type MealBreakdown struct {
	MealType      string  `json:"meal_type"`
	Calories      float64 `json:"calories"`
	Protein       float64 `json:"protein"`
	Carbs         float64 `json:"carbs"`
	Fat           float64 `json:"fat"`
	Fiber         float64 `json:"fiber"`
	ItemCount     int     `json:"item_count"`
}

// GoalComparison represents how actual intake compares to user goals
type GoalComparison struct {
	CaloriesGoal     float64 `json:"calories_goal"`
	ProteinGoal      float64 `json:"protein_goal"`
	CarbsGoal        float64 `json:"carbs_goal"`
	FatGoal          float64 `json:"fat_goal"`
	FiberGoal        float64 `json:"fiber_goal"`
	CaloriesPercent  float64 `json:"calories_percent"`
	ProteinPercent   float64 `json:"protein_percent"`
	CarbsPercent     float64 `json:"carbs_percent"`
	FatPercent       float64 `json:"fat_percent"`
	FiberPercent     float64 `json:"fiber_percent"`
}

// WeeklyTrends represents nutrition trends over a 7-day period
type WeeklyTrends struct {
	StartDate        time.Time     `json:"start_date"`
	EndDate          time.Time     `json:"end_date"`
	Timezone         string        `json:"timezone"`
	DailyData        []DailyTotals `json:"daily_data"`
	Averages         Averages      `json:"averages"`
	ConsistencyScore float64       `json:"consistency_score"` // 0-100 based on logging frequency
	Streak           int           `json:"streak"`            // Consecutive days logged
}

// Averages represents average nutrition values over a period
type Averages struct {
	AvgCalories float64 `json:"avg_calories"`
	AvgProtein  float64 `json:"avg_protein"`
	AvgCarbs    float64 `json:"avg_carbs"`
	AvgFat      float64 `json:"avg_fat"`
	AvgFiber    float64 `json:"avg_fiber"`
	DaysLogged  int     `json:"days_logged"`
}

// MacroDistribution represents the percentage breakdown of macronutrients
type MacroDistribution struct {
	Date             time.Time `json:"date"`
	Timezone         string    `json:"timezone"`
	ProteinPercent   float64   `json:"protein_percent"`
	CarbsPercent     float64   `json:"carbs_percent"`
	FatPercent       float64   `json:"fat_percent"`
	ProteinCalories  float64   `json:"protein_calories"`
	CarbsCalories    float64   `json:"carbs_calories"`
	FatCalories      float64   `json:"fat_calories"`
	TotalCalories    float64   `json:"total_calories"`
}

// NutritionInsights represents actionable insights based on nutrition data
type NutritionInsights struct {
	Period           string    `json:"period"` // "daily", "weekly", "custom"
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	Timezone         string    `json:"timezone"`
	Insights         []Insight `json:"insights"`
	OverallScore     float64   `json:"overall_score"` // 0-100 health score
}

// Insight represents a single actionable insight
type Insight struct {
	Type        string  `json:"type"`        // "success", "warning", "info", "alert"
	Category    string  `json:"category"`    // "calories", "protein", "carbs", "fat", "fiber", "consistency"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`      // "high", "medium", "low"
	Score       float64 `json:"score"`       // 0-100 for this specific metric
}

// DateRangeStats represents nutrition statistics over a custom date range
type DateRangeStats struct {
	StartDate     time.Time     `json:"start_date"`
	EndDate       time.Time     `json:"end_date"`
	Timezone      string        `json:"timezone"`
	TotalDays     int           `json:"total_days"`
	DaysLogged    int           `json:"days_logged"`
	Averages      Averages      `json:"averages"`
	Totals        Totals        `json:"totals"`
	DailyData     []DailyTotals `json:"daily_data,omitempty"`
}

// Totals represents cumulative nutrition values
type Totals struct {
	TotalCalories float64 `json:"total_calories"`
	TotalProtein  float64 `json:"total_protein"`
	TotalCarbs    float64 `json:"total_carbs"`
	TotalFat      float64 `json:"total_fat"`
	TotalFiber    float64 `json:"total_fiber"`
}

// UserGoals represents user's nutrition targets
type UserGoals struct {
	UserID        string  `json:"user_id"`
	CaloriesGoal  float64 `json:"calories_goal"`
	ProteinGoal   float64 `json:"protein_goal"`
	CarbsGoal     float64 `json:"carbs_goal"`
	FatGoal       float64 `json:"fat_goal"`
	FiberGoal     float64 `json:"fiber_goal"`
}

// DailyAnalyticsResponse represents complete daily analytics data
type DailyAnalyticsResponse struct {
	Date          time.Time          `json:"date"`
	Timezone      string             `json:"timezone"`
	Totals        DailyTotals        `json:"totals"`
	Goals         *UserGoals         `json:"goals,omitempty"`
	Progress      *GoalComparison    `json:"progress,omitempty"`
	Distribution  MacroDistribution  `json:"distribution"`
	MealBreakdown []MealBreakdown    `json:"meal_breakdown"`
}

// WeeklyAnalyticsResponse represents complete weekly analytics data
type WeeklyAnalyticsResponse struct {
	StartDate     time.Time      `json:"start_date"`
	EndDate       time.Time      `json:"end_date"`
	Timezone      string         `json:"timezone"`
	Trend         WeeklyTrends   `json:"trend"`
	Goals         *UserGoals     `json:"goals,omitempty"`
	AvgProgress   *GoalComparison `json:"avg_progress,omitempty"`
	TopFoods      []TopFood      `json:"top_foods"`
	Streak        StreakInfo     `json:"streak"`
}

// TopFood represents frequently logged food items
type TopFood struct {
	FoodName      string  `json:"food_name"`
	LogCount      int     `json:"log_count"`
	TotalCalories float64 `json:"total_calories"`
	AvgServing    float64 `json:"avg_serving"`
}

// StreakInfo represents consecutive logging days
type StreakInfo struct {
	CurrentStreak int       `json:"current_streak"`
	LongestStreak int       `json:"longest_streak"`
	LastLogDate   time.Time `json:"last_log_date"`
	TotalDays     int       `json:"total_days"`
}

// TrendsResponse represents custom date range analytics
type TrendsResponse struct {
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	Timezone     string        `json:"timezone"`
	DailyTotals  []DailyTotals `json:"daily_totals"`
	Averages     Averages      `json:"averages"`
	MinValues    DailyTotals   `json:"min_values"`
	MaxValues    DailyTotals   `json:"max_values"`
}

// DistributionResponse represents macro distribution analytics
type DistributionResponse struct {
	Date          time.Time         `json:"date"`
	Timezone      string            `json:"timezone"`
	Distribution  MacroDistribution `json:"distribution"`
	Goals         *UserGoals        `json:"goals,omitempty"`
	Totals        DailyTotals       `json:"totals"`
}

// ProgressResponse represents goal progress analytics
type ProgressResponse struct {
	Date     time.Time       `json:"date"`
	Timezone string          `json:"timezone"`
	Progress GoalComparison  `json:"progress"`
	Goals    *UserGoals      `json:"goals,omitempty"`
	Totals   DailyTotals     `json:"totals"`
}

// Trajectory represents weight trajectory prediction
type Trajectory struct {
	PredictedWeight float64   `json:"predicted_weight"`
	OnTrack         bool      `json:"on_track"`
	ProjectedDate   time.Time `json:"projected_date"`
	Confidence      float64   `json:"confidence"`      // 0-1 confidence score
	DaysAnalyzed    int       `json:"days_analyzed"`   // Number of days used in prediction
	CurrentWeight   float64   `json:"current_weight"`
	TargetWeight    float64   `json:"target_weight"`
	WeeklyRate      float64   `json:"weekly_rate"`     // kg per week
}
