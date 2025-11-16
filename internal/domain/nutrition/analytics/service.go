package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/goals"
	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/weight"
	"gonum.org/v1/gonum/stat"
)

// Service handles business logic for nutrition analytics
type Service interface {
	GetDailyAnalytics(ctx context.Context, userID string, date time.Time, timezone string) (*DailyAnalyticsResponse, error)
	GetWeeklyTrends(ctx context.Context, userID string, startDate time.Time, timezone string) (*WeeklyAnalyticsResponse, error)
	GetDateRangeStats(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*TrendsResponse, error)
	GetMacroDistribution(ctx context.Context, userID string, date time.Time, timezone string) (*MacroDistribution, error)
	GetNutritionInsights(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*NutritionInsights, error)
	CalculateTrajectory(ctx context.Context, userID string) (*Trajectory, error)
}

type service struct {
	repo        Repository
	weightRepo  weight.Repository
	goalsRepo   goals.Repository
}

// NewService creates a new analytics service
func NewService(repo Repository, weightRepo weight.Repository, goalsRepo goals.Repository) Service {
	return &service{
		repo:       repo,
		weightRepo: weightRepo,
		goalsRepo:  goalsRepo,
	}
}

// GetDailyAnalytics retrieves daily nutrition with goal comparison
func (s *service) GetDailyAnalytics(ctx context.Context, userID string, date time.Time, timezone string) (*DailyAnalyticsResponse, error) {
	dailyTotals, err := s.repo.GetDailyNutrition(ctx, userID, date, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily nutrition: %w", err)
	}

	// Get user goals for comparison
	goals, err := s.repo.GetUserGoals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user goals: %w", err)
	}

	// Calculate goal comparison
	goalComparison := s.calculateGoalComparison(dailyTotals, goals)

	// Get macro distribution
	distribution, err := s.GetMacroDistribution(ctx, userID, date, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get macro distribution: %w", err)
	}

	return &DailyAnalyticsResponse{
		Date:          date,
		Timezone:      timezone,
		Totals:        *dailyTotals,
		Goals:         goals,
		Progress:      goalComparison,
		Distribution:  *distribution,
		MealBreakdown: dailyTotals.MealBreakdown,
	}, nil
}

// GetWeeklyTrends retrieves 7-day nutrition trends
func (s *service) GetWeeklyTrends(ctx context.Context, userID string, startDate time.Time, timezone string) (*WeeklyAnalyticsResponse, error) {
	endDate := startDate.AddDate(0, 0, 6)

	// Get daily data for the week
	dailyData, err := s.repo.GetDateRangeNutrition(ctx, userID, startDate, endDate, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range nutrition: %w", err)
	}

	// Get goals for each day
	goals, err := s.repo.GetUserGoals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user goals: %w", err)
	}

	// Add goal comparison to each day
	for i := range dailyData {
		dailyData[i].GoalComparison = s.calculateGoalComparison(&dailyData[i], goals)
	}

	// Calculate averages
	averages := s.calculateAverages(dailyData)

	// Calculate consistency score
	consistencyScore := s.calculateConsistencyScore(dailyData, 7)

	// Get logging streak
	streak, err := s.repo.GetLoggingStreak(ctx, userID, endDate, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get logging streak: %w", err)
	}

	// Build weekly trend
	weeklyTrend := WeeklyTrends{
		StartDate:        startDate,
		EndDate:          endDate,
		Timezone:         timezone,
		DailyData:        dailyData,
		Averages:         averages,
		ConsistencyScore: consistencyScore,
		Streak:           streak,
	}

	// Get top foods (placeholder - implement if needed)
	topFoods := make([]TopFood, 0)

	// Build streak info
	streakInfo := StreakInfo{
		CurrentStreak: streak,
		LongestStreak: streak,
		LastLogDate:   endDate,
		TotalDays:     7,
	}

	return &WeeklyAnalyticsResponse{
		StartDate:   startDate,
		EndDate:     endDate,
		Timezone:    timezone,
		Trend:       weeklyTrend,
		Goals:       goals,
		AvgProgress: s.calculateGoalComparisonFromAverages(averages, goals),
		TopFoods:    topFoods,
		Streak:      streakInfo,
	}, nil
}

