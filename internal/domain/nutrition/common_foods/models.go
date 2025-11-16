// Package common_foods provides search functionality for common food items.
//
// This package allows users to search through a database of common foods
// with nutrition information to quickly log meals without manual entry.
package common_foods

// CommonFood represents a common food item with complete nutrition information.
type CommonFood struct {
	Name               string  `json:"name"`
	Category           string  `json:"category"`
	CaloriesPer100g    float64 `json:"calories_per_100g"`
	ProteinPer100g     float64 `json:"protein_per_100g"`
	CarbsPer100g       float64 `json:"carbs_per_100g"`
	FatPer100g         float64 `json:"fat_per_100g"`
	FiberPer100g       float64 `json:"fiber_per_100g"`
	CommonServingName  string  `json:"common_serving_name"`
	CommonServingGrams float64 `json:"common_serving_grams"`
	IsVerified         bool    `json:"is_verified"`
}

// SearchRequest represents a request to search for common foods.
type SearchRequest struct {
	Query string `json:"query" validate:"required,max=100"`
	Limit int    `json:"limit,omitempty"`
}

// SearchResponse represents the response containing matching foods.
type SearchResponse struct {
	Foods []CommonFood `json:"foods"`
}
