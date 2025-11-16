package meals

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestEstimateMeal tests the meal estimation service method
func TestServiceEstimateMeal(t *testing.T) {
	mockEstimator := &MockAIEstimator{}
	mockCache := &MockCache{items: make(map[string]interface{})}

	svc := &service{
		aiEstimator: mockEstimator,
		cache:       mockCache,
		logger:      testLogger,
	}

	tests := []struct {
		name        string
		request     *EstimateMealRequest
		setupMock   func()
		wantErr     bool
		checkResult func(*testing.T, *EstimateMealResponse)
	}{
		{
			name: "successful estimation",
			request: &EstimateMealRequest{
				Description: "chicken salad",
			},
			setupMock: func() {
				mockEstimator.EstimateFunc = func(desc string) (*AIEstimation, error) {
					return &AIEstimation{
						Calories:   350,
						Protein:    40,
						Confidence: "high",
					}, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *EstimateMealResponse) {
				if resp.Calories != 350 {
					t.Errorf("Expected calories 350, got %d", resp.Calories)
				}
				if resp.Protein != 40 {
					t.Errorf("Expected protein 40, got %d", resp.Protein)
				}
				if resp.Confidence != "high" {
					t.Errorf("Expected confidence 'high', got %s", resp.Confidence)
				}
			},
		},
		{
			name: "cached estimation",
			request: &EstimateMealRequest{
				Description: "cached meal",
			},
			setupMock: func() {
				// Pre-populate cache
				mockCache.items["meal:estimate:cached meal"] = &EstimateMealResponse{
					Calories:   600,
					Protein:    50,
					Confidence: "medium",
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *EstimateMealResponse) {
				if resp.Calories != 600 {
					t.Errorf("Expected cached calories 600, got %d", resp.Calories)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			ctx := context.Background()
			userID := uuid.New()

			result, err := svc.EstimateMeal(ctx, userID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestParseVoice tests the voice parsing service method
func TestServiceParseVoice(t *testing.T) {
	mockTranscriber := &MockAITranscriber{}
	mockCoordinator := &MockAICoordinator{}
	mockCache := &MockCache{items: make(map[string]interface{})}
	mockCostTracker := &MockCostTracker{}
	mockPhotoStorage := &MockPhotoStorage{}

	svc := &service{
		aiTranscriber: mockTranscriber,
		aiCoordinator: mockCoordinator,
		cache:         mockCache,
		costTracker:   mockCostTracker,
		photoStorage:  mockPhotoStorage,
		logger:        testLogger,
	}

	tests := []struct {
		name           string
		audioData      []byte
		contentType    string
		mealType       MealType
		consumedAt     time.Time
		idempotencyKey string
		setupMocks     func()
		wantErr        bool
		checkResult    func(*testing.T, *ParseMealResponse)
	}{
		{
			name:           "successful voice transcription and parsing",
			audioData:      []byte("fake audio data"),
			contentType:    "audio/webm",
			mealType:       MealTypeBreakfast,
			consumedAt:     time.Now().UTC(),
			idempotencyKey: "voice-test-key",
			setupMocks: func() {
				mockTranscriber.TranscribeFunc = func(audioData []byte, contentType string) (string, error) {
					return "oatmeal with banana and honey", nil
				}
				mockCoordinator.ParseMealFunc = func(ctx context.Context, desc string, photos []string) ([]DraftMealItem, float64, float64, error) {
					return []DraftMealItem{
						{
							Name:     "Oatmeal",
							Quantity: 1,
							Unit:     "serving",
							Calories: 150,
							ProteinG: 5,
							CarbsG:   27,
							FatG:     3,
							FiberG:   4,
						},
					}, 0.85, 0.0, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *ParseMealResponse) {
				if len(resp.DraftMeal.Items) == 0 {
					t.Error("Expected items in draft meal")
				}
				if resp.DraftMeal.Confidence < 0.5 {
					t.Errorf("Expected higher confidence, got %f", resp.DraftMeal.Confidence)
				}
			},
		},
		{
			name:           "cached voice parse result",
			audioData:      []byte("fake audio"),
			contentType:    "audio/mp3",
			mealType:       MealTypeLunch,
			consumedAt:     time.Now().UTC(),
			idempotencyKey: "cached-voice-key",
			setupMocks: func() {
				// Pre-populate cache
				cachedResp := &ParseMealResponse{}
				cachedResp.DraftMeal.Items = []DraftMealItem{
					{Name: "Cached Meal", Calories: 400},
				}
				mockCache.items["meal:parse:idempotency:cached-voice-key"] = cachedResp
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *ParseMealResponse) {
				if len(resp.DraftMeal.Items) == 0 {
					t.Error("Expected cached items")
				}
				if resp.DraftMeal.Items[0].Name != "Cached Meal" {
					t.Error("Expected cached meal data")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			ctx := context.Background()
			userID := uuid.New()

			result, err := svc.ParseVoice(ctx, userID, tt.audioData, tt.contentType, tt.mealType, tt.consumedAt, tt.idempotencyKey)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestNormalizeMealDescription tests the normalization functionality
func TestServiceNormalizeMealDescription(t *testing.T) {
	mockNormalizer := &MockAINormalizer{}
	mockCache := &MockCache{items: make(map[string]interface{})}

	svc := &service{
		aiNormalizer: mockNormalizer,
		cache:        mockCache,
		logger:       testLogger,
	}

	tests := []struct {
		name        string
		description string
		setupMock   func()
		expected    string
	}{
		{
			name:        "simple normalization",
			description: "chicken bowl",
			setupMock: func() {
				mockNormalizer.NormalizeFunc = func(desc string) (string, error) {
					return "Chicken Bowl", nil
				}
			},
			expected: "Chicken Bowl",
		},
		{
			name:        "cached normalization",
			description: "cached description",
			setupMock: func() {
				mockCache.items["meal:normalize:cached description"] = "Cached Description"
			},
			expected: "Cached Description",
		},
		{
			name:        "normalization failure returns original",
			description: "test meal",
			setupMock: func() {
				mockNormalizer.NormalizeFunc = func(desc string) (string, error) {
					return "", &AIError{Message: "normalization failed"}
				}
			},
			expected: "test meal", // Should return original on error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result := svc.normalizeMealDescription(tt.description)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// CacheError for mock
type CacheError struct{}

func (e *CacheError) Error() string {
	return "cache miss"
}
