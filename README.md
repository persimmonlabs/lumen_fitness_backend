# Lumen Nutrition Tracker - Backend

> AI-powered nutrition tracking and analysis platform built with Go

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Supabase](https://img.shields.io/badge/Supabase-Backend-3ECF8E?style=flat&logo=supabase)](https://supabase.com/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Overview

Lumen Nutrition Tracker is a comprehensive nutrition tracking backend that uses **AI to parse meal descriptions** into structured nutrition data. Built with Go for high performance and reliability.

### Key Features

- **AI-Powered Meal Parsing** - Parse meal descriptions using Groq/OpenRouter AI
- **Photo Upload & Management** - Upload, compress, and manage meal photos
- **Voice Transcription** - Convert voice recordings to text using Groq Whisper
- **Weight Trajectory Prediction** - AI-powered weight prediction based on trends
- **Smart Meal Suggestions** - Personalized meal recommendations based on goals
- **Draft Meal Tracking** - Track parsing status with real-time updates
- **Weight Tracking** - Track weight trends and rate of change
- **Nutrition Goals** - Set and track daily nutrition goals with TDEE calculation
- **Analytics & Insights** - Daily, weekly, and monthly nutrition analytics
- **Meal Templates** - Save and reuse common meals
- **Cost Management** - Built-in AI cost tracking and limits
- **Feature Flags** - Graceful degradation with feature toggles
- **Clean Architecture** - Modular, testable, maintainable code

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Language** | Go 1.24+ |
| **Database** | PostgreSQL (Supabase) |
| **Authentication** | Supabase Auth (JWT) |
| **File Storage** | Supabase Storage |
| **AI Providers** | Groq (primary), OpenRouter (fallback) |
| **Caching** | In-memory / Redis (optional) |
| **Web Framework** | chi router |
| **Logging** | slog (stdlib) |
| **Validation** | go-playground/validator |

---

## Quick Start

### Prerequisites

- **Go 1.24+** - [Install Go](https://golang.org/dl/)
- **Supabase Account** - [Sign up](https://supabase.com/) (free tier available)
- **AI API Keys** - [Groq](https://console.groq.com/) (free tier: 14,400 req/day)

### Installation

```bash
# Clone repository
git clone <repository-url>
cd backend

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env

# Edit .env with your credentials
nano .env
```

### Zero-Config Development

Start the server immediately with in-memory database:

```bash
go run cmd/api/main.go
```

Server runs on `http://localhost:8080` with:
- In-memory database (no Supabase required)
- Authentication disabled
- All features enabled for testing
- Debug logging

### Production Setup

1. **Configure Supabase:**

```bash
# Get credentials from: https://supabase.com/dashboard/project/_/settings/api
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key
SUPABASE_JWT_SECRET=your-jwt-secret
SUPABASE_STORAGE_BUCKET=lumen-nutrition
```

2. **Set AI Provider Keys:**

```bash
# Groq (primary provider)
GROQ_API_KEY=your-groq-api-key

# OpenRouter (fallback)
OPENROUTER_API_KEY=your-openrouter-api-key
```

3. **Run Database Migrations:**

See [DEPLOYMENT.md](docs/DEPLOYMENT.md#database-migrations) for detailed instructions.

4. **Start Server:**

```bash
APP_MODE=production go run cmd/api/main.go
```

---

## API Endpoints

### Health Check

```bash
GET /health
# Returns server status and database connection
```

### Media Upload

```bash
POST   /api/v1/media/upload     # Upload and compress meal photos
DELETE /api/v1/media            # Delete photo
GET    /api/v1/media/url        # Get photo URL
```

### Voice Transcription

```bash
POST   /api/v1/voice/transcribe # Transcribe audio to text
GET    /api/v1/voice/formats    # Get supported audio formats
```

### Weight Trajectory

```bash
POST   /api/v1/trajectory/predict # Predict future weight based on trends
```

### Meal Suggestions

```bash
POST   /api/v1/suggestions/generate # Generate AI-powered meal suggestions
GET    /api/v1/suggestions/quick    # Get quick template suggestions
```

### Meals

```bash
POST   /api/v1/meals/parse           # Parse meal description with AI (creates draft)
GET    /api/v1/meals/draft/{id}/status # Check draft processing status
POST   /api/v1/meals/confirm         # Save parsed meal
GET    /api/v1/meals                 # List user's meals
GET    /api/v1/meals/{id}            # Get meal details
PUT    /api/v1/meals/{id}            # Update meal
DELETE /api/v1/meals/{id}            # Delete meal
```

### Weight Tracking

```bash
POST   /api/v1/weight           # Create weight entry
GET    /api/v1/weight           # List weight entries
GET    /api/v1/weight/latest    # Get latest weight
GET    /api/v1/weight/trend     # Get weight trend statistics
PUT    /api/v1/weight/{id}      # Update weight entry
DELETE /api/v1/weight/{id}      # Delete weight entry
```

### Nutrition Goals

```bash
GET    /api/v1/goals            # Get user goals
PUT    /api/v1/goals            # Update goals
POST   /api/v1/goals/calculate  # Calculate TDEE from user data
GET    /api/v1/goals/daily      # Get daily goals overview
PUT    /api/v1/goals/daily/{day}  # Set custom goals for specific day
DELETE /api/v1/goals/daily/{day}  # Remove custom day goals
```

### Analytics

```bash
GET /api/v1/analytics/daily        # Daily nutrition summary
GET /api/v1/analytics/weekly       # 7-day trends
GET /api/v1/analytics/trends       # Custom date range trends
GET /api/v1/analytics/distribution # Macro distribution
GET /api/v1/analytics/progress     # Goal progress tracking
```

### Templates

```bash
POST   /api/v1/templates            # Create template
POST   /api/v1/templates/from-meal  # Create from existing meal
GET    /api/v1/templates            # List templates
GET    /api/v1/templates/{id}       # Get template details
PUT    /api/v1/templates/{id}       # Update template
DELETE /api/v1/templates/{id}       # Delete template
POST   /api/v1/templates/{id}/use   # Use template to create meal
```

**Complete API Documentation:** [docs/API.md](docs/API.md)

---

## Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
│
├── internal/
│   ├── config/                  # Configuration management
│   ├── domain/
│   │   └── nutrition/
│   │       ├── meals/           # Meal domain
│   │       ├── weight/          # Weight tracking
│   │       ├── goals/           # Goals & TDEE
│   │       ├── analytics/       # Analytics & insights
│   │       └── templates/       # Meal templates
│   ├── errors/                  # Domain error types
│   ├── handlers/                # HTTP request handlers
│   │   ├── media_handlers.go   # Media upload handlers
│   │   ├── voice_handlers.go   # Voice transcription handlers
│   │   ├── trajectory_handlers.go # Weight prediction handlers
│   │   └── suggestions_handlers.go # Meal suggestion handlers
│   ├── server/
│   │   ├── middleware/          # HTTP middleware
│   │   ├── router.go            # Route definitions
│   │   └── dependencies.go      # Dependency injection
│   ├── services/
│   │   ├── ai/                  # AI providers
│   │   ├── cache/               # Caching service
│   │   ├── cost/                # Cost tracking
│   │   ├── storage/             # File storage
│   │   ├── media/               # Photo upload & compression
│   │   ├── voice/               # Audio transcription
│   │   ├── trajectory/          # Weight prediction
│   │   └── suggestions/         # Meal recommendations
│   ├── supabase/                # Supabase client
│   └── testutil/                # Test utilities
│
├── tests/
│   └── integration/             # Integration tests
│
├── migrations/                  # Database migrations
├── docs/                        # Documentation
│   ├── API.md                   # API reference
│   ├── ARCHITECTURE.md          # System architecture
│   ├── DEPLOYMENT.md            # Deployment guide
│   ├── DEVELOPMENT.md           # Development guide
│   └── TROUBLESHOOTING.md       # Troubleshooting guide
│
├── .env.example                 # Environment template
├── go.mod                       # Go module definition
└── README.md                    # This file
```

---

## Configuration

### Application Modes

Set via `APP_MODE` environment variable:

| Mode | Description | Use Case |
|------|-------------|----------|
| `development` | Debug logging, permissive CORS, in-memory DB | Local development |
| `testing` | Minimal logging, all features disabled | Unit/integration tests |
| `production` | JSON logging, strict CORS, Supabase DB | Production deployment |

### Feature Flags

Enable/disable features for graceful degradation:

```bash
FEATURE_AUTH=true           # Authentication & authorization
FEATURE_STORAGE=true        # File storage (photos)
FEATURE_AI_ANALYSIS=true    # AI meal parsing
FEATURE_CACHE=true          # Response caching
FEATURE_RATE_LIMIT=true     # Rate limiting
FEATURE_METRICS=true        # Metrics collection
```

**When disabled:**
- Auth: Bypassed (dev mode only)
- Storage: Returns mock URLs
- AI: Returns basic fallback
- Cache: All requests hit database
- Rate Limit: No limits applied
- Metrics: No-op operations

### Database Modes

```bash
DATABASE_MODE=memory      # In-memory (development)
DATABASE_MODE=supabase    # Supabase (production)
```

**Complete Configuration:** [docs/DEVELOPMENT.md#configuration](docs/DEVELOPMENT.md)

---

## Development

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./internal/domain/nutrition/meals/... -v

# Run specific test
go test ./internal/domain/nutrition/meals -run TestCreateMeal -v
```

### Hot Reload (Development)

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run

# Check for vulnerabilities
go vet ./...
```

### Adding a New Feature

See [docs/DEVELOPMENT.md#adding-a-new-feature](docs/DEVELOPMENT.md#adding-a-new-feature) for step-by-step guide.

---

## Deployment

### Quick Deploy (Fly.io)

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
flyctl auth login

# Deploy
flyctl launch
flyctl deploy
```

### Docker

```bash
# Build image
docker build -t lumen-backend .

# Run container
docker run -p 8080:8080 --env-file .env lumen-backend
```

### Environment Variables for Production

```bash
APP_MODE=production
DATABASE_MODE=supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key
GROQ_API_KEY=your-groq-key
CORS_ALLOWED_ORIGINS=https://yourdomain.com
FEATURE_AUTH=true
FEATURE_RATE_LIMIT=true
LOG_LEVEL=info
LOG_FORMAT=json
```

**Complete Deployment Guide:** [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

---

## Documentation

| Document | Description |
|----------|-------------|
| [API.md](docs/API.md) | Complete API reference with examples |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture and design |
| [DEPLOYMENT.md](docs/DEPLOYMENT.md) | Production deployment guide |
| [DEVELOPMENT.md](docs/DEVELOPMENT.md) | Development setup and guidelines |
| [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common issues and solutions |
| [CLAUDE.md](CLAUDE.md) | AI agent development rules |

---

## Key Features Explained

### AI Meal Parsing

Parse natural language meal descriptions into structured nutrition data:

```bash
POST /api/v1/meals/parse
{
  "description": "2 scrambled eggs with spinach and whole wheat toast",
  "meal_type": "breakfast",
  "idempotency_key": "unique-request-id"
}
```

**Response:**

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
    // ... more items
  ],
  "total_calories": 387,
  "total_protein_g": 25.9,
  "ai_confidence": "high",
  "provider": "groq"
}
```

### TDEE Calculation

Calculate Total Daily Energy Expenditure using Mifflin-St Jeor equation:

```bash
POST /api/v1/goals/calculate
{
  "age": 30,
  "sex": "male",
  "weight_kg": 75.5,
  "height_cm": 180,
  "activity_level": "active"
}
```

**Response:**

```json
{
  "bmr": 1833,
  "tdee": 2841,
  "recommended_goals": {
    "calories": 2841,
    "protein_g": 151,
    "carbs_g": 356,
    "fat_g": 79,
    "fiber_g": 30
  }
}
```

### Analytics & Insights

Track nutrition progress with daily, weekly, and monthly analytics:

```bash
GET /api/v1/analytics/daily?date=2025-11-15
```

**Response:**

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
    "protein_g": 165
  },
  "progress": {
    "calories_percent": 84,
    "protein_percent": 86
  },
  "remaining": {
    "calories": 353,
    "protein_g": 23
  }
}
```

---

## Rate Limits

### AI Parsing Limits

- **Groq Free Tier:** 14,400 requests/day per API key
- **Per User Default:** 50 parses/day (configurable)
- **Cost Limit:** $10/user/month, $1000 global/month

### API Limits

- **Requests:** 1000 requests/hour per user
- **Burst:** 100 requests/minute

Configure via environment variables:

```bash
RATE_LIMIT_AI_PARSE_PER_DAY=50
RATE_LIMIT_API_CALLS_PER_HOUR=1000
RATE_LIMIT_COST_PER_USER_PER_MONTH=10.0
```

---

## Security

### Authentication

- **JWT-based authentication** via Supabase Auth
- **Row-level security (RLS)** on all database tables
- **Token validation** on protected endpoints

### Data Protection

- **User data isolation** - Users can only access their own data
- **Secure file storage** - Private Supabase storage bucket
- **Input validation** - All requests validated before processing
- **Rate limiting** - Prevent abuse and excessive costs

### Best Practices

- Never commit `.env` file (already in `.gitignore`)
- Use environment variables for all secrets
- Enable HTTPS in production
- Set strict CORS origins (no wildcards)
- Enable all security features in production:

```bash
FEATURE_AUTH=true
FEATURE_RATE_LIMIT=true
```

---

## Performance

### Optimization Strategies

1. **Database Indexes** - On frequently queried columns
2. **Connection Pooling** - Reuse database connections
3. **Response Caching** - Cache analytics and expensive queries
4. **Query Optimization** - Paginated queries, select only needed columns
5. **Idempotency** - Prevent duplicate AI parsing requests

### Benchmarks

- API response time: <100ms (cached), <500ms (uncached)
- AI parsing: 1-3 seconds
- Database queries: <50ms
- Memory usage: ~50MB (baseline), ~200MB (with cache)

---

## Troubleshooting

### Common Issues

**Database Connection Failed:**
```bash
# Verify Supabase credentials
echo $SUPABASE_URL
echo $SUPABASE_SERVICE_KEY

# Test with in-memory database
DATABASE_MODE=memory go run cmd/api/main.go
```

**AI Parsing Failed:**
```bash
# Enable debug logging
LOG_LEVEL=debug go run cmd/api/main.go

# Try different AI model
GROQ_MODEL=llama-3.3-70b-versatile
```

**Rate Limit Exceeded:**
```bash
# Increase limit for development
RATE_LIMIT_AI_PARSE_PER_DAY=100

# Or disable rate limiting
FEATURE_RATE_LIMIT=false
```

**Complete Troubleshooting Guide:** [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

---

## Contributing

### Development Workflow

1. Create feature branch: `git checkout -b feature/your-feature`
2. Make changes following code style in [CLAUDE.md](CLAUDE.md)
3. Write tests for new code
4. Run tests: `go test ./...`
5. Format code: `go fmt ./...`
6. Commit with descriptive message
7. Push and create pull request

### Code Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use [Effective Go](https://golang.org/doc/effective_go) guidelines
- See [CLAUDE.md](CLAUDE.md) for project-specific conventions

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Documentation:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/your-org/lumen-backend/issues)
- **Deployment Help:** [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- **Development Help:** [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

---

## Acknowledgments

- Built with [Go](https://golang.org/)
- Powered by [Supabase](https://supabase.com/)
- AI by [Groq](https://groq.com/) and [OpenRouter](https://openrouter.ai/)
- Web framework: [chi](https://github.com/go-chi/chi)

---

**Made with ❤️ for the nutrition tracking community**