// GetDateRangeStats retrieves statistics for a custom date range
func (s *service) GetDateRangeStats(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*TrendsResponse, error) {
	dailyData, err := s.repo.GetDateRangeNutrition(ctx, userID, startDate, endDate, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range nutrition: %w", err)
	}

	averages := s.calculateAverages(dailyData)
	minValues := s.calculateMinValues(dailyData)
	maxValues := s.calculateMaxValues(dailyData)

	return &TrendsResponse{
		StartDate:   startDate,
		EndDate:     endDate,
		Timezone:    timezone,
		DailyTotals: dailyData,
		Averages:    averages,
		MinValues:   minValues,
		MaxValues:   maxValues,
	}, nil
}

// GetMacroDistribution calculates macro percentage breakdown
func (s *service) GetMacroDistribution(ctx context.Context, userID string, date time.Time, timezone string) (*MacroDistribution, error) {
	dailyTotals, err := s.repo.GetDailyNutrition(ctx, userID, date, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily nutrition: %w", err)
	}

	// Calculate calories from each macro (protein: 4 cal/g, carbs: 4 cal/g, fat: 9 cal/g)
	proteinCalories := dailyTotals.TotalProtein * 4
	carbsCalories := dailyTotals.TotalCarbs * 4
	fatCalories := dailyTotals.TotalFat * 9

	totalMacroCalories := proteinCalories + carbsCalories + fatCalories

	distribution := &MacroDistribution{
		Date:            date,
		Timezone:        timezone,
		ProteinCalories: proteinCalories,
		CarbsCalories:   carbsCalories,
		FatCalories:     fatCalories,
		TotalCalories:   dailyTotals.TotalCalories,
	}

	// Calculate percentages (avoid division by zero)
	if totalMacroCalories > 0 {
		distribution.ProteinPercent = round((proteinCalories / totalMacroCalories) * 100)
		distribution.CarbsPercent = round((carbsCalories / totalMacroCalories) * 100)
		distribution.FatPercent = round((fatCalories / totalMacroCalories) * 100)
	}

	return distribution, nil
}

// GetNutritionInsights generates actionable insights from nutrition data
func (s *service) GetNutritionInsights(ctx context.Context, userID string, startDate, endDate time.Time, timezone string) (*NutritionInsights, error) {
	stats, err := s.GetDateRangeStats(ctx, userID, startDate, endDate, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range stats: %w", err)
	}

	goals, err := s.repo.GetUserGoals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user goals: %w", err)
	}

	insights := make([]Insight, 0)

	// Calculate total days
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1

	// Consistency insights
	consistencyScore := s.calculateConsistencyScore(stats.DailyTotals, totalDays)
	daysLogged := stats.Averages.DaysLogged
	insights = append(insights, s.generateConsistencyInsights(consistencyScore, daysLogged, totalDays)...)

	// Calorie insights
	insights = append(insights, s.generateCalorieInsights(stats.Averages, goals)...)

	// Macro insights
	insights = append(insights, s.generateMacroInsights(stats.Averages, goals)...)

	// Fiber insights
	insights = append(insights, s.generateFiberInsights(stats.Averages, goals)...)

	// Calculate overall health score
	overallScore := s.calculateOverallScore(insights)

	period := "custom"
	if totalDays == 1 {
		period = "daily"
	} else if totalDays == 7 {
		period = "weekly"
	}

	return &NutritionInsights{
		Period:       period,
		StartDate:    startDate,
		EndDate:      endDate,
		Timezone:     timezone,
		Insights:     insights,
		OverallScore: overallScore,
	}, nil
}

// Helper functions

