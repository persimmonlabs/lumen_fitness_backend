// Package trajectory provides weight prediction and trajectory analysis.
//
// This package calculates weight predictions based on current trends,
// calorie intake, and goal settings. It uses linear regression and
// energy balance calculations to estimate future weight.
package trajectory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

// Service handles weight trajectory calculations.
type Service struct {
	logger *slog.Logger
	config Config
}

// Config contains configuration for trajectory calculations.
type Config struct {
	CaloriesPerKgFat   float64 // Calories in 1kg of fat (typically 7700)
	MinDataPoints      int     // Minimum weight entries needed for prediction
	PredictionDays     int     // Default prediction horizon
	ConfidenceInterval float64 // Confidence interval (0.95 = 95%)
}

// DefaultConfig returns the default trajectory service configuration.
func DefaultConfig() Config {
	return Config{
		CaloriesPerKgFat:   7700,
		MinDataPoints:      7,
		PredictionDays:     30,
		ConfidenceInterval: 0.95,
	}
}

// NewService creates a new trajectory service instance.
func NewService(logger *slog.Logger, config Config) *Service {
	return &Service{
		logger: logger,
		config: config,
	}
}

// WeightEntry represents a historical weight measurement.
type WeightEntry struct {
	Date   time.Time
	Weight float64
}

// CalorieEntry represents daily calorie intake.
type CalorieEntry struct {
	Date     time.Time
	Calories float64
}

// PredictionRequest contains data needed for trajectory calculation.
type PredictionRequest struct {
	UserID           uuid.UUID
	CurrentWeight    float64
	GoalWeight       float64
	TargetDate       *time.Time
	WeightHistory    []WeightEntry
	CalorieHistory   []CalorieEntry
	DailyGoalCalories float64
	TDEE             float64 // Total Daily Energy Expenditure
	PredictionDays   int     // 0 uses default
}

// PredictionPoint represents a predicted weight at a specific date.
type PredictionPoint struct {
	Date              time.Time `json:"date"`
	PredictedWeight   float64   `json:"predicted_weight"`
	LowerBound        float64   `json:"lower_bound"`
	UpperBound        float64   `json:"upper_bound"`
	DailyCalorieGoal  float64   `json:"daily_calorie_goal"`
}

// PredictionResponse contains the trajectory prediction results.
type PredictionResponse struct {
	Predictions       []PredictionPoint `json:"predictions"`
	CurrentTrend      float64           `json:"current_trend_kg_per_week"`
	RequiredTrend     float64           `json:"required_trend_kg_per_week"`
	EstimatedGoalDate *time.Time        `json:"estimated_goal_date"`
	OnTrack           bool              `json:"on_track"`
	Confidence        float64           `json:"confidence"`
	Message           string            `json:"message"`
}

