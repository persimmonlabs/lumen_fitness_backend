package ai

import (
	"testing"
	"time"
)

func TestMockService_BasicFunctionality(t *testing.T) {
	mock := NewMockService(MockConfig{
		SupportsVision: true,
		SupportsAudio:  true,
	})

	req := NutritionRequest{
		Text:   "chicken and rice",
		UserID: "test-user",
	}

	resp, err := mock.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Items) == 0 {
		t.Error("Expected at least one food item")
	}

	if resp.Confidence <= 0 || resp.Confidence > 1 {
		t.Errorf("Invalid confidence: %f", resp.Confidence)
	}

	if resp.Cost != 0.0 {
		t.Errorf("Mock should be free, got cost: %f", resp.Cost)
	}

	if resp.ModelUsed != "Mock AI Service" {
		t.Errorf("Wrong model name: %s", resp.ModelUsed)
	}
}

func TestMockService_ImageSupport(t *testing.T) {
	tests := []struct {
		name           string
		supportsVision bool
		hasImage       bool
		wantError      bool
	}{
		{
			name:           "vision supported with image",
			supportsVision: true,
			hasImage:       true,
			wantError:      false,
		},
		{
			name:           "vision not supported with image",
			supportsVision: false,
			hasImage:       true,
			wantError:      true,
		},
		{
			name:           "no image",
			supportsVision: false,
			hasImage:       false,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockService(MockConfig{
				SupportsVision: tt.supportsVision,
			})

			req := NutritionRequest{
				Text:   "test",
				UserID: "test-user",
			}

			if tt.hasImage {
				req.Images = [][]byte{{1, 2, 3}}
			}

			resp, err := mock.AnalyzeNutrition(req)

			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error for unsupported vision")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response")
			}
		})
	}
}

func TestMockService_AudioSupport(t *testing.T) {
	tests := []struct {
		name          string
		supportsAudio bool
		hasAudio      bool
		wantError     bool
	}{
		{
			name:          "audio supported",
			supportsAudio: true,
			hasAudio:      true,
			wantError:     false,
		},
		{
			name:          "audio not supported",
			supportsAudio: false,
			hasAudio:      true,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockService(MockConfig{
				SupportsAudio: tt.supportsAudio,
			})

			req := NutritionRequest{
				AudioData: []byte("audio data"),
				UserID:    "test-user",
			}

			resp, err := mock.AnalyzeNutrition(req)

			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error for unsupported audio")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response")
			}
		})
	}
}

func TestMockService_SimulateDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	mock := NewMockService(MockConfig{
		SimulateDelay: delay,
	})

	req := NutritionRequest{
		Text:   "test",
		UserID: "test",
	}

	start := time.Now()
	_, err := mock.AnalyzeNutrition(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
	}
}

func TestMockService_SimulateFailures(t *testing.T) {
	mock := NewMockService(MockConfig{
		FailureRate: 1.0, // Always fail
	})

	req := NutritionRequest{
		Text:   "test",
		UserID: "test",
	}

	_, err := mock.AnalyzeNutrition(req)
	if err == nil {
		t.Fatal("Expected error with 100% failure rate")
	}

	if aiErr, ok := err.(*AIError); ok {
		if !aiErr.Retryable {
			t.Error("Mock failures should be retryable")
		}
	}
}

func TestMockService_UserPreferences(t *testing.T) {
	mock := NewMockService(MockConfig{})

	req := NutritionRequest{
		Text:   "pasta",
		UserID: "test",
		Preferences: &UserPreferences{
			DietaryRestrictions: []string{"vegetarian"},
			AllergiesWarnings:   []string{"peanuts", "dairy"},
			PreferredUnits:      "imperial",
		},
	}

	resp, err := mock.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Mock should include warnings about allergies
	hasAllergyWarning := false
	for _, warning := range resp.Warnings {
		if len(warning) > 0 {
			hasAllergyWarning = true
			break
		}
	}

	if !hasAllergyWarning && len(req.Preferences.AllergiesWarnings) > 0 {
		t.Error("Expected allergy warnings in response")
	}
}

func TestMockService_NutritionData(t *testing.T) {
	mock := NewMockService(MockConfig{})

	req := NutritionRequest{
		Text:   "test meal",
		UserID: "test",
	}

	resp, err := mock.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify nutritional data is realistic
	if resp.TotalNutrition.Calories < 0 {
		t.Error("Calories should be non-negative")
	}

	if resp.TotalNutrition.Protein < 0 {
		t.Error("Protein should be non-negative")
	}

	// Verify each item has nutrition data
	for _, item := range resp.Items {
		if item.Name == "" {
			t.Error("Item should have a name")
		}
		if item.Quantity <= 0 {
			t.Error("Quantity should be positive")
		}
		if item.Unit == "" {
			t.Error("Item should have a unit")
		}
		if item.Confidence <= 0 || item.Confidence > 1 {
			t.Errorf("Invalid confidence for item: %f", item.Confidence)
		}
	}
}

func TestMockService_Capabilities(t *testing.T) {
	tests := []struct {
		name    string
		config  MockConfig
		vision  bool
		audio   bool
	}{
		{
			name: "all capabilities",
			config: MockConfig{
				SupportsVision: true,
				SupportsAudio:  true,
			},
			vision: true,
			audio:  true,
		},
		{
			name: "vision only",
			config: MockConfig{
				SupportsVision: true,
				SupportsAudio:  false,
			},
			vision: true,
			audio:  false,
		},
		{
			name: "no capabilities",
			config: MockConfig{
				SupportsVision: false,
				SupportsAudio:  false,
			},
			vision: false,
			audio:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockService(tt.config)

			if mock.SupportsVision() != tt.vision {
				t.Errorf("Expected vision: %v, got: %v", tt.vision, mock.SupportsVision())
			}

			if mock.SupportsAudio() != tt.audio {
				t.Errorf("Expected audio: %v, got: %v", tt.audio, mock.SupportsAudio())
			}

			if mock.GetCostPerRequest() != 0.0 {
				t.Error("Mock should always be free")
			}
		})
	}
}

func TestMockService_GeneratesSuggestions(t *testing.T) {
	mock := NewMockService(MockConfig{})

	req := NutritionRequest{
		Text:   "burger and fries",
		UserID: "test",
	}

	resp, err := mock.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resp.Suggestions) == 0 {
		t.Error("Expected mock to generate suggestions")
	}
}

func TestMockService_MultipleItems(t *testing.T) {
	mock := NewMockService(MockConfig{
		SupportsVision: true,
	})

	req := NutritionRequest{
		Images: [][]byte{{1, 2, 3}}, // Image requests usually have multiple items
		UserID: "test",
	}

	resp, err := mock.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resp.Items) < 1 {
		t.Error("Expected at least one item in response")
	}

	// Total nutrition should be sum of all items
	var expectedCalories float64
	for _, item := range resp.Items {
		expectedCalories += item.Nutrition.Calories
	}

	if resp.TotalNutrition.Calories != expectedCalories {
		t.Errorf("Total calories mismatch: expected %f, got %f",
			expectedCalories, resp.TotalNutrition.Calories)
	}
}
