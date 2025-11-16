package meals

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMealWithItems(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().UTC()

	meal := &Meal{
		ID:            uuid.New(),
		UserID:        userID,
		MealType:      MealTypeBreakfast,
		ConsumedAt:    now,
		TotalCalories: 362,
		TotalProteinG: 23.2,
		TotalCarbsG:   13.2,
		TotalFatG:     22.4,
		TotalFiberG:   1.9,
	}

	items := []MealItem{
		{
			ID:       uuid.New(),
			MealID:   meal.ID,
			Name:     "Scrambled Eggs",
			Quantity: 2,
			Unit:     "large",
			Calories: 180,
			ProteinG: 12.6,
		},
	}

	repo := &MockRepository{
		CreateMealWithItemsFunc: func(ctx context.Context, uid uuid.UUID, m *Meal, itms []MealItem) (*MealWithItems, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, meal.MealType, m.MealType)
			assert.Len(t, itms, 1)
			return &MealWithItems{Meal: *m, Items: itms}, nil
		},
	}

	result, err := repo.CreateMealWithItems(ctx, userID, meal, items)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, meal.ID, result.ID)
	assert.Len(t, result.Items, 1)
}

func TestGetMealByID(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()

	repo := &MockRepository{
		GetMealByIDFunc: func(ctx context.Context, uid, mid uuid.UUID) (*MealWithItems, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return &MealWithItems{
				Meal: Meal{
					ID:       mealID,
					UserID:   userID,
					MealType: MealTypeBreakfast,
				},
				Items: []MealItem{},
			}, nil
		},
	}

	result, err := repo.GetMealByID(ctx, userID, mealID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, mealID, result.ID)
}

func TestListMealsByUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	filters := ListMealFilters{
		Page:  1,
		Limit: 20,
	}

	repo := &MockRepository{
		ListMealsByUserFunc: func(ctx context.Context, uid uuid.UUID, f ListMealFilters) ([]MealListItem, int, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 1, f.Page)
			assert.Equal(t, 20, f.Limit)
			return []MealListItem{
				{
					ID:            uuid.New(),
					MealType:      MealTypeBreakfast,
					TotalCalories: 362,
					ItemCount:     3,
				},
			}, 1, nil
		},
	}

	meals, total, err := repo.ListMealsByUser(ctx, userID, filters)
	require.NoError(t, err)
	assert.Len(t, meals, 1)
	assert.Equal(t, 1, total)
}

func TestDeleteMeal(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()

	repo := &MockRepository{
		DeleteMealFunc: func(ctx context.Context, uid, mid uuid.UUID) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			return nil
		},
	}

	err := repo.DeleteMeal(ctx, userID, mealID)
	require.NoError(t, err)
}

func TestDuplicateMeal(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	mealID := uuid.New()
	newTime := time.Now().UTC().Add(24 * time.Hour)

	repo := &MockRepository{
		DuplicateMealFunc: func(ctx context.Context, uid, mid uuid.UUID, consumedAt time.Time, mealType MealType) (*MealWithItems, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, mealID, mid)
			assert.Equal(t, MealTypeBreakfast, mealType)
			return &MealWithItems{
				Meal: Meal{
					ID:         uuid.New(),
					UserID:     userID,
					MealType:   mealType,
					ConsumedAt: consumedAt,
				},
			}, nil
		},
	}

	result, err := repo.DuplicateMeal(ctx, userID, mealID, newTime, MealTypeBreakfast)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, mealID, result.ID)
}
