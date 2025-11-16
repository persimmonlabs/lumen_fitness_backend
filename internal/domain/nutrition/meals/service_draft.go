package meals

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ParseMealAsync creates a draft meal and processes AI parsing in the background
// This is the NEW implementation that returns immediately with draft_id
func (s *service) ParseMealAsync(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error) {
	// Validate consumed_at is not in future
	if req.ConsumedAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("cannot log meals in the future")
	}

	// Validate photos if present
	if len(req.Photos) > 0 {
		if err := s.photoStorage.ValidatePhotos(ctx, req.Photos); err != nil {
			return nil, fmt.Errorf("invalid photos: %w", err)
		}
	}

	// Create draft meal with status='analyzing'
	draftMeal := &Meal{
		UserID:     userID,
		MealType:   req.MealType,
		ConsumedAt: req.ConsumedAt,
		Photos:     req.Photos,
		Notes:      "",
	}

	draftID, err := s.repo.CreateDraftMeal(ctx, userID, draftMeal)
	if err != nil {
		s.logger.Error("failed to create draft meal",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	s.logger.Info("draft meal created",
		slog.String("draft_id", draftID.String()),
		slog.String("user_id", userID.String()),
	)

	// Process AI parsing in background
	go s.processAIInBackground(userID, draftID, req.Description, req.Photos, req.IdempotencyKey)

	// Return immediately with draft_id and status='analyzing'
	return &ParseMealResponse{
		DraftID: draftID,
		Status:  DraftStatusAnalyzing,
	}, nil
}

// processAIInBackground handles AI processing asynchronously
func (s *service) processAIInBackground(userID, draftID uuid.UUID, description string, photos []string, idempotencyKey string) {
	// Create a new context with timeout for background processing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.logger.Info("starting background AI processing",
		slog.String("draft_id", draftID.String()),
		slog.String("user_id", userID.String()),
	)

	// Check idempotency cache first
	cacheKey := fmt.Sprintf("meal:parse:idempotency:%s", idempotencyKey)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if result, ok := cached.([]DraftMealItem); ok {
			s.logger.Info("using cached parse result",
				slog.String("idempotency_key", idempotencyKey),
			)
			s.updateDraftWithResults(ctx, draftID, result, 0.95, 0.0)
			return
		}
	}

	// Call AI coordinator to parse meal
	items, confidence, cost, err := s.aiCoordinator.ParseMeal(ctx, description, photos)
	if err != nil {
		s.logger.Error("AI parsing failed",
			slog.String("error", err.Error()),
			slog.String("draft_id", draftID.String()),
		)

		// Update draft status to 'error'
		errMsg := err.Error()
		updateErr := s.repo.UpdateDraftStatus(ctx, draftID, DraftStatusError, nil, &errMsg)
		if updateErr != nil {
			s.logger.Error("failed to update draft error status",
				slog.String("error", updateErr.Error()),
				slog.String("draft_id", draftID.String()),
			)
		}
		return
	}

	// Track cost
	if err := s.costTracker.TrackCost(ctx, userID, cost); err != nil {
		s.logger.Warn("failed to track cost",
			slog.String("error", err.Error()),
			slog.Float64("cost", cost),
		)
	}

	// Cache results
	s.cache.Set(ctx, cacheKey, items, 24*time.Hour)

	// Update draft with results
	s.updateDraftWithResults(ctx, draftID, items, confidence, cost)
}

// updateDraftWithResults updates the draft meal with AI parsing results
func (s *service) updateDraftWithResults(ctx context.Context, draftID uuid.UUID, items []DraftMealItem, confidence, cost float64) {
	// Convert DraftMealItem to MealItem
	mealItems := make([]MealItem, len(items))
	for i, draftItem := range items {
		mealItems[i] = MealItem{
			ID:       uuid.New(),
			MealID:   draftID,
			Name:     draftItem.Name,
			Quantity: draftItem.Quantity,
			Unit:     draftItem.Unit,
			Calories: draftItem.Calories,
			ProteinG: draftItem.ProteinG,
			CarbsG:   draftItem.CarbsG,
			FatG:     draftItem.FatG,
			FiberG:   draftItem.FiberG,
		}
	}

	// Update draft status to 'ready' with items
	err := s.repo.UpdateDraftStatus(ctx, draftID, DraftStatusReady, mealItems, nil)
	if err != nil {
		s.logger.Error("failed to update draft to ready",
			slog.String("error", err.Error()),
			slog.String("draft_id", draftID.String()),
		)
		return
	}

	s.logger.Info("draft meal ready",
		slog.String("draft_id", draftID.String()),
		slog.Int("item_count", len(items)),
		slog.Float64("confidence", confidence),
	)
}

// GetDraftStatus retrieves the current status of a draft meal
func (s *service) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*DraftStatusResponse, error) {
	meal, items, err := s.repo.GetDraftStatus(ctx, userID, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft status: %w", err)
	}

	response := &DraftStatusResponse{
		DraftID: meal.ID,
		Status:  *meal.DraftStatus,
	}

	// Include items if ready
	if *meal.DraftStatus == DraftStatusReady && len(items) > 0 {
		draftItems := make([]DraftMealItem, len(items))
		for i, item := range items {
			draftItems[i] = DraftMealItem{
				Name:     item.Name,
				Quantity: item.Quantity,
				Unit:     item.Unit,
				Calories: item.Calories,
				ProteinG: item.ProteinG,
				CarbsG:   item.CarbsG,
				FatG:     item.FatG,
				FiberG:   item.FiberG,
			}
		}
		response.Items = draftItems

		// NOTE: Database triggers (migration 012) calculate meal totals automatically
		// Use meal.Total* fields which are already calculated by database
		response.Total = &NutritionTotals{
			Calories: meal.TotalCalories,
			ProteinG: meal.TotalProteinG,
			CarbsG:   meal.TotalCarbsG,
			FatG:     meal.TotalFatG,
			FiberG:   meal.TotalFiberG,
		}
	}

	// Include error if status is error
	if *meal.DraftStatus == DraftStatusError && meal.DraftError != nil {
		response.Error = meal.DraftError
	}

	return response, nil
}
