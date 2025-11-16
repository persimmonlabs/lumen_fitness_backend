package ai

import (
	"fmt"
	"sync"
	"time"
)

const (
	MinConfidenceThreshold = 0.7  // Minimum confidence to accept Groq result
	DefaultCostLimit       = 10.0 // Default monthly cost limit in USD
)

// CoordinatorConfig contains configuration for the AI coordinator
type CoordinatorConfig struct {
	GroqAPIKey       string
	OpenRouterAPIKey string
	OpenRouterModel  string
	AppName          string
	SiteURL          string
	CostLimit        float64 // Monthly cost limit in USD
	PreferGroq       bool    // Try Groq first even for vision (after image analysis fails)
}

// Coordinator intelligently routes requests between Groq and OpenRouter
type Coordinator struct {
	groq        AIService
	openRouter  AIService
	preferGroq  bool
	costLimit   float64

	// Cost tracking
	mu              sync.RWMutex
	monthlyCost     float64
	lastResetTime   time.Time
	requestCount    int
	groqRequests    int
	openRouterRequests int
}

// NewCoordinator creates a new AI coordinator
func NewCoordinator(config CoordinatorConfig) (*Coordinator, error) {
	// Validate at least one service is configured
	if config.GroqAPIKey == "" && config.OpenRouterAPIKey == "" {
		return nil, fmt.Errorf("at least one AI service API key must be configured")
	}

	costLimit := config.CostLimit
	if costLimit == 0 {
		costLimit = DefaultCostLimit
	}

	coordinator := &Coordinator{
		preferGroq:    config.PreferGroq,
		costLimit:     costLimit,
		lastResetTime: time.Now(),
	}

	// Initialize Groq if API key is provided
	if config.GroqAPIKey != "" {
		coordinator.groq = NewGroqService(config.GroqAPIKey)
	}

	// Initialize OpenRouter if API key is provided
	if config.OpenRouterAPIKey != "" {
		coordinator.openRouter = NewOpenRouterService(OpenRouterConfig{
			APIKey:  config.OpenRouterAPIKey,
			Model:   config.OpenRouterModel,
			AppName: config.AppName,
			SiteURL: config.SiteURL,
		})
	}

	return coordinator, nil
}

// AnalyzeNutrition implements intelligent routing logic
func (c *Coordinator) AnalyzeNutrition(req NutritionRequest) (*NutritionResponse, error) {
	// Check if we need to reset monthly cost tracking
	c.checkAndResetMonthlyCost()

	// Decision tree for service selection

	// 1. If images are present, must use OpenRouter (vision capability)
	if len(req.Images) > 0 {
		if c.openRouter == nil {
			return nil, NewAIError(
				ErrorTypeInvalidInput,
				"Image analysis requires OpenRouter API key",
				"Please configure OpenRouter API key or use text-only input",
				false,
			)
		}
		return c.useOpenRouter(req, "vision required")
	}

	// 2. If only text/audio and Groq is available, try Groq first (free!)
	if c.groq != nil && c.preferGroq {
		resp, err := c.tryGroq(req)

		// If Groq succeeds with good confidence, use it
		if err == nil && resp.Confidence >= MinConfidenceThreshold {
			return resp, nil
		}

		// If Groq fails but error is not retryable, return error
		if err != nil {
			if aiErr, ok := err.(*AIError); ok && !aiErr.Retryable {
				// Check if we can fallback to OpenRouter
				if c.openRouter == nil {
					return nil, err
				}
			}
		}

		// Groq failed or low confidence - fallback to OpenRouter if available
		if c.openRouter != nil {
			return c.useOpenRouter(req, fmt.Sprintf("groq fallback (confidence: %.2f)", resp.Confidence))
		}

		// No OpenRouter available, return Groq result anyway
		if resp != nil {
			resp.Suggestions = append(resp.Suggestions,
				fmt.Sprintf("Low confidence (%.2f). Consider providing more specific details.", resp.Confidence))
			return resp, nil
		}

		return nil, err
	}

	// 3. If Groq not available or not preferred, use OpenRouter
	if c.openRouter != nil {
		return c.useOpenRouter(req, "primary service")
	}

	// 4. Fall back to Groq if it's the only option
	if c.groq != nil {
		return c.tryGroq(req)
	}

	return nil, NewAIError(
		ErrorTypeAPIError,
		"No AI service available",
		"Please configure at least one AI service API key",
		false,
	)
}

