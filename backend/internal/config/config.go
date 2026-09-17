// Package config loads application configuration from environment variables.
//
// A local backend/.env file is loaded if present (without overriding variables
// already set in the process environment), which keeps local development
// dependency-free. Load fails fast when required configuration is missing.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string (required).
	DatabaseURL string
	// Port is the HTTP listen port (default "8080").
	Port string
	// MasterPasswordHash seeds the Private Archive hash into the settings table
	// when non-empty. It is optional at boot: a hash can also be set later via
	// the settings API (T2.5 / T4.8).
	MasterPasswordHash string
	// RedisURL is optional; when empty the cache layer is disabled.
	RedisURL string
	// CORSOrigins is the list of allowed browser origins.
	CORSOrigins []string
	// ReadTimeout, WriteTimeout and ShutdownTimeout bound server I/O and
	// graceful shutdown.
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	// MaxImportBytes caps a single imported file. Defaults to 2 MiB.
	MaxImportBytes int64
}

const (
	defaultPort            = "8080"
	defaultCORSOrigin      = "http://localhost:5173"
	defaultReadTimeout     = 15 * time.Second
	defaultWriteTimeout    = 30 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultMaxImportBytes  = 2 << 20 // 2 MiB
)

// Load reads configuration from the environment, loading backend/.env first
// when present. It returns an error listing every missing required variable.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Port:               getEnv("PORT", defaultPort),
		MasterPasswordHash: os.Getenv("MASTER_PASSWORD_HASH"),
		RedisURL:           os.Getenv("REDIS_URL"),
		CORSOrigins:        splitAndTrim(getEnv("CORS_ORIGINS", defaultCORSOrigin)),
		ReadTimeout:        defaultReadTimeout,
		WriteTimeout:       defaultWriteTimeout,
		ShutdownTimeout:    defaultShutdownTimeout,
		MaxImportBytes:     defaultMaxImportBytes,
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("PORT must be an integer between 1 and 65535, got %q", cfg.Port)
	}

	return cfg, nil
}

// Addr returns the TCP address the HTTP server listens on.
func (c *Config) Addr() string {
	return ":" + c.Port
}

// RedisEnabled reports whether a Redis URL was configured.
func (c *Config) RedisEnabled() bool {
	return strings.TrimSpace(c.RedisURL) != ""
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// loadDotEnv parses a minimal KEY=VALUE .env file. Existing environment
// variables always win, and a missing file is not an error.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
