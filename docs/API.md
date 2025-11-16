# API Documentation

Complete API reference for Lumen Nutrition Tracker Backend.

**Base URL**: `http://localhost:8080`
**API Version**: v1
**Content-Type**: `application/json`

## Table of Contents

- [Authentication](#authentication)
- [Error Codes](#error-codes)
- [Rate Limits](#rate-limits)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Meals](#meals)
  - [Weight Tracking](#weight-tracking)
  - [Goals](#goals)
  - [Analytics](#analytics)
  - [Templates](#templates)

---

## Authentication

Authentication is controlled via the `FEATURE_AUTH` environment variable.

### Development Mode (Auth Disabled)
```bash
FEATURE_AUTH=false
```
No authentication required. Requests use a default development user ID.

### Production Mode (Auth Enabled)
```bash
FEATURE_AUTH=true
```

Include JWT token in Authorization header:
```http
Authorization: Bearer <your-jwt-token>
```

**Getting a Token** (via Supabase Auth):
```bash
POST https://your-project.supabase.co/auth/v1/token?grant_type=password
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password"
}
```

---

## Error Codes

All errors return JSON with consistent structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | OK - Request successful |
| 201 | Created - Resource created successfully |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid auth token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Duplicate resource |
| 422 | Unprocessable Entity - Validation failed |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server error |
| 503 | Service Unavailable - Service temporarily down |

### Error Response Examples

**Validation Error**:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "fields": {
        "weight": "must be between 1 and 500",
        "measured_at": "is required"
      }
    }
  }
}
```

**Rate Limit Exceeded**:
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Daily AI parsing limit reached",
    "details": {
      "limit": 50,
      "reset_at": "2025-11-16T00:00:00Z"
    }
  }
}
```

---

## Rate Limits

### AI Parsing Limits

- **Free Tier (Groq)**: 14,400 requests/day per API key
- **Per User**: 50 parses/day (configurable via `RATE_LIMIT_AI_PARSE_PER_DAY`)
- **Cost Limit**: $10/user/month, $1000 global/month

### General API Limits

- **Requests**: 1000 requests/hour per user
- **Burst**: 100 requests/minute

Rate limit headers included in responses:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 987
X-RateLimit-Reset: 1699999999
```

---

## Endpoints

### Health Check

#### GET /health

Check server status and availability.

**Request**:
```bash
curl http://localhost:8080/health
```

**Response** (200 OK):
```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T10:30:00Z",
  "database": "connected",
  "cache": "active"
}
```

---

### Meals

#### POST /api/v1/meals/parse

Parse meal description using AI to extract foods and nutrition.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/meals/parse \
  -H "Content-Type: application/json" \
  -d '{
    "description": "2 scrambled eggs with spinach and whole wheat toast",
    "meal_type": "breakfast",
    "idempotency_key": "unique-request-id-123"
  }'
```

**Request Body**:
```json
{
  "description": "string (required, 1-2000 chars)",
  "photos": ["photo_url_1", "photo_url_2"],
  "meal_type": "breakfast|lunch|dinner|snack (required)",
  "consumed_at": "2025-11-15T08:30:00Z (optional, defaults to now)",
  "idempotency_key": "string (required, prevents duplicate parses)"
}
```

**Response** (200 OK):
```json
{
  "items": [
    {
      "food_name": "Scrambled eggs",
      "serving_size": "2 large eggs",
      "calories": 186,
      "protein_g": 12.6,
      "carbs_g": 1.2,
      "fat_g": 14.0,
      "fiber_g": 0.0
    },
    {
      "food_name": "Spinach",
      "serving_size": "1 cup cooked",
      "calories": 41,
      "protein_g": 5.3,
      "carbs_g": 6.8,
      "fat_g": 0.5,
      "fiber_g": 4.3
    },
    {
      "food_name": "Whole wheat toast",
      "serving_size": "2 slices",
      "calories": 160,
      "protein_g": 8.0,
      "carbs_g": 28.0,
      "fat_g": 2.0,
      "fiber_g": 6.0
    }
  ],
  "total_calories": 387,
  "total_protein_g": 25.9,
  "total_carbs_g": 36.0,
  "total_fat_g": 16.5,
  "total_fiber_g": 10.3,
  "ai_confidence": "high",
  "provider": "groq",
  "cost_usd": 0.0,
  "session_id": "parse_abc123"
}
```

**Error Responses**:
- `400 Bad Request` - Invalid input (missing description, invalid meal type)
- `429 Too Many Requests` - Daily AI limit exceeded
- `503 Service Unavailable` - AI provider unavailable

---

#### POST /api/v1/meals/confirm

Confirm and save a parsed meal after user edits.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/meals/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "meal_type": "breakfast",
    "consumed_at": "2025-11-15T08:30:00Z",
    "items": [
      {
        "food_name": "Scrambled eggs",
        "serving_size": "2 large eggs",
        "calories": 186,
        "protein_g": 12.6,
        "carbs_g": 1.2,
        "fat_g": 14.0,
        "fiber_g": 0.0
      }
    ]
  }'
```

**Request Body**:
```json
{
  "meal_type": "breakfast|lunch|dinner|snack (required)",
  "consumed_at": "2025-11-15T08:30:00Z (required)",
  "items": [
    {
      "food_name": "string (required)",
      "serving_size": "string (required)",
      "calories": 0.0,
      "protein_g": 0.0,
      "carbs_g": 0.0,
      "fat_g": 0.0,
      "fiber_g": 0.0
    }
  ],
  "photos": ["photo_url_1"],
  "notes": "Optional notes"
}
```

**Response** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-uuid",
  "meal_type": "breakfast",
  "consumed_at": "2025-11-15T08:30:00Z",
  "total_calories": 387,
  "total_protein_g": 25.9,
  "total_carbs_g": 36.0,
  "total_fat_g": 16.5,
  "total_fiber_g": 10.3,
  "items": [...],
  "photos": [],
  "notes": "",
  "created_at": "2025-11-15T08:31:00Z",
  "updated_at": "2025-11-15T08:31:00Z"
}
```

---

#### GET /api/v1/meals

List user's meals with pagination and filtering.

**Request**:
```bash
curl "http://localhost:8080/api/v1/meals?page=1&page_size=20&start_date=2025-11-01&end_date=2025-11-15&meal_type=breakfast"
```

**Query Parameters**:
- `page` (int, default: 1) - Page number
- `page_size` (int, default: 20, max: 100) - Items per page
- `start_date` (string, YYYY-MM-DD) - Filter by consumed date
- `end_date` (string, YYYY-MM-DD) - Filter by consumed date
- `meal_type` (string) - Filter by meal type

**Response** (200 OK):
```json
{
  "meals": [
    {
      "id": "meal-uuid",
      "meal_type": "breakfast",
      "consumed_at": "2025-11-15T08:30:00Z",
      "total_calories": 387,
      "total_protein_g": 25.9,
      "total_carbs_g": 36.0,
      "total_fat_g": 16.5,
      "total_fiber_g": 10.3,
      "items_count": 3,
      "photos": [],
      "notes": ""
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

---

#### GET /api/v1/meals/{id}

Get detailed meal information including all items.

**Request**:
```bash
curl http://localhost:8080/api/v1/meals/550e8400-e29b-41d4-a716-446655440000
```

**Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-uuid",
  "meal_type": "breakfast",
  "consumed_at": "2025-11-15T08:30:00Z",
  "total_calories": 387,
  "total_protein_g": 25.9,
  "total_carbs_g": 36.0,
  "total_fat_g": 16.5,
  "total_fiber_g": 10.3,
  "items": [
    {
      "id": "item-uuid",
      "food_name": "Scrambled eggs",
      "serving_size": "2 large eggs",
      "calories": 186,
      "protein_g": 12.6,
      "carbs_g": 1.2,
      "fat_g": 14.0,
      "fiber_g": 0.0
    }
  ],
  "photos": [],
  "notes": "",
  "created_at": "2025-11-15T08:31:00Z",
  "updated_at": "2025-11-15T08:31:00Z"
}
```

**Error Responses**:
- `404 Not Found` - Meal doesn't exist or doesn't belong to user

---

#### PUT /api/v1/meals/{id}

Update an existing meal.

**Request**:
```bash
curl -X PUT http://localhost:8080/api/v1/meals/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "notes": "Added butter to toast"
  }'
```

**Request Body** (all fields optional):
```json
{
  "consumed_at": "2025-11-15T09:00:00Z",
  "notes": "Updated notes",
  "items": [...]  // Replace all items
}
```

**Response** (200 OK): Full meal object

---

#### DELETE /api/v1/meals/{id}

Soft delete a meal.

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/meals/550e8400-e29b-41d4-a716-446655440000
```

**Response** (204 No Content)

---

### Weight Tracking

#### POST /api/v1/weight

Record a new weight entry.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/weight \
  -H "Content-Type: application/json" \
  -d '{
    "weight": 75.5,
    "measured_at": "2025-11-15T07:00:00Z",
    "notes": "Morning weight before breakfast"
  }'
```

**Request Body**:
```json
{
  "weight": 75.5,  // kg, required, min: 1, max: 500
  "measured_at": "2025-11-15T07:00:00Z",  // required
  "notes": "Optional notes"
}
```

**Response** (201 Created):
```json
{
  "id": "weight-uuid",
  "user_id": "user-uuid",
  "weight": 75.5,
  "measured_at": "2025-11-15T07:00:00Z",
  "notes": "Morning weight before breakfast",
  "created_at": "2025-11-15T07:01:00Z",
  "updated_at": "2025-11-15T07:01:00Z"
}
```

---

#### GET /api/v1/weight

List weight entries with pagination.

**Request**:
```bash
curl "http://localhost:8080/api/v1/weight?page=1&page_size=30&start_date=2025-10-01&end_date=2025-11-15"
```

**Query Parameters**:
- `page` (int, default: 1)
- `page_size` (int, default: 30, max: 100)
- `start_date` (string, YYYY-MM-DD)
- `end_date` (string, YYYY-MM-DD)

**Response** (200 OK):
```json
{
  "entries": [
    {
      "id": "weight-uuid",
      "weight": 75.5,
      "measured_at": "2025-11-15T07:00:00Z",
      "notes": "Morning weight"
    }
  ],
  "total": 45,
  "page": 1,
  "page_size": 30,
  "total_pages": 2
}
```

---

#### GET /api/v1/weight/latest

Get the most recent weight entry.

**Request**:
```bash
curl http://localhost:8080/api/v1/weight/latest
```

**Response** (200 OK):
```json
{
  "id": "weight-uuid",
  "weight": 75.5,
  "measured_at": "2025-11-15T07:00:00Z",
  "notes": "Morning weight",
  "created_at": "2025-11-15T07:01:00Z"
}
```

---

#### GET /api/v1/weight/trend

Get weight trend statistics.

**Request**:
```bash
curl http://localhost:8080/api/v1/weight/trend
```

**Response** (200 OK):
```json
{
  "latest_weight": 75.5,
  "latest_date": "2025-11-15T07:00:00Z",
  "average_7day": 75.8,
  "average_30day": 76.2,
  "rate_of_change": -0.3,  // kg per week
  "entries_count": 45,
  "trend": "decreasing"
}
```

---

#### PUT /api/v1/weight/{id}

Update a weight entry.

**Request**:
```bash
curl -X PUT http://localhost:8080/api/v1/weight/weight-uuid \
  -H "Content-Type: application/json" \
  -d '{
    "weight": 75.6,
    "notes": "Updated measurement"
  }'
```

**Response** (200 OK): Full weight entry object

---

#### DELETE /api/v1/weight/{id}

Delete a weight entry.

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/weight/weight-uuid
```

**Response** (204 No Content)

---

### Goals

#### GET /api/v1/goals

Get user's nutrition goals.

**Request**:
```bash
curl http://localhost:8080/api/v1/goals
```

**Response** (200 OK):
```json
{
  "id": 1,
  "user_id": 1,
  "daily_calories": 2200,
  "daily_protein_g": 165,
  "daily_carbs_g": 220,
  "daily_fat_g": 73,
  "daily_fiber_g": 30,
  "tdee": 2200,
  "bmr": 1833,
  "activity_level": "active",
  "daily_goals": {
    "monday": {
      "calories": 2500,
      "protein_g": 180
    }
  },
  "created_at": "2025-11-01T00:00:00Z",
  "updated_at": "2025-11-15T00:00:00Z"
}
```

---

#### PUT /api/v1/goals

Update nutrition goals.

**Request**:
```bash
curl -X PUT http://localhost:8080/api/v1/goals \
  -H "Content-Type: application/json" \
  -d '{
    "daily_calories": 2400,
    "daily_protein_g": 180,
    "daily_fiber_g": 35
  }'
```

**Request Body** (all fields optional):
```json
{
  "daily_calories": 2400,
  "daily_protein_g": 180,
  "daily_carbs_g": 240,
  "daily_fat_g": 80,
  "daily_fiber_g": 35
}
```

**Response** (200 OK): Full goals object

---

#### POST /api/v1/goals/calculate

Calculate TDEE from user data.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/goals/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "age": 30,
    "sex": "male",
    "weight_kg": 75.5,
    "height_cm": 180,
    "activity_level": "active"
  }'
```

**Request Body**:
```json
{
  "age": 30,  // required, 13-120
  "sex": "male|female",  // required
  "weight_kg": 75.5,  // required, 30-300
  "height_cm": 180,  // required, 100-250
  "activity_level": "sedentary|lightly_active|active|very_active"  // required
}
```

**Response** (200 OK):
```json
{
  "bmr": 1833,
  "tdee": 2841,
  "activity_level": "active",
  "activity_multiplier": 1.55,
  "recommended_goals": {
    "calories": 2841,
    "protein_g": 151,  // 2g per kg bodyweight
    "carbs_g": 356,    // 50% of calories
    "fat_g": 79,       // 25% of calories
    "fiber_g": 30      // Standard recommendation
  },
  "calculation_method": "Mifflin-St Jeor"
}
```

---

#### GET /api/v1/goals/daily

Get overview of daily goals.

**Request**:
```bash
curl http://localhost:8080/api/v1/goals/daily
```

**Response** (200 OK):
```json
{
  "default_goals": {
    "calories": 2200,
    "protein_g": 165,
    "carbs_g": 220,
    "fat_g": 73,
    "fiber_g": 30
  },
  "custom_days": {
    "monday": {
      "calories": 2500,
      "protein_g": 180
    },
    "saturday": {
      "calories": 2800,
      "protein_g": 200
    }
  }
}
```

---

#### PUT /api/v1/goals/daily/{day}

Set custom goals for a specific day.

**Request**:
```bash
curl -X PUT http://localhost:8080/api/v1/goals/daily/monday \
  -H "Content-Type: application/json" \
  -d '{
    "calories": 2500,
    "protein_g": 180
  }'
```

**Path Parameters**:
- `day`: monday|tuesday|wednesday|thursday|friday|saturday|sunday

**Request Body** (all fields optional):
```json
{
  "calories": 2500,
  "protein_g": 180,
  "carbs_g": 250,
  "fat_g": 83,
  "fiber_g": 35
}
```

**Response** (200 OK): Custom day goals

---

#### DELETE /api/v1/goals/daily/{day}

Remove custom goals for a day (revert to default).

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/goals/daily/monday
```

**Response** (204 No Content)

---

### Analytics

#### GET /api/v1/analytics/daily

Get daily nutrition analytics.

**Request**:
```bash
curl "http://localhost:8080/api/v1/analytics/daily?date=2025-11-15&timezone=America/New_York"
```

**Query Parameters**:
- `date` (string, YYYY-MM-DD, optional) - Defaults to today
- `timezone` (string, optional) - IANA timezone, defaults to UTC

**Response** (200 OK):
```json
{
  "date": "2025-11-15",
  "totals": {
    "calories": 1847,
    "protein_g": 142,
    "carbs_g": 189,
    "fat_g": 58,
    "fiber_g": 28
  },
  "goals": {
    "calories": 2200,
    "protein_g": 165,
    "carbs_g": 220,
    "fat_g": 73,
    "fiber_g": 30
  },
  "progress": {
    "calories_percent": 84,
    "protein_percent": 86,
    "carbs_percent": 86,
    "fat_percent": 79,
    "fiber_percent": 93
  },
  "remaining": {
    "calories": 353,
    "protein_g": 23,
    "carbs_g": 31,
    "fat_g": 15,
    "fiber_g": 2
  },
  "meals_count": 3
}
```

---

#### GET /api/v1/analytics/weekly

Get 7-day nutrition trends.

**Request**:
```bash
curl "http://localhost:8080/api/v1/analytics/weekly?start_date=2025-11-08&timezone=UTC"
```

**Query Parameters**:
- `start_date` (string, YYYY-MM-DD, optional) - Defaults to 7 days ago
- `timezone` (string, optional) - Defaults to UTC

**Response** (200 OK):
```json
{
  "start_date": "2025-11-08",
  "end_date": "2025-11-15",
  "daily_totals": [
    {
      "date": "2025-11-08",
      "calories": 2103,
      "protein_g": 156,
      "carbs_g": 210,
      "fat_g": 71,
      "fiber_g": 32
    }
  ],
  "averages": {
    "calories": 2047,
    "protein_g": 149,
    "carbs_g": 201,
    "fat_g": 68,
    "fiber_g": 29
  },
  "goals": {
    "calories": 2200,
    "protein_g": 165,
    "carbs_g": 220,
    "fat_g": 73,
    "fiber_g": 30
  }
}
```

---

#### GET /api/v1/analytics/trends

Get nutrition trends for custom date range.

**Request**:
```bash
curl "http://localhost:8080/api/v1/analytics/trends?start_date=2025-11-01&end_date=2025-11-15&group_by=day"
```

**Query Parameters**:
- `start_date` (string, YYYY-MM-DD, required)
- `end_date` (string, YYYY-MM-DD, required, max 90 days from start)
- `group_by` (string, optional) - day|week|month (default: day)
- `timezone` (string, optional)

**Response** (200 OK):
```json
{
  "start_date": "2025-11-01",
  "end_date": "2025-11-15",
  "group_by": "day",
  "data_points": [
    {
      "date": "2025-11-01",
      "calories": 2156,
      "protein_g": 162,
      "carbs_g": 215,
      "fat_g": 72,
      "fiber_g": 31,
      "meals_count": 4
    }
  ],
  "summary": {
    "total_days": 15,
    "days_tracked": 14,
    "average_calories": 2047,
    "average_protein_g": 149
  }
}
```

---

#### GET /api/v1/analytics/distribution

Get macro distribution (for pie chart).

**Request**:
```bash
curl "http://localhost:8080/api/v1/analytics/distribution?start_date=2025-11-01&end_date=2025-11-15"
```

**Query Parameters**:
- `start_date` (string, YYYY-MM-DD, required)
- `end_date` (string, YYYY-MM-DD, required)

**Response** (200 OK):
```json
{
  "totals": {
    "calories": 28659,
    "protein_g": 2086,
    "carbs_g": 2814,
    "fat_g": 952,
    "fiber_g": 406
  },
  "distribution": {
    "protein_percent": 29,    // % of calories from protein
    "carbs_percent": 39,      // % of calories from carbs
    "fat_percent": 30,        // % of calories from fat
    "fiber_g_per_1000cal": 14.2
  },
  "recommended": {
    "protein_percent": 25,
    "carbs_percent": 50,
    "fat_percent": 25
  }
}
```

---

#### GET /api/v1/analytics/progress

Get goal progress tracking.

**Request**:
```bash
curl "http://localhost:8080/api/v1/analytics/progress?days=30"
```

**Query Parameters**:
- `days` (int, optional) - Number of days to analyze (default: 7, max: 90)

**Response** (200 OK):
```json
{
  "period_days": 30,
  "goals": {
    "calories": 2200,
    "protein_g": 165,
    "carbs_g": 220,
    "fat_g": 73,
    "fiber_g": 30
  },
  "achievements": {
    "days_tracked": 28,
    "days_hit_calorie_goal": 22,
    "days_hit_protein_goal": 25,
    "days_hit_fiber_goal": 20,
    "consistency_percent": 93
  },
  "averages": {
    "calories": 2047,
    "protein_g": 149,
    "calories_vs_goal": -7,     // % under goal
    "protein_vs_goal": -10
  }
}
```

---

### Templates

#### POST /api/v1/templates

Create a new meal template.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Protein Bowl",
    "meal_type": "breakfast",
    "items": [
      {
        "food_name": "Greek yogurt",
        "serving_size": "1 cup",
        "calories": 130,
        "protein_g": 23,
        "carbs_g": 9,
        "fat_g": 0,
        "fiber_g": 0
      }
    ]
  }'
```

**Request Body**:
```json
{
  "name": "string (required, max 100 chars)",
  "meal_type": "breakfast|lunch|dinner|snack (optional)",
  "items": [
    {
      "food_name": "string (required)",
      "serving_size": "string (required)",
      "calories": 0.0,
      "protein_g": 0.0,
      "carbs_g": 0.0,
      "fat_g": 0.0,
      "fiber_g": 0.0
    }
  ],
  "notes": "Optional description"
}
```

**Response** (201 Created):
```json
{
  "id": "template-uuid",
  "user_id": "user-uuid",
  "name": "Morning Protein Bowl",
  "meal_type": "breakfast",
  "items": [...],
  "total_calories": 130,
  "total_protein_g": 23,
  "notes": "",
  "times_used": 0,
  "created_at": "2025-11-15T10:00:00Z",
  "updated_at": "2025-11-15T10:00:00Z"
}
```

---

#### POST /api/v1/templates/from-meal

Create template from an existing meal.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/templates/from-meal \
  -H "Content-Type: application/json" \
  -d '{
    "meal_id": "meal-uuid",
    "template_name": "My Favorite Breakfast"
  }'