func (s *service) calculateGoalComparison(daily *DailyTotals, goals *UserGoals) *GoalComparison {
	return &GoalComparison{
		CaloriesGoal:    goals.CaloriesGoal,
		ProteinGoal:     goals.ProteinGoal,
		CarbsGoal:       goals.CarbsGoal,
		FatGoal:         goals.FatGoal,
		FiberGoal:       goals.FiberGoal,
		CaloriesPercent: safePercent(daily.TotalCalories, goals.CaloriesGoal),
		ProteinPercent:  safePercent(daily.TotalProtein, goals.ProteinGoal),
		CarbsPercent:    safePercent(daily.TotalCarbs, goals.CarbsGoal),
		FatPercent:      safePercent(daily.TotalFat, goals.FatGoal),
		FiberPercent:    safePercent(daily.TotalFiber, goals.FiberGoal),
	}
}

func (s *service) calculateAverages(dailyData []DailyTotals) Averages {
	if len(dailyData) == 0 {
		return Averages{}
	}

	var totalCalories, totalProtein, totalCarbs, totalFat, totalFiber float64
	daysLogged := 0

	for _, day := range dailyData {
		if day.TotalCalories > 0 {
			daysLogged++
			totalCalories += day.TotalCalories
			totalProtein += day.TotalProtein
			totalCarbs += day.TotalCarbs
			totalFat += day.TotalFat
			totalFiber += day.TotalFiber
		}
	}

	if daysLogged == 0 {
		return Averages{DaysLogged: 0}
	}

	return Averages{
		AvgCalories: round(totalCalories / float64(daysLogged)),
		AvgProtein:  round(totalProtein / float64(daysLogged)),
		AvgCarbs:    round(totalCarbs / float64(daysLogged)),
		AvgFat:      round(totalFat / float64(daysLogged)),
		AvgFiber:    round(totalFiber / float64(daysLogged)),
		DaysLogged:  daysLogged,
	}
}

func (s *service) calculateTotals(dailyData []DailyTotals) Totals {
	var totals Totals

	for _, day := range dailyData {
		totals.TotalCalories += day.TotalCalories
		totals.TotalProtein += day.TotalProtein
		totals.TotalCarbs += day.TotalCarbs
		totals.TotalFat += day.TotalFat
		totals.TotalFiber += day.TotalFiber
	}

	return totals
}

func (s *service) calculateConsistencyScore(dailyData []DailyTotals, totalDays int) float64 {
	if totalDays == 0 {
		return 0
	}

	daysLogged := 0
	for _, day := range dailyData {
		if day.TotalCalories > 0 {
			daysLogged++
		}
	}

	return round((float64(daysLogged) / float64(totalDays)) * 100)
}

func (s *service) generateConsistencyInsights(score float64, daysLogged, totalDays int) []Insight {
	insights := make([]Insight, 0)

	if score >= 80 {
		insights = append(insights, Insight{
			Type:        "success",
			Category:    "consistency",
			Title:       "Excellent Logging Consistency",
			Description: fmt.Sprintf("You've logged %d out of %d days (%.0f%%). Keep up the great work!", daysLogged, totalDays, score),
			Impact:      "high",
			Score:       score,
		})
	} else if score >= 50 {
		insights = append(insights, Insight{
			Type:        "warning",
			Category:    "consistency",
			Title:       "Moderate Logging Consistency",
			Description: fmt.Sprintf("You've logged %d out of %d days (%.0f%%). Try to log more consistently for better insights.", daysLogged, totalDays, score),
			Impact:      "medium",
			Score:       score,
		})
	} else {
		insights = append(insights, Insight{
			Type:        "alert",
			Category:    "consistency",
			Title:       "Low Logging Consistency",
			Description: fmt.Sprintf("You've only logged %d out of %d days (%.0f%%). Regular tracking is key to reaching your goals.", daysLogged, totalDays, score),
			Impact:      "high",
			Score:       score,
		})
	}

	return insights
}

