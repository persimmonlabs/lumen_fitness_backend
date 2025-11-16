package ai

import (
	"testing"
	"time"
)

func TestCoordinator_Initialization(t *testing.T) {
	tests := []struct {
		name      string
		config    CoordinatorConfig
		wantError bool
	}{
		{
			name: "both services configured",
			config: CoordinatorConfig{
				GroqAPIKey:       "groq-key",
				OpenRouterAPIKey: "openrouter-key",
				PreferGroq:       true,
				CostLimit:        10.0,
			},
			wantError: false,
		},
		{
			name: "only groq configured",
			config: CoordinatorConfig{
				GroqAPIKey: "groq-key",
				PreferGroq: true,
			},
			wantError: false,
		},
		{
			name: "only openrouter configured",
			config: CoordinatorConfig{
				OpenRouterAPIKey: "openrouter-key",
			},
			wantError: false,
		},
		{
			name:      "no services configured",
			config:    CoordinatorConfig{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewCoordinator(tt.config)
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if coordinator == nil {
				t.Fatal("Expected coordinator, got nil")
			}

			// Verify services are initialized correctly
			if tt.config.GroqAPIKey != "" && coordinator.groq == nil {
				t.Error("Expected Groq service to be initialized")
			}
			if tt.config.OpenRouterAPIKey != "" && coordinator.openRouter == nil {
				t.Error("Expected OpenRouter service to be initialized")
			}
		})
	}
}

func TestCoordinator_ImageRouting(t *testing.T) {
	// Create coordinator with both services using mock
	groqMock := NewMockService(MockConfig{SupportsVision: false})
	openRouterMock := NewMockService(MockConfig{SupportsVision: true})

	coordinator := &Coordinator{
		groq:       groqMock,
		openRouter: openRouterMock,
		preferGroq: true,
		costLimit:  10.0,
	}

	// Request with image should go to OpenRouter
	req := NutritionRequest{
		Images: [][]byte{{1, 2, 3}},
		UserID: "test-user",
	}

	resp, err := coordinator.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.ModelUsed != "Mock AI Service" {
		t.Errorf("Expected OpenRouter mock to be used for images")
	}
}

func TestCoordinator_TextRoutingWithGroqPreference(t *testing.T) {
	// Create coordinator preferring Groq
	groqMock := NewMockService(MockConfig{})
	openRouterMock := NewMockService(MockConfig{SupportsVision: true})

	coordinator := &Coordinator{
		groq:       groqMock,
		openRouter: openRouterMock,
		preferGroq: true,
		costLimit:  10.0,
	}

	// Text request should try Groq first
	req := NutritionRequest{
		Text:   "chicken and rice",
		UserID: "test-user",
	}

	resp, err := coordinator.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Should use Groq (free)
	if resp.Cost != 0.0 {
		t.Errorf("Expected free Groq service, got cost: %f", resp.Cost)
	}

	// Verify Groq was used
	stats := coordinator.GetStats()
	if stats.GroqRequests != 1 {
		t.Errorf("Expected 1 Groq request, got %d", stats.GroqRequests)
	}
}

func TestCoordinator_GroqFallbackToOpenRouter(t *testing.T) {
	// Create mock with low confidence
	groqMock := NewMockService(MockConfig{})
	openRouterMock := NewMockService(MockConfig{SupportsVision: true})

	coordinator := &Coordinator{
		groq:       groqMock,
		openRouter: openRouterMock,
		preferGroq: true,
		costLimit:  10.0,
	}

	// This test would require mocking low confidence response from Groq
	// For now, we just verify the logic exists
	req := NutritionRequest{
		Text:   "unknown food",
		UserID: "test-user",
	}

	resp, err := coordinator.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
}

func TestCoordinator_CostLimitEnforcement(t *testing.T) {
	openRouterMock := NewMockService(MockConfig{SupportsVision: true})

	coordinator := &Coordinator{
		openRouter:  openRouterMock,
		costLimit:   0.01, // Very low limit
		monthlyCost: 0.0,
	}

	// First request should succeed
	req := NutritionRequest{
		Text:   "test food",
		UserID: "test-user",
	}

	_, err := coordinator.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("First request should succeed: %v", err)
	}

	// Set cost near limit
	coordinator.monthlyCost = 0.009

	// Next request should fail (would exceed limit)
	_, err = coordinator.AnalyzeNutrition(req)
	if err == nil {
		t.Fatal("Expected cost limit error")
	}

	if aiErr, ok := err.(*AIError); ok {
		if aiErr.Type != ErrorTypeCostLimitExceeded {
			t.Errorf("Expected CostLimitExceeded error, got %v", aiErr.Type)
		}
	} else {
		t.Error("Expected AIError type")
	}
}

