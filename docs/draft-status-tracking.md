# Draft Status Tracking Implementation

## Overview

This document describes the implementation of asynchronous draft status tracking for AI meal parsing in Lumen's backend API.

## Problem Statement

The original meal parsing flow was synchronous, requiring clients to wait 3-5 seconds for AI processing to complete. This created a poor user experience with long loading times.

## Solution

Implement an asynchronous processing pattern where:
1. Client sends meal description
2. Server creates a draft meal with status='analyzing'
3. Server returns immediately with `draft_id`
4. AI processing happens in background
5. Client polls for status updates
6. When ready, client retrieves parsed items

## Architecture

### Database Changes

**Migration:** `backend/migrations/008_add_draft_status.up.sql`

```sql
-- Add draft tracking columns to meals table
ALTER TABLE meals ADD COLUMN is_draft BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE meals ADD COLUMN draft_status TEXT DEFAULT NULL;
ALTER TABLE meals ADD COLUMN draft_error TEXT DEFAULT NULL;

-- Add constraint for valid statuses
ALTER TABLE meals ADD CONSTRAINT valid_draft_status
    CHECK (draft_status IS NULL OR draft_status IN ('analyzing', 'ready', 'error'));

-- Add index for efficient querying
CREATE INDEX idx_meals_draft_status
    ON meals(user_id, draft_status)
    WHERE is_draft = TRUE;
```

### Model Changes

**File:** `backend/internal/domain/nutrition/meals/models.go`

Added new types:

```go
type DraftStatus string

const (
    DraftStatusAnalyzing DraftStatus = "analyzing" // AI is processing
    DraftStatusReady     DraftStatus = "ready"     // Processing complete
    DraftStatusError     DraftStatus = "error"     // Processing failed
)

type Meal struct {
    // ... existing fields
    IsDraft       bool         `json:"is_draft"`
    DraftStatus   *DraftStatus `json:"draft_status,omitempty"`
    DraftError    *string      `json:"draft_error,omitempty"`
}

type ParseMealResponse struct {
    DraftID   uuid.UUID   `json:"draft_id"`
    Status    DraftStatus `json:"status"`
    DraftMeal *struct {
        Items      []DraftMealItem `json:"items"`
        Total      NutritionTotals `json:"total"`
        Confidence float64         `json:"confidence"`
        CostUSD    float64         `json:"cost_usd"`
    } `json:"draft_meal,omitempty"`
}

type DraftStatusResponse struct {
    DraftID uuid.UUID        `json:"draft_id"`
    Status  DraftStatus      `json:"status"`
    Items   []DraftMealItem  `json:"items,omitempty"`
    Total   *NutritionTotals `json:"total,omitempty"`
    Error   *string          `json:"error,omitempty"`
}
```

### Repository Layer

**File:** `backend/internal/domain/nutrition/meals/repository.go`

New methods:

```go
type Repository interface {
    // Create draft meal with status='analyzing'
    CreateDraftMeal(ctx context.Context, userID uuid.UUID, meal *Meal) (uuid.UUID, error)

    // Update draft status and add items when ready
    UpdateDraftStatus(ctx context.Context, draftID uuid.UUID, status DraftStatus, items []MealItem, errMsg *string) error

    // Get current draft status and items
    GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*Meal, []MealItem, error)
}
```

### Service Layer

**File:** `backend/internal/domain/nutrition/meals/service_draft.go`

Key implementation details:

```go
// ParseMealAsync creates draft and returns immediately
func (s *service) ParseMealAsync(ctx context.Context, userID uuid.UUID, req *ParseMealRequest) (*ParseMealResponse, error) {
    // 1. Validate request
    // 2. Create draft meal with status='analyzing'
    draftID, err := s.repo.CreateDraftMeal(ctx, userID, meal)

    // 3. Start background processing
    go s.processAIInBackground(userID, draftID, req.Description, req.Photos, req.IdempotencyKey)

    // 4. Return immediately
    return &ParseMealResponse{
        DraftID: draftID,
        Status:  DraftStatusAnalyzing,
    }, nil
}

// processAIInBackground handles AI parsing asynchronously
func (s *service) processAIInBackground(userID, draftID uuid.UUID, description string, photos []string, idempotencyKey string) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Call AI coordinator
    items, confidence, cost, err := s.aiCoordinator.ParseMeal(ctx, description, photos)
    if err != nil {
        // Update status to 'error'
        errMsg := err.Error()
        s.repo.UpdateDraftStatus(ctx, draftID, DraftStatusError, nil, &errMsg)
        return
    }

    // Update status to 'ready' with items
    s.updateDraftWithResults(ctx, draftID, items, confidence, cost)
}

// GetDraftStatus retrieves current processing status
func (s *service) GetDraftStatus(ctx context.Context, userID, draftID uuid.UUID) (*DraftStatusResponse, error) {
    meal, items, err := s.repo.GetDraftStatus(ctx, userID, draftID)

    response := &DraftStatusResponse{
        DraftID: meal.ID,
        Status:  *meal.DraftStatus,
    }

    // Include items and totals if ready
    if *meal.DraftStatus == DraftStatusReady {
        response.Items = convertToD raftItems(items)
        response.Total = calculateTotals(items)
    }

    // Include error message if failed
    if *meal.DraftStatus == DraftStatusError {
        response.Error = meal.DraftError
    }

    return response, nil
}
```

### Handler Layer

**File:** `backend/internal/domain/nutrition/meals/handler_draft.go`

