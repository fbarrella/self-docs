// Package private implements Private Archive access control: master-password
// verification, short-lived sliding sessions, and unlock rate limiting
// (PRD 3.2, 5; docs/api.md section 7).
package private

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost balances security and local-first responsiveness.
const bcryptCost = bcrypt.DefaultCost

// ErrPasswordTooLong is returned when a password exceeds bcrypt's 72-byte limit.
var ErrPasswordTooLong = errors.New("password exceeds maximum length")

// HashPassword returns a bcrypt hash for storage in settings.
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		return "", ErrPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a stored bcrypt hash in
// constant time.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
