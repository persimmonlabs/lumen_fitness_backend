package user

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile represents user profile information
type UserProfile struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	FullName  string    `json:"full_name" db:"full_name"`
	Email     string    `json:"email" db:"email"`
	AvatarURL *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserSettings represents user preferences and settings
type UserSettings struct {
	UserID                    uuid.UUID `json:"user_id" db:"user_id"`
	NotificationsEnabled      bool      `json:"notifications_enabled" db:"notifications_enabled"`
	EmailNotifications        bool      `json:"email_notifications" db:"email_notifications"`
	WeeklyReports             bool      `json:"weekly_reports" db:"weekly_reports"`
	Theme                     string    `json:"theme" db:"theme"` // "light", "dark", "auto"
	Language                  string    `json:"language" db:"language"`
	TimeZone                  string    `json:"timezone" db:"timezone"`
	CreatedAt                 time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateProfileRequest represents request to update user profile
type UpdateProfileRequest struct {
	FullName  *string `json:"full_name,omitempty" validate:"omitempty,min=1,max=255"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

// UpdateSettingsRequest represents request to update user settings
type UpdateSettingsRequest struct {
	NotificationsEnabled *bool   `json:"notifications_enabled,omitempty"`
	EmailNotifications   *bool   `json:"email_notifications,omitempty"`
	WeeklyReports        *bool   `json:"weekly_reports,omitempty"`
	Theme                *string `json:"theme,omitempty" validate:"omitempty,oneof=light dark auto"`
	Language             *string `json:"language,omitempty" validate:"omitempty,len=2"`
	TimeZone             *string `json:"timezone,omitempty"`
}
