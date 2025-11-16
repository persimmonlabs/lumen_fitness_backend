// Package config provides environment-based configuration management for the fitness app.
//
// The configuration system follows a three-tier priority:
//  1. Environment variables (highest priority)
//  2. .env file values
//  3. Sensible defaults (lowest priority)
//
// This allows for zero-config development while supporting production deployments.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// AppMode represents the application running mode
type AppMode string

const (
	ModeDevelopment AppMode = "development"
	ModeTesting     AppMode = "testing"
	ModeProduction  AppMode = "production"
)

// DatabaseMode represents the database backend mode
type DatabaseMode string

const (
	DatabaseSupabase DatabaseMode = "supabase" // Use Supabase PostgreSQL
	DatabaseMemory   DatabaseMode = "memory"   // Use in-memory SQLite
)

// Config holds all application configuration
type Config struct {
	// Application mode (development/testing/production)
	Mode AppMode

	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Supabase configuration (optional)
	Supabase SupabaseConfig

	// AI service configuration
	AI AIConfig

	// Cache configuration
	Cache CacheConfig

	// Rate limiting configuration
	RateLimit RateLimitConfig

	// Storage configuration
	Storage StorageConfig

	// CORS configuration
	CORS CORSConfig

	// Feature flags
	Features FeatureFlags

	// Logging configuration
	Logging LoggingConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Mode            DatabaseMode
	ConnectionString string // For PostgreSQL
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// SupabaseConfig holds Supabase-specific settings
type SupabaseConfig struct {
	URL            string
	AnonKey        string
	ServiceKey     string
	JWTSecret      string
	StorageBucket  string
}

// CacheMode represents the cache backend mode
type CacheMode string

const (
	CacheMemory   CacheMode = "memory" // In-memory cache
	CacheRedis    CacheMode = "redis"  // Redis cache
	CacheDisabled CacheMode = "disabled"
)

// AIConfig holds AI service configuration
type AIConfig struct {
	// Groq configuration
	GroqAPIKey       string
	GroqModel        string
	GroqMaxRequests  int // Max requests per day

	// OpenRouter configuration
	OpenRouterAPIKey      string
	OpenRouterNutritionPreset string
	OpenRouterVisionPreset    string
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Mode     CacheMode
	RedisURL string
	RedisPassword string
	RedisDB  int
	DefaultTTL time.Duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	AIParsePerDay        int // AI parsing requests per user per day
	APICallsPerHour      int // API calls per user per hour
	CostPerUserPerMonth  float64 // Cost limit per user per month (USD)
	GlobalCostPerMonth   float64 // Global cost limit per month (USD)
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	BucketName          string
	MaxPhotoSizeMB      int
	AllowedPhotoFormats []string
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// FeatureFlags control optional features
type FeatureFlags struct {
	EnableAuth         bool // Authentication/authorization
	EnableStorage      bool // File storage
	EnableAIAnalysis   bool // AI workout analysis
	EnableNotifications bool // Push notifications/emails
	EnableRealtime     bool // WebSocket updates
	EnableMetrics      bool // Metrics collection
	EnableRateLimit    bool // API rate limiting
	EnableCache        bool // Response caching
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, console
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Mode: AppMode(getEnv("APP_MODE", string(ModeDevelopment))),

		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},

		Database: DatabaseConfig{
			Mode:            DatabaseMode(getEnv("DATABASE_MODE", string(DatabaseMemory))),
			ConnectionString: getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvAsInt("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		},

		Supabase: SupabaseConfig{
			URL:           getEnv("SUPABASE_URL", ""),
			AnonKey:       getEnv("SUPABASE_ANON_KEY", ""),
			ServiceKey:    getEnv("SUPABASE_SERVICE_KEY", ""),
			JWTSecret:     getEnv("SUPABASE_JWT_SECRET", ""),
			StorageBucket: getEnv("SUPABASE_STORAGE_BUCKET", "lumen-nutrition"),
		},

		AI: AIConfig{
			GroqAPIKey:                getEnv("GROQ_API_KEY", ""),
			GroqModel:                 getEnv("GROQ_MODEL", "llama-3.1-70b-versatile"),
			GroqMaxRequests:           getEnvAsInt("GROQ_MAX_REQUESTS_PER_DAY", 100),
			OpenRouterAPIKey:          getEnv("OPENROUTER_API_KEY", ""),
			OpenRouterNutritionPreset: getEnv("OPENROUTER_NUTRITION_PRESET", "anthropic/claude-3.5-sonnet"),
			OpenRouterVisionPreset:    getEnv("OPENROUTER_VISION_PRESET", "anthropic/claude-3.5-sonnet"),
		},

		Cache: CacheConfig{
			Mode:          CacheMode(getEnv("CACHE_MODE", string(CacheMemory))),
			RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:       getEnvAsInt("REDIS_DB", 0),
			DefaultTTL:    getEnvAsDuration("CACHE_DEFAULT_TTL", 5*time.Minute),
		},

		RateLimit: RateLimitConfig{
			AIParsePerDay:       getEnvAsInt("RATE_LIMIT_AI_PARSE_PER_DAY", 50),
			APICallsPerHour:     getEnvAsInt("RATE_LIMIT_API_CALLS_PER_HOUR", 1000),
			CostPerUserPerMonth: getEnvAsFloat("RATE_LIMIT_COST_PER_USER_PER_MONTH", 10.0),
			GlobalCostPerMonth:  getEnvAsFloat("RATE_LIMIT_GLOBAL_COST_PER_MONTH", 1000.0),
		},

		Storage: StorageConfig{
			BucketName:          getEnv("STORAGE_BUCKET_NAME", "lumen-nutrition"),
			MaxPhotoSizeMB:      getEnvAsInt("STORAGE_MAX_PHOTO_SIZE_MB", 10),
			AllowedPhotoFormats: getEnvAsSlice("STORAGE_ALLOWED_PHOTO_FORMATS", []string{"jpg", "jpeg", "png", "heic"}),
		},

		CORS: CORSConfig{
			AllowedOrigins:   getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
			AllowedMethods:   getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 3600),
		},

		Features: FeatureFlags{
			EnableAuth:         getEnvAsBool("FEATURE_AUTH", false),
			EnableStorage:      getEnvAsBool("FEATURE_STORAGE", false),
			EnableAIAnalysis:   getEnvAsBool("FEATURE_AI_ANALYSIS", false),
			EnableNotifications: getEnvAsBool("FEATURE_NOTIFICATIONS", false),
			EnableRealtime:     getEnvAsBool("FEATURE_REALTIME", false),
			EnableMetrics:      getEnvAsBool("FEATURE_METRICS", false),
			EnableRateLimit:    getEnvAsBool("FEATURE_RATE_LIMIT", false),
			EnableCache:        getEnvAsBool("FEATURE_CACHE", false),
		},

		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "console"),
		},
	}

	// Mode-specific defaults
	if cfg.IsDevelopment() {
		cfg.Logging.Level = getEnv("LOG_LEVEL", "debug")
		cfg.Logging.Format = getEnv("LOG_FORMAT", "console")
	} else if cfg.IsProduction() {
		cfg.Logging.Format = getEnv("LOG_FORMAT", "json")
	} else if cfg.IsTesting() {
		cfg.Logging.Level = getEnv("LOG_LEVEL", "error")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate port
	port, err := strconv.Atoi(c.Server.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %s (must be between 1-65535)", c.Server.Port)
	}

	// Validate mode
	switch c.Mode {
	case ModeDevelopment, ModeTesting, ModeProduction:
		// Valid modes
	default:
		return fmt.Errorf("invalid mode: %s (must be development, testing, or production)", c.Mode)
	}

	// Validate database mode
	switch c.Database.Mode {
	case DatabaseSupabase, DatabaseMemory:
		// Valid modes
	default:
		return fmt.Errorf("invalid database mode: %s (must be supabase or memory)", c.Database.Mode)
	}

	// Production-specific validations
	if c.IsProduction() {
		if c.Database.Mode == DatabaseSupabase && c.Database.ConnectionString == "" {
			return fmt.Errorf("DATABASE_URL is required in production mode")
		}

		if c.Features.EnableAuth && c.Supabase.JWTSecret == "" {
			return fmt.Errorf("SUPABASE_JWT_SECRET is required when authentication is enabled in production")
		}

		// Ensure no wildcard CORS origins in production
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("wildcard CORS origin not allowed in production")
			}
		}
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	// Validate log format
	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be json or console)", c.Logging.Format)
	}

	// Validate cache mode
	switch c.Cache.Mode {
	case CacheMemory, CacheRedis, CacheDisabled:
		// Valid cache modes
	default:
		return fmt.Errorf("invalid cache mode: %s (must be memory, redis, or disabled)", c.Cache.Mode)
	}

	// Validate AI configuration when AI features are enabled
	if c.Features.EnableAIAnalysis {
		if c.AI.GroqAPIKey == "" && c.AI.OpenRouterAPIKey == "" && c.IsProduction() {
			return fmt.Errorf("at least one AI API key (GROQ_API_KEY or OPENROUTER_API_KEY) is required when AI analysis is enabled in production")
		}
	}

	// Validate storage configuration when storage is enabled
	if c.Features.EnableStorage {
		if c.Storage.MaxPhotoSizeMB <= 0 || c.Storage.MaxPhotoSizeMB > 100 {
			return fmt.Errorf("invalid max photo size: %d (must be between 1-100 MB)", c.Storage.MaxPhotoSizeMB)
		}
		if len(c.Storage.AllowedPhotoFormats) == 0 {
			return fmt.Errorf("at least one photo format must be allowed")
		}
	}

	// Validate rate limit configuration when rate limiting is enabled
	if c.Features.EnableRateLimit {
		if c.RateLimit.AIParsePerDay <= 0 {
			return fmt.Errorf("invalid AI parse rate limit: %d (must be > 0)", c.RateLimit.AIParsePerDay)
		}
		if c.RateLimit.APICallsPerHour <= 0 {
			return fmt.Errorf("invalid API calls rate limit: %d (must be > 0)", c.RateLimit.APICallsPerHour)
		}
		if c.RateLimit.CostPerUserPerMonth < 0 {
			return fmt.Errorf("invalid cost per user per month: %.2f (must be >= 0)", c.RateLimit.CostPerUserPerMonth)
		}
		if c.RateLimit.GlobalCostPerMonth < 0 {
			return fmt.Errorf("invalid global cost per month: %.2f (must be >= 0)", c.RateLimit.GlobalCostPerMonth)
		}
	}

	return nil
}

// Helper methods

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Mode == ModeDevelopment
}

// IsTesting returns true if running in testing mode
func (c *Config) IsTesting() bool {
	return c.Mode == ModeTesting
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Mode == ModeProduction
}

// GetAddress returns the server address (host:port)
func (c *Config) GetAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}

// Environment variable helpers

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	// Support multiple boolean formats
	valueStr = strings.ToLower(strings.TrimSpace(valueStr))
	switch valueStr {
	case "true", "yes", "1", "on":
		return true
	case "false", "no", "0", "off":
		return false
	default:
		return defaultValue
	}
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	// Split by comma and trim whitespace
	parts := strings.Split(valueStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
