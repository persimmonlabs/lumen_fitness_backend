# Meal Flagging Job

Automated nightly job system for detecting quality issues in meal logs.

## Overview

The Meal Flagging Job automatically analyzes meal entries to detect potential quality issues such as unusual portions, macro mismatches, duplicates, and low AI confidence scores. The system runs nightly to maintain data quality and flag entries that may need manual review.

## Features

- **Automated Flagging**: Runs nightly at 2am (configurable)
- **Batch Processing**: Handles large datasets efficiently (1000 meals per batch)
- **Multiple Detection Rules**: 4 independent flagging mechanisms
- **Severity Classification**: Flags are categorized as low, medium, or high severity
- **Flexible Execution**: Run once, scheduled, or for specific users
- **Comprehensive Logging**: Detailed execution summaries and error reporting

## Flagging Rules

### 1. Unusual Portion Detection

Flags meals with portions significantly larger than typical servings.

**Thresholds:**
- Chicken/Beef/Pork: > 450g
- Fish: > 500g
- Rice/Pasta: > 300g (dry weight)
- Bread: > 400g
- Oil: > 30ml
- Butter: > 50g
- Cheese: > 200g
- Nuts: > 100g
- **Any single item: > 1000 calories**

**Severity Calculation:**
- **High**: > 5x typical serving
- **Medium**: > 3x typical serving
- **Low**: > threshold but < 3x

**Example:**
```
Meal: "Grilled Chicken"
Item: Chicken breast, 600g, 990 calories
Flag: Unusual portion (600g exceeds typical 450g)
Severity: Low
```

### 2. Macro Mismatch Detection

Validates that reported calories match macronutrient calculations.

**Formula:**
```
Calculated Calories = (Protein × 4) + (Carbs × 4) + (Fat × 9)
Tolerance = ±10% by default
```

**Severity Calculation:**
- **High**: > 30% difference
- **Medium**: > 20% difference
- **Low**: > 10% difference

**Example:**
```
Meal Macros: P=30g, C=40g, F=10g
Calculated: (30×4) + (40×4) + (10×9) = 370 cal
Reported: 500 cal
Difference: 35% → Flag as HIGH severity
```

### 3. Duplicate Detection

Identifies meals logged within 30 minutes with similar items.

**Detection Logic:**
- Time window: ±30 minutes (configurable)
- Similarity threshold: 80% item match (configurable)
- Uses fuzzy string matching for item descriptions

**Severity Calculation:**
- **High**: > 95% similarity
- **Medium**: > 90% similarity
- **Low**: > 80% similarity

**Example:**
```
Meal 1 (12:00): Chicken breast, rice, broccoli
Meal 2 (12:20): Chicken, rice, broccoli
Similarity: 100% → Flag newer meal as HIGH severity
```

### 4. Low Confidence Detection

Flags meals with low AI recognition confidence.

**Threshold:** < 0.7 (70% confidence)

**Severity Calculation:**
- **High**: < 0.5 (50%)
- **Medium**: < 0.6 (60%)
- **Low**: < 0.7 (70%)

**Example:**
```
Meal: "Mystery food"
AI Confidence: 0.45
Flag: Low confidence, manual review recommended
Severity: High
```

## Configuration

### Default Configuration

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
| `enabled` | boolean | `true` | Enable/disable the job |
| `schedule_time` | string | `"02:00"` | Daily execution time (24-hour format) |
| `batch_size` | integer | `1000` | Number of meals per batch |
| `lookback_days` | integer | `1` | Days of history to process |
| `macro_tolerance_percent` | float | `10.0` | Macro mismatch tolerance (%) |
| `confidence_threshold` | float | `0.7` | Min AI confidence threshold |
| `duplicate_window_minutes` | integer | `30` | Duplicate detection window |
| `duplicate_similarity_percent` | float | `80.0` | Min similarity for duplicates (%) |

## Usage

### Running Once

Process the last 24 hours of meals immediately:

```bash
# Set database connection
export DATABASE_URL="postgresql://user:pass@localhost:5432/lumen"

# Run the job
go run cmd/jobs/meal_flagging/main.go
```

### Scheduled Execution

Run as a daemon with cron-like scheduling:

```bash
go run cmd/jobs/meal_flagging/main.go -mode=schedule
```

The job will run daily at the configured time (default: 2am).

### User-Specific Execution

Process meals for a specific user:

```bash
go run cmd/jobs/meal_flagging/main.go \
  -mode=user \
  -user=550e8400-e29b-41d4-a716-446655440000
```

### Custom Configuration

Use a custom configuration file:

```bash
go run cmd/jobs/meal_flagging/main.go \
  -config=config.json
```

### Verbose Logging

Enable detailed logging:

```bash
go run cmd/jobs/meal_flagging/main.go -verbose
```

## Building

### Compile Binary

```bash
cd backend
go build -o bin/meal_flagging cmd/jobs/meal_flagging/main.go
```

### Run Binary

```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/lumen"
./bin/meal_flagging -mode=run-once
```

## Deployment

### Systemd Service

Create `/etc/systemd/system/meal-flagging.service`:

