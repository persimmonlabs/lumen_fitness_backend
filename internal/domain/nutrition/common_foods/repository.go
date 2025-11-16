package common_foods

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Repository defines the interface for common foods data access.
type Repository interface {
	Search(ctx context.Context, query string, limit int) ([]CommonFood, error)
	LoadFromFile(filepath string) error
}

type repository struct {
	foods []CommonFood
	mu    sync.RWMutex
}

// NewRepository creates a new in-memory common foods repository.
func NewRepository() Repository {
	return &repository{
		foods: make([]CommonFood, 0),
	}
}

// LoadFromFile loads common foods data from a JSON file.
func (r *repository) LoadFromFile(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Parse JSON
	var foodsData struct {
		Foods []CommonFood `json:"foods"`
	}
	if err := json.Unmarshal(data, &foodsData); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", path, err)
	}

	if len(foodsData.Foods) == 0 {
		return fmt.Errorf("no foods found in file %s", path)
	}

	r.foods = foodsData.Foods
	return nil
}

// Search performs case-insensitive partial matching on food names.
func (r *repository) Search(ctx context.Context, query string, limit int) ([]CommonFood, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Default limit
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	// Normalize query for case-insensitive search
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return []CommonFood{}, nil
	}

	// Search for matches
	matches := make([]CommonFood, 0, limit)
	for _, food := range r.foods {
		if len(matches) >= limit {
			break
		}

		// Case-insensitive partial match
		if strings.Contains(strings.ToLower(food.Name), normalizedQuery) {
			matches = append(matches, food)
		}
	}

	return matches, nil
}

// GetDefaultFilePath returns the default path to common_foods.json.
func GetDefaultFilePath() string {
	// Try relative path first (for development)
	relativePath := "internal/seed/data/common_foods.json"
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// Try absolute path from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		absPath := filepath.Join(cwd, "internal", "seed", "data", "common_foods.json")
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Try path relative to backend directory
	backendPath := filepath.Join("backend", "internal", "seed", "data", "common_foods.json")
	if _, err := os.Stat(backendPath); err == nil {
		return backendPath
	}

	// Default fallback
	return "data/common_foods.json"
}
