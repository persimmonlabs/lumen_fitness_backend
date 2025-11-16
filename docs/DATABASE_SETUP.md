# Database Setup - Supabase Connection

## Overview
This document describes the successful connection of the Lumen backend to the real Supabase PostgreSQL database.

## Changes Made

### 1. Environment Configuration (`.env`)
Created `.env` file in `backend/` directory with real Supabase credentials:

- **Database URL**: PostgreSQL connection string with URL-encoded password
- **Supabase Configuration**: URL, anon key, service key
- **AI Services**: OpenRouter and Groq API keys
- **Feature Flags**: Enabled auth, storage, AI analysis, cache, rate limiting
- **Connection Pool Settings**: 25 max connections, 5 idle connections

**Important**: Password special characters (`!?`) are URL-encoded as `%21%3F` in the DATABASE_URL.

### 2. Database Connection (`cmd/api/main.go`)

**Changes:**
- Uncommented database connection code (lines 62-90)
- Added import for `sqlx` and `lib/pq` PostgreSQL driver
- Switched from `pgxpool.Pool` to `sqlx.DB` (compatible with repository implementations)
- Implemented connection pool configuration:
  - `SetMaxOpenConns(25)`
  - `SetMaxIdleConns(5)`
  - `SetConnMaxLifetime(1h)`
- Added connection verification with 5-second timeout
- Added success logging with connection details

**Test Results:**
```
Successfully connected to database | host=Supabase | max_conns=25
```

### 3. Repository Initialization (`internal/server/dependencies.go`)

**Changes:**
- Changed from `pgxpool.Pool` to `sqlx.DB` throughout
- Uncommented repository initialization code
- Initialized PostgreSQL repositories:
  - ✅ Meals Repository (`meals.NewRepository`)
  - ✅ Weight Repository (`weight.NewPostgresRepository`)
  - ⏳ Templates, Analytics, Goals (TODO - not yet implemented)

**Logging Output:**
```
initialized repositories | meals=true | weight=true | templates=false | analytics=false | goals=false
```

### 4. Service Initialization

**Changes:**
- Uncommented service initialization code
- Initialized domain services:
  - ✅ Meals Service (without AI features - adapters needed)
  - ✅ Weight Service
  - ✅ Media Service
  - ✅ Trajectory Service
  - ✅ Suggestions Service

**Notes:**
- AI features are temporarily disabled (passing `nil` for AI interfaces)
- Need to create adapters to bridge `ai.Coordinator` to meals-specific interfaces
- Services will work for basic CRUD operations without AI parsing

### 5. Health Check Fix

Fixed `HealthCheck` method to use `PingContext(ctx)` instead of `Ping(ctx)` for `sqlx.DB` compatibility.

## Verification

### Startup Logs
```
[INF] Starting Go API Server mode=development supabase_configured=true
[INF] Supabase client initialized successfully url=https://ftarqjggyozzaiwkuaal.supabase.co
[INF] Successfully connected to database host=Supabase max_conns=25
INFO initialized repositories meals=true weight=true templates=false analytics=false goals=false
INFO initialized meals service (without AI features)
INFO initialized weight service
INFO initialized media service
INFO initialized trajectory service
INFO initialized suggestions service
[INF] Nutrition domain initialized meals_enabled=true weight_enabled=true
[INF] Starting HTTP server address=0.0.0.0:8000
```

### Health Check Response
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2025-11-16T05:31:36Z",
    "uptime": "12s",
    "version": "1.0.0"
  },
  "meta": {
    "timestamp": "2025-11-16T05:31:36Z"
  }
}
```

## Database Schema

The following tables are available in Supabase:
- `meals` - Meal records with nutrition data
- `meal_items` - Individual food items within meals
- `weight_entries` - Weight tracking records
- Additional tables for templates, analytics, and goals (to be implemented)

## Running the Application

```bash
cd backend
go run ./cmd/api/main.go
```

Or build and run:
```bash
go build -o bin/api.exe ./cmd/api/main.go
./bin/api.exe
```

## TODO: Next Steps

### 1. Implement Missing Repositories
- [ ] Templates PostgreSQL repository
- [ ] Analytics PostgreSQL repository
- [ ] Goals PostgreSQL repository

### 2. Create AI Service Adapters
The meals service requires specific AI interfaces:
- `AICoordinator` - Meal parsing
- `AIEstimator` - Fast meal estimation
- `AITranscriber` - Audio transcription
- `AINormalizer` - Description normalization

Current `ai.Coordinator` needs adapters to implement these interfaces.

### 3. Implement Supporting Services
- [ ] Cost Tracker for AI API usage
- [ ] Photo Storage validation
- [ ] Cache adapter for meals service

### 4. Test Database Operations
- [ ] Create meal endpoint
- [ ] List meals endpoint
- [ ] Weight tracking endpoints
- [ ] Migration verification

## Configuration Reference

### Environment Variables
```env
# Application
APP_MODE=development
PORT=8000

# Database
DATABASE_MODE=supabase
DATABASE_URL=postgresql://user:password@host:port/db
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=1h

# Supabase
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_ANON_KEY=xxx
SUPABASE_SERVICE_KEY=xxx

# AI Services
OPENROUTER_API_KEY=xxx
GROQ_API_KEY=xxx

# Features
FEATURE_AUTH=true
FEATURE_STORAGE=true
FEATURE_AI_ANALYSIS=true
FEATURE_CACHE=true
FEATURE_RATE_LIMIT=true
```

## Troubleshooting

### Password URL Encoding
If the database password contains special characters, they must be URL-encoded:
- `!` → `%21`
- `?` → `%3F`
- `@` → `%40`
- `#` → `%23`
- etc.

### Connection Pool Settings
Adjust based on your Supabase plan:
- Free tier: Keep max connections low (10-25)
- Paid tier: Can increase to 100+

### Connection Timeout
Default timeout is 5 seconds. Increase if experiencing slow connections:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
```

## Security Notes

- ⚠️ Never commit `.env` file to version control
- ⚠️ Use strong JWT secret in production
- ⚠️ Rotate API keys regularly
- ⚠️ Use Supabase RLS (Row Level Security) policies
- ⚠️ Monitor connection pool usage

## Support

For issues or questions:
1. Check Supabase dashboard for connection status
2. Verify environment variables are loaded correctly
3. Check application logs for detailed error messages
4. Test database connection with `psql` client
