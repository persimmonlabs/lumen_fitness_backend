package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/pradord/lumen_final/backend/internal/errors"
)

// Global validator instance
var validate = validator.New()

// ValidateStruct validates a struct using struct tags
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err != nil {
		// Convert validation errors to our error format
		validationErrors := err.(validator.ValidationErrors)
		valErr := errors.Validation("Request validation failed")

		for _, e := range validationErrors {
			valErr.AddField(e.Field(), formatValidationError(e))
		}

		return valErr
	}
	return nil
}

// formatValidationError converts validator error to human-readable message
func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "gte":
		return "Value must be greater than or equal to " + e.Param()
	case "lte":
		return "Value must be less than or equal to " + e.Param()
	default:
		return "Invalid value"
	}
}
