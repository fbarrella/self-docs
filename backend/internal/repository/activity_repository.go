package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
)

// ActivityRepository records and reads the activity feed.
//
// Private-only mutations must never be recorded: callers pass an empty
// document title/section pair to skip logging, and List filters private rows.
type ActivityRepository struct {
	pool *pgxpool.Pool
}

// NewActivityRepository constructs an ActivityRepository.
func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{pool: pool}
}

// LogActivityInput describes an activity entry to record.
type LogActivityInput struct {
	Action        model.Action
	DocumentID    *string
	DocumentTitle *string
	Section       *model.Section
	Actor         string
}

// Log inserts an activity entry. Rows whose section is the Private Archive are
// never written, so the feed cannot leak their existence (data-model.md 6).
func (r *ActivityRepository) Log(ctx context.Context, in LogActivityInput) error {
	if in.Section != nil && in.Section.IsPrivate() {
		return nil
	}
	actor := in.Actor
	if actor == "" {
		actor = "user"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO activity_logs (action, document_id, document_title, section, actor)
		VALUES ($1, $2::uuid, $3, $4, $5)`,
		string(in.Action), in.DocumentID, in.DocumentTitle, sectionArg(in.Section), actor)
	if err != nil {
		return fmt.Errorf("log activity: %w", err)
	}
	return nil
}

// List returns a page of activity entries, newest first, excluding private.
func (r *ActivityRepository) List(ctx context.Context, page, pageSize int) ([]model.Activity, model.Pagination, error) {
	page = model.NormalizePage(page)
	pageSize = model.NormalizePageSize(pageSize)
	offset := model.Offset(page, pageSize)

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM activity_logs
		WHERE section IS NULL OR section <> 'private'`).Scan(&total); err != nil {
		return nil, model.Pagination{}, fmt.Errorf("count activity: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, action, document_id, document_title, section, actor, created_at
		FROM activity_logs
		WHERE section IS NULL OR section <> 'private'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, model.Pagination{}, fmt.Errorf("list activity: %w", err)
	}
	defer rows.Close()

	entries := make([]model.Activity, 0)
	for rows.Next() {
		var a model.Activity
		if err := rows.Scan(&a.ID, &a.Action, &a.DocumentID, &a.DocumentTitle, &a.Section, &a.Actor, &a.CreatedAt); err != nil {
			return nil, model.Pagination{}, fmt.Errorf("scan activity: %w", err)
		}
		entries = append(entries, a)
	}
	if err := rows.Err(); err != nil {
		return nil, model.Pagination{}, err
	}

	pagination, _, _ := model.NewPagination(page, pageSize, total)
	return entries, pagination, nil
}

// Recent returns the newest activity entries for the dashboard.
func (r *ActivityRepository) Recent(ctx context.Context, limit int) ([]model.Activity, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, action, document_id, document_title, section, actor, created_at
		FROM activity_logs
		WHERE section IS NULL OR section <> 'private'
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent activity: %w", err)
	}
	defer rows.Close()

	entries := make([]model.Activity, 0)
	for rows.Next() {
		var a model.Activity
		if err := rows.Scan(&a.ID, &a.Action, &a.DocumentID, &a.DocumentTitle, &a.Section, &a.Actor, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		entries = append(entries, a)
	}
	return entries, rows.Err()
}

func sectionArg(s *model.Section) any {
	if s == nil {
		return nil
	}
	return string(*s)
}
