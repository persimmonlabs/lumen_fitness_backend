# Deployment Guide

Complete deployment guide for Lumen Nutrition Tracker Backend.

## Table of Contents

- [Environment Setup](#environment-setup)
- [Supabase Setup](#supabase-setup)
- [Database Migrations](#database-migrations)
- [Seeding Data](#seeding-data)
- [Environment Configuration](#environment-configuration)
- [Health Checks](#health-checks)
- [Monitoring](#monitoring)
- [Production Checklist](#production-checklist)
- [Troubleshooting](#troubleshooting)

---

## Environment Setup

### Prerequisites

- Go 1.24 or higher
- PostgreSQL 14+ (via Supabase)
- Supabase account (free tier available)
- AI API keys:
  - Groq API key (free tier: 14,400 requests/day)
  - OpenRouter API key (optional, for vision features)

### Quick Setup

```bash
# 1. Clone repository
git clone <repository-url>
cd backend

# 2. Install dependencies
go mod download

# 3. Copy environment template
cp .env.example .env

# 4. Edit .env with your credentials
nano .env
```

---

## Supabase Setup

### 1. Create Project

1. Go to [supabase.com](https://supabase.com)
2. Click "New Project"
3. Choose organization
4. Set project name: `lumen-nutrition`
5. Set database password (save securely!)
6. Choose region (closest to your users)
7. Wait for project to provision (~2 minutes)

### 2. Get API Credentials

1. Go to **Settings** > **API**
2. Copy these values to your `.env`:

```bash
SUPABASE_URL=https://xxxxxxxxxxxxx.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_SERVICE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_JWT_SECRET=your-jwt-secret
```

### 3. Database Connection

Get PostgreSQL connection string:

1. Go to **Settings** > **Database**
2. Find **Connection string** section
3. Copy URI (use the pooler for production)

```bash
# Connection pooler (recommended for production)
postgresql://postgres.xxxxxxxxxxxxx:[YOUR-PASSWORD]@aws-0-us-west-1.pooler.supabase.com:6543/postgres

# Direct connection (for migrations)
postgresql://postgres:[YOUR-PASSWORD]@db.xxxxxxxxxxxxx.supabase.co:5432/postgres
```

### 4. Storage Bucket Setup

1. Go to **Storage** section
2. Click **New bucket**
3. Bucket name: `lumen-nutrition`
4. Public bucket: **No** (keep private)
5. File size limit: 10 MB
6. Allowed MIME types: `image/jpeg,image/png,image/heic,image/webp`

Configure bucket policies (RLS):

```sql
-- Allow authenticated users to upload their own photos
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

-- Allow users to delete their own photos
CREATE POLICY "Users can delete own photos"
ON storage.objects FOR DELETE
TO authenticated
USING (
  bucket_id = 'lumen-nutrition'
  AND (storage.foldername(name))[1] = auth.uid()::text
);
```

### 5. Enable Row Level Security

Run in SQL Editor:

```sql
-- Enable RLS on all tables
ALTER TABLE meals ENABLE ROW LEVEL SECURITY;
ALTER TABLE meal_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE weight_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE meal_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE template_items ENABLE ROW LEVEL SECURITY;

-- Create policies for meals table
CREATE POLICY "Users can view own meals"
ON meals FOR SELECT
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY "Users can create own meals"
ON meals FOR INSERT
TO authenticated
WITH CHECK (user_id = auth.uid());

CREATE POLICY "Users can update own meals"
ON meals FOR UPDATE
TO authenticated
USING (user_id = auth.uid());

CREATE POLICY "Users can delete own meals"
ON meals FOR DELETE
TO authenticated
USING (user_id = auth.uid());

-- Repeat similar policies for other tables
-- (meal_items, weight_entries, user_goals, etc.)
```

---

## Database Migrations

### Running Migrations

Migrations are located in `backend/migrations/` directory.

#### Option 1: Via Supabase SQL Editor (Recommended)

1. Go to **SQL Editor** in Supabase dashboard
2. Copy contents of each migration file
3. Run in order:

```sql
-- migrations/001_initial_schema.sql
-- migrations/002_weight_tracking.sql
-- migrations/003_goals_and_tdee.sql
-- migrations/004_meal_templates.sql
-- migrations/005_analytics_views.sql
```

#### Option 2: Via psql Command Line

```bash
# Get connection string from Supabase Settings > Database
export DATABASE_URL="postgresql://postgres:[PASSWORD]@db.xxxxxxxxxxxxx.supabase.co:5432/postgres"

# Run migrations in order
psql $DATABASE_URL -f migrations/001_initial_schema.sql
psql $DATABASE_URL -f migrations/002_weight_tracking.sql
psql $DATABASE_URL -f migrations/003_goals_and_tdee.sql
psql $DATABASE_URL -f migrations/004_meal_templates.sql
psql $DATABASE_URL -f migrations/005_analytics_views.sql
```

#### Option 3: Automated Migration Script

Create `scripts/migrate.sh`:

```bash
#!/bin/bash
set -e

DATABASE_URL=$1

if [ -z "$DATABASE_URL" ]; then
    echo "Usage: ./scripts/migrate.sh <database-url>"
    exit 1
fi

echo "Running migrations..."

for file in migrations/*.sql; do
    echo "Applying $file..."
    psql $DATABASE_URL -f $file
done

echo "Migrations complete!"
```

Run:
```bash
chmod +x scripts/migrate.sh
./scripts/migrate.sh "$DATABASE_URL"
```

### Verify Migrations

Check tables were created:

```sql
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;
```

Expected tables:
- `meals`
- `meal_items`
- `weight_entries`
- `user_goals`
- `daily_goals`
- `meal_templates`
- `template_items`

---

## Seeding Data

### Common Foods Database

Seed the common foods table for faster parsing:

```bash
psql $DATABASE_URL -f migrations/seeds/common_foods.sql
```

This adds ~1000 common foods with accurate nutrition data.

### Test Data (Development Only)

For development/testing, create sample data:

```sql
-- Create test user
INSERT INTO auth.users (id, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'test@example.com');

-- Create test goals
INSERT INTO user_goals (user_id, daily_calories, daily_protein_g, daily_carbs_g, daily_fat_g, daily_fiber_g)
VALUES (1, 2200, 165, 220, 73, 30);

-- Create test weight entries
INSERT INTO weight_entries (user_id, weight, measured_at)
VALUES
  ('00000000-0000-0000-0000-000000000001', 75.5, NOW() - INTERVAL '7 days'),
  ('00000000-0000-0000-0000-000000000001', 75.8, NOW() - INTERVAL '6 days'),
  ('00000000-0000-0000-0000-000000000001', 75.2, NOW() - INTERVAL '5 days');
```

---

## Environment Configuration

### Production .env

```bash
##########################################################
# Application Configuration
##########################################################
APP_MODE=production
HOST=0.0.0.0
PORT=8080

##########################################################
# Database Configuration
##########################################################
DATABASE_MODE=supabase
SUPABASE_URL=https://xxxxxxxxxxxxx.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_SERVICE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
SUPABASE_JWT_SECRET=your-jwt-secret
SUPABASE_STORAGE_BUCKET=lumen-nutrition

DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m

##########################################################
# AI Service Configuration
##########################################################
GROQ_API_KEY=gsk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
GROQ_MODEL=llama-3.1-70b-versatile
GROQ_MAX_REQUESTS_PER_DAY=14400

OPENROUTER_API_KEY=sk-or-xxxxxxxxxxxxxxxxxxxxxxxxxxxx
OPENROUTER_NUTRITION_PRESET=anthropic/claude-3.5-sonnet
OPENROUTER_VISION_PRESET=anthropic/claude-3.5-sonnet

##########################################################
# Storage Configuration
##########################################################
STORAGE_BUCKET_NAME=lumen-nutrition
STORAGE_MAX_PHOTO_SIZE_MB=10
STORAGE_ALLOWED_PHOTO_FORMATS=jpg,jpeg,png,heic

##########################################################
# Cache Configuration
##########################################################
CACHE_MODE=memory
CACHE_DEFAULT_TTL=5m

##########################################################
# Logging Configuration
##########################################################
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

##########################################################
# Feature Flags
##########################################################
FEATURE_AUTH=true
FEATURE_STORAGE=true
FEATURE_AI_ANALYSIS=true
FEATURE_CACHE=true
FEATURE_RATE_LIMIT=true
FEATURE_METRICS=true

##########################################################
# Rate Limiting Configuration
##########################################################
RATE_LIMIT_AI_PARSE_PER_DAY=50
RATE_LIMIT_API_CALLS_PER_HOUR=1000
RATE_LIMIT_COST_PER_USER_PER_MONTH=10.0
RATE_LIMIT_GLOBAL_COST_PER_MONTH=1000.0

##########################################################
# CORS Configuration
##########################################################
CORS_ALLOWED_ORIGINS=https://app.example.com,https://www.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization
```

### Security Checklist

- [ ] Change all default passwords
- [ ] Use strong JWT secret (32+ random characters)
- [ ] Enable HTTPS in production
- [ ] Set strict CORS origins (no wildcards)
- [ ] Enable rate limiting
- [ ] Enable authentication (`FEATURE_AUTH=true`)
- [ ] Set appropriate log level (`info` or `warn`)
- [ ] Secure environment variables (use secrets manager)
- [ ] Enable RLS on all Supabase tables

---

## Health Checks

### Endpoint

```bash
GET /health
```

### Expected Response

```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T10:30:00Z",
  "database": "connected",
  "cache": "active"
}
```

### Health Check Script

Create `scripts/health-check.sh`:

```bash
#!/bin/bash

URL=${1:-http://localhost:8080}
RESPONSE=$(curl -s -w "\n%{http_code}" $URL/health)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Health check passed"
    echo "$BODY" | jq .
    exit 0
else
    echo "✗ Health check failed (HTTP $HTTP_CODE)"
    echo "$BODY"
    exit 1
fi
```

Run:
```bash
chmod +x scripts/health-check.sh
./scripts/health-check.sh https://api.example.com
```

### Monitoring Integration

For production monitoring (e.g., Uptime Robot, Pingdom):

- **URL**: `https://api.example.com/health`
- **Method**: GET
- **Expected Status**: 200
- **Check Interval**: 5 minutes
- **Timeout**: 30 seconds
- **Alert on**: Status ≠ 200 or timeout

---

## Monitoring

### Application Logs

Logs are output to `stdout` in JSON format (production):

```json
{
  "level": "info",
  "time": "2025-11-15T10:30:00Z",
  "message": "Request completed",
  "method": "POST",
  "path": "/api/v1/meals/parse",
  "status": 200,
  "duration_ms": 1247,
  "user_id": "user-uuid"
}
```

#### Log Aggregation

For production, aggregate logs using:

**Option 1: Supabase Logs**
- Built-in to Supabase dashboard
- Automatic for Supabase Functions
- Limited retention (7 days free tier)

**Option 2: Cloud Logging Services**
- **Datadog**: Application monitoring + logs
- **Logtail**: Structured log management
- **CloudWatch**: AWS-specific

**Option 3: Self-Hosted**
- **Loki + Grafana**: Open-source log aggregation
- **ELK Stack**: Elasticsearch, Logstash, Kibana

### Metrics to Monitor

**Application Metrics**:
- Request rate (requests/second)
- Response times (p50, p95, p99)
- Error rate (5xx responses)
- AI parsing success rate
- Cache hit rate

**Resource Metrics**:
- CPU usage
- Memory usage
- Database connections
- Storage usage

**Business Metrics**:
- Daily active users
- AI parses per day
- Cost per user
- Average meals logged per user

### Alerting Rules

Set up alerts for:

| Metric | Threshold | Action |
|--------|-----------|--------|
| Error rate | >1% for 5 minutes | Page on-call |
| Response time p95 | >2s for 5 minutes | Investigate |
| Database connections | >20 (80% of max) | Scale up |
| AI cost | >$900/month | Review usage |
| Health check | Down for 2 minutes | Page on-call |

---

## Production Checklist

### Pre-Deployment

- [ ] All migrations run successfully
- [ ] RLS enabled on all tables
- [ ] Storage bucket created with policies
- [ ] Environment variables configured
- [ ] `.env` file secured (not in git)
- [ ] API keys valid and have sufficient quota
- [ ] CORS origins set correctly
- [ ] Health check endpoint responding
- [ ] Tests passing (`go test ./...`)
- [ ] Code reviewed and approved

### Deployment

- [ ] Build binary: `go build -o bin/api cmd/api/main.go`
- [ ] Binary runs without errors
- [ ] All endpoints accessible
- [ ] Authentication working
- [ ] File uploads working
- [ ] AI parsing working
- [ ] Database queries performing well
- [ ] Logs being captured

### Post-Deployment

- [ ] Health check monitoring active
- [ ] Log aggregation configured
- [ ] Alerts set up
- [ ] Load testing completed
- [ ] Backup strategy in place
- [ ] Rollback plan documented
- [ ] Team notified of deployment
- [ ] Documentation updated

---

## Deployment Platforms

### Fly.io (Recommended)

**Advantages**:
- Free tier available
- Global edge network
- Easy scaling
- Built-in health checks

**Setup**:

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
flyctl auth login

# Initialize app
flyctl launch

# Set secrets
flyctl secrets set SUPABASE_URL=https://...
flyctl secrets set SUPABASE_SERVICE_KEY=...
flyctl secrets set GROQ_API_KEY=...

# Deploy
flyctl deploy
```

Create `fly.toml`:

```toml
app = "lumen-nutrition-api"

[build]
  builder = "paketobuildpacks/builder:base"

[env]
  APP_MODE = "production"
  PORT = "8080"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [services.concurrency]
    hard_limit = 25
    soft_limit = 20

  [[services.tcp_checks]]
    interval = "15s"
    timeout = "2s"
    grace_period = "5s"
    restart_limit = 0

  [[services.http_checks]]
    interval = "10s"
    timeout = "2s"
    grace_period = "5s"
    method = "get"
    path = "/health"
```

### Railway

**Advantages**:
- Simple deployment
- Free tier
- Auto-scaling
- GitHub integration

**Setup**:

1. Connect GitHub repository
2. Configure environment variables
3. Set start command: `./bin/api`
4. Deploy

### Docker + Your Platform

**Dockerfile**:

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api cmd/api/main.go

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
CMD ["./api"]
```

Build and run:

```bash
docker build -t lumen-backend .
docker run -p 8080:8080 --env-file .env lumen-backend
```

---

## Troubleshooting

### Common Issues

**1. Database connection fails**

```
Error: failed to initialize Supabase client: connection refused
```

**Solution**:
- Verify `SUPABASE_URL` is correct
- Check `SUPABASE_SERVICE_KEY` is valid
- Ensure database is not paused (Supabase free tier pauses after 7 days inactivity)
- Check network connectivity

**2. AI parsing fails**

```
Error: failed to parse meal: API key invalid
```

**Solution**:
- Verify `GROQ_API_KEY` is correct
- Check API key hasn't expired
- Verify rate limits not exceeded
- Try fallback provider: set `OPENROUTER_API_KEY`

**3. File upload fails**

```
Error: failed to upload photo: bucket not found
```

**Solution**:
- Verify `SUPABASE_STORAGE_BUCKET` exists in Supabase
- Check bucket name matches exactly
- Ensure RLS policies allow uploads
- Verify file size under limit (10MB default)

**4. High memory usage**

```
Warning: Memory usage at 90%
```

**Solution**:
- Reduce cache TTL: `CACHE_DEFAULT_TTL=1m`
- Lower database connection pool: `DATABASE_MAX_OPEN_CONNS=10`
- Enable garbage collection tuning
- Scale vertically (more RAM)

**5. Slow response times**

```
Warning: p95 response time >5s
```

**Solution**:
- Enable caching: `FEATURE_CACHE=true`
- Add database indexes (check migration files)
- Optimize AI provider selection
- Use connection pooler for database
- Consider CDN for static assets

### Debug Mode

Enable debug logging:

```bash
LOG_LEVEL=debug go run cmd/api/main.go
```

This will log:
- All SQL queries
- AI provider requests/responses
- Cache hits/misses
- Detailed error stack traces

### Support

If issues persist:

1. Check logs: `LOG_LEVEL=debug`
2. Run health check: `curl http://localhost:8080/health`
3. Verify migrations: Check database tables exist
4. Test AI keys: Run parse request with debug logging
5. Open GitHub issue with logs and steps to reproduce

---

**Last Updated**: 2025-11-15
