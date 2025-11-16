package templates

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service handles template business logic
type Service interface {
	CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*TemplateResponse, error)
	CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*TemplateResponse, error)
	UseTemplate(ctx context.Context, userID, templateID uuid.UUID, req UseTemplateRequest) (uuid.UUID, error)
	GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*TemplateResponse, error)
	ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) (*TemplateListResponse, error)
	UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*TemplateResponse, error)
	DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error
}

type service struct {
	repo Repository
	db   *sqlx.DB
}

// NewService creates a new template service
func NewService(repo Repository, db *sqlx.DB) Service {
	return &service{
		repo: repo,
		db:   db,
	}
}

// CreateTemplate creates a new template
func (s *service) CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*TemplateResponse, error) {
	// Validate request
	if err := s.validateCreateTemplateRequest(req); err != nil {
		return nil, err
	}

	// Create template
	template, err := s.repo.CreateTemplate(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	resp := template.ToResponse()
	return &resp, nil
}

// CreateFromMeal creates a template from an existing meal
func (s *service) CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*TemplateResponse, error) {
	// Validate request
	if err := s.validateCreateFromMealRequest(req); err != nil {
		return nil, err
	}

	// Verify meal exists and belongs to user
	var mealExists bool
	query := `SELECT EXISTS(SELECT 1 FROM nutrition.meals WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL)`
	err := s.db.GetContext(ctx, &mealExists, query, req.MealID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify meal: %w", err)
	}
	if !mealExists {
		return nil, ErrMealNotFound
	}

	// Create template from meal
	template, err := s.repo.CreateFromMeal(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	resp := template.ToResponse()
	return &resp, nil
}

// UseTemplate creates a meal from a template
func (s *service) UseTemplate(ctx context.Context, userID, templateID uuid.UUID, req UseTemplateRequest) (uuid.UUID, error) {
	// Validate request
	if err := s.validateUseTemplateRequest(req); err != nil {
		return uuid.Nil, err
	}

	// Verify template exists and belongs to user
	template, err := s.repo.GetTemplateByID(ctx, userID, templateID)
	if err != nil {
		return uuid.Nil, err
	}

	// Create meal from template
	var mealID uuid.UUID
	query := `
		INSERT INTO nutrition.meals (id, user_id, meal_type, consumed_at, photo_urls, is_draft, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, false, NOW(), NOW())
		RETURNING id
	`
	mealID = uuid.New()

	// Convert photo URL to array format
	var photoURLs []string
	if template.PhotoURL != nil && *template.PhotoURL != "" {
		photoURLs = []string{*template.PhotoURL}
	}

	_, err = s.db.ExecContext(ctx, query,
		mealID,
		userID,
		req.MealType,
		req.MealTime,
		photoURLs,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create meal: %w", err)
	}

	// Copy template items to meal items
	for _, item := range template.Items {
		itemQuery := `
			INSERT INTO nutrition.meal_items (id, meal_id, food_id, name, quantity, unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, 0, NOW(), NOW())
		`
		_, err = s.db.ExecContext(ctx, itemQuery,
			uuid.New(),
			mealID,
			item.FoodID,
			item.FoodName,
			item.ServingSize,
			item.ServingUnit,
			item.Calories,
			item.Protein,
			item.Carbs,
			item.Fat,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to create meal item: %w", err)
		}
	}

	return mealID, nil
}

// GetTemplate retrieves a single template
func (s *service) GetTemplate(ctx context.Context, userID, templateID uuid.UUID) (*TemplateResponse, error) {
	template, err := s.repo.GetTemplateByID(ctx, userID, templateID)
	if err != nil {
		return nil, err
	}

	resp := template.ToResponse()
	return &resp, nil
}

// ListTemplates retrieves all templates for a user
func (s *service) ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) (*TemplateListResponse, error) {
	// Set default pagination
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	templates, total, err := s.repo.ListTemplates(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]TemplateResponse, len(templates))
	for i, template := range templates {
		responses[i] = template.ToResponse()
	}

	return &TemplateListResponse{
		Templates: responses,
		Total:     total,
	}, nil
}

// UpdateTemplate updates a template
func (s *service) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*TemplateResponse, error) {
	// Validate request
	if err := s.validateUpdateTemplateRequest(req); err != nil {
		return nil, err
	}

	template, err := s.repo.UpdateTemplate(ctx, userID, templateID, req)
	if err != nil {
		return nil, err
	}

	resp := template.ToResponse()
	return &resp, nil
}

// DeleteTemplate deletes a template
func (s *service) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	return s.repo.DeleteTemplate(ctx, userID, templateID)
}

// validateCreateTemplateRequest validates the create template request
func (s *service) validateCreateTemplateRequest(req CreateTemplateRequest) error {
	if req.Name == "" {
		return ErrTemplateNameRequired
	}
	if len(req.Name) > 100 {
		return ErrTemplateNameTooLong
	}
	if len(req.Items) == 0 {
		return ErrTemplateItemsRequired
	}
	for i, item := range req.Items {
		if item.FoodID == uuid.Nil {
			return fmt.Errorf("item %d: %w", i, ErrFoodIDRequired)
		}
		if item.ServingSize <= 0 {
			return fmt.Errorf("item %d: %w", i, ErrInvalidServingSize)
		}
		if item.ServingUnit == "" {
			return fmt.Errorf("item %d: %w", i, ErrServingUnitRequired)
		}
	}
	return nil
}

// validateCreateFromMealRequest validates the create from meal request
func (s *service) validateCreateFromMealRequest(req CreateTemplateFromMealRequest) error {
	if req.MealID == uuid.Nil {
		return ErrMealNotFound
	}
	if req.Name == "" {
		return ErrTemplateNameRequired
	}
	if len(req.Name) > 100 {
		return ErrTemplateNameTooLong
	}
	return nil
}

// validateUseTemplateRequest validates the use template request
func (s *service) validateUseTemplateRequest(req UseTemplateRequest) error {
	if req.MealType == "" {
		return ErrInvalidMealType
	}
	validMealTypes := map[string]bool{"breakfast": true, "lunch": true, "dinner": true, "snack": true}
	if !validMealTypes[req.MealType] {
		return ErrInvalidMealType
	}
	if req.MealTime.IsZero() {
		return fmt.Errorf("meal_time is required")
	}
	if req.MealTime.After(time.Now().Add(24 * time.Hour)) {
		return ErrInvalidMealTime
	}
	return nil
}

// validateUpdateTemplateRequest validates the update template request
func (s *service) validateUpdateTemplateRequest(req UpdateTemplateRequest) error {
	if req.Name != nil {
		if *req.Name == "" {
			return ErrTemplateNameRequired
		}
		if len(*req.Name) > 100 {
			return ErrTemplateNameTooLong
		}
	}
	return nil
}
