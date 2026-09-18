package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

type tagsResponse struct {
	Data       []model.Tag      `json:"data"`
	Pagination model.Pagination `json:"pagination"`
}

// newTagTestServer reuses the document routes from newTestServer and adds the
// tag routes. It returns a helper that creates a public document.
func newTagTestServer(t *testing.T) (*gin.Engine, func(string, string, ...string)) {
	t.Helper()
	engine, pool := newTestServer(t)

	tags := NewTagHandler(repository.NewTagRepository(pool), repository.NewDocumentRepository(pool), nil)
	engine.GET("/api/tags", tags.List)
	engine.GET("/api/tags/popular", tags.Popular)
	engine.GET("/api/tags/:name/documents", tags.Documents)

	create := func(title, section string, tagNames ...string) {
		t.Helper()
		body := map[string]any{"title": title, "section": section}
		if len(tagNames) > 0 {
			body["tags"] = tagNames
		}
		rec := doJSON(t, engine, http.MethodPost, "/api/documents", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s: status %d (%s)", title, rec.Code, rec.Body.String())
		}
	}
	return engine, create
}

// insertPrivate inserts a private document directly through the repository,
// bypassing the public API which forbids section=private.
func insertPrivate(t *testing.T, pool *pgxpool.Pool, title string, tags ...string) model.Document {
	t.Helper()
	doc, err := repository.NewDocumentRepository(pool).Create(context.Background(), repository.CreateDocumentInput{
		Title:   title,
		Section: model.SectionPrivate,
		Tags:    tags,
	})
	if err != nil {
		t.Fatalf("insert private %q: %v", title, err)
	}
	return doc
}

func TestTagsListAndPopular(t *testing.T) {
	engine, create := newTagTestServer(t)

	create("One", "workflow", "Git", "CLI")
	create("Two", "cheat_sheet", "Git")

	rec := doJSON(t, engine, http.MethodGet, "/api/tags", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	resp := decodeBody[tagsResponse](t, rec)
	if resp.Pagination.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Pagination.Total)
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/tags/popular", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("popular status = %d", rec.Code)
	}
	popular := decodeBody[tagsResponse](t, rec)
	if len(popular.Data) == 0 {
		t.Fatal("expected popular tags")
	}
	if popular.Data[0].Name != "Git" || popular.Data[0].Count != 2 {
		t.Errorf("top tag = %+v, want Git count 2", popular.Data[0])
	}
}

func TestTagsDocuments(t *testing.T) {
	engine, create := newTagTestServer(t)
	create("Git Guide", "workflow", "Git")
	create("Other", "workflow", "Misc")

	rec := doJSON(t, engine, http.MethodGet, "/api/tags/Git/documents", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[listResponse](t, rec)
	if resp.Pagination.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("total=%d len=%d, want 1/1", resp.Pagination.Total, len(resp.Data))
	}
	if resp.Data[0].Title != "Git Guide" {
		t.Errorf("unexpected doc: %+v", resp.Data[0])
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/tags/Unknown/documents", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown tag status = %d, want 404", rec.Code)
	}
}

func TestTagsPrivateIsolation(t *testing.T) {
	engine, pool := newTestServer(t)

	// Public document with a shared tag.
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Public", "section": "workflow", "tags": []string{"shared"},
	})
	// Private documents inserted directly: the public API forbids private.
	insertPrivate(t, pool, "Secret One", "private-only", "shared")
	insertPrivate(t, pool, "Secret Two", "private-only")

	tags := NewTagHandler(repository.NewTagRepository(pool), repository.NewDocumentRepository(pool), nil)
	engine.GET("/api/tags", tags.List)
	engine.GET("/api/tags/popular", tags.Popular)
	engine.GET("/api/tags/:name/documents", tags.Documents)

	// Tag list must not include the private-only tag.
	rec := doJSON(t, engine, http.MethodGet, "/api/tags", nil)
	resp := decodeBody[tagsResponse](t, rec)
	for _, tag := range resp.Data {
		if tag.Name == "private-only" {
			t.Errorf("private-only tag leaked into list: %+v", tag)
		}
	}

	// Shared tag count must exclude private documents (1, not 3).
	rec = doJSON(t, engine, http.MethodGet, "/api/tags/popular", nil)
	popular := decodeBody[tagsResponse](t, rec)
	for _, tag := range popular.Data {
		if tag.Name == "shared" && tag.Count != 1 {
			t.Errorf("shared count = %d, want 1 (private excluded)", tag.Count)
		}
	}

	// Documents-by-tag must never return private documents.
	rec = doJSON(t, engine, http.MethodGet, "/api/tags/shared/documents", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("shared documents status = %d", rec.Code)
	}
	docs := decodeBody[listResponse](t, rec)
	if docs.Pagination.Total != 1 || len(docs.Data) != 1 {
		t.Fatalf("shared docs total=%d len=%d, want 1/1", docs.Pagination.Total, len(docs.Data))
	}

	// A private-only tag must look unknown.
	rec = doJSON(t, engine, http.MethodGet, "/api/tags/private-only/documents", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("private-only tag status = %d, want 404", rec.Code)
	}
}
