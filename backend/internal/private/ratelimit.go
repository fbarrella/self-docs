package private

import (
	"sync"
	"time"
)

// RateLimit constants from api.md 7.1: 5 failed unlock attempts within 15
// minutes trigger a lockout for the same window.
const (
	MaxUnlockAttempts = 5
	UnlockWindow      = 15 * time.Minute
)

// RateLimiter tracks failed unlock attempts per client key. It is a fixed
// window counter, which is sufficient for a single-user local deployment.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
	now      func() time.Time
}

// NewRateLimiter constructs a RateLimiter.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max < 1 {
		max = MaxUnlockAttempts
	}
	if window <= 0 {
		window = UnlockWindow
	}
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
		now:      time.Now,
	}
}

// Allow reports whether the key may attempt an unlock now.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	recent := r.recentLocked(key)
	return len(recent) < r.max
}

// RecordFailure records a failed attempt, returning the remaining attempts
// before lockout (0 when now locked out).
func (r *RateLimiter) RecordFailure(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	recent := append(r.recentLocked(key), r.now())
	r.attempts[key] = recent

	remaining := r.max - len(recent)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// Reset clears the failure history for a key after a successful unlock.
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.attempts, key)
}

// RetryAfter returns how long the key must wait before another attempt.
func (r *RateLimiter) RetryAfter(key string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	recent := r.recentLocked(key)
	if len(recent) < r.max {
		return 0
	}
	// The oldest attempt in the window expires first.
	oldest := recent[len(recent)-r.max]
	return oldest.Add(r.window).Sub(r.now())
}

// recentLocked returns non-expired attempts; caller must hold the mutex.
func (r *RateLimiter) recentLocked(key string) []time.Time {
	cutoff := r.now().Add(-r.window)
	all := r.attempts[key]
	kept := all[:0]
	for _, t := range all {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(r.attempts, key)
		return nil
	}
	r.attempts[key] = kept
	return kept
}
