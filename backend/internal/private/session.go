package private

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// SessionTTL is the sliding lifetime of an unlocked session (api.md 7.1).
const SessionTTL = 15 * time.Minute

// session is one unlocked Private Archive session.
type session struct {
	expiresAt time.Time
}

// SessionStore keeps in-memory session tokens. This suits a single-user,
// single-process, local-first deployment (PRD 5); sessions are intentionally
// lost on restart.
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
	ttl      time.Duration
	now      func() time.Time
}

// NewSessionStore constructs a SessionStore with the given TTL.
func NewSessionStore(ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = SessionTTL
	}
	return &SessionStore{
		sessions: make(map[string]session),
		ttl:      ttl,
		now:      time.Now,
	}
}

// Create issues a new opaque token with a sliding expiry.
func (s *SessionStore) Create() (token string, expiresAt time.Time, err error) {
	token, err = generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = s.now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	s.sessions[token] = session{expiresAt: expiresAt}
	return token, expiresAt, nil
}

// Touch validates a token and, when valid, refreshes its expiry. It reports
// whether the token is currently valid and the new expiry.
func (s *SessionStore) Touch(token string) (bool, time.Time) {
	if token == "" {
		return false, time.Time{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[token]
	if !ok {
		return false, time.Time{}
	}
	if s.now().After(sess.expiresAt) {
		delete(s.sessions, token)
		return false, time.Time{}
	}

	sess.expiresAt = s.now().Add(s.ttl)
	s.sessions[token] = sess
	return true, sess.expiresAt
}

// Valid reports whether a token is currently valid without refreshing its
// expiry. Used by read-only endpoints that only need the locked/unlocked state.
func (s *SessionStore) Valid(token string) bool {
	if token == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[token]
	if !ok {
		return false
	}
	if s.now().After(sess.expiresAt) {
		delete(s.sessions, token)
		return false
	}
	return true
}

// Invalidate removes a token (lock).
func (s *SessionStore) Invalidate(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// Clear invalidates every session. Used when the master password changes so
// previously unlocked sessions cannot continue.
func (s *SessionStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]session)
}

// prune removes expired sessions; called opportunistically by Create.
func (s *SessionStore) prune() {
	now := s.now()
	for token, sess := range s.sessions {
		if now.After(sess.expiresAt) {
			delete(s.sessions, token)
		}
	}
}

func generateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
