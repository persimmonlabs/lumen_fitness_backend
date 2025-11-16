package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles template data persistence
type Repository interface {
	CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*Template, error)
	CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*Template, error)
	GetTemplateByID(ctx context.Context, userID, templateID uuid.UUID) (*Template, error)
	ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Template, int, error)
	UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*Template, error)
	DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new template repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// CreateTemplate creates a new template using the RPC function
func (r *repository) CreateTemplate(ctx context.Context, userID uuid.UUID, req CreateTemplateRequest) (*Template, error) {
	// Convert items to JSON
	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	var templateID uuid.UUID
	query := `SELECT create_template($1, $2, $3, $4)`
	err = r.db.GetContext(ctx, &templateID, query, userID, req.Name, req.PhotoURL, itemsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Fetch the created template with items
	return r.GetTemplateByID(ctx, userID, templateID)
}

// CreateFromMeal creates a template from an existing meal
func (r *repository) CreateFromMeal(ctx context.Context, userID uuid.UUID, req CreateTemplateFromMealRequest) (*Template, error) {
	var templateID uuid.UUID
	query := `SELECT create_template_from_meal($1, $2, $3)`
	err := r.db.GetContext(ctx, &templateID, query, userID, req.MealID, req.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMealNotFound
		}
		return nil, fmt.Errorf("failed to create template from meal: %w", err)
	}

	// Fetch the created template with items
	return r.GetTemplateByID(ctx, userID, templateID)
}

// GetTemplateByID retrieves a template by ID with its items
func (r *repository) GetTemplateByID(ctx context.Context, userID, templateID uuid.UUID) (*Template, error) {
	var template Template
	query := `
		SELECT id, user_id, name, photo_url, total_calories, total_protein,
		       total_carbs, total_fat, created_at, updated_at
		FROM nutrition.templates
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &template, query, templateID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Fetch template items
	itemsQuery := `
		SELECT id, template_id, food_id, food_name, serving_size, serving_unit,
		       calories, protein, carbs, fat, food_photo_url, created_at
		FROM nutrition.template_items
		WHERE template_id = $1
		ORDER BY created_at ASC
	`
	err = r.db.SelectContext(ctx, &template.Items, itemsQuery, templateID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get template items: %w", err)
	}

	return &template, nil
}

// ListTemplates retrieves all templates for a user
func (r *repository) ListTemplates(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Template, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM nutrition.templates
		WHERE user_id = $1 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// Get templates
	var templates []Template
	query := `
		SELECT id, user_id, name, photo_url, total_calories, total_protein,
		       total_carbs, total_fat, created_at, updated_at
		FROM nutrition.templates
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err = r.db.SelectContext(ctx, &templates, query, userID, limit, offset)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	// Fetch items for each template
	for i := range templates {
		itemsQuery := `
			SELECT id, template_id, food_id, food_name, serving_size, serving_unit,
			       calories, protein, carbs, fat, food_photo_url, created_at
			FROM nutrition.template_items
			WHERE template_id = $1
			ORDER BY created_at ASC
		`
		err = r.db.SelectContext(ctx, &templates[i].Items, itemsQuery, templates[i].ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, 0, fmt.Errorf("failed to get template items: %w", err)
		}
	}

	return templates, total, nil
}

// UpdateTemplate updates a template's name and/or photo
func (r *repository) UpdateTemplate(ctx context.Context, userID, templateID uuid.UUID, req UpdateTemplateRequest) (*Template, error) {
	// Build dynamic update query
	query := `UPDATE nutrition.templates SET updated_at = NOW()`
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		query += fmt.Sprintf(`, name = $%d`, argPos)
		args = append(args, *req.Name)
		argPos++
	}

	if req.PhotoURL != nil {
		query += fmt.Sprintf(`, photo_url = $%d`, argPos)
		args = append(args, *req.PhotoURL)
		argPos++
	}

	query += fmt.Sprintf(` WHERE id = $%d AND user_id = $%d AND deleted_at IS NULL`, argPos, argPos+1)
	args = append(args, templateID, userID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrTemplateNotFound
	}

	// Fetch and return the updated template
	return r.GetTemplateByID(ctx, userID, templateID)
}

// DeleteTemplate soft deletes a template
func (r *repository) DeleteTemplate(ctx context.Context, userID, templateID uuid.UUID) error {
	query := `
		UPDATE nutrition.templates
		SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTemplateNotFound
	}

	return nil
}