```ini
[Unit]
Description=Meal Flagging Job
After=network.target postgresql.service

[Service]
Type=simple
User=lumen
WorkingDirectory=/opt/lumen
Environment="DATABASE_URL=postgresql://user:pass@localhost:5432/lumen"
ExecStart=/opt/lumen/bin/meal_flagging -mode=schedule
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable meal-flagging
sudo systemctl start meal-flagging
sudo systemctl status meal-flagging
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o meal_flagging cmd/jobs/meal_flagging/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/meal_flagging /usr/local/bin/
ENTRYPOINT ["meal_flagging"]
CMD ["-mode=schedule"]
```

Build and run:

```bash
docker build -t meal-flagging .
docker run -e DATABASE_URL="..." meal-flagging
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: meal-flagging
spec:
  schedule: "0 2 * * *"  # 2am daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: meal-flagging
            image: lumen/meal-flagging:latest
            args: ["-mode=run-once"]
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: url
          restartPolicy: OnFailure
```

## Monitoring

### Job Output

The job logs a comprehensive summary after each execution:

```
[MEAL-FLAGGING] === Meal Flagging Job Summary ===
[MEAL-FLAGGING] Meals checked: 1523
[MEAL-FLAGGING] Flags created: 47
[MEAL-FLAGGING] Duration: 2.34s
[MEAL-FLAGGING] Flags by type:
[MEAL-FLAGGING]   - unusual_portion: 23
[MEAL-FLAGGING]   - macro_mismatch: 15
[MEAL-FLAGGING]   - duplicate: 7
[MEAL-FLAGGING]   - low_confidence: 2
[MEAL-FLAGGING] Flags by severity:
[MEAL-FLAGGING]   - high: 12
[MEAL-FLAGGING]   - medium: 18
[MEAL-FLAGGING]   - low: 17
[MEAL-FLAGGING] ================================
```

### Querying Flags

```sql
-- Get recent flags by type
SELECT flag_type, severity, COUNT(*) as count
FROM meal_flags
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY flag_type, severity
ORDER BY count DESC;

-- Get high severity flags needing review
SELECT m.id, m.name, f.flag_type, f.description, f.created_at
FROM meal_flags f
JOIN meals m ON f.meal_id = m.id
WHERE f.severity = 'high'
  AND f.resolved = false
ORDER BY f.created_at DESC;

-- Get user-specific flag summary
SELECT u.email, COUNT(*) as flag_count
FROM meal_flags f
JOIN users u ON f.user_id = u.id
WHERE f.created_at > NOW() - INTERVAL '30 days'
  AND f.resolved = false
GROUP BY u.email
ORDER BY flag_count DESC;
```

## API Integration

### Trigger Job via HTTP Endpoint

Add to your API handlers:

```go
func (h *Handler) RunMealFlaggingJob(w http.ResponseWriter, r *http.Request) {
    job := meal_flagging.NewMealFlaggingJob(h.db, nil, h.logger)

    if err := job.Run(r.Context()); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "completed",
    })
}
```

### Get Flag Summary for User

```go
func (h *Handler) GetUserFlagSummary(w http.ResponseWriter, r *http.Request) {
    userID := getUserIDFromContext(r.Context())

    job := meal_flagging.NewMealFlaggingJob(h.db, nil, h.logger)
    summary, err := job.RunForUser(r.Context(), userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(summary)
}
```

## Testing

### Run All Tests

```bash
cd backend/internal/jobs/meal_flagging
go test -v
```

### Run Specific Tests

```bash
go test -v -run TestCheckUnusualPortion
go test -v -run TestCheckMacroMismatch
go test -v -run TestCheckDuplicate
```

### Coverage Report

```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Database Schema

The job requires the following tables:

```sql
-- Meal flags table
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
    created_at TIMESTAMP DEFAULT NOW(),

    INDEX idx_meal_flags_meal_id (meal_id),
    INDEX idx_meal_flags_user_id (user_id),
    INDEX idx_meal_flags_type (flag_type),
    INDEX idx_meal_flags_resolved (resolved),
    INDEX idx_meal_flags_created_at (created_at)
);
```

## Performance

- **Batch Processing**: 1000 meals per batch
- **Typical Performance**: ~500-1000 meals/second
- **Memory Usage**: ~50-100MB for 10K meals
- **Database Impact**: Minimal, uses indexed queries

## Troubleshooting

### Job Not Running

1. Check if enabled in config: `"enabled": true`
2. Verify DATABASE_URL is set
3. Check database connectivity: `psql $DATABASE_URL`
4. Review logs for errors

### Too Many Flags

Adjust tolerance settings:
- Increase `macro_tolerance_percent` to 15-20%
- Raise `confidence_threshold` to 0.8
- Increase portion thresholds in code

### Missing Flags

Lower tolerance settings:
- Decrease `macro_tolerance_percent` to 5%
- Lower `confidence_threshold` to 0.6
- Decrease portion thresholds

### High Resource Usage

- Reduce `batch_size` to 500
- Increase `lookback_days` only when needed
- Add database indexes if slow

## Future Enhancements

- [ ] Configurable portion thresholds via database
- [ ] Machine learning for adaptive thresholds
- [ ] Flag resolution workflow
- [ ] Email notifications for high severity flags
- [ ] Admin dashboard for flag management
- [ ] User feedback integration
- [ ] Flag pattern analysis
- [ ] Auto-resolve low confidence flags after user confirmation

## Support

For issues or questions:
- Check logs: `journalctl -u meal-flagging -f`
- Review database: `SELECT * FROM meal_flags ORDER BY created_at DESC LIMIT 10`
- Contact: backend-team@lumen.com
