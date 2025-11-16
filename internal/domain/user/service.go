package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidProfile  = errors.New("invalid profile data")
	ErrInvalidSettings = errors.New("invalid settings data")
)

// Service defines the business logic for user management
type Service interface {
	// GetProfile retrieves user profile
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)

	// UpdateProfile updates user profile
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserProfile, error)

	// GetSettings retrieves user settings
	GetSettings(ctx context.Context, userID uuid.UUID) (*UserSettings, error)

	// UpdateSettings updates user settings
	UpdateSettings(ctx context.Context, userID uuid.UUID, req UpdateSettingsRequest) (*UserSettings, error)
}

type service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserProfile, error) {
	// Get existing profile
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.FullName != nil {
		if err := s.validateFullName(*req.FullName); err != nil {
			return nil, err
		}
		profile.FullName = *req.FullName
	}

	if req.AvatarURL != nil {
		profile.AvatarURL = req.AvatarURL
	}

	profile.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *service) GetSettings(ctx context.Context, userID uuid.UUID) (*UserSettings, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (s *service) UpdateSettings(ctx context.Context, userID uuid.UUID, req UpdateSettingsRequest) (*UserSettings, error) {
	// Get existing settings
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.NotificationsEnabled != nil {
		settings.NotificationsEnabled = *req.NotificationsEnabled
	}

	if req.EmailNotifications != nil {
		settings.EmailNotifications = *req.EmailNotifications
	}

	if req.WeeklyReports != nil {
		settings.WeeklyReports = *req.WeeklyReports
	}

	if req.Theme != nil {
		if err := s.validateTheme(*req.Theme); err != nil {
			return nil, err
		}
		settings.Theme = *req.Theme
	}

	if req.Language != nil {
		settings.Language = *req.Language
	}

	if req.TimeZone != nil {
		settings.TimeZone = *req.TimeZone
	}

	settings.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateSettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// Helper functions

func (s *service) validateFullName(name string) error {
	if len(name) == 0 || len(name) > 255 {
		return ErrInvalidProfile
	}
	return nil
}

func (s *service) validateTheme(theme string) error {
	validThemes := map[string]bool{
		"light": true,
		"dark":  true,
		"auto":  true,
	}

	if !validThemes[theme] {
		return ErrInvalidSettings
	}

	return nil
}
