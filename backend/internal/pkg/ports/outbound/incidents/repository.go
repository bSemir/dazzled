package incidents

import (
	"context"

	"dazzled/backend/internal/domain"
)

// Repository persists incidents and serves their aggregates.
type Repository interface {
	Create(ctx context.Context, incident domain.Incident) (int64, error)
	Statistics(ctx context.Context, period string) (domain.Statistics, error)
}
