package weight

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound      = errors.New("weight entry not found")
	ErrDuplicateDate = errors.New("weight entry already exists for this date")
)

type postgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, entry *WeightEntry) error {
	query := `
		INSERT INTO weight_entries (id, user_id, weight, measured_at, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.UserID,
		entry.Weight,
		entry.MeasuredAt,
		entry.Notes,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if isDuplicateKeyError(err) {
			return ErrDuplicateDate
		}
		return err
	}

	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*WeightEntry, error) {
	query := `
		SELECT id, user_id, weight, measured_at, notes, created_at, updated_at
		FROM weight_entries
		WHERE id = $1 AND user_id = $2
	`

	var entry WeightEntry
	err := r.db.GetContext(ctx, &entry, query, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &entry, nil
}

func (r *postgresRepository) List(ctx context.Context, filter WeightListFilter) ([]WeightEntry, int, error) {
	// Build dynamic query
	baseQuery := `
		SELECT id, user_id, weight, measured_at, notes, created_at, updated_at
		FROM weight_entries
		WHERE user_id = $1
	`
	countQuery := `
		SELECT COUNT(*)
		FROM weight_entries
		WHERE user_id = $1
	`

	args := []interface{}{filter.UserID}
	argCount := 2

	if filter.StartDate != nil {
		baseQuery += ` AND measured_at >= $` + string(rune(argCount+'0'))
		countQuery += ` AND measured_at >= $` + string(rune(argCount+'0'))
		args = append(args, *filter.StartDate)
		argCount++
	}

	if filter.EndDate != nil {
		baseQuery += ` AND measured_at <= $` + string(rune(argCount+'0'))
		countQuery += ` AND measured_at <= $` + string(rune(argCount+'0'))
		args = append(args, *filter.EndDate)
		argCount++
	}

	baseQuery += ` ORDER BY measured_at DESC`

	// Add pagination
	offset := (filter.Page - 1) * filter.PageSize
	baseQuery += ` LIMIT $` + string(rune(argCount+'0')) + ` OFFSET $` + string(rune(argCount+1+'0'))
	paginationArgs := append(args, filter.PageSize, offset)

	// Get total count
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get entries
	var entries []WeightEntry
	err = r.db.SelectContext(ctx, &entries, baseQuery, paginationArgs...)
	if err != nil {
		return nil, 0, err
	}

	if entries == nil {
		entries = []WeightEntry{}
	}

	return entries, total, nil
}

func (r *postgresRepository) GetLatest(ctx context.Context, userID uuid.UUID) (*WeightEntry, error) {
	query := `
		SELECT id, user_id, weight, measured_at, notes, created_at, updated_at
		FROM weight_entries
		WHERE user_id = $1
		ORDER BY measured_at DESC
		LIMIT 1
	`

	var entry WeightEntry
	err := r.db.GetContext(ctx, &entry, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &entry, nil
}

func (r *postgresRepository) Update(ctx context.Context, entry *WeightEntry) error {
	query := `
		UPDATE weight_entries
		SET weight = $1, measured_at = $2, notes = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		entry.Weight,
		entry.MeasuredAt,
		entry.Notes,
		entry.UpdatedAt,
		entry.ID,
		entry.UserID,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateDate
		}
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

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM weight_entries
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, id, userID)
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

func (r *postgresRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]WeightEntry, error) {
	query := `
		SELECT id, user_id, weight, measured_at, notes, created_at, updated_at
		FROM weight_entries
		WHERE user_id = $1 AND measured_at >= $2 AND measured_at <= $3
		ORDER BY measured_at ASC
	`

	var entries []WeightEntry
	err := r.db.SelectContext(ctx, &entries, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if entries == nil {
		entries = []WeightEntry{}
	}

	return entries, nil
}

func (r *postgresRepository) CheckDuplicateDate(ctx context.Context, userID uuid.UUID, date time.Time, excludeID *uuid.UUID) (bool, error) {
	// Normalize date to start of day in UTC
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM weight_entries
			WHERE user_id = $1
			AND measured_at >= $2
			AND measured_at < $3
	`

	args := []interface{}{userID, startOfDay, endOfDay}

	if excludeID != nil {
		query += ` AND id != $4`
		args = append(args, *excludeID)
	}

	query += `)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, args...)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Helper function to check for duplicate key errors
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL duplicate key error code is 23505
	return contains(err.Error(), "23505") || contains(err.Error(), "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
