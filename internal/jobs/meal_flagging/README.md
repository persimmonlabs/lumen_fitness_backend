# Meal Flagging Job - Quality Control for Lumen Nutrition Tracker

Automated nightly job that detects quality issues in logged meals for manual review.

## Overview

The Meal Flagging Job analyzes meal data and flags potential quality issues across four categories:

1. **Unusual Portions** - Detects abnormally large portion sizes
2. **Macro Mismatch** - Validates calorie calculations against macronutrients
3. **Duplicate Detection** - Identifies potentially duplicate meal entries
4. **Low Confidence** - Flags meals with low AI confidence scores

## Architecture

```
backend/internal/jobs/meal_flagging/
├── models.go           # Data structures and flag types
├── flagging.go         # Flagging logic and checks
├── job.go              # Job scheduler and batch processing
├── flagging_test.go    # Flagging logic tests
├── job_test.go         # Job execution tests
├── config.example.json # Example configuration
└── README.md           # This file

backend/cmd/scheduler/
└── main.go             # Scheduler application entry point
```

## Flag Detection Rules

### 1. Unusual Portions

Flags meals with abnormally large portion sizes:

**Triggers:**
- Single item > 1000 calories
- Category-specific thresholds (e.g., chicken > 450g, oil > 30ml)
- Portion exceeds typical serving by 3x or more

**Severity Levels:**
- **High**: Item > 5x typical portion OR > 1000 calories
- **Medium**: Item > 3x typical portion
- **Low**: Item > 1x but < 3x typical portion

**Example:**
```
Portion: 600g chicken breast (990 calories)
Threshold: 450g
Flag: "Unusual portion: exceeds typical 450g"
Severity: Low (600/450 = 1.33x)
```

### 2. Macro Mismatch

Validates that reported calories match macronutrient calculations:

**Formula:** Expected Calories = (Protein × 4) + (Carbs × 4) + (Fat × 9)

**Triggers:**
- Reported calories differ from calculated by > 10% (configurable)

**Severity Levels:**
- **High**: > 30% difference
- **Medium**: > 20% difference
- **Low**: > 10% difference

**Example:**
```
Reported: 600 calories
Calculated: 370 calories (30g protein + 40g carbs + 10g fat)
Difference: 62% → High severity flag
```

### 3. Duplicate Detection

Identifies potentially duplicate meal entries:

**Triggers:**
- Meals within 30-minute window (configurable)
- Item similarity ≥ 80% (configurable)

**Similarity Algorithm:**
- Compares meal items by description
- Uses case-insensitive matching
- Checks for substring and word overlap
- Calculates match percentage

**Severity Levels:**
- **High**: > 95% similarity
- **Medium**: > 90% similarity
- **Low**: ≥ 80% similarity

**Example:**
```
Meal 1 (2:00 PM): chicken breast, rice, broccoli
Meal 2 (2:15 PM): chicken breast, rice, broccoli
Similarity: 100% → High severity duplicate flag
```

### 4. Low Confidence

Flags meals with low AI detection confidence:

**Triggers:**
- AI confidence < 0.7 (70%, configurable)

**Severity Levels:**
- **High**: < 0.5 (50%)
- **Medium**: < 0.6 (60%)
- **Low**: < 0.7 (70%)

**Example:**
```
AI Confidence: 0.55 (55%)
Threshold: 0.70 (70%)
Flag: "Low AI confidence: manual review recommended"
Severity: Medium
```

## Configuration

### Configuration File

Create `config/meal_flagging.json`:

```json
{
  "enabled": true,
  "schedule_time": "02:00",
  "batch_size": 1000,
  "lookback_days": 1,
  "macro_tolerance_percent": 10.0,
  "confidence_threshold": 0.7,
  "duplicate_window_minutes": 30,
  "duplicate_similarity_percent": 80.0
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the job |
| `schedule_time` | string | `"02:00"` | Daily run time (24-hour format) |
| `batch_size` | int | `1000` | Number of meals per batch |
| `lookback_days` | int | `1` | Days of history to check |
| `macro_tolerance_percent` | float | `10.0` | Macro calculation tolerance (%) |
| `confidence_threshold` | float | `0.7` | Minimum acceptable AI confidence |
| `duplicate_window_minutes` | int | `30` | Time window for duplicate detection |
| `duplicate_similarity_percent` | float | `80.0` | Minimum similarity for duplicates (%) |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `JOB_CONFIG_PATH` | No | Path to config file (default: `config/meal_flagging.json`) |
| `RUN_ON_STARTUP` | No | Run job immediately on startup (`true`/`false`) |

## Database Schema

The job requires the following table:

```sql
CREATE TABLE meal_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_id UUID NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    details JSONB,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    resolved_by UUID REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meal_flags_meal_id ON meal_flags(meal_id);
CREATE INDEX idx_meal_flags_user_id ON meal_flags(user_id);
CREATE INDEX idx_meal_flags_flag_type ON meal_flags(flag_type);
CREATE INDEX idx_meal_flags_resolved ON meal_flags(resolved);
CREATE INDEX idx_meal_flags_created_at ON meal_flags(created_at DESC);
```

## Usage

### Running the Scheduler

**1. Set environment variables:**

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/lumen"
export JOB_CONFIG_PATH="config/meal_flagging.json"
export RUN_ON_STARTUP="false"
```

