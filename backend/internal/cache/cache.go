// Package cache provides an optional Redis-backed cache. When Redis is not
// configured or unreachable, every operation degrades to a miss so the
// application runs normally (PRD 2, 5; DEVELOPMENT_PLAN T5.4).
package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultTTL bounds how long cached reads may be stale if an invalidation is
// ever missed.
const DefaultTTL = 5 * time.Minute

// Cache is a thin JSON cache over Redis. The zero value (nil client) is a
// valid no-op cache, so callers never need nil checks.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
	// prefix namespaces keys so multiple deployments can share a Redis.
	prefix string
}

// New connects to Redis at url. An empty url, or a failed ping, returns a
// disabled cache rather than an error: caching must never block startup.
func New(ctx context.Context, url, prefix string) *Cache {
	if url == "" {
		return &Cache{prefix: prefix, ttl: DefaultTTL}
	}

	options, err := redis.ParseURL(url)
	if err != nil {
		log.Printf("cache: invalid REDIS_URL, running without cache: %v", err)
		return &Cache{prefix: prefix, ttl: DefaultTTL}
	}

	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		log.Printf("cache: redis unreachable, running without cache: %v", err)
		_ = client.Close()
		return &Cache{prefix: prefix, ttl: DefaultTTL}
	}

	log.Printf("cache: connected to redis")
	return &Cache{client: client, ttl: DefaultTTL, prefix: prefix}
}

// Enabled reports whether a live Redis client is attached.
func (c *Cache) Enabled() bool {
	return c != nil && c.client != nil
}

// Close releases the Redis connection, if any.
func (c *Cache) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.client.Close()
}

// GetJSON loads key into dst. It returns false on a miss, disabled cache, or
// any Redis error, so callers fall through to the source of truth.
func (c *Cache) GetJSON(ctx context.Context, key string, dst any) bool {
	if !c.Enabled() {
		return false
	}
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, dst); err != nil {
		// Corrupt entry: drop it and report a miss.
		c.Delete(ctx, key)
		return false
	}
	return true
}

// SetJSON stores value under key. Errors are logged and swallowed.
func (c *Cache) SetJSON(ctx context.Context, key string, value any) {
	if !c.Enabled() {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("cache: marshal %s: %v", key, err)
		return
	}
	if err := c.client.Set(ctx, c.key(key), data, c.ttl).Err(); err != nil {
		log.Printf("cache: set %s: %v", key, err)
	}
}

// Delete removes the given keys. Errors are logged and swallowed.
func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}
	full := make([]string, len(keys))
	for i, key := range keys {
		full[i] = c.key(key)
	}
	if err := c.client.Del(ctx, full...).Err(); err != nil {
		log.Printf("cache: delete %v: %v", keys, err)
	}
}

func (c *Cache) key(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + ":" + key
}