```

**Response** (201 Created): Full template object

---

#### GET /api/v1/templates

List user's templates.

**Request**:
```bash
curl "http://localhost:8080/api/v1/templates?meal_type=breakfast&sort_by=times_used&order=desc"
```

**Query Parameters**:
- `meal_type` (string, optional) - Filter by meal type
- `sort_by` (string, optional) - name|times_used|created_at (default: created_at)
- `order` (string, optional) - asc|desc (default: desc)
- `page` (int, default: 1)
- `page_size` (int, default: 20)

**Response** (200 OK):
```json
{
  "templates": [
    {
      "id": "template-uuid",
      "name": "Morning Protein Bowl",
      "meal_type": "breakfast",
      "total_calories": 130,
      "total_protein_g": 23,
      "times_used": 15,
      "items_count": 3,
      "created_at": "2025-11-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 8,
    "total_pages": 1
  }
}
```

---

#### GET /api/v1/templates/{id}

Get template details.

**Request**:
```bash
curl http://localhost:8080/api/v1/templates/template-uuid
```

**Response** (200 OK): Full template object with all items

---

#### PUT /api/v1/templates/{id}

Update a template.

**Request**:
```bash
curl -X PUT http://localhost:8080/api/v1/templates/template-uuid \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Template Name"
  }'
