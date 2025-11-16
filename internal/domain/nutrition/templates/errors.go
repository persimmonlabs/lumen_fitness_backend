package templates

import "errors"

var (
	// ErrTemplateNotFound is returned when a template is not found
	ErrTemplateNotFound = errors.New("template not found")

	// ErrTemplateNameRequired is returned when template name is empty
	ErrTemplateNameRequired = errors.New("template name is required")

	// ErrTemplateNameTooLong is returned when template name exceeds max length
	ErrTemplateNameTooLong = errors.New("template name too long (max 100 characters)")

	// ErrTemplateItemsRequired is returned when no items are provided
	ErrTemplateItemsRequired = errors.New("at least one template item is required")

	// ErrFoodIDRequired is returned when food_id is missing
	ErrFoodIDRequired = errors.New("food_id is required")

	// ErrInvalidServingSize is returned when serving size is invalid
	ErrInvalidServingSize = errors.New("serving_size must be greater than 0")

	// ErrServingUnitRequired is returned when serving unit is empty
	ErrServingUnitRequired = errors.New("serving_unit is required")

	// ErrUnauthorized is returned when user doesn't own the template
	ErrUnauthorized = errors.New("unauthorized access to template")

	// ErrMealNotFound is returned when referenced meal is not found
	ErrMealNotFound = errors.New("meal not found")

	// ErrInvalidMealType is returned when meal type is invalid
	ErrInvalidMealType = errors.New("invalid meal type")

	// ErrInvalidMealTime is returned when meal time is invalid
	ErrInvalidMealTime = errors.New("meal_time cannot be more than 24 hours in the future")
)
