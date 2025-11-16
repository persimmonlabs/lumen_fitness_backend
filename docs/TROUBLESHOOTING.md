# Troubleshooting Guide

Complete troubleshooting guide for Lumen Nutrition Tracker Backend.

## Table of Contents

- [Common Issues](#common-issues)
  - [Database Connection](#database-connection)
  - [AI Parsing Failures](#ai-parsing-failures)
  - [Photo Upload Issues](#photo-upload-issues)
  - [Authentication Problems](#authentication-problems)
  - [Cost Limit Errors](#cost-limit-errors)
- [Performance Issues](#performance-issues)
- [API Errors](#api-errors)
- [Development Issues](#development-issues)
- [Debugging Tools](#debugging-tools)
- [Getting Help](#getting-help)

---

## Common Issues

### Database Connection

#### Error: Connection Refused

```
Error: failed to initialize Supabase client: dial tcp: connect: connection refused
```

**Possible Causes:**
1. Incorrect `SUPABASE_URL` in environment variables
2. Supabase project is paused (free tier pauses after 7 days inactivity)
3. Network connectivity issues
4. Firewall blocking connection

**Solutions:**

```bash
# 1. Verify environment variables
echo $SUPABASE_URL
echo $SUPABASE_SERVICE_KEY

# 2. Test connection manually
curl https://your-project.supabase.co/rest/v1/

# 3. Check if project is paused
# Go to Supabase dashboard and resume project

# 4. Test with in-memory database for development
DATABASE_MODE=memory go run cmd/api/main.go
```

**Verify Fix:**
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy","database":"connected"}
```

---

#### Error: Invalid JWT Secret

```
Error: token validation failed: signature is invalid
```

**Cause:** `SUPABASE_JWT_SECRET` doesn't match your Supabase project.

**Solution:**

1. Go to Supabase Dashboard → Settings → API
2. Copy **JWT Secret** (not the anon key or service key!)
3. Update `.env`:

```bash
SUPABASE_JWT_SECRET=your-actual-jwt-secret-from-dashboard
```

4. Restart server

---

#### Error: Row Level Security Prevents Access

```
Error: new row violates row-level security policy
```

**Cause:** RLS policies not set up correctly or missing.

**Solution:**

Run RLS setup in Supabase SQL Editor:

```sql
-- Enable RLS on all tables
ALTER TABLE meals ENABLE ROW LEVEL SECURITY;
ALTER TABLE meal_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE weight_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE meal_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE template_items ENABLE ROW LEVEL SECURITY;

-- Create policies for meals (example)
CREATE POLICY "Users can view own meals"
ON meals FOR SELECT
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY "Users can create own meals"
ON meals FOR INSERT
TO authenticated
WITH CHECK (user_id = auth.uid());

-- Repeat for other tables
```

**For Development:** Temporarily disable RLS:

```sql
ALTER TABLE meals DISABLE ROW LEVEL SECURITY;
```

---

### AI Parsing Failures

#### Error: AI Provider Unavailable

```
Error: failed to parse meal: connection timeout
```

**Possible Causes:**
1. AI provider API is down
2. API key is invalid or expired
3. Rate limits exceeded
4. Network connectivity issues

**Solutions:**

```bash
# 1. Verify API keys are set
echo $GROQ_API_KEY
echo $OPENROUTER_API_KEY

# 2. Test API key manually
curl https://api.groq.com/openai/v1/models \
  -H "Authorization: Bearer $GROQ_API_KEY"

# 3. Check Groq status
curl https://status.groq.com/

# 4. Try with fallback provider
# Set both API keys in .env:
GROQ_API_KEY=your-groq-key
OPENROUTER_API_KEY=your-openrouter-key

# Server automatically fails over to OpenRouter if Groq fails
```

---

#### Error: Rate Limit Exceeded

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

**Solutions:**

```bash
# 1. Increase user limit (temporary)
RATE_LIMIT_AI_PARSE_PER_DAY=100

# 2. For development, disable rate limiting
FEATURE_RATE_LIMIT=false

# 3. Use multiple API keys and rotate them
# Implement in code or wait for daily reset
```

---

#### Error: Invalid AI Response

```
Error: failed to parse AI response: unexpected JSON structure
```

**Cause:** AI model returned malformed JSON or unexpected format.

**Solutions:**

```bash
# 1. Enable debug logging to see raw AI response
LOG_LEVEL=debug go run cmd/api/main.go

# 2. Try different AI model
GROQ_MODEL=llama-3.3-70b-versatile  # or
GROQ_MODEL=llama-3.1-70b-versatile

# 3. Simplify the request
# Use shorter meal descriptions (< 500 chars)
# Avoid complex or ambiguous descriptions
```

**Example Debug Output:**
```
DEBUG: AI provider response: {"items":[...],"confidence":"high"}
```

---

### Photo Upload Issues

#### Error: Bucket Not Found

```
Error: failed to upload photo: storage bucket 'lumen-nutrition' not found
```

**Solutions:**

1. **Create Storage Bucket in Supabase:**
   - Go to Supabase Dashboard → Storage
   - Click "New Bucket"
   - Name: `lumen-nutrition`
   - Public: **No** (keep private)
   - Click "Create Bucket"

2. **Verify bucket name in `.env`:**

```bash
SUPABASE_STORAGE_BUCKET=lumen-nutrition
```

3. **Set up RLS policies for bucket:**

```sql
-- Allow users to upload their own photos
CREATE POLICY "Users can upload own photos"
ON storage.objects FOR INSERT
TO authenticated
WITH CHECK (
  bucket_id = 'lumen-nutrition'
  AND (storage.foldername(name))[1] = auth.uid()::text
);

-- Allow users to read their own photos
CREATE POLICY "Users can read own photos"
ON storage.objects FOR SELECT
TO authenticated
USING (
  bucket_id = 'lumen-nutrition'
  AND (storage.foldername(name))[1] = auth.uid()::text
);
```

---

#### Error: File Too Large

```json
{
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "File size exceeds maximum allowed size",
    "details": {
      "size_mb": 15,
      "max_allowed_mb": 10
    }
  }
}
```

**Solutions:**

```bash
# 1. Increase max file size (not recommended >50MB)
STORAGE_MAX_PHOTO_SIZE_MB=20

# 2. Compress photos on client side before upload
# Use client-side image compression libraries

# 3. Resize images on server (future enhancement)
```

---

#### Error: Unsupported File Format

```json
{
  "error": {
    "code": "UNSUPPORTED_FORMAT",
    "message": "File format not allowed",
    "details": {
      "format": "gif",
      "allowed": ["jpg", "jpeg", "png", "heic"]
    }
  }
}
```

**Solution:**

```bash
# Add format to allowed list
STORAGE_ALLOWED_PHOTO_FORMATS=jpg,jpeg,png,heic,webp
```

---

### Authentication Problems

#### Error: Unauthorized - Missing Token

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid authentication token"
  }
}
```

**Solutions:**

```bash
# 1. For development, disable authentication
FEATURE_AUTH=false

# 2. Include token in request
curl http://localhost:8080/api/v1/meals \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 3. Get token from Supabase Auth
# Login via Supabase client library and use returned access_token
```

---

#### Error: Token Expired

```json
{
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "Authentication token has expired"
  }
}
```

**Solution:**

Use refresh token to get new access token:

```bash
curl -X POST https://your-project.supabase.co/auth/v1/token?grant_type=refresh_token \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your-refresh-token"}'
```

---

### Cost Limit Errors

#### Error: User Cost Limit Exceeded

```json
{
  "error": {
    "code": "COST_LIMIT_EXCEEDED",
    "message": "Monthly AI cost limit exceeded for user",
    "details": {
      "spent_usd": 10.50,
      "limit_usd": 10.00,
      "reset_at": "2025-12-01T00:00:00Z"
    }
  }
}
```

**Solutions:**

```bash
# 1. Increase user cost limit
RATE_LIMIT_COST_PER_USER_PER_MONTH=20.0

# 2. For development, disable cost tracking
FEATURE_RATE_LIMIT=false

# 3. Wait for monthly reset
# Cost limits reset on the 1st of each month
```

---

## Performance Issues

### Slow Response Times

#### Symptom: API requests taking >2 seconds

**Diagnosis:**

```bash
# Enable debug logging to see query times
LOG_LEVEL=debug go run cmd/api/main.go

# Check logs for slow queries:
# DEBUG: query duration=1847ms table=meals
```

**Solutions:**

1. **Enable Caching:**

```bash
FEATURE_CACHE=true
CACHE_DEFAULT_TTL=5m
```

2. **Add Database Indexes:**

```sql
-- Index frequently queried columns
CREATE INDEX idx_meals_user_id_consumed_at ON meals(user_id, consumed_at DESC);
CREATE INDEX idx_weight_entries_user_id_measured_at ON weight_entries(user_id, measured_at DESC);
CREATE INDEX idx_meal_items_meal_id ON meal_items(meal_id);
```

3. **Use Connection Pooler:**

```bash
# Use Supabase connection pooler URL instead of direct connection
# Format: pooler.supabase.com:6543 instead of db.supabase.com:5432
```

4. **Optimize Queries:**

```sql
-- Limit results
SELECT * FROM meals ORDER BY consumed_at DESC LIMIT 20;

-- Select only needed columns
SELECT id, meal_type, consumed_at FROM meals;

-- Use materialized views for analytics
```

---

### High Memory Usage

#### Symptom: Server memory usage >500MB

**Solutions:**

```bash
# 1. Reduce cache TTL
CACHE_DEFAULT_TTL=1m

# 2. Lower database connection pool
DATABASE_MAX_OPEN_CONNS=10
DATABASE_MAX_IDLE_CONNS=2

# 3. Enable garbage collection tuning
# Add to main.go:
# runtime.GC()
```

---

### High CPU Usage

#### Symptom: CPU usage constantly >80%

**Diagnosis:**

```bash
# Profile CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

**Solutions:**

1. Reduce AI parsing frequency
2. Implement request queuing
3. Use async processing for heavy operations
4. Scale horizontally (multiple instances)

---

## API Errors

### 400 Bad Request

**Common Causes:**

1. **Missing required fields:**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "fields": {
        "description": "is required",
        "meal_type": "is required"
      }
    }
  }
}
```

**Fix:** Include all required fields in request body.

2. **Invalid data types:**

```json
{
  "weight": "75.5kg"  // ❌ Should be number: 75.5
}
```

---

### 404 Not Found

**Common Causes:**

1. **Resource doesn't exist:**

```bash
# Verify resource ID is correct
curl http://localhost:8080/api/v1/meals/550e8400-e29b-41d4-a716-446655440000
```

2. **Resource belongs to different user:**

RLS policies prevent access to other users' data. This is expected behavior.

---

### 409 Conflict

**Common Cause:** Duplicate resource creation.

**Example:**

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Resource already exists"
  }
}
```

**Fix:** Check if resource already exists before creating.

---

### 500 Internal Server Error

**Always indicates a server bug.** Check logs:

```bash
# View error logs
LOG_LEVEL=debug go run cmd/api/main.go

# Logs will show stack trace:
# ERROR: panic recovered: runtime error: invalid memory address
```

**Report to developers with:**
- Request details (endpoint, method, body)
- Error logs
- Steps to reproduce

---

## Development Issues

### Port Already in Use

```
Error: listen tcp :8080: bind: address already in use
```

**Solutions:**

```bash
# 1. Use different port
PORT=8081 go run cmd/api/main.go

# 2. Kill process using port (Linux/Mac)
lsof -ti:8080 | xargs kill -9

# 3. Kill process using port (Windows)
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

---

### Module Not Found

```
Error: package github.com/some/package: cannot find module providing package
```

**Solution:**

```bash
# Download missing dependencies
go mod download

# Or tidy and download
go mod tidy
go mod download
```

---

### CORS Errors (from frontend)

```
Access to fetch at 'http://localhost:8080' from origin 'http://localhost:3000'
has been blocked by CORS policy
```

**Solution:**

```bash
# Add frontend origin to allowed list
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# For development, allow all origins (NOT for production!)
CORS_ALLOWED_ORIGINS=*
```

---

## Debugging Tools

### Enable Debug Logging

```bash
LOG_LEVEL=debug
LOG_FORMAT=console
go run cmd/api/main.go
```

**Shows:**
- All SQL queries with execution times
- AI provider requests/responses
- Cache hits/misses
- Detailed error stack traces

---

### Health Check Endpoint

```bash
curl http://localhost:8080/health
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T10:30:00Z",
  "database": "connected",
  "cache": "active"
}
```

---

### Manual Database Queries

```bash
# Connect to Supabase database
psql "postgresql://postgres:[PASSWORD]@db.xxxxx.supabase.co:5432/postgres"

# Check user's meals
SELECT * FROM meals WHERE user_id = 'user-uuid' ORDER BY consumed_at DESC LIMIT 10;

# Check weight entries
SELECT * FROM weight_entries WHERE user_id = 'user-uuid' ORDER BY measured_at DESC;

# Check goals
SELECT * FROM user_goals WHERE user_id = 'user-uuid';
```

---

### Test AI Parsing

```bash
curl -X POST http://localhost:8080/api/v1/meals/parse \
  -H "Content-Type: application/json" \
  -d '{
    "description": "chicken breast with rice and broccoli",
    "meal_type": "lunch",
    "idempotency_key": "test-123"
  }'
```

---

### Monitor Logs in Real-Time

```bash
# Run server with verbose logging
LOG_LEVEL=debug go run cmd/api/main.go 2>&1 | tee server.log

# In another terminal, watch logs
tail -f server.log | grep ERROR
```

---

## Getting Help

### Before Opening an Issue

1. **Check this troubleshooting guide**
2. **Enable debug logging:** `LOG_LEVEL=debug`
3. **Try with minimal configuration:**

```bash
APP_MODE=development
DATABASE_MODE=memory
FEATURE_AUTH=false
FEATURE_RATE_LIMIT=false
LOG_LEVEL=debug
```

4. **Verify environment variables:**

```bash
env | grep SUPABASE
env | grep GROQ
env | grep FEATURE
```

---

### Information to Include

When reporting issues, include:

1. **Error message** (full stack trace)
2. **Configuration:**
   ```bash
   APP_MODE=development
   DATABASE_MODE=supabase
   FEATURE_AUTH=true
   # etc.
   ```
3. **Steps to reproduce:**
   ```bash
   1. Start server with: go run cmd/api/main.go
   2. Send request: curl -X POST http://localhost:8080/api/v1/meals/parse ...
   3. Observe error: ...
   ```
4. **Expected behavior:** What should happen?
5. **Actual behavior:** What actually happened?
6. **Environment:**
   - OS: Ubuntu 22.04 / macOS 14 / Windows 11
   - Go version: `go version`
   - Database: Supabase / In-memory

---

### Quick Diagnostic Script

Create `scripts/diagnose.sh`:

```bash
#!/bin/bash

echo "=== Environment Check ==="
go version
echo ""

echo "=== Configuration ==="
echo "APP_MODE=$APP_MODE"
echo "DATABASE_MODE=$DATABASE_MODE"
echo "SUPABASE_URL=$SUPABASE_URL"
echo "FEATURE_AUTH=$FEATURE_AUTH"
echo ""

echo "=== Health Check ==="
curl -s http://localhost:8080/health | jq .
echo ""

echo "=== Database Tables ==="
psql $DATABASE_URL -c "\dt" 2>/dev/null || echo "Cannot connect to database"
echo ""

echo "=== Logs (last 20 lines) ==="
tail -20 server.log 2>/dev/null || echo "No log file found"
```

Run:
```bash
chmod +x scripts/diagnose.sh
./scripts/diagnose.sh > diagnostic-report.txt
```

---

### Support Channels

- **Documentation:** `backend/docs/`
- **GitHub Issues:** [Repository Issues Page]
- **Development Guide:** `backend/docs/DEVELOPMENT.md`
- **Deployment Guide:** `backend/docs/DEPLOYMENT.md`

---

**Last Updated:** 2025-11-15