func (s *service) generateCalorieInsights(avg Averages, goals *UserGoals) []Insight {
	insights := make([]Insight, 0)
	percent := safePercent(avg.AvgCalories, goals.CaloriesGoal)

	if percent >= 90 && percent <= 110 {
		insights = append(insights, Insight{
			Type:        "success",
			Category:    "calories",
			Title:       "On Target with Calories",
			Description: fmt.Sprintf("Your average of %.0f calories is right on track with your %.0f calorie goal.", avg.AvgCalories, goals.CaloriesGoal),
			Impact:      "high",
			Score:       100,
		})
	} else if percent < 80 {
		insights = append(insights, Insight{
			Type:        "warning",
			Category:    "calories",
			Title:       "Calories Below Target",
			Description: fmt.Sprintf("Your average of %.0f calories is %.0f%% of your goal. Consider adding nutrient-dense foods.", avg.AvgCalories, percent),
			Impact:      "high",
			Score:       percent,
		})
	} else if percent > 120 {
		insights = append(insights, Insight{
			Type:        "alert",
			Category:    "calories",
			Title:       "Calories Above Target",
			Description: fmt.Sprintf("Your average of %.0f calories is %.0f%% of your goal. Review portion sizes and food choices.", avg.AvgCalories, percent),
			Impact:      "high",
			Score:       math.Max(0, 100-(percent-100)),
		})
	}

	return insights
}

func (s *service) generateMacroInsights(avg Averages, goals *UserGoals) []Insight {
	insights := make([]Insight, 0)

	// Protein insights
	proteinPercent := safePercent(avg.AvgProtein, goals.ProteinGoal)
	if proteinPercent < 80 {
		insights = append(insights, Insight{
			Type:        "warning",
			Category:    "protein",
			Title:       "Low Protein Intake",
			Description: fmt.Sprintf("Your average protein (%.0fg) is only %.0f%% of your goal. Include more lean meats, fish, or plant-based proteins.", avg.AvgProtein, proteinPercent),
			Impact:      "medium",
			Score:       proteinPercent,
		})
	} else if proteinPercent >= 90 {
		insights = append(insights, Insight{
			Type:        "success",
			Category:    "protein",
			Title:       "Great Protein Intake",
			Description: fmt.Sprintf("Your average protein (%.0fg) meets your goal. Protein supports muscle maintenance and satiety.", avg.AvgProtein),
			Impact:      "medium",
			Score:       100,
		})
	}

	// Carbs insights
	carbsPercent := safePercent(avg.AvgCarbs, goals.CarbsGoal)
	if carbsPercent > 130 {
		insights = append(insights, Insight{
			Type:        "info",
			Category:    "carbs",
			Title:       "High Carbohydrate Intake",
			Description: fmt.Sprintf("Your average carbs (%.0fg) is %.0f%% of your goal. Focus on complex carbs and fiber-rich options.", avg.AvgCarbs, carbsPercent),
			Impact:      "low",
			Score:       math.Max(0, 100-(carbsPercent-100)/2),
		})
	}

	return insights
}

func (s *service) generateFiberInsights(avg Averages, goals *UserGoals) []Insight {
	insights := make([]Insight, 0)
	fiberPercent := safePercent(avg.AvgFiber, goals.FiberGoal)

	if fiberPercent < 70 {
		insights = append(insights, Insight{
			Type:        "alert",
			Category:    "fiber",
			Title:       "Low Fiber Intake",
			Description: fmt.Sprintf("Your average fiber (%.0fg) is only %.0f%% of your goal. Add more vegetables, fruits, and whole grains.", avg.AvgFiber, fiberPercent),
			Impact:      "high",
			Score:       fiberPercent,
		})
	} else if fiberPercent >= 90 {
		insights = append(insights, Insight{
			Type:        "success",
			Category:    "fiber",
			Title:       "Excellent Fiber Intake",
			Description: fmt.Sprintf("Your average fiber (%.0fg) meets your goal. Fiber supports digestive health and satiety.", avg.AvgFiber),
			Impact:      "medium",
			Score:       100,
		})
	}

	return insights
}

func (s *service) calculateOverallScore(insights []Insight) float64 {
	if len(insights) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, insight := range insights {
		totalScore += insight.Score
	}

	return round(totalScore / float64(len(insights)))
}

// Utility functions

