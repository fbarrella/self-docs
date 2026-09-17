// Package migrations embeds the SQL migration files and applies them in order.
//
// Applied versions are tracked in a schema_migrations table so startup is
// idempotent. Only *.up.sql files are applied automatically; *.down.sql files
// are available for manual rollback.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var files embed.FS

const (
	upSuffix   = ".up.sql"
	downSuffix = ".down.sql"
)

// Migration is a single ordered schema change.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// Apply runs every pending up migration inside its own transaction.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    integer     PRIMARY KEY,
			name       text        NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	migs, err := Load()
	if err != nil {
		return err
	}

	for _, m := range migs {
		if applied[m.Version] {
			continue
		}
		if err := applyOne(ctx, pool, m); err != nil {
			return fmt.Errorf("migration %04d_%s: %w", m.Version, m.Name, err)
		}
	}
	return nil
}

// Load returns all up migrations parsed from the embedded files, ordered by
// version. It is exported so tests can assert on the discovered set.
func Load() ([]Migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var migs []Migration
	seen := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), upSuffix) {
			continue
		}
		version, name, err := parseName(entry.Name())
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", version, prev, entry.Name())
		}
		seen[version] = entry.Name()

		content, err := files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		migs = append(migs, Migration{Version: version, Name: name, SQL: string(content)})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	return migs, nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
		m.Version, m.Name,
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return tx.Commit(ctx)
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// parseName splits a file like "0001_init.up.sql" into version 1 and "init".
func parseName(filename string) (int, string, error) {
	base := strings.TrimSuffix(filename, upSuffix)
	prefix, name, ok := strings.Cut(base, "_")
	if !ok {
		return 0, "", fmt.Errorf("invalid migration filename %q: expected <version>_<name>%s", filename, upSuffix)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, "", fmt.Errorf("invalid migration version in %q: %w", filename, err)
	}
	return version, name, nil
}
