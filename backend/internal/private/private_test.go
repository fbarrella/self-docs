package private

import (
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash must not equal the plaintext")
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Error("VerifyPassword rejected the correct password")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Error("VerifyPassword accepted a wrong password")
	}
}

func TestHashPasswordTooLong(t *testing.T) {
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := HashPassword(string(long)); err == nil {
		t.Error("expected error for password longer than 72 bytes")
	}
}

func TestSessionCreateAndTouch(t *testing.T) {
	store := NewSessionStore(time.Minute)
	token, _, err := store.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	valid, _ := store.Touch(token)
	if !valid {
		t.Error("new token should be valid")
	}
	store.Invalidate(token)
	if valid, _ := store.Touch(token); valid {
		t.Error("invalidated token should be rejected")
	}
}

func TestSessionExpiry(t *testing.T) {
	store := NewSessionStore(time.Minute)
	base := time.Now()
	store.now = func() time.Time { return base }

	token, _, _ := store.Create()
	store.now = func() time.Time { return base.Add(30 * time.Second) }
	if valid, _ := store.Touch(token); !valid {
		t.Error("token should be valid before expiry")
	}

	store.now = func() time.Time { return base.Add(2 * time.Minute) }
	if valid, _ := store.Touch(token); valid {
		t.Error("token should be expired")
	}
}

func TestSessionSlidingRefresh(t *testing.T) {
	store := NewSessionStore(time.Minute)
	base := time.Now()
	store.now = func() time.Time { return base }

	token, first, _ := store.Create()
	store.now = func() time.Time { return base.Add(50 * time.Second) }
	valid, refreshed := store.Touch(token)
	if !valid {
		t.Fatal("token should be valid")
	}
	if !refreshed.After(first) {
		t.Error("touch should extend the expiry (sliding session)")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)
	base := time.Now()
	limiter.now = func() time.Time { return base }

	for i := 0; i < 3; i++ {
		if !limiter.Allow("client") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		limiter.RecordFailure("client")
	}
	if limiter.Allow("client") {
		t.Error("client should be locked out after max failures")
	}

	// A different key is unaffected.
	if !limiter.Allow("other") {
		t.Error("unrelated key should still be allowed")
	}

	// After the window elapses the lock clears.
	limiter.now = func() time.Time { return base.Add(2 * time.Minute) }
	if !limiter.Allow("client") {
		t.Error("lockout should expire after the window")
	}
}

func TestRateLimiterReset(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	limiter.RecordFailure("client")
	limiter.RecordFailure("client")
	if limiter.Allow("client") {
		t.Fatal("expected lockout")
	}
	limiter.Reset("client")
	if !limiter.Allow("client") {
		t.Error("Reset should clear failures")
	}
}
