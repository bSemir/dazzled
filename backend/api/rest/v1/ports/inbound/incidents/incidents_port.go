// Package incidents defines use cases exposed to incident HTTP handlers.
package incidents

import (
	"context"

	"dazzled/backend/internal/domain"
)

// Service defines incident and statistics use cases consumed by REST handlers.
type Service interface {
	Create(ctx context.Context, incident domain.Incident) (int64, error)
	Count(ctx context.Context) (int, error)
	Statistics(ctx context.Context, period string) (domain.Statistics, error)
}
