package analytics

import "time"

// Analytics calculation constants
// These values control the behavior of analytics calculations and should be
// tuned based on user feedback and data analysis.
const (
	// WeightTrajectoryWindowDays is the number of days to look back when
	// calculating weight trajectory trends. Default: 60 days (2 months)
	WeightTrajectoryWindowDays = 60

	// MinimumWeightDataPoints is the minimum number of weight entries
	// required to calculate a meaningful trajectory. Default: 7 (1 week)
	MinimumWeightDataPoints = 7

	// MovingAverageWindowDays is the window size for smoothing weight data
	// using a moving average. This helps reduce noise from daily fluctuations.
	// Default: 7 days (1 week)
	MovingAverageWindowDays = 7

	// StreakRecursionLimit is the maximum number of days to check when
	// calculating logging streaks. This prevents excessive database load
	// for long-term users. Default: 365 days (1 year)
	StreakRecursionLimit = 365

	// CalorieDeficitThreshold is the minimum daily calorie deficit (in kcal)
	// to be considered "significant" for weight loss insights.
	// Default: 100 kcal/day
	CalorieDeficitThreshold = 100

	// CalorieSurplusThreshold is the minimum daily calorie surplus (in kcal)
	// to be considered "significant" for weight gain insights.
	// Default: 100 kcal/day
	CalorieSurplusThreshold = 100

	// NutritionComplianceThreshold is the percentage of days that must meet
	// nutrition goals to be considered "consistent". Default: 80% (0.80)
	NutritionComplianceThreshold = 0.80

	// ProteinTargetPercentage is the recommended percentage of total calories
	// from protein for general health. Default: 25% (0.25)
	ProteinTargetPercentage = 0.25

	// CarbsTargetPercentage is the recommended percentage of total calories
	// from carbohydrates for general health. Default: 45% (0.45)
	CarbsTargetPercentage = 0.45

	// FatTargetPercentage is the recommended percentage of total calories
	// from fat for general health. Default: 30% (0.30)
	FatTargetPercentage = 0.30

	// MinMealsPerDayForInsights is the minimum number of meals logged per day
	// to generate meaningful nutrition insights. Default: 2 meals
	MinMealsPerDayForInsights = 2

	// WeeklyAnalysisDays is the number of days in a week for weekly analytics
	WeeklyAnalysisDays = 7

	// DailyAnalysisHours is the number of hours in a day for daily analytics
	DailyAnalysisHours = 24
)

// Time constants for analytics calculations
const (
	// DefaultQueryTimeout is the maximum time allowed for analytics queries
	DefaultQueryTimeout = 30 * time.Second

	// CacheExpiration is how long to cache analytics results
	CacheExpiration = 5 * time.Minute
)

// Insight message templates
const (
	InsightTypeStreak              = "streak"
	InsightTypeCalorieDeficit      = "calorie_deficit"
	InsightTypeCalorieSurplus      = "calorie_surplus"
	InsightTypeProteinIntake       = "protein_intake"
	InsightTypeConsistency         = "consistency"
	InsightTypeWeightProgress      = "weight_progress"
	InsightTypeMacroBalance        = "macro_balance"
	InsightTypeHydration           = "hydration"
	InsightTypeMealTiming          = "meal_timing"
	InsightTypeNutrientDeficiency  = "nutrient_deficiency"
)

// Priority levels for insights
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)
