// Package main wires app dependencies and manages startup, serving, and graceful shutdown.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dazzled/backend/api/rest/v1/core"
	"dazzled/backend/internal/pkg/config"
	"dazzled/backend/internal/pkg/database/migrations"
	"dazzled/backend/internal/pkg/database/sqlite"
	incidentservice "dazzled/backend/internal/pkg/services/incidents"
	leaderboardservice "dazzled/backend/internal/pkg/services/leaderboard"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	settings := config.Load()
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStartup()
	db, err := sqlite.Open(startupContext, settings.DatabasePath)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()
	if err := migrations.Apply(startupContext, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if len(os.Args) > 1 && os.Args[1] == "--migrate-only" {
		return nil
	}

	repository := sqlite.New(db)
	incidentUseCases := incidentservice.New(repository)
	tokenSecret := []byte(settings.GameTokenSecret)
	if len(tokenSecret) == 0 {
		tokenSecret = make([]byte, 32)
		if _, err := rand.Read(tokenSecret); err != nil {
			return fmt.Errorf("generate game token signing key: %w", err)
		}
		slog.Warn("using an ephemeral game token key; unfinished runs will expire after restart")
	} else if len(tokenSecret) < 32 {
		return fmt.Errorf("GAME_TOKEN_SECRET must contain at least 32 bytes")
	}
	leaderboardUseCases := leaderboardservice.New(repository, tokenSecret)
	handler := core.NewRouter(db, settings.WebDirectory, incidentUseCases, leaderboardUseCases, settings.TrustProxyHeaders)
	server := &http.Server{
		Addr:              settings.ListenAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("server listening", "address", settings.ListenAddress, "database", settings.DatabasePath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("serve HTTP: %w", err)
		}
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}
