// Package middleware provides cross-cutting HTTP concerns for the API:
// request IDs, structured access logging, security headers, request body
// limits, and panic recovery.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID is the response header carrying the per-request identifier.
const HeaderRequestID = "X-Request-ID"

// RequestID assigns a request identifier, honoring an inbound X-Request-ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(HeaderRequestID, id)
		c.Next()
	}
}

// Logger emits one structured line per request, including the request ID.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		requestID, _ := c.Get("request_id")
		log.Printf("request method=%s path=%s status=%d size=%d ip=%s request_id=%v",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(),
			c.Writer.Size(), c.ClientIP(), requestID)
	}
}

// NormalizeSameOriginOrigin drops the Origin header when it matches the
// request host. Same-origin requests are never subject to CORS, but browsers
// still send Origin on non-GET requests; leaving it in place makes the CORS
// middleware reject an otherwise valid same-origin POST (e.g. behind the nginx
// reverse proxy). It runs before the CORS handler.
func NormalizeSameOriginOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && sameHost(origin, c.Request.Host) {
			c.Request.Header.Del("Origin")
		}
		c.Next()
	}
}

// sameHost reports whether an Origin header's host matches the request host,
// ignoring scheme and default ports.
func sameHost(origin, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}

// SecurityHeaders sets conservative defaults for a self-hosted JSON API.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// MaxBodyBytes rejects oversized request bodies for JSON endpoints. Multipart
// imports enforce their own per-file limits.
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func newRequestID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(raw)
}
