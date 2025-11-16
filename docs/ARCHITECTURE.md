# Architecture Documentation

Complete system architecture overview for Lumen Nutrition Tracker Backend.

## Table of Contents

- [System Overview](#system-overview)
- [Architecture Layers](#architecture-layers)
- [Domain Structure](#domain-structure)
- [Data Flow](#data-flow)
- [Service Dependencies](#service-dependencies)
- [Technology Stack](#technology-stack)
- [Scalability & Performance](#scalability--performance)
- [Security Architecture](#security-architecture)

---

## System Overview

Lumen Nutrition Tracker is a **nutrition tracking and analysis platform** that uses AI to parse meal descriptions and photos into structured nutrition data.

```
┌─────────────────────────────────────────────────────────┐
│                    Client Applications                   │
│         (Web App, Mobile App, API Consumers)            │
└────────────────────┬────────────────────────────────────┘
                     │ HTTPS / REST API
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  Go Backend Server                       │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Handlers   │  │   Services   │  │ Repositories │  │
│  │  (HTTP API)  │→ │  (Business)  │→ │    (Data)    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Middleware   │  │   Config     │  │    Cache     │  │
│  │ (Auth, CORS) │  │ (Env Vars)   │  │  (Memory)    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────┬───────────────────┬───────────────┘
                      │                   │
                      │                   │
         ┌────────────▼────────┐  ┌──────▼─────────┐
         │   Supabase Cloud    │  │  AI Providers  │
         │ ┌─────────────────┐ │  │ ┌────────────┐ │
         │ │   PostgreSQL    │ │  │ │   Groq     │ │
         │ │    Database     │ │  │ │  (Primary) │ │
         │ └─────────────────┘ │  │ └────────────┘ │
         │ ┌─────────────────┐ │  │ ┌────────────┐ │
         │ │  File Storage   │ │  │ │ OpenRouter │ │
         │ │  (Photos/Media) │ │  │ │ (Fallback) │ │
         │ └─────────────────┘ │  │ └────────────┘ │
         │ ┌─────────────────┐ │  └────────────────┘
         │ │  Auth Service   │ │
         │ │  (JWT Tokens)   │ │
         │ └─────────────────┘ │
         └─────────────────────┘
```

### Key Principles

1. **Clean Architecture** - Separation of concerns across layers
2. **Domain-Driven Design** - Business logic organized by domain
3. **Repository Pattern** - Abstract data access
4. **Dependency Injection** - Loose coupling, easy testing
5. **Feature Flags** - Graceful degradation and flexible deployment

---

## Architecture Layers

### 1. HTTP Layer (Handlers)

**Location:** `internal/domain/nutrition/*/handler.go`, `internal/server/handlers/`

**Responsibilities:**
- Parse HTTP requests
- Validate request format
- Call service layer
- Format HTTP responses
- Handle errors

**Example:**

```go
func (h *MealHandler) CreateMeal(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req CreateMealRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.BadRequestError(w, "invalid request", nil)
        return
    }

    // 2. Call service layer
    meal, err := h.service.CreateMeal(r.Context(), userID, req)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }

    // 3. Return response
    h.SuccessResponse(w, http.StatusCreated, meal, nil)
}
```

**Key Components:**
- `BaseHandler` - Common HTTP utilities
- Request/Response DTOs
- Error formatting
- Authentication extraction

---

### 2. Service Layer (Business Logic)

**Location:** `internal/domain/nutrition/*/service.go`

**Responsibilities:**
- Implement business rules
- Orchestrate operations
- Validate business logic
- Coordinate between repositories
- Handle transactions

**Example:**

```go
func (s *MealService) CreateMeal(ctx context.Context, userID uuid.UUID, req CreateMealRequest) (*Meal, error) {
    // 1. Validate business rules
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // 2. Check user goals
    goals, _ := s.goalsRepo.GetByUserID(ctx, userID)

    // 3. Calculate totals
    meal := &Meal{
        ID:            uuid.New(),
        UserID:        userID,
        Items:         req.Items,
        TotalCalories: calculateTotalCalories(req.Items),
    }

    // 4. Save to database
    if err := s.repo.Create(ctx, meal); err != nil {
        return nil, errors.Wrap(err, "failed to create meal")
    }

    return meal, nil
}
```

**Key Patterns:**
- Domain validation
- Cross-domain coordination
- Error wrapping
- Context propagation

---

### 3. Repository Layer (Data Access)

**Location:** `internal/domain/nutrition/*/repository.go`

**Responsibilities:**
- Abstract database operations
- Execute queries
- Map database rows to models
- Handle database errors

**Example:**

```go
type MealRepository interface {
    Create(ctx context.Context, meal *Meal) error
    GetByID(ctx context.Context, userID, mealID uuid.UUID) (*Meal, error)
    List(ctx context.Context, userID uuid.UUID, filters ListFilters) ([]Meal, int, error)
    Update(ctx context.Context, meal *Meal) error
    Delete(ctx context.Context, userID, mealID uuid.UUID) error
}
```

**Implementations:**
- Supabase repository (production)
- In-memory repository (testing)
- Future: Redis cache layer

---

### 4. Middleware Layer

**Location:** `internal/server/middleware/`

**Components:**

```
┌─────────────────────────────────────────┐
│             HTTP Request                │
└───────────────┬─────────────────────────┘
                │
         ┌──────▼──────┐
         │   CORS      │ ← Allow specific origins
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │  Logging    │ ← Log all requests
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │    Auth     │ ← Validate JWT token
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │ Rate Limit  │ ← Check rate limits
         └──────┬──────┘
                │
         ┌──────▼──────┐
         │   Handler   │ ← Execute business logic
         └─────────────┘
```

**Middleware:**
- **CORS** - Cross-origin resource sharing
- **Auth** - JWT token validation
- **Logging** - Request/response logging
- **Rate Limiting** - Prevent abuse
- **Recovery** - Panic recovery
- **Request ID** - Trace requests

---

## Domain Structure

### Nutrition Domain

The application is organized into **5 sub-domains** under `internal/domain/nutrition/`:

```
nutrition/
├── meals/          # Meal tracking and parsing
├── weight/         # Weight tracking
├── goals/          # Nutrition goals and TDEE
├── analytics/      # Analytics and trends
└── templates/      # Meal templates
```

Each domain follows the **same structure:**

```
domain/
├── models.go           # Data models and DTOs
├── repository.go       # Repository interface
├── repository_impl.go  # Supabase implementation
├── service.go          # Business logic
├── handler.go          # HTTP handlers
└── *_test.go          # Tests
```

---

### Domain Boundaries

```
┌─────────────────────────────────────────────────────┐
│                    MEALS DOMAIN                      │
│  - Parse meal descriptions                          │
│  - Store meal data                                  │
│  - Manage meal items                                │
│  - Photo uploads                                    │
└──────────┬──────────────────────────────────────────┘
           │ depends on
           ▼
┌──────────────────────┐      ┌───────────────────────┐
│   GOALS DOMAIN       │      │  ANALYTICS DOMAIN     │
│  - Daily goals       │ ◄─── │  - Daily summaries    │
│  - TDEE calculation  │      │  - Weekly trends      │
│  - Macro targets     │      │  - Progress tracking  │
└──────────────────────┘      └───────────────────────┘
           │                           ▲
           │                           │
           │ depends on                │ reads from
           ▼                           │
┌──────────────────────┐              │
│   WEIGHT DOMAIN      │              │
│  - Weight entries    │──────────────┘
│  - Weight trends     │
│  - Rate of change    │
└──────────────────────┘

           ▲
           │ references
           │
┌──────────┴───────────┐
│  TEMPLATES DOMAIN    │
│  - Meal templates    │
│  - Template items    │
│  - Reusable meals    │
└──────────────────────┘
```

**Dependency Rules:**
- Services can depend on other repositories (not services)
- No circular dependencies
- Analytics reads from all domains (read-only)

---

## Data Flow

### 1. AI Meal Parsing Flow

```
┌─────────┐
│ Client  │ POST /api/v1/meals/parse
└────┬────┘ { "description": "chicken and rice", ... }
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Handler: ParseMeal                                  │
│  1. Validate request                               │
│  2. Check idempotency key (prevent duplicates)     │
│  3. Extract user ID from auth token                │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Service: MealService.ParseMeal                     │
│  1. Check rate limits                              │
│  2. Check cost limits                              │
│  3. Prepare AI prompt                              │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ AI Coordinator                                      │
│  1. Select provider (Groq → OpenRouter fallback)   │
│  2. Send request to AI provider                    │
│  3. Parse AI response                              │
│  4. Validate nutrition data                        │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Cost Tracker                                        │
│  1. Calculate request cost                         │
│  2. Update user cost counter                       │
│  3. Update global cost counter                     │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Cache Service                                       │
│  1. Cache parse result (idempotency)               │
│  2. Set TTL: 24 hours                              │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌─────────┐
│ Client  │ { "items": [...], "total_calories": 500 }
└─────────┘
```

---

### 2. Meal Confirmation Flow

```
┌─────────┐
│ Client  │ POST /api/v1/meals/confirm
└────┬────┘ { "items": [...], "consumed_at": "..." }
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Handler: ConfirmMeal                                │
│  1. Validate request                               │
│  2. Extract user ID                                │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Service: MealService.ConfirmMeal                   │
│  1. Validate items                                 │
│  2. Calculate totals                               │
│  3. Create meal entity                             │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Repository: MealRepository.Create                  │
│  1. Begin transaction                              │
│  2. Insert meal record                             │
│  3. Insert meal items                              │
│  4. Commit transaction                             │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Supabase Database                                   │
│  - Insert into meals table                         │
│  - Insert into meal_items table                    │
│  - Trigger: update updated_at timestamp            │
│  - RLS: Verify user_id = auth.uid()                │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Cache Invalidation                                  │
│  1. Invalidate daily analytics cache               │
│  2. Invalidate meal list cache                     │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌─────────┐
│ Client  │ { "id": "...", "total_calories": 500, ... }
└─────────┘
```

---

### 3. Analytics Calculation Flow

```
┌─────────┐
│ Client  │ GET /api/v1/analytics/daily?date=2025-11-15
└────┬────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Handler: GetDailyAnalytics                         │
│  1. Parse query parameters (date, timezone)        │
│  2. Validate date format                           │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Cache: Check Cache                                  │
│  Key: analytics:daily:{user_id}:{date}            │
│  TTL: 5 minutes                                    │
└────┬───────────────────────────────────────────────┘
     │ cache miss
     ▼
┌────────────────────────────────────────────────────┐
│ Service: AnalyticsService.GetDailyAnalytics        │
│  1. Get user goals                                 │
│  2. Get meals for date                             │
│  3. Sum nutrition totals                           │
│  4. Calculate progress percentages                 │
│  5. Calculate remaining amounts                    │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Repository: MealRepository.ListByDate              │
│  SELECT * FROM meals                               │
│  WHERE user_id = $1                                │
│    AND DATE(consumed_at) = $2                      │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Repository: GoalsRepository.GetByUserID            │
│  SELECT * FROM user_goals                          │
│  WHERE user_id = $1                                │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Business Logic: Aggregate & Calculate              │
│  totals.calories = SUM(meals.total_calories)       │
│  progress.calories_percent = (totals / goals) * 100│
│  remaining.calories = goals - totals               │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Cache: Store Result                                 │
│  Store for 5 minutes                               │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌─────────┐
│ Client  │ { "totals": {...}, "goals": {...}, ... }
└─────────┘
```

---

## Service Dependencies

### Core Services

#### 1. AI Coordinator

**Location:** `internal/services/ai/coordinator.go`

**Responsibilities:**
- Manage multiple AI providers
- Failover between providers
- Track provider health
- Format requests/responses

**Providers:**
- **Groq** (Primary) - Fast, free tier: 14,400 req/day
- **OpenRouter** (Fallback) - Reliable, pay-per-use

**Flow:**

```
Request → Groq API ──success──→ Return result
               │
               └──fail──→ OpenRouter API ──→ Return result
```

---

#### 2. Storage Service

**Location:** `internal/services/storage/`

**Responsibilities:**
- Upload photos to Supabase Storage
- Generate signed URLs
- Validate file types and sizes
- Organize files by user

**File Structure:**

```
lumen-nutrition/
├── {user_id}/
│   ├── meals/
│   │   ├── {meal_id}_1.jpg
│   │   ├── {meal_id}_2.jpg
│   │   └── ...
│   └── profile/
│       └── avatar.jpg
```

---

#### 3. Cache Service

**Location:** `internal/services/cache/`

**Strategies:**
- **Memory Cache** (Default) - Fast, ephemeral
- **Redis Cache** (Optional) - Persistent, distributed

**Cache Keys:**

```
analytics:daily:{user_id}:{date}
analytics:weekly:{user_id}:{start_date}
meals:list:{user_id}:{page}
goals:{user_id}
weight:trend:{user_id}
```

**Cache Invalidation:**
- On meal create/update/delete → Invalidate analytics
- On goals update → Invalidate goals, analytics
- On weight entry → Invalidate weight trends

---

#### 4. Cost Tracker

**Location:** `internal/services/cost/`

**Tracks:**
- AI parsing costs per user
- Daily/monthly cost limits
- Global cost caps

**Storage:**
- In-memory (development)
- Database (production)

---

## Technology Stack

### Backend

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | **Go 1.24+** | High-performance, type-safe |
| Web Framework | **chi** | Lightweight HTTP router |
| Database | **PostgreSQL** (via Supabase) | Relational data storage |
| Authentication | **Supabase Auth** | JWT-based auth |
| File Storage | **Supabase Storage** | Photo/media uploads |
| Caching | **In-memory** / Redis | Response caching |
| Logging | **slog** (stdlib) | Structured logging |
| Validation | **validator/v10** | Request validation |

### External Services

| Service | Purpose | Tier |
|---------|---------|------|
| **Groq** | AI nutrition parsing (primary) | Free: 14,400 req/day |
| **OpenRouter** | AI parsing (fallback) | Pay-per-use |
| **Supabase** | Database, Auth, Storage | Free tier available |

### Development Tools

| Tool | Purpose |
|------|---------|
| **Air** | Hot reload during development |
| **golangci-lint** | Code linting |
| **delve** | Debugging |
| **testify** | Testing framework |

---

## Scalability & Performance

### Current Architecture

- **Single-instance deployment**
- **Connection pooling** (25 max connections)
- **In-memory caching**
- **Async AI processing** (where applicable)

### Performance Optimizations

1. **Database Indexes:**

```sql
-- Frequently queried columns
CREATE INDEX idx_meals_user_consumed ON meals(user_id, consumed_at DESC);
CREATE INDEX idx_weight_user_measured ON weight_entries(user_id, measured_at DESC);
CREATE INDEX idx_meal_items_meal_id ON meal_items(meal_id);
```

2. **Query Optimization:**
   - Paginated queries (LIMIT/OFFSET)
   - Select only needed columns
   - Use database views for complex analytics

3. **Caching Strategy:**
   - Cache analytics (5-minute TTL)
   - Cache user goals (until updated)
   - Idempotency cache (24-hour TTL)

4. **Connection Pooling:**

```go
db.SetMaxOpenConns(25)  // Max concurrent connections
db.SetMaxIdleConns(5)   // Keep 5 idle connections
db.SetConnMaxLifetime(5 * time.Minute)
```

### Scaling Strategies

#### Horizontal Scaling

```
┌──────────────┐
│ Load Balancer│
└──────┬───────┘
       │
       ├────→ Server Instance 1
       ├────→ Server Instance 2
       └────→ Server Instance 3
              │
              ▼
        ┌─────────────┐
        │  Supabase   │
        │  (Shared)   │
        └─────────────┘
```

**Requirements:**
- Stateless server instances
- Redis for distributed cache
- Session storage in database (not memory)

#### Vertical Scaling

Increase resources for single instance:
- More CPU cores
- More RAM (for larger cache)
- Faster network

#### Database Scaling

- **Read Replicas** - Separate read/write
- **Connection Pooler** - Use Supabase pooler (6543 port)
- **Partitioning** - Partition large tables by date

---

## Security Architecture

### Authentication Flow

```
┌─────────┐
│ Client  │ POST /auth/login
└────┬────┘ { "email": "...", "password": "..." }
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Supabase Auth                                       │
│  1. Verify credentials                             │
│  2. Generate JWT access token                      │
│  3. Generate refresh token                         │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌─────────┐
│ Client  │ Store tokens
└────┬────┘ { "access_token": "...", "refresh_token": "..." }
     │
     │ Subsequent requests
     ▼
┌────────────────────────────────────────────────────┐
│ Request with Authorization header                   │
│ Authorization: Bearer <access_token>               │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Auth Middleware                                     │
│  1. Extract token from header                      │
│  2. Verify JWT signature (using SUPABASE_JWT_SECRET)│
│  3. Check token expiration                         │
│  4. Extract user ID from token claims              │
│  5. Add user ID to request context                 │
└────┬───────────────────────────────────────────────┘
     │
     ▼
┌────────────────────────────────────────────────────┐
│ Handler executes with authenticated user context    │
└────────────────────────────────────────────────────┘
```

### Row-Level Security (RLS)

**All database queries automatically filtered by user:**

```sql
-- RLS Policy Example
CREATE POLICY "users_select_own_meals"
ON meals FOR SELECT
TO authenticated
USING (user_id = auth.uid());
```

**Prevents:**
- Users accessing other users' data
- Accidental data leaks
- SQL injection attacks (with RLS + parameterized queries)

### API Security

1. **CORS Protection:**

```go
CORS_ALLOWED_ORIGINS=https://app.example.com,https://www.example.com
// No wildcard (*) in production
```

2. **Rate Limiting:**

```go
// Per user limits
RATE_LIMIT_AI_PARSE_PER_DAY=50
RATE_LIMIT_API_CALLS_PER_HOUR=1000

// Global limits
RATE_LIMIT_COST_PER_USER_PER_MONTH=10.0
RATE_LIMIT_GLOBAL_COST_PER_MONTH=1000.0
```

3. **Input Validation:**

```go
type CreateMealRequest struct {
    Description string `validate:"required,min=1,max=2000"`
    MealType    string `validate:"required,oneof=breakfast lunch dinner snack"`
}
```

4. **Secret Management:**

```bash
# Never commit secrets
# Use environment variables
SUPABASE_SERVICE_KEY=xxxxx
GROQ_API_KEY=xxxxx
```

---

### Feature Flags for Security

```go
// Disable features in production if not needed
FEATURE_AUTH=true           // Always true in production
FEATURE_RATE_LIMIT=true     // Always true in production
FEATURE_METRICS=true        // Monitor usage
FEATURE_STORAGE=true        // Enable photo uploads

// Development shortcuts
FEATURE_AUTH=false          // Only for local dev
FEATURE_RATE_LIMIT=false    // Only for testing
```

---

## Future Enhancements

### Planned Improvements

1. **Microservices Architecture:**
   - Separate AI parsing service
   - Dedicated analytics service
   - Background job processor

2. **Event-Driven Architecture:**
   - Meal created → Trigger analytics update
   - Weight entry → Recalculate TDEE
   - Cost limit → Send notification

3. **Advanced Caching:**
   - Redis cluster for distributed cache
   - Cache warming strategies
   - Intelligent cache invalidation

4. **Enhanced AI:**
   - Vision model for photo analysis
   - Food recognition from images
   - Personalized nutrition suggestions

5. **Real-time Features:**
   - WebSocket support for live updates
   - Collaborative meal planning
   - Real-time progress tracking

---

**Last Updated:** 2025-11-15
