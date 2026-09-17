package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

type searchResponse struct {
	Data       []model.SearchResult `json:"data"`
	Pagination model.Pagination     `json:"pagination"`
	Query      string               `json:"query"`
}

// newSearchTestServer adds the search route to the shared test engine.
func newSearchTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	engine, pool := newTestServer(t)
	handler := NewSearchHandler(repository.NewDocumentRepository(pool))
	engine.GET("/api/search", handler.Search)
	return engine, pool
}

func TestSearchRanksAndSnippets(t *testing.T) {
	engine, _ := newSearchTestServer(t)

	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Git Rebase Guide", "content": "Learn to rebase interactively.", "section": "workflow",
	})
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Unrelated", "content": "No matching terms here.", "section": "workflow",
	})

	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=rebase", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[searchResponse](t, rec)
	if resp.Query != "rebase" {
		t.Errorf("query = %q, want rebase", resp.Query)
	}
	if resp.Pagination.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("total=%d len=%d, want 1/1", resp.Pagination.Total, len(resp.Data))
	}
	if resp.Data[0].Title != "Git Rebase Guide" {
		t.Errorf("unexpected result: %+v", resp.Data[0])
	}
	if resp.Data[0].Rank <= 0 {
		t.Errorf("rank = %f, want > 0", resp.Data[0].Rank)
	}
	if resp.Data[0].Snippet == "" {
		t.Error("expected a non-empty snippet")
	}
}

func TestSearchPrivateOnlyTermReturnsZero(t *testing.T) {
	engine, pool := newSearchTestServer(t)

	// Public term.
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Public Notes", "content": "publickeyword content", "section": "workflow",
	})
	// Private document with a unique term, inserted via repository.
	insertPrivate(t, pool, "Private Notes", "secretkeyword")

	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=secretkeyword", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBody[searchResponse](t, rec)
	if resp.Pagination.Total != 0 || len(resp.Data) != 0 {
		t.Fatalf("private-only term returned %d results, want 0", len(resp.Data))
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/search?q=publickeyword", nil)
	resp = decodeBody[searchResponse](t, rec)
	if resp.Pagination.Total != 1 {
		t.Fatalf("public term total = %d, want 1", resp.Pagination.Total)
	}
}

func TestSearchRejectsPrivateSection(t *testing.T) {
	engine, _ := newSearchTestServer(t)
	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=x&section=private", nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	engine, _ := newSearchTestServer(t)
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Anything", "section": "workflow",
	})

	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeBody[searchResponse](t, rec)
	if len(resp.Data) != 0 || resp.Pagination.Total != 0 {
		t.Errorf("empty query returned %d results, want 0", len(resp.Data))
	}
}

func TestSearchTitleAndTagMatching(t *testing.T) {
	engine, _ := newSearchTestServer(t)

	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Kubernetes Basics", "content": "pods and services", "section": "workflow", "tags": []string{"devops"},
	})

	// Title match.
	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=kubernetes", nil)
	if resp := decodeBody[searchResponse](t, rec); resp.Pagination.Total != 1 {
		t.Errorf("title search total = %d, want 1", resp.Pagination.Total)
	}
	// Tag match (document has no content containing "devops").
	rec = doJSON(t, engine, http.MethodGet, "/api/search?q=devops", nil)
	if resp := decodeBody[searchResponse](t, rec); resp.Pagination.Total != 1 {
		t.Errorf("tag search total = %d, want 1", resp.Pagination.Total)
	}
}

func TestSearchPagination(t *testing.T) {
	engine, _ := newSearchTestServer(t)
	for _, title := range []string{"keyword one", "keyword two", "keyword three"} {
		doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
			"title": title, "section": "workflow",
		})
	}

	rec := doJSON(t, engine, http.MethodGet, "/api/search?q=keyword&page=1&page_size=2", nil)
	resp := decodeBody[searchResponse](t, rec)
	if resp.Pagination.Total != 3 || resp.Pagination.TotalPages != 2 || len(resp.Data) != 2 {
		t.Errorf("unexpected pagination: %+v (len=%d)", resp.Pagination, len(resp.Data))
	}
}
