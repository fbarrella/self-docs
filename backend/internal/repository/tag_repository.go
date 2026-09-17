package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
)

// TagRepository provides tag persistence and aggregation queries.
type TagRepository struct {
	pool *pgxpool.Pool
}

// NewTagRepository constructs a TagRepository.
func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

// ListTagsOptions controls tag listing and pagination.
type ListTagsOptions struct {
	Query    string
	Sort     string
	Order    string
	Page     int
	PageSize int
}

// NormalizeTag produces the lookup key for a tag: trimmed, lowercased, with a
// leading '#' removed and internal whitespace collapsed.
func NormalizeTag(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.TrimPrefix(n, "#")
	n = strings.Join(strings.Fields(n), " ")
	return n
}

// CanonicalTagName returns the display form of a tag: trimmed, leading '#'
// removed, internal whitespace collapsed, original casing preserved.
func CanonicalTagName(name string) string {
	n := strings.TrimSpace(name)
	n = strings.TrimPrefix(n, "#")
	return strings.Join(strings.Fields(n), " ")
}

// Upsert inserts a tag or returns the existing one, keyed by normalized name.
func (r *TagRepository) Upsert(ctx context.Context, name string) (model.Tag, error) {
	return upsertTag(ctx, r.pool, name)
}

// List returns tags with non-private usage counts.
func (r *TagRepository) List(ctx context.Context, opts ListTagsOptions) ([]model.Tag, model.Pagination, error) {
	query := strings.TrimSpace(opts.Query)
	page := model.NormalizePage(opts.Page)
	pageSize := model.NormalizePageSize(opts.PageSize)
	offset := model.Offset(page, pageSize)

	// A tag attached exclusively to private documents must not be listed, since
	// its very existence would leak. Tags with no attachments are still shown.
	notPrivateOnly := `NOT (
		EXISTS (SELECT 1 FROM document_tags x WHERE x.tag_id = t.id)
		AND NOT EXISTS (
			SELECT 1 FROM document_tags y
			JOIN documents dy ON dy.id = y.document_id
			WHERE y.tag_id = t.id AND dy.is_private = false
		)
	)`

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM tags t WHERE ($1 = '' OR t.name ILIKE '%%' || $1 || '%%') AND %s",
		notPrivateOnly),
		query,
	).Scan(&total); err != nil {
		return nil, model.Pagination{}, fmt.Errorf("count tags: %w", err)
	}

	order := tagOrderClause(opts.Sort, opts.Order)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT t.name, COUNT(d.id) AS count
		FROM tags t
		LEFT JOIN document_tags dt ON dt.tag_id = t.id
		LEFT JOIN documents d ON d.id = dt.document_id AND d.is_private = false
		WHERE ($1 = '' OR t.name ILIKE '%%' || $1 || '%%') AND %s
		GROUP BY t.id, t.name
		ORDER BY %s
		LIMIT $2 OFFSET $3`, notPrivateOnly, order),
		query, pageSize, offset,
	)
	if err != nil {
		return nil, model.Pagination{}, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	tags, err := scanTags(rows)
	if err != nil {
		return nil, model.Pagination{}, err
	}
	pagination, _, _ := model.NewPagination(page, pageSize, total)
	return tags, pagination, nil
}

// Popular returns the most-used tags with non-private usage counts.
func (r *TagRepository) Popular(ctx context.Context, limit int) ([]model.Tag, error) {
	if limit < 1 || limit > 100 {
		limit = 12
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.name, COUNT(d.id) AS count
		FROM tags t
		JOIN document_tags dt ON dt.tag_id = t.id
		JOIN documents d ON d.id = dt.document_id AND d.is_private = false
		GROUP BY t.id, t.name
		ORDER BY count DESC, t.name ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("popular tags: %w", err)
	}
	defer rows.Close()

	return scanTags(rows)
}

// GetByNormalized returns a tag by its normalized key.
func (r *TagRepository) GetByNormalized(ctx context.Context, normalized string) (model.Tag, error) {
	var tag model.Tag
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM tags WHERE normalized = $1`,
		NormalizeTag(normalized),
	).Scan(&tag.Name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Tag{}, ErrNotFound
		}
		return model.Tag{}, fmt.Errorf("get tag: %w", err)
	}
	return tag, nil
}

