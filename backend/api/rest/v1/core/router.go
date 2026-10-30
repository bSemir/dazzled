package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dazzled/backend/api/rest/v1/handlers/incidents"
	"dazzled/backend/api/rest/v1/handlers/leaderboard"
	incidentport "dazzled/backend/api/rest/v1/ports/inbound/incidents"
	leaderboardport "dazzled/backend/api/rest/v1/ports/inbound/leaderboard"
)

func NewRouter(db *sql.DB, webDirectory string, incidentService incidentport.Service, leaderboardService leaderboardport.Service, trustProxyHeaders bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", readiness(db))
	incidents.New(incidentService).RegisterRoutes(mux)
	leaderboard.New(leaderboardService).RegisterRoutes(mux)
	if webDirectory != "" {
		mux.Handle("/", staticHandler(webDirectory))
	}
	return securityHeaders(requestLog(newWriteRateLimiter(trustProxyHeaders).handler(rejectPathTraversal(mux))))
}

func rejectPathTraversal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		for _, segment := range strings.Split(request.URL.Path, "/") {
			if segment == ".." {
				http.NotFound(writer, request)
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

func readiness(db *sql.DB) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), time.Second)
		defer cancel()
		var migrationCount int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
			http.Error(writer, "not ready", http.StatusServiceUnavailable)
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ready"))
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(writer, request)
	})
}

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		var requestID [16]byte
		if _, err := rand.Read(requestID[:]); err != nil {
			slog.ErrorContext(request.Context(), "generate request id failed", "error", err)
		}
		id := hex.EncodeToString(requestID[:])
		writer.Header().Set("X-Request-ID", id)
		status := &statusWriter{ResponseWriter: writer}
		next.ServeHTTP(status, request)
		slog.InfoContext(request.Context(), "http request",
			"request_id", id,
			"method", request.Method,
			"path", request.URL.Path,
			"status", status.code(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (writer *statusWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.status = status
	writer.wroteHeader = true
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *statusWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func (writer *statusWriter) code() int {
	if !writer.wroteHeader {
		return http.StatusOK
	}
	return writer.status
}

func staticHandler(root string) http.Handler {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return http.NotFoundHandler()
	}
	files := http.FileServer(http.Dir(absoluteRoot))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(writer, request)
			return
		}
		cleanPath := strings.TrimPrefix(filepath.Clean("/"+request.URL.Path), string(filepath.Separator))
		path := filepath.Join(absoluteRoot, cleanPath)
		relative, err := filepath.Rel(absoluteRoot, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			http.NotFound(writer, request)
			return
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			if filepath.Ext(path) == ".html" {
				writer.Header().Set("Cache-Control", "no-cache, must-revalidate")
			}
			files.ServeHTTP(writer, request)
			return
		}
		if cleanPath != "" {
			if _, err := os.Stat(path + ".html"); err == nil {
				writer.Header().Set("Cache-Control", "no-cache, must-revalidate")
				http.ServeFile(writer, request, path+".html")
				return
			}
		}
		indexFile := filepath.Join(path, "index.html")
		if _, err := os.Stat(indexFile); err == nil {
			writer.Header().Set("Cache-Control", "no-cache, must-revalidate")
			http.ServeFile(writer, request, indexFile)
			return
		}
		http.NotFound(writer, request)
	})
}
