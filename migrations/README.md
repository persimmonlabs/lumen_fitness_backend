# Lumen Nutrition Tracker - Database Migrations

This directory contains SQL migration files for the Lumen Nutrition Tracker database schema.

## Migration Files

### 001_initial_schema
**Up**: Creates all core tables with Row Level Security (RLS) policies
- `users` - User profiles extending Supabase auth
- `user_daily_goals` - Daily nutrition targets
- `meals` - Meal entries with timestamps
- `meal_items` - Individual foods within meals (denormalized nutrition)
- `meal_flags` - User-reported data issues
- `weight_entries` - Weight tracking
- `templates` - Reusable meal templates
- `template_items` - Foods within templates
- `common_foods` - Global food database
- `ai_usage` - AI API usage tracking
- `idempotency_keys` - Request deduplication

**Down**: Drops all tables

### 002_rpc_functions
**Up**: Creates database functions for atomic operations
- `create_meal_with_items(...)` - Create meal + items in single transaction
- `get_daily_nutrition(...)` - Get daily totals with timezone support
- `create_template_from_meal(...)` - Convert meal to reusable template
- `search_common_foods(...)` - Fuzzy search for common foods
- `get_nutrition_trends(...)` - Daily nutrition over date range

**Down**: Drops all functions

### 003_indexes
**Up**: Creates performance indexes and enables extensions
- pg_trgm extension for fuzzy text search
- Indexes on all foreign keys
- Composite indexes for common query patterns
- GIN indexes for full-text search

**Down**: Drops all indexes and extensions

## Running Migrations in Supabase

### Method 1: SQL Editor (Recommended)

1. Open Supabase Dashboard → SQL Editor
2. Run migrations in order:
   ```sql
   -- 1. Initial Schema
   -- Copy/paste contents of 001_initial_schema.up.sql

   -- 2. RPC Functions
   -- Copy/paste contents of 002_rpc_functions.up.sql

   -- 3. Indexes
   -- Copy/paste contents of 003_indexes.up.sql
   ```

3. Verify tables created:
   ```sql
   SELECT tablename FROM pg_tables WHERE schemaname = 'public';
   ```

### Method 2: Supabase CLI

```bash
# Install Supabase CLI
npm install -g supabase

# Initialize project (if not already done)
supabase init

# Link to your project
supabase link --project-ref your-project-ref

# Run migrations
supabase db push
```

### Method 3: Manual SQL Files

1. Connect to database using psql or any PostgreSQL client
2. Run in order:
   ```bash
   psql "postgresql://..." -f 001_initial_schema.up.sql
   psql "postgresql://..." -f 002_rpc_functions.up.sql
   psql "postgresql://..." -f 003_indexes.up.sql
   ```

## Rollback Migrations

To rollback, run `.down.sql` files in **reverse order**:

```sql
-- 1. Drop indexes first
-- Run 003_indexes.down.sql

-- 2. Drop functions
-- Run 002_rpc_functions.down.sql

-- 3. Drop tables
-- Run 001_initial_schema.down.sql
```

## Key Features

### Row Level Security (RLS)
All tables have RLS enabled with policies ensuring:
- Users can only access their own data
- No user can access other users' meals, templates, or weight entries
- Common foods are read-only for all users

### Timezone Handling
- All timestamps stored as `TIMESTAMPTZ` (UTC)
- RPC functions accept timezone parameter
- Date boundaries calculated in user's local timezone
- Example: `get_daily_nutrition(user_id, '2024-01-15', 'America/New_York')`

### Denormalized Nutrition
- Nutrition values stored directly in `meal_items` and `template_items`
- No joins needed for daily totals (fast queries)
- Maintains data consistency even if AI models change

### Atomic Transactions
- `create_meal_with_items()` ensures meal + all items created together
- Prevents orphaned meals or partial data

### Fuzzy Search
- `pg_trgm` extension enables similarity search
- `search_common_foods()` finds foods even with typos
- Example: "chiken breast" → "Chicken Breast"

### Performance Indexes
- Composite indexes on common query patterns
- GIN indexes for full-text search
- Date-based indexes for aggregation queries

## Data Validation

All tables include CHECK constraints:
- Positive values for nutrition (no negative calories)
- Reasonable ranges (calories < 10000, weight < 500kg)
- String length limits
- Email format validation
- Valid enums for flag types and sources

## Testing Migrations

After running migrations, verify:

```sql
-- 1. Check all tables exist
SELECT tablename FROM pg_tables WHERE schemaname = 'public';

-- 2. Verify RLS enabled
SELECT tablename, rowsecurity
FROM pg_tables
WHERE schemaname = 'public';

-- 3. Test RPC function
SELECT create_meal_with_items(
    auth.uid(),
    'Test Meal',
    NOW(),
    'Test notes',
    '[{"food_name":"Apple","quantity":1,"unit":"medium","calories":95,"protein":0.5,"carbs":25,"fat":0.3}]'::jsonb
);

-- 4. Test fuzzy search
SELECT * FROM search_common_foods('chiken breast', 5);

-- 5. Check indexes
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public';
```

## Common Issues

### Issue: "relation already exists"
**Solution**: Table already created. Either skip or run down migration first.

### Issue: "must be owner of extension uuid-ossp"
**Solution**: Extension already exists (Supabase enables by default). Safe to ignore.

### Issue: RLS policy errors
**Solution**: Ensure `auth.uid()` returns valid UUID. Test with authenticated user.

### Issue: Function execution denied
**Solution**: Run `GRANT EXECUTE` statements from migration file.

## Schema Version Tracking

To track applied migrations:

```sql
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Mark migrations as applied
INSERT INTO schema_migrations (version) VALUES
    ('001_initial_schema'),
    ('002_rpc_functions'),
    ('003_indexes');
```

## Next Steps

After migrations:
1. Populate `common_foods` table with nutrition data
2. Configure Supabase Auth (email/password or OAuth)
3. Set up Edge Functions for AI integration
4. Configure Storage buckets if needed for photos
5. Test API endpoints with authenticated users

## Support

For issues:
- Check Supabase Dashboard → Logs
- Review migration output for errors
- Verify user authentication before testing RLS
- Ensure project has necessary extensions enabled

## Notes

- **IMPORTANT**: Always backup database before running migrations
- Migrations are idempotent where possible (IF NOT EXISTS)
- RLS policies tested with authenticated users only
- Timezone handling requires user timezone stored in `users.timezone`
- AI usage tracking is for internal monitoring (no user access)