// Calculate generates weight trajectory predictions.
func (s *Service) Calculate(ctx context.Context, req *PredictionRequest) (*PredictionResponse, error) {
	// Validate input
	if req.CurrentWeight <= 0 {
		return nil, fmt.Errorf("current weight must be positive")
	}
	if req.TDEE <= 0 {
		return nil, fmt.Errorf("TDEE must be positive")
	}

	// Determine prediction horizon
	predictionDays := req.PredictionDays
	if predictionDays <= 0 {
		predictionDays = s.config.PredictionDays
	}

	// Calculate current trend from weight history
	currentTrend := s.calculateWeightTrend(req.WeightHistory)

	// Calculate required trend to reach goal by target date
	var requiredTrend float64
	var estimatedGoalDate *time.Time

	if req.TargetDate != nil {
		daysToGoal := time.Until(*req.TargetDate).Hours() / 24
		if daysToGoal > 0 {
			weightToLose := req.CurrentWeight - req.GoalWeight
			requiredTrend = (weightToLose / daysToGoal) * 7 // kg per week
		}
	}

	// Calculate estimated goal date based on current trend
	if currentTrend != 0 {
		weightToLose := req.CurrentWeight - req.GoalWeight
		daysNeeded := (weightToLose / currentTrend) * 7
		goalDate := time.Now().Add(time.Duration(daysNeeded*24) * time.Hour)
		estimatedGoalDate = &goalDate
	}

	// Generate predictions
	predictions := make([]PredictionPoint, 0, predictionDays)
	currentDate := time.Now()
	currentPredictedWeight := req.CurrentWeight

	for i := 0; i < predictionDays; i++ {
		date := currentDate.Add(time.Duration(i) * 24 * time.Hour)

		// Calculate daily calorie deficit/surplus
		calorieBalance := req.DailyGoalCalories - req.TDEE

		// Convert to weight change (kg)
		// Negative balance = weight loss, positive = weight gain
		dailyWeightChange := calorieBalance / s.config.CaloriesPerKgFat

		// Apply weight change
		currentPredictedWeight += dailyWeightChange

		// Calculate confidence bounds (wider as we predict further out)
		uncertaintyFactor := 1.0 + (float64(i) / float64(predictionDays) * 0.1)
		stdDev := 0.5 * uncertaintyFactor // kg
		margin := stdDev * 1.96 // 95% confidence interval

		predictions = append(predictions, PredictionPoint{
			Date:             date,
			PredictedWeight:  roundToDecimal(currentPredictedWeight, 2),
			LowerBound:       roundToDecimal(currentPredictedWeight-margin, 2),
			UpperBound:       roundToDecimal(currentPredictedWeight+margin, 2),
			DailyCalorieGoal: req.DailyGoalCalories,
		})
	}

	// Determine if user is on track
	onTrack := false
	message := "Insufficient data to determine progress"

	if req.TargetDate != nil && currentTrend != 0 && requiredTrend != 0 {
		// Consider on track if within 20% of required trend
		trendRatio := math.Abs(currentTrend / requiredTrend)
		onTrack = trendRatio >= 0.8 && trendRatio <= 1.2

		if onTrack {
			message = "You are on track to reach your goal!"
		} else if currentTrend < requiredTrend {
			message = "Current progress is slower than needed. Consider reducing calorie intake."
		} else {
			message = "Current progress is faster than needed. You may want to adjust your goal."
		}
	}

	// Calculate confidence based on data availability
	confidence := s.calculateConfidence(req.WeightHistory, req.CalorieHistory)

	s.logger.Info("trajectory calculated",
		slog.String("user_id", req.UserID.String()),
		slog.Float64("current_trend", currentTrend),
		slog.Float64("required_trend", requiredTrend),
		slog.Bool("on_track", onTrack),
		slog.Float64("confidence", confidence),
	)

	return &PredictionResponse{
		Predictions:       predictions,
		CurrentTrend:      roundToDecimal(currentTrend, 3),
		RequiredTrend:     roundToDecimal(requiredTrend, 3),
		EstimatedGoalDate: estimatedGoalDate,
		OnTrack:           onTrack,
		Confidence:        confidence,
		Message:           message,
	}, nil
}

// calculateWeightTrend calculates the current weight trend (kg/day) using linear regression.
func (s *Service) calculateWeightTrend(history []WeightEntry) float64 {
	if len(history) < s.config.MinDataPoints {
		return 0
	}

	// Use last 30 days of data
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	var recentHistory []WeightEntry
	for _, entry := range history {
		if entry.Date.After(cutoff) {
			recentHistory = append(recentHistory, entry)
		}
	}

	if len(recentHistory) < 2 {
		return 0
	}

	// Simple linear regression
	n := float64(len(recentHistory))
	var sumX, sumY, sumXY, sumX2 float64
	baseTime := recentHistory[0].Date

	for _, entry := range recentHistory {
		x := entry.Date.Sub(baseTime).Hours() / 24 // days since first entry
		y := entry.Weight
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (weight change per day)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	return slope
}

// calculateConfidence determines confidence in predictions based on data quality.
func (s *Service) calculateConfidence(weightHistory []WeightEntry, calorieHistory []CalorieEntry) float64 {
	confidence := 0.0

	// Factor 1: Amount of weight data (0-40 points)
	if len(weightHistory) >= s.config.MinDataPoints {
		dataPoints := float64(len(weightHistory))
		confidence += math.Min(40, dataPoints*2)
	}

	// Factor 2: Consistency of weight measurements (0-30 points)
	// Check if measurements are regular (not implemented for brevity)
	confidence += 20

	// Factor 3: Amount of calorie data (0-30 points)
	if len(calorieHistory) >= 14 {
		confidence += 30
	} else if len(calorieHistory) >= 7 {
		confidence += 20
	} else if len(calorieHistory) > 0 {
		confidence += 10
	}

	return math.Min(100, confidence) / 100.0
}

// roundToDecimal rounds a float to specified decimal places.
func roundToDecimal(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier
}
