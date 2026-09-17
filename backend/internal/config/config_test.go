package config

import "testing"

func TestLoadMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/selfdocs")
	t.Setenv("PORT", "")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("MASTER_PASSWORD_HASH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.Addr() != ":8080" {
		t.Errorf("Addr() = %q, want :8080", cfg.Addr())
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "http://localhost:5173" {
		t.Errorf("CORSOrigins = %v, want default origin", cfg.CORSOrigins)
	}
	if cfg.RedisEnabled() {
		t.Error("RedisEnabled() = true, want false")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/selfdocs")
	t.Setenv("PORT", "not-a-port")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid PORT")
	}
}

func TestLoadCORSOriginsSplit(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/selfdocs")
	t.Setenv("CORS_ORIGINS", "http://a.test, http://b.test ,")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("CORSOrigins = %v, want 2 entries", cfg.CORSOrigins)
	}
	if cfg.CORSOrigins[0] != "http://a.test" || cfg.CORSOrigins[1] != "http://b.test" {
		t.Errorf("CORSOrigins = %v, want trimmed entries", cfg.CORSOrigins)
	}
}

func TestRedisEnabled(t *testing.T) {
	cfg := &Config{RedisURL: "redis://localhost:6379"}
	if !cfg.RedisEnabled() {
		t.Error("RedisEnabled() = false, want true")
	}
}
