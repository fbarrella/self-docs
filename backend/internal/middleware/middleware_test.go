package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func newEngine(mw gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.Use(mw)
	engine.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func TestRequestIDGenerates(t *testing.T) {
	engine := newEngine(RequestID())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	id := rec.Header().Get(HeaderRequestID)
	if id == "" {
		t.Fatal("expected an X-Request-ID response header")
	}
}

func TestRequestIDPreservesInbound(t *testing.T) {
	engine := newEngine(RequestID())
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "inbound-id")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderRequestID); got != "inbound-id" {
		t.Errorf("request id = %q, want inbound-id", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	engine := newEngine(SecurityHeaders())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestMaxBodyBytesRejects(t *testing.T) {
	engine := newEngine(MaxBodyBytes(8))
	engine.POST("/x", func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	body := strings.NewReader("this body is definitely longer than eight bytes")
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", body))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestMaxBodyBytesAllows(t *testing.T) {
	engine := newEngine(MaxBodyBytes(1024))
	engine.POST("/x", func(c *gin.Context) {
		_, _ = c.GetRawData()
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("small")))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMaxBodyBytesDisabled(t *testing.T) {
	engine := newEngine(MaxBodyBytes(0))
	engine.POST("/x", func(c *gin.Context) {
		_, _ = c.GetRawData()
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	big := strings.NewReader(strings.Repeat("a", 4096))
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", big))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 with limit disabled", rec.Code)
	}
}
