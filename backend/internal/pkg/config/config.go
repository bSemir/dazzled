package config

import (
	"os"
	"strconv"
)

type Config struct {
	ListenAddress     string
	DatabasePath      string
	WebDirectory      string
	TrustProxyHeaders bool
	GameTokenSecret   string
}

func Load() Config {
	return Config{
		ListenAddress:     valueOrDefault("LISTEN_ADDR", ":8080"),
		DatabasePath:      valueOrDefault("DATABASE_PATH", "./data/dazzled.db"),
		WebDirectory:      valueOrDefault("WEB_DIR", "../frontend/out"),
		TrustProxyHeaders: boolValueOrDefault("TRUST_PROXY_HEADERS", false),
		GameTokenSecret:   os.Getenv("GAME_TOKEN_SECRET"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolValueOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
