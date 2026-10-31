// Package leaderboard manages signed game sessions, score validation, and rankings.
package leaderboard

import "errors"

var ErrInvalidScore = errors.New("invalid score")

var ErrInvalidGameToken = errors.New("invalid, expired, or already-used game token")

var ErrScoreExceedsElapsedTime = errors.New("score exceeds the maximum possible for the elapsed time")
