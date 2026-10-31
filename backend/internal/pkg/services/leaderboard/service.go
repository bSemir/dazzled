// Package leaderboard manages signed game sessions, score validation, and rankings.
package leaderboard

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	inbound "dazzled/backend/api/rest/v1/ports/inbound/leaderboard"
	"dazzled/backend/internal/domain"
	outbound "dazzled/backend/internal/pkg/ports/outbound/leaderboard"
)

type Service struct {
	repository  outbound.Repository
	tokenSecret []byte
}

var _ inbound.Service = (*Service)(nil)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9 _-]{2,20}$`)

const (
	gameTokenVersion    = "v1"
	gameTokenNonceBytes = 32
	maxGameDuration     = 20 * time.Minute
	// This is slightly above the fastest frame-rate-independent obstacle-pass rate.
	maxDistancePerSecond   = 0.35
	gameDistanceSafetyRoom = 0.25
)

func New(repository outbound.Repository, tokenSecret []byte) *Service {
	return &Service{repository: repository, tokenSecret: append([]byte(nil), tokenSecret...)}
}

func (s *Service) StartGame(ctx context.Context) (string, error) {
	if len(s.tokenSecret) < sha256.Size {
		return "", errors.New("game token signing key must contain at least 32 bytes")
	}
	// The random nonce is stored server-side; the signature prevents clients inventing sessions.
	nonceBytes := make([]byte, gameTokenNonceBytes)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("generate game nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	startedAt := time.Now().UTC()
	if err := s.repository.StartGame(ctx, nonce, startedAt); err != nil {
		return "", fmt.Errorf("record game start: %w", err)
	}
	return s.signGameToken(nonce), nil
}

func (s *Service) Submit(ctx context.Context, score domain.Score, token string) error {
	score.Username = strings.TrimSpace(score.Username)
	if !usernamePattern.MatchString(score.Username) || math.IsNaN(score.Distance) || math.IsInf(score.Distance, 0) || score.Distance < 0 || score.Distance > 10000 {
		return ErrInvalidScore
	}
	nonce, err := s.verifyGameToken(token)
	if err != nil {
		return ErrInvalidGameToken
	}
	now := time.Now().UTC()
	startedAt, err := s.repository.ConsumeGame(ctx, nonce, now)
	if err != nil {
		if errors.Is(err, outbound.ErrGameSessionUnavailable) {
			return ErrInvalidGameToken
		}
		return fmt.Errorf("consume game token: %w", err)
	}
	elapsed := now.Sub(startedAt)
	if elapsed < 0 || elapsed > maxGameDuration {
		return ErrInvalidGameToken
	}
	// Allow a small safety margin for browser timing while bounding impossible score claims.
	maximumDistance := elapsed.Seconds()*maxDistancePerSecond + gameDistanceSafetyRoom
	if score.Distance > maximumDistance {
		return ErrScoreExceedsElapsedTime
	}
	if err := s.repository.Submit(ctx, score); err != nil {
		return fmt.Errorf("submit leaderboard score: %w", err)
	}
	return nil
}

func (s *Service) Top(ctx context.Context) ([]domain.Score, error) {
	scores, err := s.repository.Top(ctx)
	if err != nil {
		return nil, fmt.Errorf("get top leaderboard scores: %w", err)
	}
	return scores, nil
}

func (s *Service) signGameToken(nonce string) string {
	payload := gameTokenVersion + "." + nonce
	mac := hmac.New(sha256.New, s.tokenSecret)
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyGameToken(token string) (string, error) {
	if len(s.tokenSecret) < sha256.Size {
		return "", ErrInvalidGameToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != gameTokenVersion {
		return "", ErrInvalidGameToken
	}
	nonceBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(nonceBytes) != gameTokenNonceBytes {
		return "", ErrInvalidGameToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", ErrInvalidGameToken
	}
	mac := hmac.New(sha256.New, s.tokenSecret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", ErrInvalidGameToken
	}
	return parts[1], nil
}
