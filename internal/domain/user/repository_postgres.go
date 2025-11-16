package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound = errors.New("user not found")
)

type postgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	query := `
		SELECT user_id, full_name, email, avatar_url, created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`

	var profile UserProfile
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &profile, nil
}

func (r *postgresRepository) UpdateProfile(ctx context.Context, profile *UserProfile) error {
	query := `
		UPDATE user_profiles
		SET full_name = $1, avatar_url = $2, updated_at = $3
		WHERE user_id = $4
	`

	result, err := r.db.ExecContext(ctx, query,
		profile.FullName,
		profile.AvatarURL,
		profile.UpdatedAt,
		profile.UserID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *postgresRepository) GetSettings(ctx context.Context, userID uuid.UUID) (*UserSettings, error) {
	query := `
		SELECT user_id, notifications_enabled, email_notifications, weekly_reports,
		       theme, language, timezone, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	var settings UserSettings
	err := r.db.GetContext(ctx, &settings, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &settings, nil
}

func (r *postgresRepository) UpdateSettings(ctx context.Context, settings *UserSettings) error {
	query := `
		UPDATE user_settings
		SET notifications_enabled = $1, email_notifications = $2, weekly_reports = $3,
		    theme = $4, language = $5, timezone = $6, updated_at = $7
		WHERE user_id = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.NotificationsEnabled,
		settings.EmailNotifications,
		settings.WeeklyReports,
		settings.Theme,
		settings.Language,
		settings.TimeZone,
		settings.UpdatedAt,
		settings.UserID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *postgresRepository) CreateProfile(ctx context.Context, profile *UserProfile) error {
	query := `
		INSERT INTO user_profiles (user_id, full_name, email, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		profile.UserID,
		profile.FullName,
		profile.Email,
		profile.AvatarURL,
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	return err
}

func (r *postgresRepository) CreateSettings(ctx context.Context, settings *UserSettings) error {
	query := `
		INSERT INTO user_settings (user_id, notifications_enabled, email_notifications,
		                          weekly_reports, theme, language, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.UserID,
		settings.NotificationsEnabled,
		settings.EmailNotifications,
		settings.WeeklyReports,
		settings.Theme,
		settings.Language,
		settings.TimeZone,
		settings.CreatedAt,
		settings.UpdatedAt,
	)

	return err
}