func safePercent(actual, goal float64) float64 {
	if goal == 0 {
		return 0
	}
	return round((actual / goal) * 100)
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

// calculateGoalComparisonFromAverages calculates goal comparison from average values
func (s *service) calculateGoalComparisonFromAverages(avg Averages, goals *UserGoals) *GoalComparison {
	return &GoalComparison{
		CaloriesGoal:    goals.CaloriesGoal,
		ProteinGoal:     goals.ProteinGoal,
		CarbsGoal:       goals.CarbsGoal,
		FatGoal:         goals.FatGoal,
		FiberGoal:       goals.FiberGoal,
		CaloriesPercent: safePercent(avg.AvgCalories, goals.CaloriesGoal),
		ProteinPercent:  safePercent(avg.AvgProtein, goals.ProteinGoal),
		CarbsPercent:    safePercent(avg.AvgCarbs, goals.CarbsGoal),
		FatPercent:      safePercent(avg.AvgFat, goals.FatGoal),
		FiberPercent:    safePercent(avg.AvgFiber, goals.FiberGoal),
	}
}

// calculateMinValues calculates minimum values across daily data
func (s *service) calculateMinValues(dailyData []DailyTotals) DailyTotals {
	if len(dailyData) == 0 {
		return DailyTotals{}
	}

	minVals := dailyData[0]
	for _, day := range dailyData[1:] {
		if day.TotalCalories > 0 && day.TotalCalories < minVals.TotalCalories {
			minVals.TotalCalories = day.TotalCalories
		}
		if day.TotalProtein > 0 && day.TotalProtein < minVals.TotalProtein {
			minVals.TotalProtein = day.TotalProtein
		}
		if day.TotalCarbs > 0 && day.TotalCarbs < minVals.TotalCarbs {
			minVals.TotalCarbs = day.TotalCarbs
		}
		if day.TotalFat > 0 && day.TotalFat < minVals.TotalFat {
			minVals.TotalFat = day.TotalFat
		}
		if day.TotalFiber > 0 && day.TotalFiber < minVals.TotalFiber {
			minVals.TotalFiber = day.TotalFiber
		}
	}

	return minVals
}

// calculateMaxValues calculates maximum values across daily data
func (s *service) calculateMaxValues(dailyData []DailyTotals) DailyTotals {
	if len(dailyData) == 0 {
		return DailyTotals{}
	}

	maxVals := dailyData[0]
	for _, day := range dailyData[1:] {
		if day.TotalCalories > maxVals.TotalCalories {
			maxVals.TotalCalories = day.TotalCalories
		}
		if day.TotalProtein > maxVals.TotalProtein {
			maxVals.TotalProtein = day.TotalProtein
		}
		if day.TotalCarbs > maxVals.TotalCarbs {
			maxVals.TotalCarbs = day.TotalCarbs
		}
		if day.TotalFat > maxVals.TotalFat {
			maxVals.TotalFat = day.TotalFat
		}
		if day.TotalFiber > maxVals.TotalFiber {
			maxVals.TotalFiber = day.TotalFiber
		}
	}

	return maxVals
}

// CalculateTrajectory predicts weight trajectory based on historical data
func (s *service) CalculateTrajectory(ctx context.Context, userID string) (*Trajectory, error) {
	// Parse userID to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user's goal to know target weight and date
	userGoal, err := s.goalsRepo.GetByUserID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user goal: %w", err)
	}

	// Get weight entries for the last N days minimum
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -WeightTrajectoryWindowDays)

	weightEntries, err := s.weightRepo.GetByDateRange(ctx, userUUID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get weight entries: %w", err)
	}

	// Need at least minimum data points for meaningful prediction
	if len(weightEntries) < MinimumWeightDataPoints {
		return nil, fmt.Errorf("insufficient data: need at least %d weight entries", MinimumWeightDataPoints)
	}

	// Extract weights and dates
	weights := make([]float64, len(weightEntries))
	dates := make([]time.Time, len(weightEntries))
	for i, entry := range weightEntries {
		weights[i] = entry.Weight
		dates[i] = entry.MeasuredAt
	}

	// Apply moving average smoothing
	smoothedWeights := applyMovingAverage(weights, MovingAverageWindowDays)

	// Convert dates to days since first measurement for regression
	xValues := make([]float64, len(smoothedWeights))
	for i := range smoothedWeights {
		xValues[i] = float64(i)
	}

	// Calculate linear regression (y = mx + b)
	alpha, beta := stat.LinearRegression(xValues, smoothedWeights, nil, false)

	// alpha is intercept (b), beta is slope (m)
	// y = alpha + beta*x

	// Calculate current weight (most recent entry)
	currentWeight := weightEntries[len(weightEntries)-1].Weight

	// Calculate weekly rate of change
	// beta is in kg/day, so multiply by 7 for kg/week
	weeklyRate := beta * 7

	// Project to target date
	daysToTarget := int(userGoal.TargetDate.Sub(endDate).Hours() / 24)
	if daysToTarget < 0 {
		daysToTarget = 0
	}

	// Predict weight at target date
	// x value for prediction is current day count + days to target
	xPredict := float64(len(smoothedWeights)-1) + float64(daysToTarget)
	predictedWeight := alpha + beta*xPredict

	// Calculate when target weight will be reached
	// targetWeight = alpha + beta*x
	// x = (targetWeight - alpha) / beta
	var projectedDate time.Time
	if beta != 0 {
		xTarget := (userGoal.TargetWeight - alpha) / beta
		daysToReach := int(xTarget) - (len(smoothedWeights) - 1)
		if daysToReach < 0 {
			daysToReach = 0
		}
		projectedDate = endDate.AddDate(0, 0, daysToReach)
	} else {
		// No change in weight, use far future date
		projectedDate = endDate.AddDate(10, 0, 0)
	}

	// Determine if on track (within 10% margin)
	onTrack := false
	if userGoal.TargetWeight > 0 {
		// Calculate acceptable range (±10%)
		margin := math.Abs(userGoal.TargetWeight * 0.10)
		onTrack = math.Abs(predictedWeight-userGoal.TargetWeight) <= margin

		// Also check if trajectory direction is correct
		if userGoal.TargetWeight < currentWeight {
			// Want to lose weight, need negative weekly rate
			onTrack = onTrack && weeklyRate < 0
		} else if userGoal.TargetWeight > currentWeight {
			// Want to gain weight, need positive weekly rate
			onTrack = onTrack && weeklyRate > 0
		}
	}

	// Calculate confidence based on data quality
	// More data points and consistent trend = higher confidence
	confidence := calculateConfidence(len(weightEntries), weights, smoothedWeights)

	return &Trajectory{
		PredictedWeight: round(predictedWeight),
		OnTrack:         onTrack,
		ProjectedDate:   projectedDate,
		Confidence:      round(confidence),
		DaysAnalyzed:    len(weightEntries),
		CurrentWeight:   round(currentWeight),
		TargetWeight:    round(userGoal.TargetWeight),
		WeeklyRate:      round(weeklyRate),
	}, nil
}

