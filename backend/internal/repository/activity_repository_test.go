package repository

import (
	"context"
	"testing"

	"github.com/self-docs/backend/internal/model"
)

func TestActivityLogAndList(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewActivityRepository(pool)

	section := model.SectionWorkflow
	if err := repo.Log(ctx, LogActivityInput{
		Action:        model.ActionCreated,
		DocumentTitle: strptr("Frontend Guide"),
		Section:       &section,
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if err := repo.Log(ctx, LogActivityInput{
		Action:        model.ActionUpdated,
		DocumentTitle: strptr("Frontend Guide"),
		Section:       &section,
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entries, pag, err := repo.List(ctx, 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pag.Total != 2 || len(entries) != 2 {
		t.Fatalf("total=%d len=%d, want 2/2", pag.Total, len(entries))
	}
	if entries[0].CreatedAt.Before(entries[1].CreatedAt) {
		t.Error("entries are not newest-first")
	}
	if entries[0].Actor != "user" {
		t.Errorf("Actor = %q, want user", entries[0].Actor)
	}
}

func TestActivityPrivateNeverLogged(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewActivityRepository(pool)

	private := model.SectionPrivate
	if err := repo.Log(ctx, LogActivityInput{
		Action:        model.ActionCreated,
		DocumentTitle: strptr("Secret"),
		Section:       &private,
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entries, pag, err := repo.List(ctx, 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pag.Total != 0 || len(entries) != 0 {
		t.Fatalf("private activity leaked: total=%d len=%d", pag.Total, len(entries))
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM activity_logs`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("private row was physically written (%d rows)", count)
	}

	recent, err := repo.Recent(ctx, 10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("Recent returned private rows: %d", len(recent))
	}
}

func TestActivitySurvivesDocumentDelete(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	docs := NewDocumentRepository(pool)
	activity := NewActivityRepository(pool)

	doc, _ := docs.Create(ctx, CreateDocumentInput{Title: "Ephemeral", Section: model.SectionWorkflow})
	section := model.SectionWorkflow
	if err := activity.Log(ctx, LogActivityInput{
		Action: model.ActionDeleted, DocumentID: &doc.ID, DocumentTitle: &doc.Title, Section: &section,
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if err := docs.Delete(ctx, doc.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	entries, _, err := activity.List(ctx, 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].DocumentID != nil {
		t.Error("document_id should be NULL after delete")
	}
	if entries[0].DocumentTitle == nil || *entries[0].DocumentTitle != "Ephemeral" {
		t.Error("title snapshot should survive delete")
	}
}