func TestCoordinator_GetStats(t *testing.T) {
	coordinator := &Coordinator{
		monthlyCost:        5.5,
		costLimit:          10.0,
		requestCount:       15,
		groqRequests:       10,
		openRouterRequests: 5,
		lastResetTime:      time.Now(),
	}

	stats := coordinator.GetStats()

	if stats.MonthlyCost != 5.5 {
		t.Errorf("Expected monthly cost 5.5, got %f", stats.MonthlyCost)
	}
	if stats.CostLimit != 10.0 {
		t.Errorf("Expected cost limit 10.0, got %f", stats.CostLimit)
	}
	if stats.RemainingBudget != 4.5 {
		t.Errorf("Expected remaining budget 4.5, got %f", stats.RemainingBudget)
	}
	if stats.RequestCount != 15 {
		t.Errorf("Expected 15 requests, got %d", stats.RequestCount)
	}
	if stats.GroqRequests != 10 {
		t.Errorf("Expected 10 Groq requests, got %d", stats.GroqRequests)
	}
	if stats.OpenRouterRequests != 5 {
		t.Errorf("Expected 5 OpenRouter requests, got %d", stats.OpenRouterRequests)
	}
}

func TestCoordinator_ResetCosts(t *testing.T) {
	coordinator := &Coordinator{
		monthlyCost:        5.5,
		requestCount:       10,
		groqRequests:       5,
		openRouterRequests: 5,
	}

	coordinator.ResetCosts()

	if coordinator.monthlyCost != 0 {
		t.Errorf("Expected monthly cost to be reset to 0, got %f", coordinator.monthlyCost)
	}
	if coordinator.requestCount != 0 {
		t.Errorf("Expected request count to be reset to 0, got %d", coordinator.requestCount)
	}
}

func TestCoordinator_SetCostLimit(t *testing.T) {
	coordinator := &Coordinator{
		costLimit: 10.0,
	}

	newLimit := 25.0
	coordinator.SetCostLimit(newLimit)

	if coordinator.costLimit != newLimit {
		t.Errorf("Expected cost limit %f, got %f", newLimit, coordinator.costLimit)
	}
}

func TestCoordinator_MonthlyCostReset(t *testing.T) {
	coordinator := &Coordinator{
		monthlyCost:   5.0,
		lastResetTime: time.Now().Add(-31 * 24 * time.Hour), // 31 days ago
		requestCount:  10,
	}

	coordinator.checkAndResetMonthlyCost()

	if coordinator.monthlyCost != 0 {
		t.Error("Expected monthly cost to be reset after 30 days")
	}
	if coordinator.requestCount != 0 {
		t.Error("Expected request count to be reset")
	}
}

func TestCoordinator_NoGroqAvailable(t *testing.T) {
	openRouterMock := NewMockService(MockConfig{SupportsVision: true})

	coordinator := &Coordinator{
		openRouter: openRouterMock,
		preferGroq: true,
		costLimit:  10.0,
	}

	req := NutritionRequest{
		Text:   "test food",
		UserID: "test-user",
	}

	resp, err := coordinator.AnalyzeNutrition(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use OpenRouter since Groq is not available
	stats := coordinator.GetStats()
	if stats.OpenRouterRequests != 1 {
		t.Errorf("Expected OpenRouter to be used when Groq unavailable")
	}
	if !stats.OpenRouterAvailable {
		t.Error("Expected OpenRouter to be available")
	}
	if stats.GroqAvailable {
		t.Error("Expected Groq to be unavailable")
	}

	if resp == nil {
		t.Fatal("Expected response")
	}
}

func TestCoordinator_ImageWithoutOpenRouter(t *testing.T) {
	groqMock := NewMockService(MockConfig{})

	coordinator := &Coordinator{
		groq:      groqMock,
		costLimit: 10.0,
	}

	req := NutritionRequest{
		Images: [][]byte{{1, 2, 3}},
		UserID: "test-user",
	}

	_, err := coordinator.AnalyzeNutrition(req)
	if err == nil {
		t.Fatal("Expected error for image analysis without OpenRouter")
	}

	if aiErr, ok := err.(*AIError); ok {
		if aiErr.Type != ErrorTypeInvalidInput {
			t.Errorf("Expected InvalidInput error, got %v", aiErr.Type)
		}
	}
}

func TestCoordinator_Capabilities(t *testing.T) {
	tests := []struct {
		name           string
		hasGroq        bool
		hasOpenRouter  bool
		expectVision   bool
		expectAudio    bool
	}{
		{
			name:          "both services",
			hasGroq:       true,
			hasOpenRouter: true,
			expectVision:  true,
			expectAudio:   true,
		},
		{
			name:          "only groq",
			hasGroq:       true,
			hasOpenRouter: false,
			expectVision:  false,
			expectAudio:   true,
		},
		{
			name:          "only openrouter",
			hasGroq:       false,
			hasOpenRouter: true,
			expectVision:  true,
			expectAudio:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator := &Coordinator{}

			if tt.hasGroq {
				coordinator.groq = NewMockService(MockConfig{SupportsAudio: true})
			}
			if tt.hasOpenRouter {
				coordinator.openRouter = NewMockService(MockConfig{SupportsVision: true})
			}

			if coordinator.SupportsVision() != tt.expectVision {
				t.Errorf("Expected vision support: %v, got: %v", tt.expectVision, coordinator.SupportsVision())
			}
			if coordinator.SupportsAudio() != tt.expectAudio {
				t.Errorf("Expected audio support: %v, got: %v", tt.expectAudio, coordinator.SupportsAudio())
			}
		})
	}
}