// GetVisibleByNormalized is like GetByNormalized but treats a tag attached
// exclusively to private documents as not found, so public callers cannot
// detect its existence.
func (r *TagRepository) GetVisibleByNormalized(ctx context.Context, normalized string) (model.Tag, error) {
	var tag model.Tag
	err := r.pool.QueryRow(ctx, `
		SELECT t.name, COUNT(d.id) AS count
		FROM tags t
		LEFT JOIN document_tags dt ON dt.tag_id = t.id
		LEFT JOIN documents d ON d.id = dt.document_id AND d.is_private = false
		WHERE t.normalized = $1
		  AND NOT (
			EXISTS (SELECT 1 FROM document_tags x WHERE x.tag_id = t.id)
			AND NOT EXISTS (
				SELECT 1 FROM document_tags y
				JOIN documents dy ON dy.id = y.document_id
				WHERE y.tag_id = t.id AND dy.is_private = false
			)
		  )
		GROUP BY t.id, t.name`,
		NormalizeTag(normalized),
	).Scan(&tag.Name, &tag.Count)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Tag{}, ErrNotFound
		}
		return model.Tag{}, fmt.Errorf("get visible tag: %w", err)
	}
	return tag, nil
}

// Delete removes a tag and its document links (document_tags cascades).
func (r *TagRepository) Delete(ctx context.Context, name string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE normalized = $1`, NormalizeTag(name))
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func tagOrderClause(sort, order string) string {
	dir := "ASC"
	if strings.EqualFold(order, "desc") {
		dir = "DESC"
	}
	switch sort {
	case "count":
		return "count " + dir + ", t.name ASC"
	default:
		return "t.name " + dir
	}
}

func scanTags(rows pgx.Rows) ([]model.Tag, error) {
	tags := make([]model.Tag, 0)
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// upsertTag inserts or fetches a tag, preserving the existing display name.
func upsertTag(ctx context.Context, q querier, name string) (model.Tag, error) {
	normalized := NormalizeTag(name)
	if normalized == "" {
		return model.Tag{}, fmt.Errorf("tag name must not be empty")
	}
	canonical := CanonicalTagName(name)

	var tag model.Tag
	err := q.QueryRow(ctx, `
		INSERT INTO tags (name, normalized)
		VALUES ($1, $2)
		ON CONFLICT (normalized) DO UPDATE SET name = tags.name
		RETURNING name`, canonical, normalized,
	).Scan(&tag.Name)
	if err != nil {
		return model.Tag{}, fmt.Errorf("upsert tag: %w", err)
	}
	return tag, nil
}

// setDocumentTagsTx replaces the tag set of a document within a transaction.
func setDocumentTagsTx(ctx context.Context, q querier, documentID string, names []string) error {
	if _, err := q.Exec(ctx, `DELETE FROM document_tags WHERE document_id = $1::uuid`, documentID); err != nil {
		return fmt.Errorf("clear document tags: %w", err)
	}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := NormalizeTag(name)
		if normalized == "" {
			continue
		}
		if _, dup := seen[normalized]; dup {
			continue
		}
		seen[normalized] = struct{}{}

		tag, err := upsertTag(ctx, q, name)
		if err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO document_tags (document_id, tag_id)
			SELECT $1::uuid, id FROM tags WHERE normalized = $2
			ON CONFLICT DO NOTHING`, documentID, normalized); err != nil {
			return fmt.Errorf("attach tag %q: %w", tag.Name, err)
		}
	}
	return nil
}

// loadTagsForDocuments returns tag names keyed by document id.
func loadTagsForDocuments(ctx context.Context, q querier, ids []string) (map[string][]string, error) {
	out := make(map[string][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT dt.document_id::text, t.name
		FROM document_tags dt
		JOIN tags t ON t.id = dt.tag_id
		WHERE dt.document_id = ANY($1::uuid[])
		ORDER BY t.name`, ids)
	if err != nil {
		return nil, fmt.Errorf("load document tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docID, name string
		if err := rows.Scan(&docID, &name); err != nil {
			return nil, fmt.Errorf("scan document tag: %w", err)
		}
		out[docID] = append(out[docID], name)
	}
	return out, rows.Err()
}
