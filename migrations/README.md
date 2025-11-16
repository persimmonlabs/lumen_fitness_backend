# Database Migrations

This directory contains all database migrations for the Lumen Fitness application.

## Migration Naming Convention

Migrations follow the pattern: `{number}_{description}.{up|down}.sql`

- **Number**: Sequential version number (001, 002, etc.)
- **Description**: Short kebab-case description
- **Direction**:
  - `.up.sql` - Apply migration (creates/modifies schema)
  - `.down.sql` - Rollback migration (reverses changes)

## Migration Order

Migrations must be run in sequential order (001 → 008):

### Core Schema (001-003)
1. `001_initial_schema.up.sql` - Creates all base tables (meals, weight, templates, goals, analytics, common_foods, etc.)
2. `002_rpc_functions.up.sql` - Creates stored procedures for analytics calculations
3. `003_indexes.up.sql` - Creates performance indexes

### Feature Additions (004-006)
4. `004_add_normalized_description.up.sql` - Adds normalized_description column to meals
5. `005_add_draft_status.up.sql` - Adds draft meal workflow support
6. `006_add_onboarding_tracking.up.sql` - Adds onboarding state tracking

### User Management (007)
7. `007_user_profiles_and_settings.up.sql` - Creates user_profiles and user_settings tables

### Search Optimization (008)
8. `008_common_foods_indexes.up.sql` - Adds full-text search indexes to common_foods

## Running Migrations

### Using psql directly
```bash
# Run migrations in order
psql $DATABASE_URL < migrations/001_initial_schema.up.sql
psql $DATABASE_URL < migrations/002_rpc_functions.up.sql
psql $DATABASE_URL < migrations/003_indexes.up.sql
psql $DATABASE_URL < migrations/004_add_normalized_description.up.sql
psql $DATABASE_URL < migrations/005_add_draft_status.up.sql
psql $DATABASE_URL < migrations/006_add_onboarding_tracking.up.sql
psql $DATABASE_URL < migrations/007_user_profiles_and_settings.up.sql
psql $DATABASE_URL < migrations/008_common_foods_indexes.up.sql
```

### Using migrate tool
```bash
# Install migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply all up migrations
migrate -path ./migrations -database $DATABASE_URL up

# Rollback last migration
migrate -path ./migrations -database $DATABASE_URL down 1
```

## Migration Dependencies

| Migration | Requires | Description |
|-----------|----------|-------------|
| 001 | None | Base schema creation |
| 002 | 001 | References tables from 001 |
| 003 | 001 | Indexes tables from 001 |
| 004 | 001 | Modifies meals table |
| 005 | 001 | Modifies meals table |
| 006 | 001 | Modifies meals table |
| 007 | 001 | References auth.users from Supabase |
| 008 | 001 | Indexes common_foods table |

## Rollback Order

When rolling back, run migrations in **reverse order** (008 → 001):

```bash
psql $DATABASE_URL < migrations/008_common_foods_indexes.down.sql
psql $DATABASE_URL < migrations/007_user_profiles_and_settings.down.sql
psql $DATABASE_URL < migrations/006_add_onboarding_tracking.down.sql
psql $DATABASE_URL < migrations/005_add_draft_status.down.sql
psql $DATABASE_URL < migrations/004_add_normalized_description.down.sql
psql $DATABASE_URL < migrations/003_indexes.down.sql
psql $DATABASE_URL < migrations/002_rpc_functions.down.sql
psql $DATABASE_URL < migrations/001_initial_schema.down.sql
```

## Schema Overview

### Core Tables
- `nutrition.meals` - User meal entries with items
- `nutrition.meal_items` - Individual food items in meals
- `nutrition.weight_entries` - User weight tracking
- `nutrition.templates` - Reusable meal templates
- `nutrition.template_items` - Items in templates
- `user_goals` - User nutrition goals
- `daily_goals` - Day-specific goal overrides
- `common_foods` - Database of verified common foods

### User Management (Migration 007)
- `user_profiles` - Extended user information
- `user_settings` - User preferences and configuration

### Utility Functions (Migration 002)
- `get_daily_nutrition()` - Calculate daily nutrition totals
- `get_date_range_nutrition()` - Calculate nutrition over date range
- Various analytics and trajectory functions

## Important Notes

1. **Never modify existing migrations** after they've been applied to production
2. **Always create new migrations** for schema changes
3. **Test rollbacks** in development before applying to production
4. **Backup database** before running migrations in production
5. Common foods data is **loaded from JSON at startup** (`internal/seed/data/common_foods.json`), not seeded via migration

## Troubleshooting

### Error: "column does not exist"
- Check migration dependencies are run in correct order
- Verify column name matches schema (e.g., `verified` not `is_verified` in common_foods)

### Error: "relation already exists"
- Migration may have already been applied
- Use `CREATE TABLE IF NOT EXISTS` for idempotency

### Error: "function does not exist"
- RPC functions migration (002) may not be applied
- Check function definitions exist in database

## Quick Start for New Deployment

Run all 8 migrations in order for a fresh database:

```bash
cd backend/migrations

# Run all migrations (001-008)
for i in {1..8}; do
  num=$(printf "%03d" $i)
  file=$(ls ${num}_*.up.sql 2>/dev/null)
  if [ -f "$file" ]; then
    echo "Running $file..."
    psql $DATABASE_URL < "$file"
  fi
done
```

All migrations complete successfully = ✅ Database ready for production!

## Current Schema Version

**Latest:** 008 (Common foods indexes)
**Total Migrations:** 8 (001-008, sequential)
