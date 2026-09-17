package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/model/slug"
)

// DocumentRepository provides document persistence and query operations.
//
// Methods that serve public content always filter is_private = false. Methods
// that accept an includePrivate flag are only used by the Private Archive
// endpoints behind a master-password session (T2.5).
type DocumentRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentRepository constructs a DocumentRepository.
func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// CreateDocumentInput carries the fields required to create a document.
type CreateDocumentInput struct {
	Title    string
	Content  string
	Excerpt  *string
	Section  model.Section
	ParentID *string
	Position int
	Tags     []string
}

// UpdateDocumentInput carries a partial update. Nil pointers leave the field
// unchanged; SetParent distinguishes "leave parent" from "clear parent".
type UpdateDocumentInput struct {
	Title      *string
	Content    *string
	Excerpt    *string
	SetExcerpt bool
	Section    *model.Section
	ParentID   *string
	SetParent  bool
	Position   *int
	Tags       *[]string
}

// ListDocumentsOptions controls filtering, sorting, and pagination.
type ListDocumentsOptions struct {
	Sections       []model.Section
	Tags           []string
	ParentID       *string
	Query          string
	IncludeContent bool
	Sort           string
	Order          string
	Page           int
	PageSize       int
	IncludePrivate bool
}

// SearchOptions controls the full-text search query.
type SearchOptions struct {
	Query    string
	Sections []model.Section
	Page     int
	PageSize int
}

