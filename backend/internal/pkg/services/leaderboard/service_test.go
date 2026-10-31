// Package leaderboard tests score validation, token use, and elapsed-time checks.
package leaderboard

import (
	"context"
	"testing"
	"time"

	"dazzled/backend/internal/domain"
	outbound "dazzled/backend/internal/pkg/ports/outbound/leaderboard"
)

type testRepository struct {
	score    domain.Score
	sessions map[string]time.Time
	used     map[string]bool
}

func (r *testRepository) StartGame(_ context.Context, nonce string, startedAt time.Time) error {
	if r.sessions == nil {
		r.sessions = make(map[string]time.Time)
		r.used = make(map[string]bool)
	}
	r.sessions[nonce] = startedAt
	return nil
}

func (r *testRepository) ConsumeGame(_ context.Context, nonce string, _ time.Time) (time.Time, error) {
	startedAt, ok := r.sessions[nonce]
	if !ok || r.used[nonce] {
		return time.Time{}, outbound.ErrGameSessionUnavailable
	}
	r.used[nonce] = true
	return startedAt, nil
}

func (r *testRepository) Submit(_ context.Context, score domain.Score) error {
	r.score = score
	return nil
}

func (r *testRepository) Top(context.Context) ([]domain.Score, error) { return nil, nil }

func TestSubmitValidatesAndNormalizesScore(t *testing.T) {
	repository := &testRepository{}
	service := New(repository, []byte("test-game-token-secret-that-is-long-enough"))
	token, err := service.StartGame(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Submit(context.Background(), domain.Score{Username: " road-runner ", Distance: 0.2}, token); err != nil {
		t.Fatal(err)
	}
	if repository.score.Username != "road-runner" {
		t.Fatalf("score username not normalized: %+v", repository.score)
	}
	for _, invalid := range []domain.Score{
		{Username: "x", Distance: 1},
		{Username: "ok", Distance: -1},
		{Username: "ok", Distance: 10001},
	} {
		if err := service.Submit(context.Background(), invalid, token); err == nil {
			t.Errorf("invalid score accepted: %+v", invalid)
		}
	}
}

func TestSubmitRequiresSingleUseServerTimedToken(t *testing.T) {
	repository := &testRepository{}
	service := New(repository, []byte("test-game-token-secret-that-is-long-enough"))
	token, err := service.StartGame(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Submit(context.Background(), domain.Score{Username: "night owl", Distance: 3}, token); err != ErrScoreExceedsElapsedTime {
		t.Fatalf("large immediate score error=%v, want elapsed-time rejection", err)
	}
	if err := service.Submit(context.Background(), domain.Score{Username: "night owl", Distance: 0.1}, token); err != ErrInvalidGameToken {
		t.Fatalf("replayed token error=%v, want invalid-token rejection", err)
	}
	newToken, err := service.StartGame(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tamperedTokenBytes := []byte(newToken)
	if tamperedTokenBytes[3] == 'A' {
		tamperedTokenBytes[3] = 'B'
	} else {
		tamperedTokenBytes[3] = 'A'
	}
	tamperedToken := string(tamperedTokenBytes)
	if err := service.Submit(context.Background(), domain.Score{Username: "night owl", Distance: 0.1}, tamperedToken); err != ErrInvalidGameToken {
		t.Fatalf("tampered token error=%v, want invalid-token rejection", err)
	}
}