**2. Run the scheduler:**

```bash
cd backend
go run cmd/scheduler/main.go
```

**3. The job will:**
- Start immediately if `RUN_ON_STARTUP=true`
- Otherwise run at the configured schedule time daily
- Continue running until interrupted (Ctrl+C)

### Running Programmatically

```go
import (
    "database/sql"
    "log"

    "github.com/lumen/fitness-app/internal/jobs/meal_flagging"
)

func main() {
    // Connect to database
    db, _ := sql.Open("postgres", "your-connection-string")
    defer db.Close()

    // Create job with custom config
    config := &meal_flagging.Config{
        Enabled:                    true,
        ScheduleTime:               "03:00",
        BatchSize:                  500,
        LookbackDays:               2,
        MacroTolerancePercent:      15.0,
        ConfidenceThreshold:        0.6,
        DuplicateWindowMinutes:     60,
        DuplicateSimilarityPercent: 85.0,
    }

    logger := log.Default()
    job := meal_flagging.NewMealFlaggingJob(db, config, logger)

    // Start scheduled job
    if err := job.Start(); err != nil {
        log.Fatal(err)
    }

    // Or run immediately
    if err := job.Run(context.Background()); err != nil {
        log.Printf("Job failed: %v", err)
    }

    // Or run for specific user
    summary, err := job.RunForUser(context.Background(), userID)
    if err != nil {
        log.Printf("User job failed: %v", err)
    }

    log.Printf("Checked %d meals, created %d flags",
        summary.TotalMealsChecked,
        summary.TotalFlagsCreated)
}
```

### Running Tests

```bash
# Run all tests
go test ./internal/jobs/meal_flagging/... -v

# Run with coverage
go test ./internal/jobs/meal_flagging/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./internal/jobs/meal_flagging -run TestCheckUnusualPortion -v

# Run tests with race detection
go test ./internal/jobs/meal_flagging/... -race
```

## Job Execution Flow

```
1. Job starts at scheduled time (e.g., 2:00 AM)
   │
2. Calculate time range (lookback_days)
   │
3. Fetch meals in batches (batch_size)
   │
4. For each meal:
   ├── Load meal items from database
   ├── Run unusual portion check
   ├── Run macro mismatch check
   ├── Run low confidence check
   ├── Run duplicate check (queries database)
   └── Insert flags if any issues detected
   │
5. Log summary:
   ├── Total meals checked
   ├── Total flags created
   ├── Flags by type
   ├── Flags by severity
   └── Any errors encountered
   │
6. Complete (wait for next scheduled run)
```

## Performance Considerations

### Batch Processing

- Processes meals in configurable batches (default: 1000)
- Prevents memory issues with large datasets
- Allows for progress tracking

### Database Optimization

```sql
-- Ensure indexes exist for efficient queries
CREATE INDEX idx_meals_logged_at ON meals(logged_at DESC);
CREATE INDEX idx_meals_user_id_logged_at ON meals(user_id, logged_at DESC);
CREATE INDEX idx_meal_items_meal_id ON meal_items(meal_id);
```

### Connection Pooling

The scheduler configures database connections:
- Max open connections: 10
- Max idle connections: 5
- Connection max lifetime: 5 minutes

### Duplicate Detection

Duplicate checking is the most expensive operation:
- Queries nearby meals within time window
- Loads items for similarity comparison
- Consider increasing `duplicate_window_minutes` cautiously

## Monitoring

### Log Output

The job logs detailed execution information:

```
[SCHEDULER] 2025-01-15 02:00:00 Starting meal flagging job
[SCHEDULER] 2025-01-15 02:00:05 === Meal Flagging Job Summary ===
[SCHEDULER] 2025-01-15 02:00:05 Meals checked: 1250
[SCHEDULER] 2025-01-15 02:00:05 Flags created: 47
[SCHEDULER] 2025-01-15 02:00:05 Duration: 4.523s
[SCHEDULER] 2025-01-15 02:00:05 Flags by type:
[SCHEDULER] 2025-01-15 02:00:05   - unusual_portion: 18
[SCHEDULER] 2025-01-15 02:00:05   - macro_mismatch: 15
[SCHEDULER] 2025-01-15 02:00:05   - duplicate: 8
[SCHEDULER] 2025-01-15 02:00:05   - low_confidence: 6
[SCHEDULER] 2025-01-15 02:00:05 Flags by severity:
[SCHEDULER] 2025-01-15 02:00:05   - high: 12
[SCHEDULER] 2025-01-15 02:00:05   - medium: 20
[SCHEDULER] 2025-01-15 02:00:05   - low: 15
[SCHEDULER] 2025-01-15 02:00:05 ================================
```

