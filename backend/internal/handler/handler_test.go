package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/repository"
	"github.com/self-docs/backend/migrations"
)

// newTestServer wires the document routes against the database in
// TEST_DATABASE_URL and truncates domain tables. The suite is skipped without
// the variable.
func newTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping handler integration tests")
	}

	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	// Serialize with other test packages that share this database.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock connection: %v", err)
	}
	const integrationLockID = int64(0x5e1fd0c5)
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, integrationLockID); err != nil {
		conn.Release()
		t.Fatalf("acquire advisory lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, integrationLockID)
		conn.Release()
	})

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE documents, tags, document_tags, activity_logs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	engine := gin.New()
	docs := NewDocumentHandler(
		repository.NewDocumentRepository(pool),
		repository.NewActivityRepository(pool),
	)
	engine.POST("/api/documents", docs.Create)
	engine.GET("/api/documents", docs.List)
	engine.GET("/api/documents/tree", docs.Tree)
	engine.GET("/api/documents/:id", docs.Get)
	engine.PUT("/api/documents/:id", docs.Update)
	engine.DELETE("/api/documents/:id", docs.Delete)

	return engine, pool
}

// doJSON issues a JSON request and decodes the response body when it is JSON.
func doJSON(t *testing.T, engine *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// httptestNewRequest builds a GET request carrying the private session cookie.
func httptestNewRequest(t *testing.T, method, path, token string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: PrivateCookieName, Value: token})
	return req
}

// serve runs a prepared request through the engine.
func serve(engine *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return out
}
