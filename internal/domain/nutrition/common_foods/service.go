package common_foods

import (
	"context"
	"fmt"
	"log/slog"
)

// Service defines the business logic for common foods search.
type Service interface {
	Search(ctx context.Context, query string, limit int) ([]CommonFood, error)
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new common foods service.
func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// Search performs a search for common foods matching the query.
func (s *service) Search(ctx context.Context, query string, limit int) ([]CommonFood, error) {
	if query == "" {
		return []CommonFood{}, nil
	}

	foods, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		s.logger.Error("failed to search common foods",
			slog.String("query", query),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	s.logger.Info("common foods search completed",
		slog.String("query", query),
		slog.Int("results", len(foods)),
	)

	return foods, nil
}
