package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pradord/lumen_final/backend/internal/config"
	"github.com/pradord/lumen_final/backend/internal/server"
	"github.com/pradord/lumen_final/backend/internal/supabase"
	"github.com/pradord/lumen_final/backend/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// TestDB holds the test database connection
type TestDB struct {
	DB *gorm.DB
}

// TestServer holds the test server instance
type TestServer struct {
	Server     *server.Server
	Config     *config.Config
	Logger     *logger.Logger
	Supabase   *supabase.Client
	DB         *TestDB
	HTTPServer *httptest.Server
}

// TestUser represents a test user
type TestUser struct {
	ID       uuid.UUID
	Email    string
	Token    string
	Password string
}

// Global test server instance
var testServer *TestServer

// TestMain sets up and tears down test infrastructure
func TestMain(m *testing.M) {
	var err error

	// Setup test environment
	testServer, err = setupTestServer()
	if err != nil {
		log.Fatalf("Failed to setup test server: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := testServer.Cleanup(); err != nil {
		log.Printf("Failed to cleanup test server: %v", err)
	}

	os.Exit(code)
}

// setupTestServer creates and configures a test server instance
func setupTestServer() (*TestServer, error) {
	// Set test environment
	os.Setenv("APP_MODE", "testing")
	os.Setenv("DATABASE_MODE", "memory")
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("PORT", "0") // Random port

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Pretty: false,
	})

	// Setup test database
	testDB, err := setupTestDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	// Initialize Supabase client (fake for testing)
	supabaseClient := supabase.NewFakeClient()

	// Create server
	srv := server.NewServer(cfg, log.WithComponent("server"), supabaseClient)

	// Create HTTP test server
	httpServer := httptest.NewServer(srv.Router)

	return &TestServer{
		Server:     srv,
		Config:     cfg,
		Logger:     log,
		Supabase:   supabaseClient,
		DB:         testDB,
		HTTPServer: httpServer,
	}, nil
}

// setupTestDatabase creates an in-memory SQLite database for testing
func setupTestDatabase() (*TestDB, error) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &TestDB{DB: db}, nil
}

