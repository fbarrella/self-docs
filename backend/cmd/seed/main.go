// Command seed inserts sample content so a fresh self-docs instance is
// demo-ready (T6.4). It is idempotent: a document is skipped when one with the
// same slug already exists in its section (and, for children, under the same
// parent).
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed
//
// Inside Docker Compose the binary is shipped at /usr/local/bin/seed:
//
//	docker compose exec backend /usr/local/bin/seed
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"

	"github.com/self-docs/backend/internal/config"
	"github.com/self-docs/backend/internal/db"
	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/model/slug"
	"github.com/self-docs/backend/internal/repository"
	"github.com/self-docs/backend/migrations"
)

//go:embed content/*.md
var contentFS embed.FS

// seedMeta is the front-matter of a seed file.
type seedMeta struct {
	Title    string   `yaml:"title"`
	Section  string   `yaml:"section"`
	Parent   string   `yaml:"parent"`
	Tags     []string `yaml:"tags"`
	Excerpt  string   `yaml:"excerpt"`
	Position int      `yaml:"position"`
}

type seedDoc struct {
	meta    seedMeta
	content string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Apply migrations first so seeding works on a fresh database even without
	// the server having started.
	if err := migrations.Apply(ctx, pool.Pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	docs, err := loadSeedDocs()
	if err != nil {
		log.Fatalf("load seed content: %v", err)
	}

	docRepo := repository.NewDocumentRepository(pool.Pool)
	tagRepo := repository.NewTagRepository(pool.Pool)

	created, skipped := 0, 0
	// Roots first, then children (which resolve their parent's id).
	for _, pass := range []bool{false, true} {
		for _, seed := range docs {
			isChild := seed.meta.Parent != ""
			if isChild != pass {
				continue
			}

			exists, err := documentExists(ctx, pool, seed)
			if err != nil {
				log.Fatalf("check %q: %v", seed.meta.Title, err)
			}
			if exists {
				skipped++
				continue
			}

			var parentID *string
			if isChild {
				id, err := findDocumentID(ctx, pool, seed.meta.Section, seed.meta.Parent)
				if err != nil {
					log.Fatalf("resolve parent %q: %v", seed.meta.Parent, err)
				}
				if id == "" {
					log.Printf("skipping %q: parent %q not found", seed.meta.Title, seed.meta.Parent)
					skipped++
					continue
				}
				parentID = &id
			}

			var excerpt *string
			if seed.meta.Excerpt != "" {
				excerpt = &seed.meta.Excerpt
			}

			if _, err := docRepo.Create(ctx, repository.CreateDocumentInput{
				Title:    seed.meta.Title,
				Content:  seed.content,
				Excerpt:  excerpt,
				Section:  model.Section(seed.meta.Section),
				ParentID: parentID,
				Position: seed.meta.Position,
				Tags:     seed.meta.Tags,
			}); err != nil {
				log.Fatalf("create %q: %v", seed.meta.Title, err)
			}
			created++
		}
	}

	// Ensure sample tags exist even when all documents already did.
	for _, seed := range docs {
		for _, tag := range seed.meta.Tags {
			if _, err := tagRepo.Upsert(ctx, tag); err != nil {
				log.Fatalf("tag %q: %v", tag, err)
			}
		}
	}

	fmt.Printf("seed complete: %d created, %d skipped (already present)\n", created, skipped)
}

// loadSeedDocs reads and parses the embedded Markdown files in a stable order.
func loadSeedDocs() ([]seedDoc, error) {
	entries, err := fs.ReadDir(contentFS, "content")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	docs := make([]seedDoc, 0, len(names))
	for _, name := range names {
		raw, err := contentFS.ReadFile("content/" + name)
		if err != nil {
			return nil, err
		}

		meta, body, ok := splitFrontMatter(string(raw))
		if !ok {
			return nil, fmt.Errorf("%s: missing front-matter", name)
		}
		var parsed seedMeta
		if err := yaml.Unmarshal([]byte(meta), &parsed); err != nil {
			return nil, fmt.Errorf("%s: invalid front-matter: %w", name, err)
		}
		if parsed.Title == "" || parsed.Section == "" {
			return nil, fmt.Errorf("%s: title and section are required", name)
		}
		docs = append(docs, seedDoc{meta: parsed, content: strings.TrimLeft(body, "\n")})
	}
	return docs, nil
}

// splitFrontMatter separates the leading YAML block from the body, stripping
// the front-matter from the stored content (so documents match the import
// behavior).
func splitFrontMatter(content string) (string, string, bool) {
	normalized := strings.ReplaceAll(strings.TrimPrefix(content, "\ufeff"), "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", content, false
	}
	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", content, false
	}
	meta := rest[:end]
	body := rest[end+len("\n---"):]
	return meta, body, true
}

// documentExists reports whether a document with the seed's slug and section
// already exists (root, or under its parent for children).
func documentExists(ctx context.Context, pool *db.Pool, seed seedDoc) (bool, error) {
	s := slug.Make(seed.meta.Title)
	var count int
	if seed.meta.Parent == "" {
		err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM documents WHERE section = $1 AND slug = $2 AND parent_id IS NULL`,
			seed.meta.Section, s,
		).Scan(&count)
		return count > 0, err
	}
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM documents d
		JOIN documents p ON p.id = d.parent_id
		WHERE d.section = $1 AND d.slug = $2 AND p.slug = $3`,
		seed.meta.Section, s, slug.Make(seed.meta.Parent),
	).Scan(&count)
	return count > 0, err
}

// findDocumentID returns the id of a root document by section and slug.
func findDocumentID(ctx context.Context, pool *db.Pool, section, title string) (string, error) {
	var id string
	err := pool.QueryRow(ctx,
		`SELECT id FROM documents WHERE section = $1 AND slug = $2 AND parent_id IS NULL`,
		section, slug.Make(title),
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}
