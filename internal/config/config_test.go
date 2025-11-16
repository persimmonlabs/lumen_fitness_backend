package config

import (
	"os"
	"testing"
	"time"
)

// TestDefaultConfiguration tests that default configuration loads without errors
func TestDefaultConfiguration(t *testing.T) {
	// Clear all environment variables
	clearTestEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load default configuration: %v", err)
	}

	// Verify default values
	if cfg.Mode != ModeDevelopment {
		t.Errorf("Expected development mode, got %s", cfg.Mode)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", cfg.Server.Port)
	}

	if cfg.Database.Mode != DatabaseMemory {
		t.Errorf("Expected memory database mode, got %s", cfg.Database.Mode)
	}

	if cfg.Features.EnableAuth {
		t.Error("Expected auth to be disabled in development mode")
	}
}

// TestEnvironmentVariableOverride tests environment variable priority
func TestEnvironmentVariableOverride(t *testing.T) {
	clearTestEnv()

	// Set environment variables
	os.Setenv("PORT", "3000")
	os.Setenv("APP_MODE", "production")
	os.Setenv("DATABASE_MODE", "supabase")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("APP_MODE")
		os.Unsetenv("DATABASE_MODE")
	}()

	// Load configuration (will fail validation without Supabase credentials)
	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for missing Supabase credentials")
	}
}

// TestProductionModeValidation tests production mode requirements
func TestProductionModeValidation(t *testing.T) {
	clearTestEnv()

	os.Setenv("APP_MODE", "production")
	os.Setenv("DATABASE_MODE", "supabase")
	os.Setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
	os.Setenv("SUPABASE_URL", "https://test.supabase.co")
	os.Setenv("SUPABASE_SERVICE_KEY", "test-key")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	defer clearTestEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load production configuration: %v", err)
	}

	if !cfg.IsProduction() {
		t.Error("Expected production mode")
	}

	if cfg.Database.Mode != DatabaseSupabase {
		t.Error("Expected Supabase database mode in production")
	}

	if cfg.Logging.Format != "json" {
		t.Error("Expected JSON log format in production mode")
	}
}

// TestDevelopmentModeDefaults tests development mode defaults
func TestDevelopmentModeDefaults(t *testing.T) {
	clearTestEnv()

	os.Setenv("APP_MODE", "development")
	defer os.Unsetenv("APP_MODE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load development configuration: %v", err)
	}

	if !cfg.IsDevelopment() {
		t.Error("Expected development mode")
	}

	if cfg.Database.Mode != DatabaseMemory {
		t.Error("Expected memory database mode in development")
	}

	if cfg.Features.EnableAuth {
		t.Error("Expected auth to be disabled in development")
	}

	if cfg.Logging.Level != "debug" {
		t.Error("Expected debug log level in development")
	}

	if cfg.Logging.Format != "console" {
		t.Error("Expected console log format in development")
	}
}

// TestTestingModeDefaults tests testing mode defaults
func TestTestingModeDefaults(t *testing.T) {
	clearTestEnv()

	os.Setenv("APP_MODE", "testing")
	defer os.Unsetenv("APP_MODE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load testing configuration: %v", err)
	}

	if !cfg.IsTesting() {
		t.Error("Expected testing mode")
	}

	if cfg.Database.Mode != DatabaseMemory {
		t.Error("Expected memory database mode in testing")
	}

	if cfg.Features.EnableAuth {
		t.Error("Expected auth to be disabled in testing")
	}
}

