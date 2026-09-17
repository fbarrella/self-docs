package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/migrations"
)

// testPool connects to the database named by TEST_DATABASE_URL and applies the
// migrations. The whole suite is skipped when the variable is unset, so
// `go test ./...` stays green without a database.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping repository integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	// Different test packages share one database and each truncates tables, so
	// serialize them with a session-level advisory lock held on a pinned
	// connection. This keeps a plain `go test ./...` reliable without -p 1.
	lockIntegrationDB(t, pool)

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

// integrationLockID namespaces the test advisory lock.
const integrationLockID = int64(0x5e1fd0c5)

// lockIntegrationDB takes a session-level advisory lock and holds it until the
// test finishes, then releases it on the same pinned connection.
func lockIntegrationDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock connection: %v", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, integrationLockID); err != nil {
		conn.Release()
		t.Fatalf("acquire advisory lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, integrationLockID)
		conn.Release()
	})
}

// truncate clears all domain tables between tests so they are independent.
func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		TRUNCATE documents, tags, document_tags, activity_logs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func strptr(s string) *string { return &s }
