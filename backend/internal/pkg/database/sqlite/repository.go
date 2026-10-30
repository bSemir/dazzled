package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"dazzled/backend/internal/domain"
	incidentsport "dazzled/backend/internal/pkg/ports/outbound/incidents"
	leaderboardport "dazzled/backend/internal/pkg/ports/outbound/leaderboard"
)

type Repository struct {
	db *sql.DB
}

var _ incidentsport.Repository = (*Repository)(nil)
var _ leaderboardport.Repository = (*Repository)(nil)

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, incident domain.Incident) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO incidents(vehicle_type, brand, location, severity, occurred_local_hour) VALUES(?, ?, ?, ?, ?)`,
		incident.VehicleType, incident.Brand, incident.Location, incident.Severity, incident.LocalHour,
	)
	if err != nil {
		return 0, fmt.Errorf("insert incident: %w", err)
	}
	return result.LastInsertId()
}

func (r *Repository) Statistics(ctx context.Context, period string) (domain.Statistics, error) {
	cutoff := "1970-01-01T00:00:00Z"
	switch period {
	case "today":
		cutoff = time.Now().UTC().Format("2006-01-02") + "T00:00:00Z"
	case "week":
		cutoff = time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z")
	}

	statistics := domain.Statistics{
		Period:   period,
		Brands:   make([]domain.Ranking, 0),
		Vehicles: make([]domain.Ranking, 0),
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(created_at >= strftime('%Y-%m-%dT00:00:00Z','now')),0) FROM incidents`,
	).Scan(&statistics.Total, &statistics.Today); err != nil {
		return domain.Statistics{}, fmt.Errorf("count incidents: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM incidents WHERE created_at >= ?`, cutoff).Scan(&statistics.FilteredCount); err != nil {
		return domain.Statistics{}, fmt.Errorf("count filtered incidents: %w", err)
	}

	var err error
	statistics.Brands, err = r.rankings(ctx, "brand", cutoff)
	if err != nil {
		return domain.Statistics{}, fmt.Errorf("rank incident brands: %w", err)
	}
	statistics.Vehicles, err = r.rankings(ctx, "vehicle_type", cutoff)
	if err != nil {
		return domain.Statistics{}, fmt.Errorf("rank incident vehicle types: %w", err)
	}
	statistics.LocalHours, err = r.localHours(ctx, "1970-01-01T00:00:00Z")
	if err != nil {
		return domain.Statistics{}, fmt.Errorf("count incidents by local hour: %w", err)
	}
	return statistics, nil
}

func (r *Repository) localHours(ctx context.Context, cutoff string) ([]domain.HourlyCount, error) {
	hours := make([]domain.HourlyCount, 24)
	for hour := range hours {
		hours[hour].Hour = hour
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT occurred_local_hour, COUNT(*) FROM incidents WHERE occurred_local_hour IS NOT NULL AND created_at >= ? GROUP BY occurred_local_hour ORDER BY occurred_local_hour`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query hourly counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var hour domain.HourlyCount
		if err := rows.Scan(&hour.Hour, &hour.Count); err != nil {
			return nil, fmt.Errorf("scan hourly count: %w", err)
		}
		hours[hour.Hour] = hour
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly counts: %w", err)
	}
	return hours, nil
}

func (r *Repository) rankings(ctx context.Context, column, cutoff string) ([]domain.Ranking, error) {
	if column != "brand" && column != "vehicle_type" {
		return nil, fmt.Errorf("unsupported ranking column %q", column)
	}
	query := `SELECT ` + column + `, COUNT(*) FROM incidents WHERE created_at >= ? GROUP BY ` + column + ` ORDER BY COUNT(*) DESC, ` + column + ` ASC LIMIT 8`
	rows, err := r.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query rankings: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Ranking, 0)
	for rows.Next() {
		var item domain.Ranking
		if err := rows.Scan(&item.Name, &item.Value); err != nil {
			return nil, fmt.Errorf("scan ranking: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rankings: %w", err)
	}
	return items, nil
}

func (r *Repository) Submit(ctx context.Context, score domain.Score) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO leaderboard_scores(username, distance) VALUES(?, ?)`,
		score.Username, score.Distance,
	); err != nil {
		return fmt.Errorf("insert leaderboard score: %w", err)
	}
	return nil
}

func (r *Repository) StartGame(ctx context.Context, nonce string, startedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin game session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM leaderboard_game_sessions WHERE started_at_ns < ?`, startedAt.Add(-24*time.Hour).UnixNano()); err != nil {
		return fmt.Errorf("remove expired game sessions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO leaderboard_game_sessions(nonce, started_at_ns) VALUES(?, ?)`, nonce, startedAt.UnixNano()); err != nil {
		return fmt.Errorf("insert game session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit game session: %w", err)
	}
	return nil
}

func (r *Repository) ConsumeGame(ctx context.Context, nonce string, now time.Time) (time.Time, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("begin game session use: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var startedAtNS int64
	var usedAtNS sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT started_at_ns, used_at_ns FROM leaderboard_game_sessions WHERE nonce = ?`, nonce).Scan(&startedAtNS, &usedAtNS); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, leaderboardport.ErrGameSessionUnavailable
		}
		return time.Time{}, fmt.Errorf("read game session: %w", err)
	}
	if usedAtNS.Valid {
		return time.Time{}, leaderboardport.ErrGameSessionUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE leaderboard_game_sessions SET used_at_ns = ? WHERE nonce = ? AND used_at_ns IS NULL`, now.UnixNano(), nonce)
	if err != nil {
		return time.Time{}, fmt.Errorf("mark game session used: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, fmt.Errorf("check game session update: %w", err)
	}
	if rowsAffected != 1 {
		return time.Time{}, leaderboardport.ErrGameSessionUnavailable
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, fmt.Errorf("commit game session use: %w", err)
	}
	return time.Unix(0, startedAtNS).UTC(), nil
}

func (r *Repository) Top(ctx context.Context) ([]domain.Score, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT username, distance FROM leaderboard_scores ORDER BY distance DESC, created_at ASC LIMIT 10`,
	)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	scores := make([]domain.Score, 0)
	for rows.Next() {
		var score domain.Score
		if err := rows.Scan(&score.Username, &score.Distance); err != nil {
			return nil, fmt.Errorf("scan leaderboard score: %w", err)
		}
		scores = append(scores, score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard scores: %w", err)
	}
	return scores, nil
}
