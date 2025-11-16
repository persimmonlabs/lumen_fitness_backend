package weight

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	return sqlxDB, mock
}

func TestRepository_Create(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		entry := &WeightEntry{
			ID:         uuid.New(),
			UserID:     uuid.New(),
			Weight:     75.5,
			MeasuredAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO weight_entries").
			WithArgs(entry.ID, entry.UserID, entry.Weight, entry.MeasuredAt, entry.Notes, entry.CreatedAt, entry.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, entry)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate date error", func(t *testing.T) {
		entry := &WeightEntry{
			ID:         uuid.New(),
			UserID:     uuid.New(),
			Weight:     75.5,
			MeasuredAt: time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO weight_entries").
			WithArgs(entry.ID, entry.UserID, entry.Weight, entry.MeasuredAt, entry.Notes, entry.CreatedAt, entry.UpdatedAt).
			WillReturnError(sqlmock.ErrCancelled) // Simulate duplicate key error

		err := repo.Create(ctx, entry)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetByID(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "weight", "measured_at", "notes", "created_at", "updated_at"}).
			AddRow(id, userID, 75.5, now, nil, now, now)

		mock.ExpectQuery("SELECT (.+) FROM weight_entries WHERE id = \\$1 AND user_id = \\$2").
			WithArgs(id, userID).
			WillReturnRows(rows)

		entry, err := repo.GetByID(ctx, id, userID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, id, entry.ID)
		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, 75.5, entry.Weight)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM weight_entries WHERE id = \\$1 AND user_id = \\$2").
			WithArgs(id, userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "weight", "measured_at", "notes", "created_at", "updated_at"}))

		entry, err := repo.GetByID(ctx, id, userID)
		assert.Error(t, err)
		assert.Nil(t, entry)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_List(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful list with pagination", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now().UTC()

		filter := WeightListFilter{
			UserID:   userID,
			Page:     1,
			PageSize: 10,
		}

		// Mock count query
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM weight_entries WHERE user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(countRows)

		// Mock select query
		rows := sqlmock.NewRows([]string{"id", "user_id", "weight", "measured_at", "notes", "created_at", "updated_at"}).
			AddRow(uuid.New(), userID, 75.5, now, nil, now, now).
			AddRow(uuid.New(), userID, 76.0, now.Add(-24*time.Hour), nil, now, now)

		mock.ExpectQuery("SELECT (.+) FROM weight_entries WHERE user_id = \\$1 ORDER BY measured_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(userID, 10, 0).
			WillReturnRows(rows)

		entries, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, entries, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetLatest(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "weight", "measured_at", "notes", "created_at", "updated_at"}).
			AddRow(uuid.New(), userID, 75.5, now, nil, now, now)

		mock.ExpectQuery("SELECT (.+) FROM weight_entries WHERE user_id = \\$1 ORDER BY measured_at DESC LIMIT 1").
			WithArgs(userID).
			WillReturnRows(rows)

		entry, err := repo.GetLatest(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, 75.5, entry.Weight)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_Delete(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()

		mock.ExpectExec("DELETE FROM weight_entries WHERE id = \\$1 AND user_id = \\$2").
			WithArgs(id, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, id, userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()

		mock.ExpectExec("DELETE FROM weight_entries WHERE id = \\$1 AND user_id = \\$2").
			WithArgs(id, userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, id, userID)
		assert.Error(t, err)
		assert.Equal(t, ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetByDateRange(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now().UTC()
		startDate := now.Add(-7 * 24 * time.Hour)
		endDate := now

		rows := sqlmock.NewRows([]string{"id", "user_id", "weight", "measured_at", "notes", "created_at", "updated_at"}).
			AddRow(uuid.New(), userID, 75.0, now.Add(-5*24*time.Hour), nil, now, now).
			AddRow(uuid.New(), userID, 75.5, now, nil, now, now)

		mock.ExpectQuery("SELECT (.+) FROM weight_entries WHERE user_id = \\$1 AND measured_at >= \\$2 AND measured_at <= \\$3 ORDER BY measured_at ASC").
			WithArgs(userID, startDate, endDate).
			WillReturnRows(rows)

		entries, err := repo.GetByDateRange(ctx, userID, startDate, endDate)
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_CheckDuplicateDate(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	t.Run("duplicate exists", func(t *testing.T) {
		userID := uuid.New()
		date := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

		mock.ExpectQuery("SELECT EXISTS").
			WillReturnRows(rows)

		exists, err := repo.CheckDuplicateDate(ctx, userID, date, nil)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no duplicate", func(t *testing.T) {
		userID := uuid.New()
		date := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)

		mock.ExpectQuery("SELECT EXISTS").
			WillReturnRows(rows)

		exists, err := repo.CheckDuplicateDate(ctx, userID, date, nil)
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
