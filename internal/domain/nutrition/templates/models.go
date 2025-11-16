package templates

import (
	"time"

	"github.com/google/uuid"
)

// Template represents a reusable meal template
type Template struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	UserID        uuid.UUID      `json:"user_id" db:"user_id"`
	Name          string         `json:"name" db:"name"`
	PhotoURL      *string        `json:"photo_url,omitempty" db:"photo_url"`
	TotalCalories float64        `json:"total_calories" db:"total_calories"`
	TotalProtein  float64        `json:"total_protein" db:"total_protein"`
	TotalCarbs    float64        `json:"total_carbs" db:"total_carbs"`
	TotalFat      float64        `json:"total_fat" db:"total_fat"`
	Items         []TemplateItem `json:"items,omitempty" db:"-"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// TemplateItem represents a food item in a template
type TemplateItem struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TemplateID   uuid.UUID `json:"template_id" db:"template_id"`
	FoodID       uuid.UUID `json:"food_id" db:"food_id"`
	FoodName     string    `json:"food_name" db:"food_name"`
	ServingSize  float64   `json:"serving_size" db:"serving_size"`
	ServingUnit  string    `json:"serving_unit" db:"serving_unit"`
	Calories     float64   `json:"calories" db:"calories"`
	Protein      float64   `json:"protein" db:"protein"`
	Carbs        float64   `json:"carbs" db:"carbs"`
	Fat          float64   `json:"fat" db:"fat"`
	FoodPhotoURL *string   `json:"food_photo_url,omitempty" db:"food_photo_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateTemplateRequest represents a request to create a new template
type CreateTemplateRequest struct {
	Name     string                      `json:"name" validate:"required,max=100"`
	PhotoURL *string                     `json:"photo_url,omitempty"`
	Items    []CreateTemplateItemRequest `json:"items" validate:"required,min=1,dive"`
}

// CreateTemplateItemRequest represents an item in a create template request
type CreateTemplateItemRequest struct {
	FoodID      uuid.UUID `json:"food_id" validate:"required"`
	ServingSize float64   `json:"serving_size" validate:"required,gt=0"`
	ServingUnit string    `json:"serving_unit" validate:"required"`
}

// CreateTemplateFromMealRequest represents a request to create a template from an existing meal
type CreateTemplateFromMealRequest struct {
	MealID uuid.UUID `json:"meal_id" validate:"required"`
	Name   string    `json:"name" validate:"required,max=100"`
}

// UseTemplateRequest represents a request to create a meal from a template
type UseTemplateRequest struct {
	MealType string    `json:"meal_type" validate:"required,oneof=breakfast lunch dinner snack"`
	MealTime time.Time `json:"meal_time" validate:"required"`
}

// UpdateTemplateRequest represents a request to update a template
type UpdateTemplateRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,max=100"`
	PhotoURL *string `json:"photo_url,omitempty"`
}

// TemplateResponse represents a single template response
type TemplateResponse struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	PhotoURL      *string                `json:"photo_url,omitempty"`
	TotalCalories float64                `json:"total_calories"`
	TotalProtein  float64                `json:"total_protein"`
	TotalCarbs    float64                `json:"total_carbs"`
	TotalFat      float64                `json:"total_fat"`
	Items         []TemplateItemResponse `json:"items"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// TemplateItemResponse represents a template item in a response
type TemplateItemResponse struct {
	ID           uuid.UUID `json:"id"`
	FoodID       uuid.UUID `json:"food_id"`
	FoodName     string    `json:"food_name"`
	ServingSize  float64   `json:"serving_size"`
	ServingUnit  string    `json:"serving_unit"`
	Calories     float64   `json:"calories"`
	Protein      float64   `json:"protein"`
	Carbs        float64   `json:"carbs"`
	Fat          float64   `json:"fat"`
	FoodPhotoURL *string   `json:"food_photo_url,omitempty"`
}

// TemplateListResponse represents a list of templates response
type TemplateListResponse struct {
	Templates []TemplateResponse `json:"templates"`
	Total     int                `json:"total"`
}

// ToResponse converts a Template to TemplateResponse
func (t *Template) ToResponse() TemplateResponse {
	items := make([]TemplateItemResponse, len(t.Items))
	for i, item := range t.Items {
		items[i] = TemplateItemResponse{
			ID:           item.ID,
			FoodID:       item.FoodID,
			FoodName:     item.FoodName,
			ServingSize:  item.ServingSize,
			ServingUnit:  item.ServingUnit,
			Calories:     item.Calories,
			Protein:      item.Protein,
			Carbs:        item.Carbs,
			Fat:          item.Fat,
			FoodPhotoURL: item.FoodPhotoURL,
		}
	}

	return TemplateResponse{
		ID:            t.ID,
		Name:          t.Name,
		PhotoURL:      t.PhotoURL,
		TotalCalories: t.TotalCalories,
		TotalProtein:  t.TotalProtein,
		TotalCarbs:    t.TotalCarbs,
		TotalFat:      t.TotalFat,
		Items:         items,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}