// TestFeatureFlags tests feature flag parsing
func TestFeatureFlags(t *testing.T) {
	clearTestEnv()

	testCases := []struct {
		name     string
		envVar   string
		envValue string
		expected bool
	}{
		{"true value", "FEATURE_AUTH", "true", true},
		{"false value", "FEATURE_AUTH", "false", false},
		{"yes value", "FEATURE_STORAGE", "yes", true},
		{"no value", "FEATURE_STORAGE", "no", false},
		{"1 value", "FEATURE_CACHE", "1", true},
		{"0 value", "FEATURE_CACHE", "0", false},
		{"on value", "FEATURE_METRICS", "on", true},
		{"off value", "FEATURE_METRICS", "off", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearTestEnv()
			os.Setenv(tc.envVar, tc.envValue)
			defer os.Unsetenv(tc.envVar)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Failed to load configuration: %v", err)
			}

			var actual bool
			switch tc.envVar {
			case "FEATURE_AUTH":
				actual = cfg.Features.EnableAuth
			case "FEATURE_STORAGE":
				actual = cfg.Features.EnableStorage
			case "FEATURE_CACHE":
				actual = cfg.Features.EnableCache
			case "FEATURE_METRICS":
				actual = cfg.Features.EnableMetrics
			}

			if actual != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

// TestInvalidPortValidation tests port validation
func TestInvalidPortValidation(t *testing.T) {
	testCases := []struct {
		name string
		port string
	}{
		{"negative port", "-1"},
		{"zero port", "0"},
		{"port too large", "99999"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearTestEnv()
			os.Setenv("PORT", tc.port)
			defer os.Unsetenv("PORT")

			_, err := Load()
			if err == nil {
				t.Error("Expected validation error for invalid port")
			}
		})
	}
}

// TestInvalidDatabaseModeValidation tests database mode validation
func TestInvalidDatabaseModeValidation(t *testing.T) {
	clearTestEnv()

	os.Setenv("DATABASE_MODE", "invalid")
	defer os.Unsetenv("DATABASE_MODE")

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for invalid database mode")
	}
}

// TestSupabaseValidation tests Supabase configuration validation
func TestSupabaseValidation(t *testing.T) {
	clearTestEnv()

	// Missing DATABASE_URL in production with supabase mode
	t.Run("missing DATABASE_URL in production", func(t *testing.T) {
		clearTestEnv()
		os.Setenv("APP_MODE", "production")
		os.Setenv("DATABASE_MODE", "supabase")
		os.Setenv("SUPABASE_URL", "https://test.supabase.co")
		os.Setenv("SUPABASE_SERVICE_KEY", "test-key")
		defer clearTestEnv()

		_, err := Load()
		if err == nil {
			t.Error("Expected validation error for missing DATABASE_URL in production")
		}
	})

	// Valid Supabase config in development (no DATABASE_URL required)
	t.Run("valid config in development", func(t *testing.T) {
		clearTestEnv()
		os.Setenv("DATABASE_MODE", "supabase")
		os.Setenv("SUPABASE_URL", "https://test.supabase.co")
		os.Setenv("SUPABASE_SERVICE_KEY", "test-key")
		defer clearTestEnv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected valid configuration: %v", err)
		}

		if cfg.Supabase.URL == "" {
			t.Error("Expected Supabase URL to be set")
		}
	})
}

// TestCORSValidation tests CORS configuration validation
func TestCORSValidation(t *testing.T) {
	// Production mode should reject wildcard CORS
	t.Run("production wildcard rejection", func(t *testing.T) {
		clearTestEnv()
		os.Setenv("APP_MODE", "production")
		os.Setenv("DATABASE_MODE", "memory")
		os.Setenv("CORS_ALLOWED_ORIGINS", "*")
		defer clearTestEnv()

		_, err := Load()
		if err == nil {
			t.Error("Expected validation error for wildcard CORS in production")
		}
	})

	// Production mode should require specific origins
	t.Run("production specific origins", func(t *testing.T) {
		clearTestEnv()
		os.Setenv("APP_MODE", "production")
		os.Setenv("DATABASE_MODE", "memory")
		os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
		defer clearTestEnv()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected valid configuration: %v", err)
		}

		if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "https://example.com" {
			t.Error("Expected specific CORS origin in production")
		}
	})
}

// TestLogLevelValidation tests log level validation
func TestLogLevelValidation(t *testing.T) {
	clearTestEnv()

	os.Setenv("LOG_LEVEL", "invalid")
	defer os.Unsetenv("LOG_LEVEL")

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for invalid log level")
	}
}

// TestLogFormatValidation tests log format validation
func TestLogFormatValidation(t *testing.T) {
	clearTestEnv()

	os.Setenv("LOG_FORMAT", "invalid")
	defer os.Unsetenv("LOG_FORMAT")

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for invalid log format")
	}
}

