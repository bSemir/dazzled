package leaderboard

import (
	"context"
	"errors"
	"time"

	"dazzled/backend/internal/domain"
)

var ErrGameSessionUnavailable = errors.New("game session is unavailable")

// Repository persists and ranks Blind Drive scores.
type Repository interface {
	StartGame(ctx context.Context, nonce string, startedAt time.Time) error
	ConsumeGame(ctx context.Context, nonce string, now time.Time) (time.Time, error)
	Submit(ctx context.Context, score domain.Score) error
	Top(ctx context.Context) ([]domain.Score, error)
}
