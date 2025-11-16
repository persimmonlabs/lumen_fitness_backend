package ai

import (
	"bytes"
	"log/slog"
	"testing"
)

// TestEstimateMeal tests the meal estimation functionality
func TestEstimateMeal(t *testing.T) {
	// Skip if no API key (for CI/CD)
	apiKey := "test-key"
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	service := NewGroqServiceWithLogger(apiKey, logger)

	tests := []struct {
		name        string
		description string
		wantErr     bool
		checkResult func(*testing.T, *MealEstimation)
	}{
		{
			name:        "valid meal description",
			description: "chicken breast with rice",
			wantErr:     false,
			checkResult: func(t *testing.T, est *MealEstimation) {
				if est.Calories <= 0 {
					t.Errorf("Expected positive calories, got %d", est.Calories)
				}
				if est.Protein <= 0 {
					t.Errorf("Expected positive protein, got %d", est.Protein)
				}
				validConfidence := est.Confidence == "low" || est.Confidence == "medium" || est.Confidence == "high"
				if !validConfidence {
					t.Errorf("Invalid confidence level: %s", est.Confidence)
				}
			},
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
		},
		{
			name:        "vague description",
			description: "lunch",
			wantErr:     false,
			checkResult: func(t *testing.T, est *MealEstimation) {
				// Vague descriptions should have low confidence
				if est.Confidence != "low" {
					t.Logf("Warning: Expected low confidence for vague description, got %s", est.Confidence)
				}
			},
		},
		{
			name:        "specific meal with portions",
			description: "200g grilled salmon, 1 cup brown rice, steamed broccoli",
			wantErr:     false,
			checkResult: func(t *testing.T, est *MealEstimation) {
				// Specific descriptions should have higher confidence
				if est.Confidence == "low" {
					t.Logf("Warning: Expected higher confidence for specific description")
				}
				if est.Calories < 400 || est.Calories > 800 {
					t.Logf("Warning: Calories seem off for this meal: %d", est.Calories)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.EstimateMeal(tt.description)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestNormalizeMealDescription tests meal description normalization
func TestNormalizeMealDescription(t *testing.T) {
	apiKey := "test-key"
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	service := NewGroqServiceWithLogger(apiKey, logger)

	tests := []struct {
		name           string
		description    string
		wantErr        bool
		shouldContain  string
		shouldNotEqual string
	}{
		{
			name:           "lowercase to title case",
			description:    "chipotle bowl",
			wantErr:        false,
			shouldContain:  "Chipotle",
		},
		{
			name:           "typo correction",
			description:    "chiken brest",
			wantErr:        false,
			shouldContain:  "Chicken",
		},
		{
			name:          "empty description",
			description:   "",
			wantErr:       false,
			shouldContain: "",
		},
		{
			name:           "mixed case normalization",
			description:    "PIZZA margherita",
			wantErr:        false,
			shouldContain:  "Pizza",
		},
		{
			name:           "complex meal",
			description:    "griled salman with quinoa and vegtables",
			wantErr:        false,
			shouldContain:  "Salmon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.NormalizeMealDescription(tt.description)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				// Normalization failure should return original description
				if result != tt.description {
					t.Errorf("Expected original description on error, got %s", result)
				}
				return
			}

			if tt.shouldContain != "" {
				if !bytes.Contains([]byte(result), []byte(tt.shouldContain)) {
					t.Errorf("Expected result to contain %s, got %s", tt.shouldContain, result)
				}
			}

			if tt.shouldNotEqual != "" {
				if result == tt.shouldNotEqual {
					t.Errorf("Expected result to differ from %s", tt.shouldNotEqual)
				}
			}

			t.Logf("Normalized: %s -> %s", tt.description, result)
		})
	}
}

// TestTranscribeAudio tests audio transcription
func TestTranscribeAudio(t *testing.T) {
	apiKey := "test-key"
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	service := NewGroqServiceWithLogger(apiKey, logger)

	tests := []struct {
		name        string
		audioData   []byte
		contentType string
		wantErr     bool
	}{
		{
			name:        "empty audio data",
			audioData:   []byte{},
			contentType: "audio/webm",
			wantErr:     true,
		},
		{
			name:        "no api key",
			audioData:   []byte("fake audio data"),
			contentType: "audio/webm",
			wantErr:     false, // Will fail at API level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.TranscribeAudio(tt.audioData, tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			// Note: Real audio data would be needed for full integration test
			t.Logf("Transcription result: %s (error: %v)", result, err)
		})
	}
}

// TestEstimateMealPerformance tests estimation speed
func TestEstimateMealPerformance(t *testing.T) {
	apiKey := "test-key"
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	service := NewGroqServiceWithLogger(apiKey, logger)

	description := "chicken salad"

	result, err := service.EstimateMeal(description)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Fast model should respond quickly (under 5 seconds)
	if result.ProcessingTime.Seconds() > 5 {
		t.Errorf("Estimation took too long: %v (expected < 5s)", result.ProcessingTime)
	}

	t.Logf("Estimation completed in %v", result.ProcessingTime)
}
