# AI Service Layer

Complete AI service implementation for nutrition analysis with Groq, OpenRouter, and intelligent coordination.

## Overview

This package provides a flexible AI service layer that intelligently routes requests between free (Groq) and paid (OpenRouter) AI models based on capabilities, confidence, and cost constraints.

## Architecture

```
NutritionRequest → Coordinator → Decision Logic
                                     ↓
                        ┌────────────┴────────────┐
                        ↓                         ↓
                    Groq (Free)            OpenRouter (Paid)
                    - Text only            - Text + Vision
                    - High speed           - Higher accuracy
                    - No cost              - ~$0.003/request
```

## Components

### 1. Models (`models.go`)
Shared types and interfaces used across all services:

- `AIService` - Interface all services implement
- `NutritionRequest` - Input structure
- `NutritionResponse` - Output structure with confidence scoring
- `ParsedItem` - Individual food items
- `NutritionData` - Nutritional information
- `AIError` - Structured error handling

### 2. Groq Service (`groq.go`)
Free AI service for text-based nutrition analysis:

```go
service := NewGroqService("your-groq-api-key")
resp, err := service.AnalyzeNutrition(NutritionRequest{
    Text: "grilled chicken breast and steamed broccoli",
})
```

**Features:**
- Free tier with generous limits
- Fast response times (< 2 seconds)
- Text and audio support
- Structured JSON responses
- Automatic retry logic

**Limitations:**
- No image analysis support
- May have lower confidence for ambiguous descriptions

### 3. OpenRouter Service (`openrouter.go`)
Paid AI service with vision capabilities:

```go
service := NewOpenRouterService(OpenRouterConfig{
    APIKey: "your-openrouter-key",
    Model: "anthropic/claude-3.5-sonnet", // Optional: override default
})

resp, err := service.AnalyzeNutrition(NutritionRequest{
    Images: [][]byte{imageData},
})
```

**Features:**
- Vision support for food photos
- Higher accuracy for complex meals
- Multiple model options
- Detailed cost tracking

**Cost:**
- Approximately $0.003 per request
- Varies by model and token usage
- Automatically calculated and tracked

### 4. Coordinator (`coordinator.go`)
Intelligent routing between services:

```go
coordinator, err := NewCoordinator(CoordinatorConfig{
    GroqAPIKey:       "groq-key",
    OpenRouterAPIKey: "openrouter-key",
    PreferGroq:       true,
    CostLimit:        10.0, // Monthly limit in USD
})

// Automatically routes to best service
resp, err := coordinator.AnalyzeNutrition(request)
```

**Decision Logic:**

1. **Has images?** → Use OpenRouter (vision required)
2. **Text only** → Try Groq first (free!)
   - If confidence ≥ 0.7 → Return Groq result
   - If confidence < 0.7 → Fallback to OpenRouter
3. **Check cost limit** → Prevent exceeding budget
4. **Track usage** → Monitor costs and request counts

### 5. Mock Service (`mock.go`)
Testing implementation without API keys:

```go
mock := NewMockService(MockConfig{
    SupportsVision: true,
    SupportsAudio: true,
    SimulateDelay: 500 * time.Millisecond,
    FailureRate: 0.1, // 10% random failures for testing
})
```

**Use Cases:**
- Development without API keys
- Automated testing
- Load testing
- Offline development

## Usage Examples

### Basic Text Analysis

```go
req := NutritionRequest{
    Text: "2 scrambled eggs, toast with butter, orange juice",
    UserID: "user-123",
    Preferences: &UserPreferences{
        DietaryRestrictions: []string{"vegetarian"},
        AllergiesWarnings: []string{"dairy"},
        PreferredUnits: "imperial",
    },
}

resp, err := coordinator.AnalyzeNutrition(req)
if err != nil {
    // Handle error
    if aiErr, ok := err.(*AIError); ok {
        switch aiErr.Type {
        case ErrorTypeCostLimitExceeded:
            // Offer manual entry option
        case ErrorTypeRateLimited:
            // Retry later
        }
    }
}

// Use response
fmt.Printf("Total Calories: %.0f\n", resp.TotalNutrition.Calories)
fmt.Printf("Confidence: %.2f\n", resp.Confidence)
fmt.Printf("Cost: $%.4f\n", resp.Cost)
```

### Image Analysis

```go
imageData, _ := ioutil.ReadFile("meal-photo.jpg")

req := NutritionRequest{
    Images: [][]byte{imageData},
    UserID: "user-123",
}

resp, err := coordinator.AnalyzeNutrition(req)
// Will automatically use OpenRouter for vision
```

### Monitoring Usage

```go
stats := coordinator.GetStats()
fmt.Printf("Monthly Cost: $%.2f / $%.2f\n",
    stats.MonthlyCost, stats.CostLimit)
fmt.Printf("Requests: %d (%d Groq, %d OpenRouter)\n",
    stats.RequestCount, stats.GroqRequests, stats.OpenRouterRequests)
```