// applyMovingAverage applies a simple moving average with the given window size
func applyMovingAverage(data []float64, window int) []float64 {
	if len(data) < window {
		window = len(data)
	}

	result := make([]float64, len(data))
	for i := range data {
		start := i - window + 1
		if start < 0 {
			start = 0
		}

		sum := 0.0
		count := 0
		for j := start; j <= i; j++ {
			sum += data[j]
			count++
		}
		result[i] = sum / float64(count)
	}

	return result
}

// calculateConfidence determines prediction confidence based on data quality
func calculateConfidence(dataPoints int, original, smoothed []float64) float64 {
	// Base confidence on number of data points (more is better)
	baseConfidence := math.Min(float64(dataPoints)/60.0, 1.0)

	// Calculate variance ratio (smoothed vs original)
	// Lower variance in smoothed data compared to original = higher confidence
	originalVar := stat.Variance(original, nil)
	smoothedVar := stat.Variance(smoothed, nil)

	varianceRatio := 1.0
	if originalVar > 0 {
		varianceRatio = smoothedVar / originalVar
		// Invert so that lower variance ratio = higher confidence
		varianceRatio = 1.0 - math.Min(varianceRatio, 1.0)
	}

	// Combine factors (weighted average)
	confidence := (baseConfidence * 0.7) + (varianceRatio * 0.3)

	// Clamp to 0-1 range
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}