// TestGetEnvAsInt tests integer environment variable parsing
func TestGetEnvAsInt(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{"valid integer", "42", 10, 42},
		{"empty string", "", 10, 10},
		{"invalid integer", "abc", 10, 10},
		{"negative integer", "-5", 10, -5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_INT", tc.value)
				defer os.Unsetenv("TEST_INT")
			}

			result := getEnvAsInt("TEST_INT", tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// TestGetEnvAsBool tests boolean environment variable parsing
func TestGetEnvAsBool(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue bool
		expected     bool
	}{
		{"true", "true", false, true},
		{"false", "false", true, false},
		{"yes", "yes", false, true},
		{"no", "no", true, false},
		{"1", "1", false, true},
		{"0", "0", true, false},
		{"on", "on", false, true},
		{"off", "off", true, false},
		{"empty", "", true, true},
		{"invalid", "invalid", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_BOOL", tc.value)
				defer os.Unsetenv("TEST_BOOL")
			}

			result := getEnvAsBool("TEST_BOOL", tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestGetEnvAsDuration tests duration environment variable parsing
func TestGetEnvAsDuration(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"seconds", "30s", 10 * time.Second, 30 * time.Second},
		{"minutes", "2m", 10 * time.Second, 2 * time.Minute},
		{"hours", "1h", 10 * time.Second, 1 * time.Hour},
		{"combined", "1h30m", 10 * time.Second, 90 * time.Minute},
		{"empty", "", 10 * time.Second, 10 * time.Second},
		{"invalid", "invalid", 10 * time.Second, 10 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_DURATION", tc.value)
				defer os.Unsetenv("TEST_DURATION")
			}

			result := getEnvAsDuration("TEST_DURATION", tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestGetEnvAsSlice tests slice environment variable parsing
func TestGetEnvAsSlice(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		defaultValue []string
		expected     []string
	}{
		{"single value", "value1", []string{"default"}, []string{"value1"}},
		{"multiple values", "value1,value2,value3", []string{"default"}, []string{"value1", "value2", "value3"}},
		{"with spaces", "value1, value2, value3", []string{"default"}, []string{"value1", "value2", "value3"}},
		{"empty", "", []string{"default"}, []string{"default"}},
		{"empty values", ",,", []string{"default"}, []string{"default"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv("TEST_SLICE", tc.value)
				defer os.Unsetenv("TEST_SLICE")
			}

			result := getEnvAsSlice("TEST_SLICE", tc.defaultValue)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(result))
				return
			}

			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("At index %d: expected %s, got %s", i, tc.expected[i], result[i])
				}
			}
		})
	}
}

// TestGetAddress tests the GetAddress helper method
func TestGetAddress(t *testing.T) {
	clearTestEnv()

	os.Setenv("HOST", "localhost")
	os.Setenv("PORT", "3000")
	defer clearTestEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	expected := "localhost:3000"
	if cfg.GetAddress() != expected {
		t.Errorf("Expected address %s, got %s", expected, cfg.GetAddress())
	}
}

// Helper function to clear test environment variables
func clearTestEnv() {
	// Clear all config-related environment variables
	envVars := []string{
		"APP_MODE",
		"PORT",
		"HOST",
		"READ_TIMEOUT",
		"WRITE_TIMEOUT",
		"IDLE_TIMEOUT",
		"SHUTDOWN_TIMEOUT",
		"DATABASE_MODE",
		"DATABASE_URL",
		"DB_MAX_OPEN_CONNS",
		"DB_MAX_IDLE_CONNS",
		"DB_CONN_MAX_LIFETIME",
		"SUPABASE_URL",
		"SUPABASE_ANON_KEY",
		"SUPABASE_SERVICE_KEY",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS",
		"CORS_ALLOW_CREDENTIALS",
		"CORS_MAX_AGE",
		"FEATURE_AUTH",
		"FEATURE_STORAGE",
		"FEATURE_AI_ANALYSIS",
		"FEATURE_NOTIFICATIONS",
		"FEATURE_REALTIME",
		"FEATURE_METRICS",
		"FEATURE_RATE_LIMIT",
		"FEATURE_CACHE",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"LOG_ENABLE_CALLER",
		"LOG_ENABLE_STACKTRACE",
		"LOG_OUTPUT_PATH",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// BenchmarkLoad benchmarks configuration loading
func BenchmarkLoad(b *testing.B) {
	clearTestEnv()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load()
		if err != nil {
			b.Fatalf("Failed to load configuration: %v", err)
		}
	}
}