## Error Handling

All errors implement the `AIError` type with these fields:

- `Type` - Error category (see `ErrorType` constants)
- `Message` - Human-readable message
- `Details` - Additional context
- `Retryable` - Whether the operation can be retried

```go
resp, err := service.AnalyzeNutrition(req)
if err != nil {
    if aiErr, ok := err.(*AIError); ok {
        log.Printf("AI Error [%s]: %s", aiErr.Type, aiErr.Message)
        if aiErr.Retryable {
            // Implement retry logic
        }
    }
}
```

## Configuration

### Environment Variables

```bash
GROQ_API_KEY=your-groq-api-key
OPENROUTER_API_KEY=your-openrouter-api-key
OPENROUTER_MODEL=anthropic/claude-3.5-sonnet
AI_COST_LIMIT=10.0
AI_PREFER_GROQ=true
```

### Coordinator Setup

```go
config := CoordinatorConfig{
    GroqAPIKey:       os.Getenv("GROQ_API_KEY"),
    OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
    OpenRouterModel:  os.Getenv("OPENROUTER_MODEL"),
    CostLimit:        10.0,
    PreferGroq:       true,
}

coordinator, err := NewCoordinator(config)
```

## Testing

Run all tests:
```bash
cd backend/internal/services/ai
go test -v ./...
```

Run specific test:
```bash
go test -v -run TestCoordinator_ImageRouting
```

Run with coverage:
```bash
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Performance Considerations

### Response Times
- Groq: ~1-2 seconds for text
- OpenRouter: ~2-4 seconds for text, ~3-6 seconds for images
- Mock: Instant (configurable delay for testing)

### Cost Optimization
1. Use Groq for simple text queries (free)
2. Reserve OpenRouter for:
   - Image analysis (required)
   - Low-confidence Groq results
   - Complex multi-item meals

### Rate Limits
- Groq: 30 requests/minute (free tier)
- OpenRouter: Varies by model and account
- Implement exponential backoff for retries

## Integration Example

```go
package main

import (
    "log"
    "github.com/yourorg/lumen/backend/internal/services/ai"
)

func main() {
    // Initialize coordinator
    coordinator, err := ai.NewCoordinator(ai.CoordinatorConfig{
        GroqAPIKey:       "your-groq-key",
        OpenRouterAPIKey: "your-openrouter-key",
        PreferGroq:       true,
        CostLimit:        25.0,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Make request
    resp, err := coordinator.AnalyzeNutrition(ai.NutritionRequest{
        Text: "chicken salad with ranch dressing",
        UserID: "user-123",
        Preferences: &ai.UserPreferences{
            PreferredUnits: "metric",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Process response
    for _, item := range resp.Items {
        log.Printf("%s: %.0f calories (confidence: %.2f)",
            item.Name, item.Nutrition.Calories, item.Confidence)
    }

    // Check usage
    stats := coordinator.GetStats()
    log.Printf("Monthly cost: $%.2f", stats.MonthlyCost)
}
```

## API Keys

### Getting API Keys

**Groq:**
1. Visit https://console.groq.com
2. Sign up for free account
3. Generate API key in dashboard
4. Free tier: 30 requests/minute

**OpenRouter:**
1. Visit https://openrouter.ai
2. Create account
3. Add credits (pay-as-you-go)
4. Generate API key
5. Cost: ~$0.003/request for Claude 3.5 Sonnet

## Best Practices

1. **Always use the Coordinator** - Don't call Groq/OpenRouter directly
2. **Set appropriate cost limits** - Prevent unexpected charges
3. **Handle all error types** - Provide fallback for manual entry
4. **Cache results** - Avoid duplicate API calls for same input
5. **Monitor usage** - Track costs and optimize routing logic
6. **Validate input** - Check for empty requests before API calls
7. **Use mock for development** - Faster iteration without API costs

## Troubleshooting

### "No API key" errors
- Verify environment variables are set
- Check API key format (no spaces or quotes)
- Ensure keys are valid (test on provider dashboards)

### "Cost limit exceeded" errors
- Increase `CostLimit` in configuration
- Implement manual entry fallback
- Monitor usage with `GetStats()`

### Low confidence scores
- Provide more specific food descriptions
- Use portion sizes ("1 cup" vs "some rice")
- Include preparation methods ("grilled" vs "chicken")
- Consider using OpenRouter for better accuracy

### Rate limit errors
- Implement exponential backoff
- Use caching to reduce duplicate requests
- Spread requests over time
- Upgrade API tier if needed

## Future Enhancements

- [ ] Audio transcription with Groq Whisper
- [ ] Batch processing for multiple items
- [ ] Caching layer for common foods
- [ ] Custom fine-tuning for specific diets
- [ ] Multi-language support
- [ ] Barcode scanning integration
- [ ] Historical accuracy tracking
- [ ] A/B testing between models
