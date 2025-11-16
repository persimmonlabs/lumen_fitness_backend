package meals

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/pradord/lumen_final/backend/internal/domain/nutrition/constants"
)

// Service defines the business logic interface for meals
type Service interface {
	ParseMeal(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error)
	ConfirmMeal(ctx context.Context, userID uuid.UUID, req *ConfirmMealRequest) (*MealResponse, error)
	GetMeal(ctx context.Context, userID, mealID uuid.UUID) (*MealResponse, error)
	ListMeals(ctx context.Context, userID uuid.UUID, filters ListMealFilters) (*MealListResponse, error)
	UpdateMeal(ctx context.Context, userID, mealID uuid.UUID, req *UpdateMealRequest) (*MealResponse, error)
	DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error
	CopyMeal(ctx context.Context, userID, mealID uuid.UUID, req *CopyMealRequest) (*MealResponse, error)
	EstimateMeal(ctx context.Context, userID uuid.UUID, req *EstimateMealRequest) (*EstimateMealResponse, error)
	ParseVoice(ctx context.Context, userID uuid.UUID, audioData []byte, contentType string, mealType MealType, consumedAt time.Time, idempotencyKey string) (*ParseMealResponse, error)
	GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*DraftStatusResponse, error)
	GetMealSuggestions(ctx context.Context, userID uuid.UUID, mealType MealType) (*MealSuggestionsResponse, error)
}

// AICoordinator defines the interface for AI meal parsing
type AICoordinator interface {
	ParseMeal(ctx context.Context, description string, photos []string) ([]DraftMealItem, float64, float64, error)
}

// AIEstimator defines the interface for fast meal estimation
type AIEstimator interface {
	EstimateMeal(description string) (*AIEstimation, error)
}

// AITranscriber defines the interface for audio transcription
type AITranscriber interface {
	TranscribeAudio(audioData []byte, contentType string) (string, error)
}

// AINormalizer defines the interface for meal description normalization
type AINormalizer interface {
	NormalizeMealDescription(description string) (string, error)
}

// AIEstimation represents a quick estimation from AI
type AIEstimation struct {
	Calories   int    `json:"calories"`
	Protein    int    `json:"protein"`
	Confidence string `json:"confidence"`
}

// Cache defines the interface for caching
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// CostTracker defines the interface for tracking API costs
type CostTracker interface {
	TrackCost(ctx context.Context, userID uuid.UUID, cost float64) error
}

// PhotoStorage defines the interface for photo validation
type PhotoStorage interface {
	ValidatePhotos(ctx context.Context, photoIDs []string) error
}

type service struct {
	repo          Repository
	aiCoordinator AICoordinator
	aiEstimator   AIEstimator
	aiTranscriber AITranscriber
	aiNormalizer  AINormalizer
	cache         Cache
	costTracker   CostTracker
	photoStorage  PhotoStorage
	logger        *slog.Logger
}

// NewService creates a new meal service
func NewService(
	repo Repository,
	aiCoordinator AICoordinator,
	aiEstimator AIEstimator,
	aiTranscriber AITranscriber,
	aiNormalizer AINormalizer,
	cache Cache,
	costTracker CostTracker,
	photoStorage PhotoStorage,
	logger *slog.Logger,
) Service {
	return &service{
		repo:          repo,
		aiCoordinator: aiCoordinator,
		aiEstimator:   aiEstimator,
		aiTranscriber: aiTranscriber,
		aiNormalizer:  aiNormalizer,
		cache:         cache,
		costTracker:   costTracker,
		photoStorage:  photoStorage,
		logger:        logger,
	}
}

