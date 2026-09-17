package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/private"
)

// PrivateCookieName is the HttpOnly cookie carrying a private session token.
const PrivateCookieName = "selfdocs_private"

// PrivateAuth holds the session store used to guard private routes.
type PrivateAuth struct {
	Sessions *private.SessionStore
	// SecureCookies sets the Secure flag; false for local HTTP development.
	SecureCookies bool
}

// NewPrivateAuth constructs the middleware helper.
func NewPrivateAuth(sessions *private.SessionStore, secureCookies bool) *PrivateAuth {
	return &PrivateAuth{Sessions: sessions, SecureCookies: secureCookies}
}

// tokenFromRequest reads the bearer token, falling back to the cookie.
func tokenFromRequest(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); header != "" {
		const prefix = "Bearer "
		if len(header) > len(prefix) && header[:len(prefix)] == prefix {
			return header[len(prefix):]
		}
	}
	if cookie, err := c.Cookie(PrivateCookieName); err == nil {
		return cookie
	}
	return ""
}

// Middleware rejects requests without a valid, unexpired private session. It
// slides the session expiry on each successful request.
func (a *PrivateAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := tokenFromRequest(c)
		valid, expiresAt := a.Sessions.Touch(token)
		if !valid {
			respondError(c, http.StatusUnauthorized, CodeUnauthorized,
				"private archive is locked")
			return
		}
		c.Set("private_token", token)
		c.Set("private_expires_at", expiresAt)
		// Refresh the cookie so browser sessions keep sliding.
		a.SetCookie(c, token, expiresAt)
		c.Next()
	}
}

// OptionalMiddleware resolves a session token when present but never rejects
// the request. It is used by the lock route so an expired session can still
// clear its cookie.
func (a *PrivateAuth) OptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := tokenFromRequest(c)
		if valid, expiresAt := a.Sessions.Touch(token); valid {
			c.Set("private_token", token)
			c.Set("private_expires_at", expiresAt)
		}
		c.Next()
	}
}

// SetCookie writes the sliding HttpOnly session cookie.
func (a *PrivateAuth) SetCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetCookie(PrivateCookieName, token, int(time.Until(expiresAt).Seconds()), "/api/private", "", a.SecureCookies, true)
}

// ClearCookie removes the session cookie on lock.
func (a *PrivateAuth) ClearCookie(c *gin.Context) {
	c.SetCookie(PrivateCookieName, "", -1, "/api/private", "", a.SecureCookies, true)
}