### Metrics to Track

- **Meals checked per run**: Should match expected volume
- **Flags created**: Unusual spike indicates data quality issues
- **Execution duration**: Should be consistent; spikes indicate performance issues
- **Error rate**: Should be near zero

### Querying Flags

```sql
-- Get all unresolved flags
SELECT * FROM meal_flags WHERE resolved = FALSE ORDER BY created_at DESC;

-- Flags by type
SELECT flag_type, COUNT(*) as count
FROM meal_flags
WHERE resolved = FALSE
GROUP BY flag_type;

-- High severity flags
SELECT * FROM meal_flags
WHERE severity = 'high' AND resolved = FALSE
ORDER BY created_at DESC;

-- Flags for specific user
SELECT * FROM meal_flags
WHERE user_id = 'user-uuid' AND resolved = FALSE;
```

## Deployment

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o scheduler cmd/scheduler/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/scheduler .
COPY config/meal_flagging.json config/

CMD ["./scheduler"]
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: meal-flagging-job
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: scheduler
            image: lumen/meal-flagging:latest
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: url
            - name: RUN_ON_STARTUP
              value: "true"
          restartPolicy: OnFailure
```

### systemd Service

```ini
[Unit]
Description=Lumen Meal Flagging Scheduler
After=network.target postgresql.service

[Service]
Type=simple
User=lumen
WorkingDirectory=/opt/lumen
Environment="DATABASE_URL=postgresql://user:pass@localhost/lumen"
Environment="JOB_CONFIG_PATH=/opt/lumen/config/meal_flagging.json"
ExecStart=/opt/lumen/scheduler
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Job not running

**Check configuration:**
```bash
# Verify enabled flag
cat config/meal_flagging.json | grep enabled

# Check schedule time format
cat config/meal_flagging.json | grep schedule_time
```

**Check logs:**
```bash
# Look for errors on startup
tail -f /var/log/lumen/scheduler.log | grep ERROR

# Verify cron expression
# Schedule "02:00" becomes cron "0 2 * * *"
```

### No flags created

**Possible causes:**
- No meals in lookback period
- All meals pass quality checks
- Thresholds too lenient

**Debug:**
```sql
-- Check meal count in lookback period
SELECT COUNT(*) FROM meals
WHERE logged_at >= NOW() - INTERVAL '1 day';

-- Check if any flags exist
SELECT COUNT(*) FROM meal_flags;
```

### High error rate

**Check database connection:**
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check connection limits
psql $DATABASE_URL -c "SHOW max_connections"
```

**Check database indexes:**
```sql
-- Verify indexes exist
SELECT indexname FROM pg_indexes
WHERE tablename = 'meals' OR tablename = 'meal_items';
```

### Performance issues

**Reduce batch size:**
```json
{
  "batch_size": 500
}
```

**Reduce duplicate window:**
```json
{
  "duplicate_window_minutes": 15
}
```

**Check query performance:**
```sql
-- Enable query logging
SET log_statement = 'all';

-- Analyze slow queries
EXPLAIN ANALYZE SELECT ... FROM meals ...
```

## API Integration

Flags can be accessed via REST API endpoints:

```go
// Get flags for meal
GET /api/v1/meals/{meal_id}/flags

// Get flags for user
GET /api/v1/users/{user_id}/flags

// Resolve flag
POST /api/v1/flags/{flag_id}/resolve

// Get flag statistics
GET /api/v1/flags/stats
```

Response example:
```json
{
  "flags": [
    {
      "id": "uuid",
      "meal_id": "uuid",
      "user_id": "uuid",
      "flag_type": "unusual_portion",
      "severity": "high",
      "description": "Single item 'chicken breast' has excessive calories (990 cal)",
      "details": {
        "item_description": "chicken breast",
        "calories": 990,
        "threshold": 1000
      },
      "resolved": false,
      "created_at": "2025-01-15T02:30:00Z"
    }
  ]
}
```

## Best Practices

1. **Run during low-traffic hours**: Default 2:00 AM minimizes user impact
2. **Monitor flag volume**: Sudden spikes indicate data quality issues
3. **Tune thresholds gradually**: Start lenient, tighten based on false positive rate
4. **Review resolved flags**: Learn from patterns to improve detection
5. **Archive old flags**: Prevent table bloat, keep last 90 days
6. **Test configuration changes**: Use `RUN_ON_STARTUP` to validate before deployment

## Support

For issues or questions:
- Check logs in `/var/log/lumen/scheduler.log`
- Review database query performance
- Verify configuration settings
- Check database connection and indexes
- Review test cases for expected behavior

## License

Copyright © 2025 Lumen Nutrition Tracker
