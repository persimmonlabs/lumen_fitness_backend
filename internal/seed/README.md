# Common Foods Seeding System

A comprehensive seeding system for populating the Lumen Nutrition Tracker database with verified common food items. Includes 100+ foods across 8 categories with accurate USDA nutrition data.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Directory Structure](#directory-structure)
- [Quick Start](#quick-start)
- [CLI Usage](#cli-usage)
- [Food Data Structure](#food-data-structure)
- [Categories](#categories)
- [Programmatic Usage](#programmatic-usage)
- [Testing](#testing)
- [Data Sources](#data-sources)
- [Contributing](#contributing)

## Overview

The Common Foods Seeding System provides:

- **100+ verified foods** with accurate nutrition data
- **8 food categories** (Protein, Carbs, Vegetables, Fruits, Dairy, Fats, Snacks, Beverages)
- **CLI tool** for easy database seeding
- **Programmatic API** for integration
- **Comprehensive tests** with 80%+ coverage
- **Progress tracking** and statistics

## Features

- Accurate nutrition data per 100g (calories, protein, carbs, fat, fiber)
- Common serving sizes (e.g., "medium apple", "cup", "slice")
- Category-based organization
- Verification status for data quality
- Idempotent seeding (safe to run multiple times)
- Progress reporting
- Statistics and validation
- Custom food data support

## Directory Structure

```
backend/internal/seed/
├── common_foods.go          # Seeding logic and API
├── common_foods_test.go     # Comprehensive test suite
├── data/
│   └── common_foods.json    # 100+ food items with nutrition data
└── README.md                # This file

backend/cmd/seed/
└── main.go                  # CLI tool
```

## Quick Start

### 1. Seed the Database

```bash
# Basic seeding (uses default common_foods.json)
go run backend/cmd/seed/main.go

# With progress tracking
go run backend/cmd/seed/main.go --progress

# Clear existing data first
go run backend/cmd/seed/main.go --clear

# With validation
go run backend/cmd/seed/main.go --validate
```

### 2. View Statistics

```bash
# Show database statistics only
go run backend/cmd/seed/main.go --stats
```

### 3. Custom Data File

```bash
# Seed from custom JSON file
go run backend/cmd/seed/main.go --file path/to/custom_foods.json
```

## CLI Usage

### Command Line Flags

```
Usage: seed [options]

Options:
  -db string
        Database URL (defaults to DATABASE_URL env var)
  -file string
        Custom JSON file to seed from
  -clear
        Clear existing data before seeding
  -stats
        Show statistics only (no seeding)
  -validate
        Validate foods before seeding
  -progress
        Show progress during seeding
```

### Examples

```bash
# Seed with default data
go run backend/cmd/seed/main.go

# Clear existing data and seed
go run backend/cmd/seed/main.go --clear

# Seed from custom file
go run backend/cmd/seed/main.go --file custom.json

# Show statistics only
go run backend/cmd/seed/main.go --stats

# Validate and show progress
go run backend/cmd/seed/main.go --validate --progress

# Custom database URL
go run backend/cmd/seed/main.go --db "postgres://user:pass@localhost:5432/lumen"
```

### Output Example

```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║         Lumen Nutrition Tracker                           ║
║         Common Foods Seeder                               ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

Connecting to database...
Connected successfully!

Seeding common foods database...

Seeding completed successfully in 245ms!

Database Statistics:
═══════════════════════════════════════════════════════════
Total Foods:     108
Verified Foods:  108

By Category:
  Protein:        18
  Carbs:          20
  Vegetables:     12
  Fruits:         12
  Dairy:          12
  Fats:           14
  Snacks:         12
  Beverages:       8
═══════════════════════════════════════════════════════════
```

## Food Data Structure

### JSON Format

```json
{
  "foods": [
    {
      "name": "Chicken breast, grilled",
      "category": "Protein",
      "calories_per_100g": 165,
      "protein_per_100g": 31.0,
      "carbs_per_100g": 0.0,
      "fat_per_100g": 3.6,
      "fiber_per_100g": 0.0,
      "common_serving_name": "medium breast",
      "common_serving_grams": 150,
      "is_verified": true
    }
  ]
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Food item name (must be unique) |
| `category` | string | One of 8 valid categories |
| `calories_per_100g` | float64 | Calories per 100 grams |
| `protein_per_100g` | float64 | Protein in grams per 100g |
| `carbs_per_100g` | float64 | Carbohydrates in grams per 100g |
| `fat_per_100g` | float64 | Fat in grams per 100g |
| `fiber_per_100g` | float64 | Fiber in grams per 100g |
| `common_serving_name` | string | Common serving description |
| `common_serving_grams` | float64 | Weight of common serving in grams |
| `is_verified` | bool | Data verification status |

### Validation Rules

- **Name**: Required, must be unique
- **Category**: Must be one of: Protein, Carbs, Vegetables, Fruits, Dairy, Fats, Snacks, Beverages
- **Calories**: Must be > 0
- **Macronutrients**: Must be >= 0
- **Serving Size**: Must be > 0
- **Serving Name**: Required

## Categories

### Protein
Meat, fish, eggs, tofu, legumes, tempeh, seitan
- Examples: Chicken breast, Salmon, Eggs, Black beans, Tofu

### Carbs
Grains, rice, pasta, bread, cereals, potatoes
- Examples: White rice, Whole wheat bread, Oats, Sweet potato, Quinoa

### Vegetables
All vegetables including leafy greens
- Examples: Broccoli, Spinach, Carrots, Bell peppers, Kale

### Fruits
Fresh fruits and berries
- Examples: Apple, Banana, Strawberries, Blueberries, Orange

### Dairy
Milk, yogurt, cheese, and dairy alternatives
- Examples: Milk (whole/skim), Greek yogurt, Cheddar cheese, Almond milk

### Fats
Healthy fats, nuts, seeds, oils, nut butters
- Examples: Avocado, Almonds, Olive oil, Peanut butter, Chia seeds

### Snacks
Processed snacks, bars, crackers, treats
- Examples: Protein bar, Granola bar, Popcorn, Dark chocolate, Trail mix

### Beverages
Drinks including juices, sodas, protein shakes
- Examples: Orange juice, Protein shake, Coffee, Sports drink, Tea

## Programmatic Usage

### Basic Seeding

```go
package main

import (
    "context"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "lumen/internal/seed"
)

func main() {
    // Connect to database
    pool, err := pgxpool.New(context.Background(), dbURL)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create seeder
    seeder := seed.NewCommonFoodsSeeder(pool)

    // Seed database
    ctx := context.Background()
    if err := seeder.Seed(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Seeding completed successfully")
}
```

### Custom File Seeding

```go
// Seed from custom file
err := seeder.SeedFromFile(ctx, "path/to/custom_foods.json")
if err != nil {
    log.Fatal(err)
}
```

### With Progress Tracking

```go
// Create progress channel
progressChan := make(chan int, 200)

// Seed with progress
go func() {
    if err := seeder.SeedWithProgress(ctx, progressChan); err != nil {
        log.Fatal(err)
    }
}()

// Monitor progress
for processed := range progressChan {
    fmt.Printf("\rProcessed: %d foods", processed)
}
fmt.Println()
```

### With Validation

```go
// Seed with validation
err := seeder.SeedWithValidation(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Clear Data

```go
// Clear existing data
err := seeder.Clear(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Get Statistics

```go
// Get seeding statistics
stats, err := seeder.GetStats(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total foods: %d\n", stats["total"])
fmt.Printf("Verified: %d\n", stats["verified"])
fmt.Printf("Protein foods: %d\n", stats["Protein"])
```

### Load Food Data

```go
// Load foods from JSON file
foods, err := seeder.LoadFromFile("data/common_foods.json")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded %d foods\n", len(foods))
```

## Testing

### Run All Tests

```bash
# Run all tests
go test ./internal/seed/...

# With coverage
go test ./internal/seed/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Verbose output
go test ./internal/seed/... -v
```

### Test Coverage

The test suite includes:

- **File loading tests**: Valid/invalid JSON, missing files
- **Validation tests**: Data integrity, required fields
- **Seeding tests**: Database insertion, idempotency
- **Category tests**: All 8 categories represented
- **Serving size tests**: Valid serving data
- **Statistics tests**: Accurate counts and metrics
- **Progress tests**: Progress reporting functionality
- **Benchmarks**: Performance testing

### Example Test Run

```bash
$ go test ./internal/seed/... -v

=== RUN   TestCommonFoodsSeeder_New
--- PASS: TestCommonFoodsSeeder_New (0.00s)
=== RUN   TestCommonFoodsSeeder_LoadFromFile
--- PASS: TestCommonFoodsSeeder_LoadFromFile (0.01s)
=== RUN   TestCommonFoodsSeeder_Seed
--- PASS: TestCommonFoodsSeeder_Seed (0.15s)
=== RUN   TestCommonFoodsSeeder_Seed_Categories
--- PASS: TestCommonFoodsSeeder_Seed_Categories (0.12s)
=== RUN   TestCommonFoodsSeeder_Seed_Idempotent
--- PASS: TestCommonFoodsSeeder_Seed_Idempotent (0.23s)
PASS
coverage: 87.3% of statements
ok      lumen/internal/seed     0.512s
```

## Data Sources

All nutrition data is sourced from the **USDA FoodData Central** database:
- https://fdc.nal.usda.gov/

Data accuracy:
- **Verified**: All foods marked as `is_verified: true`
- **Updated**: Data reflects current USDA standards
- **Precision**: Nutrition values rounded to 1 decimal place

## Contributing

### Adding New Foods

1. **Find accurate data** from USDA FoodData Central
2. **Add to JSON file** following the structure
3. **Validate format**:
   ```bash
   go run backend/cmd/seed/main.go --validate --file data/common_foods.json
   ```
4. **Run tests**:
   ```bash
   go test ./internal/seed/...
   ```
5. **Submit pull request**

### Custom Categories

To add new categories:

1. Update `ValidateFood()` in `common_foods.go`
2. Update category list in `showStats()` in `cmd/seed/main.go`
3. Update this README
4. Add test coverage

### Best Practices

- Use accurate USDA data
- Include common serving sizes
- Mark foods as verified
- Follow JSON structure exactly
- Test before submitting
- Document unusual foods

## Database Schema

The seeder expects this table structure:

```sql
CREATE TABLE common_foods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL,
    calories_per_100g DECIMAL(10,2) NOT NULL,
    protein_per_100g DECIMAL(10,2) NOT NULL,
    carbs_per_100g DECIMAL(10,2) NOT NULL,
    fat_per_100g DECIMAL(10,2) NOT NULL,
    fiber_per_100g DECIMAL(10,2) NOT NULL,
    common_serving_name VARCHAR(100) NOT NULL,
    common_serving_grams DECIMAL(10,2) NOT NULL,
    is_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_common_foods_category ON common_foods(category);
CREATE INDEX idx_common_foods_verified ON common_foods(is_verified);
```

## Troubleshooting

### Connection Errors

```
Error: failed to connect to database
```

**Solution**: Check your `DATABASE_URL` environment variable or use `--db` flag

### File Not Found

```
Error: failed to read file data/common_foods.json
```

**Solution**: Run from the `backend/` directory or use absolute path with `--file`

### Validation Errors

```
Error: validation error: invalid category
```

**Solution**: Check that all foods use valid categories (case-sensitive)

### Duplicate Entries

The seeder handles duplicates automatically by updating existing records.

## License

This seeding system is part of the Lumen Nutrition Tracker project.

## Support

For issues or questions:
- Check this README
- Review test cases for examples
- Check database connection and schema
- Validate JSON file format