// ParseMeal parses a meal description using AI
func (s *service) ParseMeal(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error) {
	// Validate consumed_at is not in future
	if req.ConsumedAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("cannot log meals in the future")
	}

	// Check idempotency cache first
	cacheKey := fmt.Sprintf("meal:parse:idempotency:%s", req.IdempotencyKey)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if result, ok := cached.(*ParseMealResponse); ok {
			s.logger.Info("returning cached parse result",
				slog.String("idempotency_key", req.IdempotencyKey),
			)
			return result, nil
		}
	}

	// Check content hash cache
	hashKey := s.generateCacheKey(req.Description, req.Photos)
	if cached, err := s.cache.Get(ctx, hashKey); err == nil {
		if result, ok := cached.(*ParseMealResponse); ok {
			// Save to idempotency cache
			s.cache.Set(ctx, cacheKey, result, 24*time.Hour)
			return result, nil
		}
	}

	// Validate photos if present
	if len(req.Photos) > 0 {
		if err := s.photoStorage.ValidatePhotos(ctx, req.Photos); err != nil {
			return nil, fmt.Errorf("invalid photos: %w", err)
		}
	}

	// Call AI coordinator to parse meal
	items, confidence, cost, err := s.aiCoordinator.ParseMeal(ctx, req.Description, req.Photos)
	if err != nil {
		s.logger.Error("AI parsing failed",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
		)
		return nil, fmt.Errorf("failed to parse meal: %w", err)
	}

	// Track cost
	if err := s.costTracker.TrackCost(ctx, userID, cost); err != nil {
		s.logger.Warn("failed to track cost",
			slog.String("error", err.Error()),
			slog.Float64("cost", cost),
		)
	}

	// NOTE: Totals are calculated by database triggers (migration 012)
	// We calculate here only for API response preview before DB persistence
	totals := s.calculateTotalsForPreview(items)

	// Build response
	response := &ParseMealResponse{}
	response.DraftMeal.Items = items
	response.DraftMeal.Total = totals
	response.DraftMeal.Confidence = confidence
	response.DraftMeal.CostUSD = cost

	// Cache results
	s.cache.Set(ctx, cacheKey, response, 24*time.Hour)
	s.cache.Set(ctx, hashKey, response, 1*time.Hour)

	s.logger.Info("meal parsed successfully",
		slog.String("user_id", userID.String()),
		slog.Int("item_count", len(items)),
		slog.Float64("confidence", confidence),
	)

	return response, nil
}

