package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/router"
	"github.com/self-docs/backend/migrations"
)

// newEngine builds the fully wired router against TEST_DATABASE_URL. The suite
// is skipped when the variable is unset so `go test ./...` stays green without
// a database.
func newEngine(t *testing.T) *gin.Engine {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping router integration tests")
	}
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Serialize with the other integration packages sharing this database.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock connection: %v", err)
	}
	const integrationLockID = int64(0x5e1fd0c5)
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, integrationLockID); err != nil {
		conn.Release()
		t.Fatalf("advisory lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, integrationLockID)
		conn.Release()
	})

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE documents, tags, document_tags, activity_logs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return router.New(router.Deps{Pool: pool, Version: "test", MaxImportBytes: 1 << 20})
}

func TestFullEngineSmoke(t *testing.T) {
	engine := newEngine(t)

	// Health endpoint works and middleware adds a request ID.
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header from middleware")
	}

	// Create then list a document through the full stack.
	body := strings.NewReader(`{"title":"Router Smoke","section":"workflow"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/documents", body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d (%s)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/documents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Router Smoke") {
		t.Errorf("list did not include the created document: %s", rec.Body.String())
	}

	// Private routes are locked without a session.
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/private/documents", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("private locked status = %d, want 401", rec.Code)
	}
}

func TestOversizedJSONRejected(t *testing.T) {
	engine := newEngine(t)

	big := `{"title":"` + strings.Repeat("a", 2<<20) + `","section":"workflow"}`
	req := httptest.NewRequest(http.MethodPost, "/api/documents", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 for oversized body", rec.Code)
	}
}