// Create inserts a document and replaces its tag set.
func (r *DocumentRepository) Create(ctx context.Context, in CreateDocumentInput) (model.Document, error) {
	if err := validateSectionAndParent(in.Section, in.ParentID); err != nil {
		return model.Document{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Document{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if in.ParentID != nil {
		if err := ensureParentTx(ctx, tx, *in.ParentID, in.Section); err != nil {
			return model.Document{}, err
		}
	}

	s := slug.Make(in.Title)
	var doc model.Document
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (title, slug, content, excerpt, section, parent_id, position, is_private)
		VALUES ($1, $2, $3, $4, $5, $6::uuid, $7, $8)
		RETURNING id, title, slug, content, excerpt, section, parent_id, position, created_at, updated_at`,
		in.Title, s, in.Content, in.Excerpt, string(in.Section), in.ParentID, in.Position, in.Section.IsPrivate(),
	).Scan(&doc.ID, &doc.Title, &doc.Slug, &doc.Content, &doc.Excerpt, &doc.Section,
		&doc.ParentID, &doc.Position, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.Document{}, ErrConflict
		}
		return model.Document{}, fmt.Errorf("insert document: %w", err)
	}

	if err := setDocumentTagsTx(ctx, tx, doc.ID, in.Tags); err != nil {
		return model.Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Document{}, fmt.Errorf("commit: %w", err)
	}

	tags, err := r.tagsFor(ctx, doc.ID)
	if err != nil {
		return model.Document{}, err
	}
	doc.Tags = tags
	return doc, nil
}

// GetByID returns a document. When includePrivate is false, private documents
// are treated as not found.
func (r *DocumentRepository) GetByID(ctx context.Context, id string, includePrivate bool) (model.Document, error) {
	var doc model.Document
	err := scanDocument(r.pool.QueryRow(ctx, `
		SELECT id, title, slug, content, excerpt, section, parent_id, position, created_at, updated_at
		FROM documents
		WHERE id = $1::uuid AND ($2 OR is_private = false)`, id, includePrivate),
		&doc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Document{}, ErrNotFound
		}
		return model.Document{}, fmt.Errorf("get document: %w", err)
	}
	tags, err := r.tagsFor(ctx, doc.ID)
	if err != nil {
		return model.Document{}, err
	}
	doc.Tags = tags
	return doc, nil
}

// List returns a page of documents matching the filters.
func (r *DocumentRepository) List(ctx context.Context, opts ListDocumentsOptions) ([]model.Document, model.Pagination, error) {
	where, args := buildDocumentFilters(opts)
	page := model.NormalizePage(opts.Page)
	pageSize := model.NormalizePageSize(opts.PageSize)
	offset := model.Offset(page, pageSize)

	var total int
	countSQL := "SELECT COUNT(*) FROM documents d WHERE " + where
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, model.Pagination{}, fmt.Errorf("count documents: %w", err)
	}

	nextArg := len(args) + 1
	args = append(args, pageSize, offset)
	listSQL := fmt.Sprintf(`
		SELECT d.id, d.title, d.slug, %s, d.excerpt, d.section, d.parent_id, d.position, d.created_at, d.updated_at
		FROM documents d
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		contentColumn(opts.IncludeContent), where,
		documentOrderClause(opts.Sort, opts.Order), nextArg, nextArg+1)

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, model.Pagination{}, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	docs := make([]model.Document, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var doc model.Document
		if err := scanDocumentRow(rows, &doc); err != nil {
			return nil, model.Pagination{}, err
		}
		docs = append(docs, doc)
		ids = append(ids, doc.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, model.Pagination{}, err
	}

	tagMap, err := loadTagsForDocuments(ctx, r.pool, ids)
	if err != nil {
		return nil, model.Pagination{}, err
	}
	for i := range docs {
		docs[i].Tags = tagMap[docs[i].ID]
	}

	pagination, _, _ := model.NewPagination(page, pageSize, total)
	return docs, pagination, nil
}

// Update applies a partial update and returns the updated document.
func (r *DocumentRepository) Update(ctx context.Context, id string, in UpdateDocumentInput, includePrivate bool) (model.Document, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Document{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := getDocumentTx(ctx, tx, id, includePrivate)
	if err != nil {
		return model.Document{}, err
	}

	sets := make([]string, 0, 8)
	args := make([]any, 0, 8)

	if in.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", len(args)+1))
		args = append(args, *in.Title)
		sets = append(sets, fmt.Sprintf("slug = $%d", len(args)+1))
		args = append(args, slug.Make(*in.Title))
	}
	if in.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", len(args)+1))
		args = append(args, *in.Content)
	}
	if in.SetExcerpt {
		sets = append(sets, fmt.Sprintf("excerpt = $%d", len(args)+1))
		args = append(args, in.Excerpt)
	}

	section := current.Section
	if in.Section != nil {
		section = *in.Section
		sets = append(sets, fmt.Sprintf("section = $%d", len(args)+1))
		args = append(args, string(*in.Section))
		sets = append(sets, fmt.Sprintf("is_private = $%d", len(args)+1))
		args = append(args, section.IsPrivate())
	}

	parent := current.ParentID
	if in.SetParent {
		parent = in.ParentID
	}
	if section != model.SectionProjectNote {
		parent = nil
	}
	if in.SetParent || (in.Section != nil && section != model.SectionProjectNote) {
		if parent != nil {
			if err := ensureParentTx(ctx, tx, *parent, section); err != nil {
				return model.Document{}, err
			}
			if err := ensureNoCycleTx(ctx, tx, id, *parent); err != nil {
				return model.Document{}, err
			}
		}
		sets = append(sets, fmt.Sprintf("parent_id = $%d::uuid", len(args)+1))
		args = append(args, parent)
	}
	if in.Position != nil {
		sets = append(sets, fmt.Sprintf("position = $%d", len(args)+1))
		args = append(args, *in.Position)
	}

	if len(sets) == 0 && in.Tags == nil {
		tags, err := r.tagsFor(ctx, current.ID)
		if err != nil {
			return model.Document{}, err
		}
		current.Tags = tags
		return current, nil
	}

	if len(sets) > 0 {
		args = append(args, id)
		_, err = tx.Exec(ctx, fmt.Sprintf(
			"UPDATE documents SET %s WHERE id = $%d::uuid", strings.Join(sets, ", "), len(args)),
			args...)
		if err != nil {
			if isUniqueViolation(err) {
				return model.Document{}, ErrConflict
			}
			return model.Document{}, fmt.Errorf("update document: %w", err)
		}
	}

	if in.Tags != nil {
		if err := setDocumentTagsTx(ctx, tx, id, *in.Tags); err != nil {
			return model.Document{}, err
		}
	}

	updated, err := getDocumentTx(ctx, tx, id, includePrivate)
	if err != nil {
		return model.Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Document{}, fmt.Errorf("commit: %w", err)
	}

	tags, err := r.tagsFor(ctx, updated.ID)
	if err != nil {
		return model.Document{}, err
	}
	updated.Tags = tags
	return updated, nil
}

// Delete removes a document; project-note children cascade.
func (r *DocumentRepository) Delete(ctx context.Context, id string, includePrivate bool) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM documents WHERE id = $1::uuid AND ($2 OR is_private = false)`,
		id, includePrivate)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Tree returns the Project Notes hierarchy ordered by position and title.
func (r *DocumentRepository) Tree(ctx context.Context, section model.Section, includePrivate bool) ([]*model.DocumentNode, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, section, parent_id, position
		FROM documents
		WHERE section = $1 AND ($2 OR is_private = false)
		ORDER BY position ASC, title ASC`, string(section), includePrivate)
	if err != nil {
		return nil, fmt.Errorf("tree query: %w", err)
	}
	defer rows.Close()

	nodes := make(map[string]*model.DocumentNode)
	parentOf := make(map[string]*string)
	order := make([]string, 0)
	for rows.Next() {
		var n model.DocumentNode
		var parent *string
		if err := rows.Scan(&n.ID, &n.Title, &n.Slug, &n.Section, &parent, &n.Position); err != nil {
			return nil, fmt.Errorf("scan tree node: %w", err)
		}
		n.Children = []*model.DocumentNode{}
		nodes[n.ID] = &n
		parentOf[n.ID] = parent
		order = append(order, n.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	roots := make([]*model.DocumentNode, 0)
	for _, id := range order {
		node := nodes[id]
		parent := parentOf[id]
		if parent == nil || nodes[*parent] == nil {
			roots = append(roots, node)
			continue
		}
		nodes[*parent].Children = append(nodes[*parent].Children, node)
	}
	return roots, nil
}

// CountBySection returns document counts per section. Private counts are only
// included when includePrivate is true.
func (r *DocumentRepository) CountBySection(ctx context.Context, includePrivate bool) (map[model.Section]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT section, COUNT(*)
		FROM documents
		WHERE ($1 OR is_private = false)
		GROUP BY section`, includePrivate)
	if err != nil {
		return nil, fmt.Errorf("count by section: %w", err)
	}
	defer rows.Close()

	counts := make(map[model.Section]int)
	for rows.Next() {
		var s model.Section
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, fmt.Errorf("scan section count: %w", err)
		}
		counts[s] = n
	}
	return counts, rows.Err()
}

// RecentlyUpdated returns the most recently modified documents, newest first.
func (r *DocumentRepository) RecentlyUpdated(ctx context.Context, limit int, includePrivate bool) ([]model.Document, error) {
	if limit < 1 || limit > 50 {
		limit = 8
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, slug, '', excerpt, section, parent_id, position, created_at, updated_at
		FROM documents
		WHERE ($2 OR is_private = false)
		ORDER BY updated_at DESC
		LIMIT $1`, limit, includePrivate)
	if err != nil {
		return nil, fmt.Errorf("recently updated: %w", err)
	}
	defer rows.Close()

	docs := make([]model.Document, 0)
	for rows.Next() {
		var doc model.Document
		if err := scanDocumentRow(rows, &doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// Search runs a ranked full-text search over title, content, and tags. Private
// documents are always excluded, regardless of session (PRD 3.3).
func (r *DocumentRepository) Search(ctx context.Context, opts SearchOptions) ([]model.SearchResult, model.Pagination, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		pagination, _, _ := model.NewPagination(opts.Page, opts.PageSize, 0)
		return []model.SearchResult{}, pagination, nil
	}

	page := model.NormalizePage(opts.Page)
	pageSize := model.NormalizePageSize(opts.PageSize)
	offset := model.Offset(page, pageSize)

	sectionFilter := "true"
	args := []any{query}
	if len(opts.Sections) > 0 {
		names := make([]string, len(opts.Sections))
		for i, s := range opts.Sections {
			names[i] = string(s)
		}
		sectionFilter = "d.section = ANY($2)"
		args = append(args, names)
	}

	matchExpr := `(d.search_vector @@ websearch_to_tsquery('simple', $1)
		OR EXISTS (
			SELECT 1 FROM document_tags dt
			JOIN tags t ON t.id = dt.tag_id
			WHERE dt.document_id = d.id AND t.normalized ILIKE '%' || $1 || '%'
		))`

	var total int
	countSQL := fmt.Sprintf(
		"SELECT COUNT(*) FROM documents d WHERE d.is_private = false AND %s AND %s",
		matchExpr, sectionFilter)
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, model.Pagination{}, fmt.Errorf("count search: %w", err)
	}

	nextArg := len(args) + 1
	args = append(args, pageSize, offset)
	listSQL := fmt.Sprintf(`
		SELECT d.id, d.title, d.slug, d.section,
		       ts_rank(d.search_vector, websearch_to_tsquery('simple', $1)) AS rank,
		       ts_headline('simple', coalesce(nullif(d.content, ''), d.title),
		                   websearch_to_tsquery('simple', $1),
		                   'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15') AS snippet,
		       d.updated_at
		FROM documents d
		WHERE d.is_private = false AND %s AND %s
		ORDER BY rank DESC, d.updated_at DESC
		LIMIT $%d OFFSET $%d`, matchExpr, sectionFilter, nextArg, nextArg+1)

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, model.Pagination{}, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	results := make([]model.SearchResult, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var sr model.SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title, &sr.Slug, &sr.Section, &sr.Rank, &sr.Snippet, &sr.UpdatedAt); err != nil {
			return nil, model.Pagination{}, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, sr)
		ids = append(ids, sr.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, model.Pagination{}, err
	}

	tagMap, err := loadTagsForDocuments(ctx, r.pool, ids)
	if err != nil {
		return nil, model.Pagination{}, err
	}
	for i := range results {
		results[i].Tags = tagMap[results[i].ID]
	}

	pagination, _, _ := model.NewPagination(page, pageSize, total)
	return results, pagination, nil
}

// tagsFor loads tag names for a single document.
func (r *DocumentRepository) tagsFor(ctx context.Context, id string) ([]string, error) {
	m, err := loadTagsForDocuments(ctx, r.pool, []string{id})
	if err != nil {
		return nil, err
	}
	if tags, ok := m[id]; ok {
		return tags, nil
	}
	return []string{}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func validateSectionAndParent(section model.Section, parentID *string) error {
	if !section.Valid() {
		return fmt.Errorf("%w: invalid section %q", ErrInvalidHierarchy, section)
	}
	if parentID != nil && section != model.SectionProjectNote {
		return fmt.Errorf("%w: parent_id is only allowed for project_note documents", ErrInvalidHierarchy)
	}
	return nil
}

// ensureParentTx verifies that the parent exists, is a project_note, and is not
// private when the child is public.
func ensureParentTx(ctx context.Context, q querier, parentID string, section model.Section) error {
	var parentSection model.Section
	err := q.QueryRow(ctx, `SELECT section FROM documents WHERE id = $1::uuid`, parentID).Scan(&parentSection)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: parent not found", ErrInvalidHierarchy)
		}
		return fmt.Errorf("load parent: %w", err)
	}
	if parentSection != model.SectionProjectNote {
		return fmt.Errorf("%w: parent must be a project_note", ErrInvalidHierarchy)
	}
	return nil
}

// ensureNoCycleTx rejects moving a document under one of its own descendants.
func ensureNoCycleTx(ctx context.Context, q querier, documentID, parentID string) error {
	var descendant bool
	err := q.QueryRow(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id FROM documents WHERE id = $1::uuid
			UNION ALL
			SELECT d.id FROM documents d JOIN descendants x ON d.parent_id = x.id
		)
		SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2::uuid)`,
		documentID, parentID,
	).Scan(&descendant)
	if err != nil {
		return fmt.Errorf("cycle check: %w", err)
	}
	if descendant {
		return fmt.Errorf("%w: parent would create a cycle", ErrInvalidHierarchy)
	}
	return nil
}

func getDocumentTx(ctx context.Context, q querier, id string, includePrivate bool) (model.Document, error) {
	var doc model.Document
	err := scanDocument(q.QueryRow(ctx, `
		SELECT id, title, slug, content, excerpt, section, parent_id, position, created_at, updated_at
		FROM documents
		WHERE id = $1::uuid AND ($2 OR is_private = false)`, id, includePrivate),
		&doc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Document{}, ErrNotFound
		}
		return model.Document{}, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanDocument(s scanner, doc *model.Document) error {
	return s.Scan(&doc.ID, &doc.Title, &doc.Slug, &doc.Content, &doc.Excerpt, &doc.Section,
		&doc.ParentID, &doc.Position, &doc.CreatedAt, &doc.UpdatedAt)
}

// scanDocumentRow scans a row where content may be selected as an empty
// literal, allowing a shared column list for list and detail queries.
func scanDocumentRow(s scanner, doc *model.Document) error {
	return s.Scan(&doc.ID, &doc.Title, &doc.Slug, &doc.Content, &doc.Excerpt, &doc.Section,
		&doc.ParentID, &doc.Position, &doc.CreatedAt, &doc.UpdatedAt)
}

func contentColumn(include bool) string {
	if include {
		return "d.content"
	}
	return "''"
}

func buildDocumentFilters(opts ListDocumentsOptions) (string, []any) {
	args := []any{opts.IncludePrivate}
	clauses := []string{fmt.Sprintf("($1 OR d.is_private = false)")}

	if len(opts.Sections) > 0 {
		names := make([]string, len(opts.Sections))
		for i, s := range opts.Sections {
			names[i] = string(s)
		}
		args = append(args, names)
		clauses = append(clauses, fmt.Sprintf("d.section = ANY($%d)", len(args)))
	}
	if opts.ParentID != nil {
		args = append(args, *opts.ParentID)
		clauses = append(clauses, fmt.Sprintf("d.parent_id = $%d::uuid", len(args)))
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		args = append(args, q)
		clauses = append(clauses, fmt.Sprintf("d.title ILIKE '%%' || $%d || '%%'", len(args)))
	}
	if len(opts.Tags) > 0 {
		normalized := make([]string, 0, len(opts.Tags))
		for _, t := range opts.Tags {
			if n := NormalizeTag(t); n != "" {
				normalized = append(normalized, n)
			}
		}
		if len(normalized) > 0 {
			args = append(args, normalized)
			clauses = append(clauses, fmt.Sprintf(`
				(SELECT COUNT(DISTINCT t.normalized)
				 FROM document_tags dt
				 JOIN tags t ON t.id = dt.tag_id
				 WHERE dt.document_id = d.id AND t.normalized = ANY($%d)) = %d`,
				len(args), len(normalized)))
		}
	}
	return strings.Join(clauses, " AND "), args
}

func documentOrderClause(sort, order string) string {
	dir := "DESC"
	if strings.EqualFold(order, "asc") {
		dir = "ASC"
	}
	switch sort {
	case "created_at":
		return "d.created_at " + dir
	case "title":
		return "d.title " + dir
	default:
		return "d.updated_at " + dir
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