// tryGroq attempts to use Groq service
func (c *Coordinator) tryGroq(req NutritionRequest) (*NutritionResponse, error) {
	c.mu.Lock()
	c.requestCount++
	c.groqRequests++
	c.mu.Unlock()

	resp, err := c.groq.AnalyzeNutrition(req)
	if err != nil {
		return nil, err
	}

	resp.Cost = 0.0 // Groq is free
	return resp, nil
}

// useOpenRouter uses OpenRouter service with cost tracking
func (c *Coordinator) useOpenRouter(req NutritionRequest, reason string) (*NutritionResponse, error) {
	// Check cost limit before making paid request
	c.mu.RLock()
	currentCost := c.monthlyCost
	c.mu.RUnlock()

	estimatedCost := c.openRouter.GetCostPerRequest()
	if currentCost+estimatedCost > c.costLimit {
		return nil, NewAIError(
			ErrorTypeCostLimitExceeded,
			fmt.Sprintf("Monthly cost limit of $%.2f exceeded", c.costLimit),
			fmt.Sprintf("Current: $%.2f, Estimated request: $%.2f. Consider manual entry or increase limit.",
				currentCost, estimatedCost),
			false,
		)
	}

	// Make request
	c.mu.Lock()
	c.requestCount++
	c.openRouterRequests++
	c.mu.Unlock()

	resp, err := c.openRouter.AnalyzeNutrition(req)
	if err != nil {
		return nil, err
	}

	// Track actual cost
	c.mu.Lock()
	c.monthlyCost += resp.Cost
	c.mu.Unlock()

	// Add routing info to suggestions
	resp.Suggestions = append([]string{
		fmt.Sprintf("Analyzed using %s (%s). Cost: $%.4f", c.openRouter.Name(), reason, resp.Cost),
	}, resp.Suggestions...)

	return resp, nil
}

// checkAndResetMonthlyCost resets cost tracking if a month has passed
func (c *Coordinator) checkAndResetMonthlyCost() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.lastResetTime) >= 30*24*time.Hour { // Approximate month
		c.monthlyCost = 0
		c.lastResetTime = now
		c.requestCount = 0
		c.groqRequests = 0
		c.openRouterRequests = 0
	}
}

// GetStats returns current usage statistics
func (c *Coordinator) GetStats() CoordinatorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CoordinatorStats{
		MonthlyCost:        c.monthlyCost,
		CostLimit:          c.costLimit,
		RemainingBudget:    c.costLimit - c.monthlyCost,
		RequestCount:       c.requestCount,
		GroqRequests:       c.groqRequests,
		OpenRouterRequests: c.openRouterRequests,
		LastResetTime:      c.lastResetTime,
		GroqAvailable:      c.groq != nil,
		OpenRouterAvailable: c.openRouter != nil,
	}
}

// CoordinatorStats contains usage statistics
type CoordinatorStats struct {
	MonthlyCost         float64   `json:"monthly_cost"`
	CostLimit           float64   `json:"cost_limit"`
	RemainingBudget     float64   `json:"remaining_budget"`
	RequestCount        int       `json:"request_count"`
	GroqRequests        int       `json:"groq_requests"`
	OpenRouterRequests  int       `json:"openrouter_requests"`
	LastResetTime       time.Time `json:"last_reset_time"`
	GroqAvailable       bool      `json:"groq_available"`
	OpenRouterAvailable bool      `json:"openrouter_available"`
}

// ResetCosts resets the monthly cost tracking (useful for testing)
func (c *Coordinator) ResetCosts() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.monthlyCost = 0
	c.lastResetTime = time.Now()
	c.requestCount = 0
	c.groqRequests = 0
	c.openRouterRequests = 0
}

// SetCostLimit updates the monthly cost limit
func (c *Coordinator) SetCostLimit(limit float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.costLimit = limit
}

// SupportsVision returns true if any service supports vision
func (c *Coordinator) SupportsVision() bool {
	return c.openRouter != nil && c.openRouter.SupportsVision()
}

// SupportsAudio returns true if any service supports audio
func (c *Coordinator) SupportsAudio() bool {
	if c.groq != nil && c.groq.SupportsAudio() {
		return true
	}
	if c.openRouter != nil && c.openRouter.SupportsAudio() {
		return true
	}
	return false
}

// GetCostPerRequest returns the estimated cost (returns 0 if Groq available)
func (c *Coordinator) GetCostPerRequest() float64 {
	if c.groq != nil && c.preferGroq {
		return 0.0 // Groq is tried first and is free
	}
	if c.openRouter != nil {
		return c.openRouter.GetCostPerRequest()
	}
	return 0.0
}

// Name returns the coordinator name
func (c *Coordinator) Name() string {
	return "AI Coordinator"
}