// ConfirmMeal saves a parsed meal to the database
func (s *service) ConfirmMeal(ctx context.Context, userID uuid.UUID, req *ConfirmMealRequest) (*MealResponse, error) {
	// Validate consumed_at is not in future
	if req.ConsumedAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("cannot log meals in the future")
	}

	// Validate meal type
	if !req.MealType.IsValid() {
		return nil, fmt.Errorf("invalid meal type: %s", req.MealType)
	}

	// Validate items
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("at least one item is required")
	}

	// Validate macros using constants package for item-level validation
	if err := s.validateMacros(req.Items); err != nil {
		return nil, err
	}

	// NOTE: Database triggers (migration 012) automatically calculate totals
	// DO NOT set total_* fields - they are calculated from meal_items
	meal := &Meal{
		ID:         uuid.New(),
		UserID:     userID,
		MealType:   req.MealType,
		ConsumedAt: req.ConsumedAt,
		Photos:     req.Photos,
		Notes:      req.Notes,
		// Total nutrition fields intentionally omitted - database triggers handle this
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Convert draft items to meal items
	items := make([]MealItem, len(req.Items))
	for i, draftItem := range req.Items {
		items[i] = MealItem{
			ID:       uuid.New(),
			MealID:   meal.ID,
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

	// Save to database
	result, err := s.repo.CreateMealWithItems(ctx, userID, meal, items)
	if err != nil {
		return nil, fmt.Errorf("failed to save meal: %w", err)
	}

	s.logger.Info("meal confirmed",
		slog.String("meal_id", meal.ID.String()),
		slog.String("user_id", userID.String()),
		slog.String("meal_type", string(req.MealType)),
	)

	return &MealResponse{MealWithItems: *result}, nil
}

// GetMeal retrieves a single meal by ID
func (s *service) GetMeal(ctx context.Context, userID, mealID uuid.UUID) (*MealResponse, error) {
	meal, err := s.repo.GetMealByID(ctx, userID, mealID)
	if err != nil {
		return nil, err
	}

	return &MealResponse{MealWithItems: *meal}, nil
}

// ListMeals retrieves paginated meals for a user
func (s *service) ListMeals(ctx context.Context, userID uuid.UUID, filters ListMealFilters) (*MealListResponse, error) {
	// Set default pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 20
	}

	meals, total, err := s.repo.ListMealsByUser(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list meals: %w", err)
	}

	totalPages := (total + filters.Limit - 1) / filters.Limit

	return &MealListResponse{
		Meals: meals,
		Pagination: Pagination{
			Page:       filters.Page,
			Limit:      filters.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateMeal updates an existing meal
func (s *service) UpdateMeal(ctx context.Context, userID, mealID uuid.UUID, req *UpdateMealRequest) (*MealResponse, error) {
	// Validate consumed_at is not in future
	if req.ConsumedAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("cannot log meals in the future")
	}

	// Validate macros using constants package for item-level validation
	if err := s.validateMacros(req.Items); err != nil {
		return nil, err
	}

	// NOTE: Database triggers (migration 012) automatically calculate totals
	// DO NOT set total_* fields - they are calculated from meal_items
	meal := &Meal{
		ID:         mealID,
		UserID:     userID,
		MealType:   req.MealType,
		ConsumedAt: req.ConsumedAt,
		Photos:     req.Photos,
		Notes:      req.Notes,
		// Total nutrition fields intentionally omitted - database triggers handle this
	}

	// Convert to meal items
	items := make([]MealItem, len(req.Items))
	for i, draftItem := range req.Items {
		items[i] = MealItem{
			ID:       uuid.New(),
			MealID:   mealID,
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

	result, err := s.repo.UpdateMeal(ctx, userID, meal, items)
	if err != nil {
		return nil, err
	}

	s.logger.Info("meal updated",
		slog.String("meal_id", mealID.String()),
		slog.String("user_id", userID.String()),
	)

	return &MealResponse{MealWithItems: *result}, nil
}

// DeleteMeal soft deletes a meal
func (s *service) DeleteMeal(ctx context.Context, userID, mealID uuid.UUID) error {
	if err := s.repo.DeleteMeal(ctx, userID, mealID); err != nil {
		return err
	}

	s.logger.Info("meal deleted",
		slog.String("meal_id", mealID.String()),
		slog.String("user_id", userID.String()),
	)

	return nil
}

// CopyMeal duplicates a meal to a new time
func (s *service) CopyMeal(ctx context.Context, userID, mealID uuid.UUID, req *CopyMealRequest) (*MealResponse, error) {
	// Validate consumed_at is not in future
	if req.ConsumedAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("cannot log meals in the future")
	}

	result, err := s.repo.DuplicateMeal(ctx, userID, mealID, req.ConsumedAt, req.MealType)
	if err != nil {
		return nil, err
	}

	s.logger.Info("meal copied",
		slog.String("source_meal_id", mealID.String()),
		slog.String("new_meal_id", result.ID.String()),
		slog.String("user_id", userID.String()),
	)

	return &MealResponse{MealWithItems: *result}, nil
}

// Helper functions

func (s *service) generateCacheKey(description string, photos []string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"description": description,
		"photos":      photos,
	})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("meal:parse:hash:%x", hash)
}

// calculateTotalsForPreview calculates nutrition totals for API response previews.
// NOTE: This is ONLY for preview purposes before database persistence.
// Once data is saved, database triggers (migration 012) are the SINGLE source of truth.
// DO NOT use this function to set meal.total_* fields - let the database handle it.
func (s *service) calculateTotalsForPreview(items []DraftMealItem) NutritionTotals {
	var totals NutritionTotals
	for _, item := range items {
		totals.Calories += item.Calories
		totals.ProteinG += item.ProteinG
		totals.CarbsG += item.CarbsG
		totals.FatG += item.FatG
		totals.FiberG += item.FiberG
	}
	return totals
}

// validateMacros validates item-level macronutrient consistency using constants package.
// Uses ValidateMacroCalories from constants package to ensure calories match macros within tolerance.
// This validation is performed at the ITEM level only - meal totals are calculated by database triggers.
func (s *service) validateMacros(items []DraftMealItem) error {
	for _, item := range items {
		// Use constants package for validation - single source of truth for macro coefficients
		// NOTE: This validates ITEM data only, NOT meal totals (database triggers handle meal totals)
		if !constants.ValidateMacroCalories(item.Calories, item.ProteinG, item.CarbsG, item.FatG) {
			expectedCals := constants.CalculateMacroCalories(item.ProteinG, item.CarbsG, item.FatG)
			return fmt.Errorf("macro validation failed for %s: calories %.1f not within ±%.0f%% of calculated %.1f",
				item.Name, item.Calories, constants.MacroCalorieTolerancePercent*100, expectedCals)
		}
	}
	return nil
}

// EstimateMeal provides fast meal calorie estimation
func (s *service) EstimateMeal(ctx context.Context, userID uuid.UUID, req *EstimateMealRequest) (*EstimateMealResponse, error) {
	// Check cache first (5 min TTL)
	cacheKey := fmt.Sprintf("meal:estimate:%s", req.Description)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if result, ok := cached.(*EstimateMealResponse); ok {
			s.logger.Debug("returning cached estimation",
				slog.String("description", req.Description),
			)
			return result, nil
		}
	}

	// Call AI estimator
	estimation, err := s.aiEstimator.EstimateMeal(req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate meal: %w", err)
	}

	response := &EstimateMealResponse{
		Calories:   estimation.Calories,
		Protein:    estimation.Protein,
		Confidence: estimation.Confidence,
	}

	// Cache result for 5 minutes
	s.cache.Set(ctx, cacheKey, response, 5*time.Minute)

	s.logger.Info("meal estimated",
		slog.String("user_id", userID.String()),
		slog.Int("calories", estimation.Calories),
		slog.String("confidence", estimation.Confidence),
	)

	return response, nil
}

// ParseVoice transcribes audio and parses meal
func (s *service) ParseVoice(ctx context.Context, userID uuid.UUID, audioData []byte, contentType string, mealType MealType, consumedAt time.Time, idempotencyKey string) (*ParseMealResponse, error) {
	// Check idempotency cache first
	cacheKey := fmt.Sprintf("meal:parse:idempotency:%s", idempotencyKey)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if result, ok := cached.(*ParseMealResponse); ok {
			s.logger.Info("returning cached voice parse result",
				slog.String("idempotency_key", idempotencyKey),
			)
			return result, nil
		}
	}

	// Transcribe audio
	transcription, err := s.aiTranscriber.TranscribeAudio(audioData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	s.logger.Info("audio transcribed",
		slog.String("user_id", userID.String()),
		slog.String("transcription", transcription),
		slog.Int("audio_bytes", len(audioData)),
	)

	// Create parse request from transcription
	parseReq := &ParseMealRequest{
		Description:    transcription,
		MealType:       mealType,
		ConsumedAt:     consumedAt,
		Photos:         []string{},
		IdempotencyKey: idempotencyKey,
	}

	// Parse meal using existing logic
	return s.ParseMeal(ctx, userID, parseReq)
}

// normalizeMealDescription normalizes meal description using AI
func (s *service) normalizeMealDescription(description string) string {
	// Check cache first
	cacheKey := fmt.Sprintf("meal:normalize:%s", description)
	ctx := context.Background()

	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if normalized, ok := cached.(string); ok {
			return normalized
		}
	}

	// Call AI normalizer
	normalized, err := s.aiNormalizer.NormalizeMealDescription(description)
	if err != nil {
		// If normalization fails, return original description
		s.logger.Warn("failed to normalize meal description",
			slog.String("error", err.Error()),
			slog.String("description", description),
		)
		return description
	}

	// Cache result for 1 hour
	s.cache.Set(ctx, cacheKey, normalized, 1*time.Hour)

	return normalized
}
