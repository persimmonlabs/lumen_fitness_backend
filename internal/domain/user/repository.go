package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user data access
type Repository interface {
	// GetProfile retrieves user profile by user ID
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)

	// UpdateProfile updates user profile
	UpdateProfile(ctx context.Context, profile *UserProfile) error

	// GetSettings retrieves user settings by user ID
	GetSettings(ctx context.Context, userID uuid.UUID) (*UserSettings, error)

	// UpdateSettings updates user settings
	UpdateSettings(ctx context.Context, settings *UserSettings) error

	// CreateProfile creates a new user profile (used during registration)
	CreateProfile(ctx context.Context, profile *UserProfile) error

	// CreateSettings creates default user settings (used during registration)
	CreateSettings(ctx context.Context, settings *UserSettings) error
}