```go
// GET /api/v1/meals/draft/{id}/status
func (h *Handler) GetDraftStatus(w http.ResponseWriter, r *http.Request) {
    userID := extractUserID(r)
    draftID := extractDraftID(r)

    result, err := h.service.GetDraftStatus(r.Context(), userID, draftID)
    if err != nil {
        h.errorResponse(w, http.StatusNotFound, "draft not found", nil)
        return
    }

    h.successResponse(w, http.StatusOK, result)
}
```

## API Flow

### 1. Client Submits Meal for Parsing

**Request:**
```http
POST /api/v1/meals/parse
Content-Type: application/json

{
  "description": "2 scrambled eggs with toast and butter",
  "meal_type": "breakfast",
  "consumed_at": "2024-01-15T08:30:00Z",
  "photos": [],
  "idempotency_key": "user123-20240115-0830"
}
```

**Response (HTTP 202 Accepted):**
```json
{
  "draft_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "analyzing"
}
```

### 2. Client Polls for Status

**Request:**
```http
GET /api/v1/meals/draft/550e8400-e29b-41d4-a716-446655440000/status
```

**Response (Still Processing):**
```json
{
  "draft_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "analyzing"
}
```

**Response (Ready):**
```json
{
  "draft_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "ready",
  "items": [
    {
      "name": "Scrambled Eggs",
      "quantity": 2,
      "unit": "whole",
      "calories": 140,
      "protein_g": 12,
      "carbs_g": 2,
      "fat_g": 10,
      "fiber_g": 0
    },
    {
      "name": "Toast with Butter",
      "quantity": 2,
      "unit": "slices",
      "calories": 220,
      "protein_g": 6,
      "carbs_g": 30,
      "fat_g": 8,
      "fiber_g": 2
    }
  ],
  "total": {
    "calories": 360,
    "protein_g": 18,
    "carbs_g": 32,
    "fat_g": 18,
    "fiber_g": 2
  }
}
```

**Response (Error):**
```json
{
  "draft_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "error",
  "error": "AI service timeout - please try again"
}
```

## Error Handling

### Background Processing Errors

When AI processing fails:
1. Goroutine catches error
2. Updates draft status to 'error'
3. Stores error message in `draft_error` column
4. Logs error for monitoring
5. Client receives error on next status poll

### Repository Errors

- Draft not found → HTTP 404
- Invalid draft_id → HTTP 400
- Unauthorized access → HTTP 401

## Performance Considerations

### Concurrency Safety

- Each draft processed in separate goroutine
- Database updates are atomic
- Context timeout prevents hanging goroutines
- No shared state between concurrent requests

### Resource Management

- Background context with 30-second timeout
- Goroutines cleaned up after completion
- Database connections returned to pool
- Proper error logging for debugging

### Polling Strategy

Recommended client-side polling:
- Initial delay: 500ms
- Max attempts: 20
- Backoff: Exponential (500ms, 1s, 2s, 4s, 8s)
- Max interval: 8s
- Total timeout: ~60s

## Testing

### Unit Tests

**File:** `backend/internal/domain/nutrition/meals/service_draft_test.go`

Test coverage includes:
- Draft creation with status='analyzing'
- Background AI processing success
- Background AI processing failure
- Status polling (analyzing, ready, error)
- Invalid requests (future date, invalid photos)
- Concurrent draft processing
- Error handling and recovery

Run tests:
```bash
cd backend
go test ./internal/domain/nutrition/meals -v -run TestDraft
```

### Integration Tests

1. Create draft meal
2. Verify draft_id returned
3. Poll status endpoint
4. Verify status transitions
5. Verify items populated when ready
6. Test error scenarios

## Monitoring

### Metrics to Track

- Draft creation rate
- Average AI processing time
- Success vs error rate
- Background goroutine count
- Database query performance

### Logging

Key log events:
```go
logger.Info("draft meal created", "draft_id", draftID)
logger.Info("starting background AI processing", "draft_id", draftID)
logger.Info("draft meal ready", "draft_id", draftID, "item_count", len(items))
logger.Error("AI parsing failed", "draft_id", draftID, "error", err)
```

## Migration Path

### Backward Compatibility

The new async flow is **compatible** with existing sync flow:
- Old clients: Continue using synchronous parsing
- New clients: Use async draft status tracking
- Gradual migration supported

### Deployment Steps

1. Run database migration: `008_add_draft_status.up.sql`
2. Deploy backend code with new endpoints
3. Update frontend to use draft status polling
4. Monitor error rates and performance
5. Deprecate synchronous endpoint (optional)

## Security Considerations

### Authorization

- Draft meals are user-scoped
- Only owner can check draft status
- RLS policies prevent cross-user access

### Rate Limiting

- Apply rate limits to draft creation
- Prevent polling abuse with backoff
- Monitor for suspicious patterns

### Data Validation

- Validate all inputs before creating draft
- Sanitize error messages before storing
- Prevent injection attacks

## Future Enhancements

1. **WebSocket Support**
   - Real-time status updates
   - No polling required
   - Lower latency

2. **Draft Expiration**
   - Auto-delete drafts after 24 hours
   - Cleanup job for abandoned drafts

3. **Batch Processing**
   - Process multiple meals in single request
   - Improved throughput

4. **Retry Logic**
   - Automatic retry on transient failures
   - Exponential backoff

5. **Progress Reporting**
   - Fine-grained status (parsing, analyzing, validating)
   - Percentage complete

## References

- Controller-Service-Repository Pattern
- Background Job Processing with Goroutines
- Database Indexing Best Practices
- RESTful API Design
- Idempotency in Distributed Systems