// runMigrations creates test database schema
func runMigrations(db *gorm.DB) error {
	// Create tables for nutrition tracking
	migrations := []string{
		// Users table (simplified for testing)
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Meals table
		`CREATE TABLE IF NOT EXISTS meals (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			meal_type TEXT NOT NULL,
			consumed_at TIMESTAMP NOT NULL,
			photos TEXT,
			notes TEXT,
			total_calories REAL DEFAULT 0,
			total_protein_g REAL DEFAULT 0,
			total_carbs_g REAL DEFAULT 0,
			total_fat_g REAL DEFAULT 0,
			total_fiber_g REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// Meal items table
		`CREATE TABLE IF NOT EXISTS meal_items (
			id TEXT PRIMARY KEY,
			meal_id TEXT NOT NULL,
			name TEXT NOT NULL,
			quantity REAL NOT NULL,
			unit TEXT NOT NULL,
			calories REAL DEFAULT 0,
			protein_g REAL DEFAULT 0,
			carbs_g REAL DEFAULT 0,
			fat_g REAL DEFAULT 0,
			fiber_g REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (meal_id) REFERENCES meals(id) ON DELETE CASCADE
		)`,
		// Weight entries table
		`CREATE TABLE IF NOT EXISTS weight_entries (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			weight REAL NOT NULL,
			measured_at TIMESTAMP NOT NULL,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(user_id, measured_at)
		)`,
		// Goals table
		`CREATE TABLE IF NOT EXISTS user_goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			daily_calories INTEGER NOT NULL,
			protein_grams INTEGER NOT NULL,
			carbs_grams INTEGER NOT NULL,
			fat_grams INTEGER NOT NULL,
			auto_calculate BOOLEAN DEFAULT false,
			activity_level TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_calculated_at TIMESTAMP
		)`,
		// Daily goals table
		`CREATE TABLE IF NOT EXISTS daily_goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			day_of_week TEXT NOT NULL,
			daily_calories INTEGER NOT NULL,
			protein_grams INTEGER NOT NULL,
			carbs_grams INTEGER NOT NULL,
			fat_grams INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, day_of_week)
		)`,
		// Templates table
		`CREATE TABLE IF NOT EXISTS templates (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			photo_url TEXT,
			total_calories REAL DEFAULT 0,
			total_protein REAL DEFAULT 0,
			total_carbs REAL DEFAULT 0,
			total_fat REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		// Template items table
		`CREATE TABLE IF NOT EXISTS template_items (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL,
			food_id TEXT NOT NULL,
			food_name TEXT NOT NULL,
			serving_size REAL NOT NULL,
			serving_unit TEXT NOT NULL,
			calories REAL DEFAULT 0,
			protein REAL DEFAULT 0,
			carbs REAL DEFAULT 0,
			fat REAL DEFAULT 0,
			food_photo_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE
		)`,
		// AI cache table for idempotency
		`CREATE TABLE IF NOT EXISTS ai_parse_cache (
			idempotency_key TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			request_data TEXT NOT NULL,
			response_data TEXT NOT NULL,
			cost_usd REAL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		)`,
		// Cost tracking table
		`CREATE TABLE IF NOT EXISTS ai_cost_tracking (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			cost_usd REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migrations {
		if err := db.Exec(migration).Error; err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}

// Cleanup closes database connections and cleans up resources
func (ts *TestServer) Cleanup() error {
	if ts.HTTPServer != nil {
		ts.HTTPServer.Close()
	}

	if ts.DB != nil && ts.DB.DB != nil {
		sqlDB, err := ts.DB.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	if ts.Supabase != nil {
		ts.Supabase.Close()
	}

	return nil
}

// CreateTestUser creates a test user in the database
func (ts *TestServer) CreateTestUser(t *testing.T) *TestUser {
	t.Helper()

	user := &TestUser{
		ID:       uuid.New(),
		Email:    fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8]),
		Password: "TestPassword123!",
		Token:    "test-token-" + uuid.New().String(),
	}

	// Insert user into database
	err := ts.DB.DB.Exec(
		"INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)",
		user.ID.String(),
		user.Email,
		time.Now(),
		time.Now(),
	).Error

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CleanDatabase removes all data from test tables
func (ts *TestServer) CleanDatabase(t *testing.T) {
	t.Helper()

	tables := []string{
		"meal_items",
		"meals",
		"weight_entries",
		"daily_goals",
		"user_goals",
		"template_items",
		"templates",
		"ai_parse_cache",
		"ai_cost_tracking",
		"users",
	}

	for _, table := range tables {
		if err := ts.DB.DB.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			t.Logf("Warning: Failed to clean table %s: %v", table, err)
		}
	}
}

// MakeRequest sends an HTTP request to the test server
func (ts *TestServer) MakeRequest(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, ts.HTTPServer.URL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return http.DefaultClient.Do(req)
}

// MakeAuthenticatedRequest sends an authenticated HTTP request
func (ts *TestServer) MakeAuthenticatedRequest(method, path string, body interface{}, user *TestUser) (*http.Response, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + user.Token,
	}
	return ts.MakeRequest(method, path, body, headers)
}

// AssertResponse checks response status and decodes JSON body
func AssertResponse(t *testing.T, resp *http.Response, expectedStatus int, result interface{}) {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s", expectedStatus, resp.StatusCode, string(body))
	}

	if result != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
}

// SeedMeals creates test meal data for a user
func (ts *TestServer) SeedMeals(t *testing.T, user *TestUser, count int, startDate time.Time) []uuid.UUID {
	t.Helper()

	mealIDs := make([]uuid.UUID, count)
	mealTypes := []string{"breakfast", "lunch", "dinner", "snack"}

	for i := 0; i < count; i++ {
		mealID := uuid.New()
		mealIDs[i] = mealID

		consumedAt := startDate.Add(time.Duration(i) * 24 * time.Hour)
		mealType := mealTypes[i%len(mealTypes)]

		// Insert meal
		err := ts.DB.DB.Exec(`
			INSERT INTO meals (id, user_id, meal_type, consumed_at, total_calories, total_protein_g, total_carbs_g, total_fat_g, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			mealID.String(),
			user.ID.String(),
			mealType,
			consumedAt,
			500.0+float64(i*50),
			30.0+float64(i*2),
			50.0+float64(i*3),
			15.0+float64(i),
			time.Now(),
			time.Now(),
		).Error

		if err != nil {
			t.Fatalf("Failed to seed meal: %v", err)
		}

		// Insert meal items
		for j := 0; j < 2; j++ {
			itemID := uuid.New()
			err := ts.DB.DB.Exec(`
				INSERT INTO meal_items (id, meal_id, name, quantity, unit, calories, protein_g, carbs_g, fat_g, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				itemID.String(),
				mealID.String(),
				fmt.Sprintf("Food Item %d-%d", i, j),
				100.0,
				"g",
				250.0,
				15.0,
				25.0,
				7.5,
				time.Now(),
			).Error

			if err != nil {
				t.Fatalf("Failed to seed meal item: %v", err)
			}
		}
	}

	return mealIDs
}

// SeedWeightEntries creates test weight data for a user
func (ts *TestServer) SeedWeightEntries(t *testing.T, user *TestUser, count int, startDate time.Time, startWeight float64) []uuid.UUID {
	t.Helper()

	entryIDs := make([]uuid.UUID, count)

	for i := 0; i < count; i++ {
		entryID := uuid.New()
		entryIDs[i] = entryID

		measuredAt := startDate.Add(time.Duration(i) * 24 * time.Hour)
		weight := startWeight - float64(i)*0.1 // Gradual weight loss

		err := ts.DB.DB.Exec(`
			INSERT INTO weight_entries (id, user_id, weight, measured_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			entryID.String(),
			user.ID.String(),
			weight,
			measuredAt,
			time.Now(),
			time.Now(),
		).Error

		if err != nil {
			t.Fatalf("Failed to seed weight entry: %v", err)
		}
	}

	return entryIDs
}

// NewTestContext creates a context for testing with timeout
func NewTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// LoggerForTest creates a test logger
func LoggerForTest() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
