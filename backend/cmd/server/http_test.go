// Package main tests the assembled server through its public HTTP routes.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"dazzled/backend/api/rest/v1/core"
	"dazzled/backend/internal/domain"
	"dazzled/backend/internal/pkg/database/migrations"
	"dazzled/backend/internal/pkg/database/sqlite"
	incidentservice "dazzled/backend/internal/pkg/services/incidents"
	leaderboardservice "dazzled/backend/internal/pkg/services/leaderboard"
)

func testServer(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "test.db")+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.Apply(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatal("SQLite foreign-key enforcement is disabled")
	}
	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode=%q, want wal", journalMode)
	}
	repository := sqlite.New(db)
	return db, core.NewRouter(db, "", incidentservice.New(repository), leaderboardservice.New(repository, []byte("test-game-token-secret-that-is-long-enough")), false)
}

func call(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestIncidentSaveAndStatistics(t *testing.T) {
	db, handler := testServer(t)
	response := call(handler, http.MethodPost, "/api/incidents", `{"vehicle_type":"SUV / crossover","brand":"BMW","severity":4,"location":"Sarajevo","local_hour":19}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d; body=%s", response.Code, response.Body.String())
	}
	var stored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE brand = ? AND severity = ?`, "BMW", 4).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 1 {
		t.Fatalf("persisted count = %d, want 1", stored)
	}
	response = call(handler, http.MethodGet, "/api/stats?period=all", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET stats status = %d", response.Code)
	}
	var result struct {
		Total      int                  `json:"total"`
		Brands     []domain.Ranking     `json:"brands"`
		Vehicles   []domain.Ranking     `json:"vehicles"`
		LocalHours []domain.HourlyCount `json:"local_hours"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || len(result.Brands) != 1 || result.Brands[0].Name != "BMW" || len(result.Vehicles) != 1 {
		t.Fatalf("unexpected statistics: %+v", result)
	}
	if len(result.LocalHours) != 24 || result.LocalHours[19].Count != 1 {
		t.Fatalf("unexpected hourly aggregates: %+v", result.LocalHours)
	}
	if strings.Contains(response.Body.String(), "Sarajevo") {
		t.Fatal("statistics API exposed the individual report location")
	}
}

func TestIncidentRejectsInvalidAndUnknownInput(t *testing.T) {
	_, handler := testServer(t)
	for _, body := range []string{`{"vehicle_type":"SUV","brand":"BMW","severity":9}`, `{"vehicle_type":"SUV","brand":"BMW","severity":2,"local_hour":24}`, `{"vehicle_type":"SUV","brand":"BMW","severity":2,"owner":"extra"}`, `{}`} {
		response := call(handler, http.MethodPost, "/api/incidents", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("body %s: status=%d want 400", body, response.Code)
		}
	}
}

func TestRequestDecodingRejectsWrongContentTypeAndOversizedBody(t *testing.T) {
	_, handler := testServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewBufferString(`{"vehicle_type":"SUV","brand":"BMW","severity":2}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type status=%d, want 415", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewBufferString(strings.Repeat(" ", 17*1024)))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request status=%d, want 413", response.Code)
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	_, handler := testServer(t)
	response := call(handler, http.MethodGet, "/healthz", "")
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("response is missing X-Content-Type-Options: nosniff")
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("response is missing X-Frame-Options: DENY")
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal("response CSP is missing frame-ancestors 'none'")
	}
}

func TestLeaderboardPersistsAndRanksScores(t *testing.T) {
	db, handler := testServer(t)
	for index, score := range []domain.Score{{Username: "slow driver", Distance: 0.1}, {Username: "night owl", Distance: 0.2}} {
		startResponse := call(handler, http.MethodPost, "/api/leaderboard/start", "")
		if startResponse.Code != http.StatusCreated {
			t.Fatalf("POST game start status=%d body=%s", startResponse.Code, startResponse.Body.String())
		}
		var session struct {
			Token string `json:"game_token"`
		}
		if err := json.Unmarshal(startResponse.Body.Bytes(), &session); err != nil {
			t.Fatal(err)
		}
		bodyBytes, err := json.Marshal(map[string]any{"username": score.Username, "distance": score.Distance, "game_token": session.Token})
		if err != nil {
			t.Fatal(err)
		}
		response := call(handler, http.MethodPost, "/api/leaderboard", string(bodyBytes))
		if response.Code != http.StatusCreated {
			t.Fatalf("POST score status = %d body=%s", response.Code, response.Body.String())
		}
		if index == 0 {
			replay := call(handler, http.MethodPost, "/api/leaderboard", string(bodyBytes))
			if replay.Code != http.StatusBadRequest {
				t.Fatalf("replayed game token status=%d, want 400", replay.Code)
			}
		}
	}
	response := call(handler, http.MethodGet, "/api/leaderboard", "")
	var payload struct {
		Scores []domain.Score `json:"scores"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(payload.Scores) != 2 || payload.Scores[0].Username != "night owl" {
		t.Fatalf("unexpected leaderboard: status=%d scores=%+v", response.Code, payload.Scores)
	}
	if _, err := db.Exec(`INSERT INTO leaderboard_scores(username, distance) VALUES(?, ?)`, "invalid", -1); err == nil {
		t.Fatal("database accepted negative distance")
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	db, _ := testServer(t)
	if err := migrations.Apply(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("migration count=%d, want 5", count)
	}
}

func TestDataSurvivesDatabaseReopen(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "persistent.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite3", "file:"+databasePath+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		if err := migrations.Apply(context.Background(), db); err != nil {
			t.Fatal(err)
		}
		return db
	}
	first := open()
	service := incidentservice.New(sqlite.New(first))
	if _, err := service.Create(context.Background(), domain.Incident{VehicleType: "Sedan", Brand: "Audi", Severity: 2}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second := open()
	defer second.Close()
	stats, err := incidentservice.New(sqlite.New(second)).Statistics(context.Background(), "all")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 1 {
		t.Fatalf("after reopening, total=%d, want 1", stats.Total)
	}
}

func TestStaticRoutesAndTraversalProtection(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "fine"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("home"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fine", "index.html"), []byte("fine page"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := core.NewRouter(nil, root, nil, nil, false)
	for path, expected := range map[string]string{"/": "home", "/fine": "fine page"} {
		response := call(handler, http.MethodGet, path, "")
		if response.Code != http.StatusOK || response.Body.String() != expected {
			t.Errorf("GET %s: status=%d body=%q", path, response.Code, response.Body.String())
		}
	}
	for _, path := range []string{"/../../etc/passwd", "/%2e%2e/%2e%2e/etc/passwd"} {
		response := call(handler, http.MethodGet, path, "")
		if response.Code != http.StatusNotFound {
			t.Errorf("traversal path %q returned status %d", path, response.Code)
		}
	}
}

func TestConcurrentIncidentSubmissionsRemainConsistent(t *testing.T) {
	db, handler := testServer(t)
	const submissions = 32
	var group sync.WaitGroup
	errors := make(chan string, submissions)
	for i := 0; i < submissions; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			body := `{"vehicle_type":"SUV / crossover","brand":"BMW","severity":` + strconv.Itoa(index%5+1) + `}`
			response := call(handler, http.MethodPost, "/api/incidents", body)
			if response.Code != http.StatusCreated {
				errors <- response.Body.String()
			}
		}(i)
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent submit failed: %s", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM incidents`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != submissions {
		t.Fatalf("stored %d concurrent incidents, want %d", count, submissions)
	}
}
