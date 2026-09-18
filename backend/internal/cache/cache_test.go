package cache

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDisabledCacheIsNoOp(t *testing.T) {
	ctx := context.Background()

	// Nil cache must behave as a permanent miss.
	var nilCache *Cache
	if nilCache.Enabled() {
		t.Error("nil cache should be disabled")
	}
	var dst any
	if nilCache.GetJSON(ctx, "x", &dst) {
		t.Error("nil cache should always miss")
	}
	nilCache.SetJSON(ctx, "x", map[string]string{"a": "b"})
	nilCache.Delete(ctx, "x")
	nilCache.InvalidateDocuments(ctx)
	if err := nilCache.Close(); err != nil {
		t.Errorf("nil cache Close: %v", err)
	}

	// Empty-URL cache (disabled) must also be a no-op.
	disabled := New(ctx, "", "test")
	if disabled.Enabled() {
		t.Error("cache with empty URL should be disabled")
	}
	if disabled.GetJSON(ctx, "x", &dst) {
		t.Error("disabled cache should always miss")
	}
}

func TestInvalidRedisURLFallsBack(t *testing.T) {
	c := New(context.Background(), "not-a-redis-url", "test")
	if c.Enabled() {
		t.Error("invalid URL should yield a disabled cache")
	}
}

func TestCacheRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping cache integration test")
	}

	ctx := context.Background()
	c := New(ctx, url, "selfdocs-test")
	if !c.Enabled() {
		t.Fatal("expected Redis to be enabled")
	}
	t.Cleanup(func() { _ = c.Close() })

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	want := payload{Name: "git", Count: 3}
	c.SetJSON(ctx, "roundtrip", want)

	var got payload
	if !c.GetJSON(ctx, "roundtrip", &got) {
		t.Fatal("expected a cache hit")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	c.Delete(ctx, "roundtrip")
	if c.GetJSON(ctx, "roundtrip", &got) {
		t.Error("expected a miss after delete")
	}
}

func TestInvalidateDocuments(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping cache integration test")
	}

	ctx := context.Background()
	c := New(ctx, url, "selfdocs-test")
	if !c.Enabled() {
		t.Fatal("expected Redis to be enabled")
	}
	t.Cleanup(func() { _ = c.Close() })

	c.SetJSON(ctx, KeyDashboard, map[string]int{"x": 1})
	c.SetJSON(ctx, KeyPopularTags, map[string]int{"y": 2})
	c.InvalidateDocuments(ctx)

	var dst map[string]int
	if c.GetJSON(ctx, KeyDashboard, &dst) {
		t.Error("dashboard should be invalidated")
	}
	if c.GetJSON(ctx, KeyPopularTags, &dst) {
		t.Error("popular tags should be invalidated")
	}
}

// TestCacheTTLConfigured documents the default TTL constant.
func TestCacheTTLConfigured(t *testing.T) {
	if DefaultTTL <= 0 {
		t.Error("DefaultTTL must be positive")
	}
	if DefaultTTL > time.Hour {
		t.Error("DefaultTTL should stay short to bound staleness")
	}
}
