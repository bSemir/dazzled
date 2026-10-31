// Package leaderboard defines use cases exposed to leaderboard HTTP handlers.
package leaderboard

import (
	"context"

	"dazzled/backend/internal/domain"
)

// Service defines leaderboard use cases consumed by REST handlers.
type Service interface {
	StartGame(ctx context.Context) (string, error)
	Submit(ctx context.Context, score domain.Score, token string) error
	Top(ctx context.Context) ([]domain.Score, error)
}
