package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

type activityResponse struct {
	Data       []model.Activity `json:"data"`
	Pagination model.Pagination `json:"pagination"`
}

// newActivityTestServer adds the activity route to the shared test engine.
func newActivityTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	engine, pool := newTestServer(t)
	activity := NewActivityHandler(repository.NewActivityRepository(pool))
	engine.GET("/api/activity", activity.List)
	return engine, pool
}

func TestActivityChronologicalOrder(t *testing.T) {
	engine, _ := newActivityTestServer(t)

	// Create two documents; the handler logs created/updated/deleted.
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "First", "section": "workflow",
	})
	first := decodeBody[model.Document](t, rec)
	time.Sleep(10 * time.Millisecond)
	doJSON(t, engine, http.MethodPut, "/api/documents/"+first.ID, map[string]any{"content": "changed"})

	rec = doJSON(t, engine, http.MethodGet, "/api/activity", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBody[activityResponse](t, rec)
	if resp.Pagination.Total != 2 || len(resp.Data) != 2 {
		t.Fatalf("total=%d len=%d, want 2/2", resp.Pagination.Total, len(resp.Data))
	}
	if resp.Data[0].CreatedAt.Before(resp.Data[1].CreatedAt) {
		t.Error("activity is not newest-first")
	}
	if resp.Data[0].Action != model.ActionUpdated {
		t.Errorf("newest action = %q, want updated", resp.Data[0].Action)
	}
}

func TestActivityExcludesPrivate(t *testing.T) {
	engine, pool := newActivityTestServer(t)

	// A public mutation logs an entry.
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Public Action", "section": "workflow",
	})
	// Directly insert private activity rows; the repository refuses them.
	insertPrivate(t, pool, "Secret", "hidden-tag")

	rec := doJSON(t, engine, http.MethodGet, "/api/activity", nil)
	resp := decodeBody[activityResponse](t, rec)
	if resp.Pagination.Total != 1 {
		t.Fatalf("total = %d, want 1 (private excluded)", resp.Pagination.Total)
	}
	for _, entry := range resp.Data {
		if entry.Section != nil && *entry.Section == model.SectionPrivate {
			t.Errorf("private activity leaked: %+v", entry)
		}
	}
}

func TestActivityPagination(t *testing.T) {
	engine, _ := newActivityTestServer(t)

	for _, title := range []string{"One", "Two", "Three"} {
		doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
			"title": title, "section": "workflow",
		})
	}

	rec := doJSON(t, engine, http.MethodGet, "/api/activity?page=1&page_size=2", nil)
	resp := decodeBody[activityResponse](t, rec)
	if resp.Pagination.Total != 3 || resp.Pagination.TotalPages != 2 || len(resp.Data) != 2 {
		t.Errorf("unexpected pagination: %+v (len=%d)", resp.Pagination, len(resp.Data))
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/activity?page=2&page_size=2", nil)
	resp = decodeBody[activityResponse](t, rec)
	if len(resp.Data) != 1 {
		t.Errorf("page 2 len = %d, want 1", len(resp.Data))
	}
}

func TestActivityInvalidPagination(t *testing.T) {
	engine, _ := newActivityTestServer(t)
	rec := doJSON(t, engine, http.MethodGet, "/api/activity?page=0", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
