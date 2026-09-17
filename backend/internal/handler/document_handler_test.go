package handler

import (
	"net/http"
	"testing"

	"github.com/self-docs/backend/internal/model"
)

type listResponse struct {
	Data       []model.Document `json:"data"`
	Pagination model.Pagination `json:"pagination"`
}

type treeResponse struct {
	Data []*model.DocumentNode `json:"data"`
}

func TestCreateDocument(t *testing.T) {
	engine, _ := newTestServer(t)

	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title":   "Git Commands",
		"content": "# Git Commands",
		"section": "cheat_sheet",
		"tags":    []string{"Git", "CLI"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	doc := decodeBody[model.Document](t, rec)
	if doc.ID == "" || doc.Slug != "git-commands" {
		t.Errorf("unexpected document: %+v", doc)
	}
	if len(doc.Tags) != 2 {
		t.Errorf("tags = %v, want 2", doc.Tags)
	}
}

func TestCreateDocumentValidation(t *testing.T) {
	engine, _ := newTestServer(t)

	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing title", map[string]any{"section": "workflow"}, http.StatusBadRequest},
		{"invalid section", map[string]any{"title": "X", "section": "nope"}, http.StatusBadRequest},
		{"private forbidden", map[string]any{"title": "X", "section": "private"}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodPost, "/api/documents", tt.body)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (body=%s)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestCreateDocumentSlugConflict(t *testing.T) {
	engine, _ := newTestServer(t)
	body := map[string]any{"title": "Same", "section": "workflow"}

	if rec := doJSON(t, engine, http.MethodPost, "/api/documents", body); rec.Code != http.StatusCreated {
		t.Fatalf("first create status = %d", rec.Code)
	}
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want 409", rec.Code)
	}
	env := decodeBody[struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}](t, rec)
	if env.Error.Code != CodeConflict {
		t.Errorf("error code = %q, want %q", env.Error.Code, CodeConflict)
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	engine, _ := newTestServer(t)
	rec := doJSON(t, engine, http.MethodGet, "/api/documents/00000000-0000-0000-0000-000000000000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListDocuments(t *testing.T) {
	engine, _ := newTestServer(t)

	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Alpha", "section": "workflow", "content": "a",
	})
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Beta", "section": "cheat_sheet", "content": "b",
	})

	rec := doJSON(t, engine, http.MethodGet, "/api/documents?section=workflow", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeBody[listResponse](t, rec)
	if resp.Pagination.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("total=%d len=%d, want 1/1", resp.Pagination.Total, len(resp.Data))
	}
	if resp.Data[0].Content != "" {
		t.Error("content should be omitted by default")
	}

	withContent := doJSON(t, engine, http.MethodGet, "/api/documents?include=content", nil)
	resp = decodeBody[listResponse](t, withContent)
	if len(resp.Data) == 0 || resp.Data[0].Content == "" {
		t.Error("content should be present with include=content")
	}
}

func TestListRejectsPrivateSection(t *testing.T) {
	engine, _ := newTestServer(t)
	rec := doJSON(t, engine, http.MethodGet, "/api/documents?section=private", nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestUpdateDocument(t *testing.T) {
	engine, _ := newTestServer(t)
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Original", "section": "workflow", "content": "v1",
	})
	doc := decodeBody[model.Document](t, rec)

	rec = doJSON(t, engine, http.MethodPut, "/api/documents/"+doc.ID, map[string]any{
		"title": "Renamed", "content": "v2",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	updated := decodeBody[model.Document](t, rec)
	if updated.Title != "Renamed" || updated.Slug != "renamed" || updated.Content != "v2" {
		t.Errorf("unexpected update: %+v", updated)
	}
}

func TestDeleteDocument(t *testing.T) {
	engine, _ := newTestServer(t)
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Temp", "section": "workflow",
	})
	doc := decodeBody[model.Document](t, rec)

	rec = doJSON(t, engine, http.MethodDelete, "/api/documents/"+doc.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	rec = doJSON(t, engine, http.MethodGet, "/api/documents/"+doc.ID, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d, want 404", rec.Code)
	}
}

func TestProjectNotesTree(t *testing.T) {
	engine, _ := newTestServer(t)
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Frontend", "section": "project_note",
	})
	root := decodeBody[model.Document](t, rec)

	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Routing", "section": "project_note", "parent_id": root.ID,
	})

	rec = doJSON(t, engine, http.MethodGet, "/api/documents/tree?section=project_note", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	tree := decodeBody[treeResponse](t, rec)
	if len(tree.Data) != 1 || len(tree.Data[0].Children) != 1 {
		t.Fatalf("unexpected tree: %+v", tree.Data)
	}
}

func TestActivityRecordedOnMutations(t *testing.T) {
	engine, pool := newTestServer(t)
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Tracked", "section": "workflow",
	})
	doc := decodeBody[model.Document](t, rec)
	doJSON(t, engine, http.MethodPut, "/api/documents/"+doc.ID, map[string]any{"content": "changed"})
	doJSON(t, engine, http.MethodDelete, "/api/documents/"+doc.ID, nil)

	var created, updated, deleted int
	rows, err := pool.Query(t.Context(), `SELECT action, COUNT(*) FROM activity_logs GROUP BY action`)
	if err != nil {
		t.Fatalf("query activity: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			t.Fatalf("scan: %v", err)
		}
		switch action {
		case "created":
			created = count
		case "updated":
			updated = count
		case "deleted":
			deleted = count
		}
	}
	if created != 1 || updated != 1 || deleted != 1 {
		t.Errorf("activity counts created=%d updated=%d deleted=%d, want 1/1/1", created, updated, deleted)
	}
}