```

**Response** (200 OK): Full template object

---

#### DELETE /api/v1/templates/{id}

Delete a template.

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/templates/template-uuid
```

**Response** (204 No Content)

---

#### POST /api/v1/templates/{id}/use

Use a template to create a meal.

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/templates/template-uuid/use \
  -H "Content-Type: application/json" \
  -d '{
    "consumed_at": "2025-11-15T08:30:00Z"
  }'
```

**Request Body**:
```json
{
  "consumed_at": "2025-11-15T08:30:00Z (optional, defaults to now)"
}
```

**Response** (201 Created):
```json
{
  "meal_id": "new-meal-uuid",
  "template_id": "template-uuid",
  "message": "Meal created from template"
}
```

---

## Pagination

List endpoints support pagination with consistent parameters:

**Query Parameters**:
- `page` (int, default: 1) - Page number (1-indexed)
- `page_size` (int, default: 20, max: 100) - Items per page

**Response Format**:
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 87,
    "total_pages": 5
  }
}
```

## Idempotency

AI parsing endpoint requires an idempotency key to prevent duplicate charges:

```json
{
  "description": "...",
  "idempotency_key": "unique-request-id-123"
}
```

Duplicate requests with the same key within 24 hours return the cached result.

## Timestamps

All timestamps are in ISO 8601 format with UTC timezone:
```
2025-11-15T10:30:00Z
```

Clients should convert to local timezone for display.

## CORS

Allowed origins configured via `CORS_ALLOWED_ORIGINS` environment variable:
```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
```

---

**Last Updated**: 2025-11-15
**API Version**: v1
