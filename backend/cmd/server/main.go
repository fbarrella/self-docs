// Command server is the self-docs API entrypoint. It loads configuration,
// connects to PostgreSQL, applies pending migrations, and serves the HTTP API
// with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/cache"
	"github.com/self-docs/backend/internal/config"
	"github.com/self-docs/backend/internal/db"
	"github.com/self-docs/backend/internal/router"
	"github.com/self-docs/backend/migrations"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.Apply(ctx, pool.Pool); err != nil {
		return err
	}
	log.Printf("migrations up to date")

	if err := seedMasterPassword(ctx, pool, cfg.MasterPasswordHash); err != nil {
		return err
	}

	// Optional Redis cache; a disabled cache is a safe no-op.
	c := cache.New(ctx, cfg.RedisURL, "selfdocs")
	defer func() { _ = c.Close() }()
	if cfg.RedisEnabled() && !c.Enabled() {
		log.Printf("warning: REDIS_URL set but Redis is unreachable; continuing without cache")
	}

	engine := router.New(router.Deps{
		Pool:           pool.Pool,
		RedisEnabled:   cfg.RedisEnabled(),
		Version:        version,
		CORSOrigins:    cfg.CORSOrigins,
		Cache:          c,
		TrustedProxies: cfg.TrustedProxies,
	})

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("self-docs backend %s listening on %s", version, cfg.Addr())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Printf("server stopped")
	return nil
}

// seedMasterPassword stores MASTER_PASSWORD_HASH in settings when provided and
// not already set. An empty value is ignored so a hash can be configured later
// through the settings API.
func seedMasterPassword(ctx context.Context, pool *db.Pool, hash string) error {
	if hash == "" {
		return nil
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ('master_password_hash', $1, now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()
		WHERE settings.value = ''`, hash)
	if err != nil {
		return err
	}
	return nil
}

// ensure gin uses release mode when not in development.
func init() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
}
