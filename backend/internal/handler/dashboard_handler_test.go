package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

type dashboardResponse struct {
	Cards           []dashboardCard `json:"cards"`
	RecentlyUpdated []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Section string `json:"section"`
	} `json:"recently_updated"`
	PopularTags []model.Tag      `json:"popular_tags"`
	Activity    []model.Activity `json:"activity"`
}

// newDashboardTestServer wires the dashboard route plus document routes and
// returns the session store so tests can simulate an unlocked session.
func newDashboardTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool, *private.SessionStore, *PrivateAuth) {
	t.Helper()
	engine, pool := newTestServer(t)

	documentRepo := repository.NewDocumentRepository(pool)
	activityRepo := repository.NewActivityRepository(pool)
	tags := repository.NewTagRepository(pool)

	sessions := private.NewSessionStore(private.SessionTTL)
	auth := NewPrivateAuth(sessions, false)
	dashboard := NewDashboardHandler(documentRepo, tags, activityRepo, sessions, auth)
	engine.GET("/api/dashboard", dashboard.Dashboard)
	return engine, pool, sessions, auth
}

func TestDashboardCardsAndCounts(t *testing.T) {
	engine, _, _, _ := newDashboardTestServer(t)

	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "W1", "section": "workflow",
	})
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "W2", "section": "workflow",
	})
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "C1", "section": "cheat_sheet",
	})

	rec := doJSON(t, engine, http.MethodGet, "/api/dashboard", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[dashboardResponse](t, rec)
	if len(resp.Cards) != 4 {
		t.Fatalf("cards = %d, want 4", len(resp.Cards))
	}

	byID := map[string]dashboardCard{}
	for _, card := range resp.Cards {
		byID[string(card.ID)] = card
	}
	if card := byID["workflow"]; card.Count == nil || *card.Count != 2 {
		t.Errorf("workflow card = %+v, want count 2", card)
	}
	if card := byID["cheat_sheet"]; card.Count == nil || *card.Count != 1 {
		t.Errorf("cheat_sheet card = %+v, want count 1", card)
	}
	if card := byID["private"]; card.Count != nil {
		t.Errorf("private card count = %v, want null", *card.Count)
	}
	if card := byID["private"]; !card.Locked {
		t.Error("private card should be locked without a session")
	}
}

func TestDashboardExcludesPrivateEverywhere(t *testing.T) {
	engine, pool, _, _ := newDashboardTestServer(t)

	insertPrivate(t, pool, "Secret Doc", "secret-tag")
	doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Public Doc", "section": "workflow", "tags": []string{"public-tag"},
	})

	rec := doJSON(t, engine, http.MethodGet, "/api/dashboard", nil)
	resp := decodeBody[dashboardResponse](t, rec)

	// Recently updated must not include the private doc.
	for _, item := range resp.RecentlyUpdated {
		if item.Section == "private" {
			t.Errorf("private document in recently_updated: %+v", item)
		}
	}
	if len(resp.RecentlyUpdated) != 1 {
		t.Errorf("recently_updated = %d, want 1", len(resp.RecentlyUpdated))
	}

	// Popular tags must not include the private-only tag and count public only.
	for _, tag := range resp.PopularTags {
		if tag.Name == "secret-tag" {
			t.Errorf("private-only tag leaked: %+v", tag)
		}
	}

	// Activity must be empty (private insert does not log).
	for _, a := range resp.Activity {
		if a.Section != nil && *a.Section == model.SectionPrivate {
			t.Errorf("private activity leaked: %+v", a)
		}
	}
}

func TestDashboardPrivateLockedState(t *testing.T) {
	engine, pool, sessions, _ := newDashboardTestServer(t)
	_ = pool

	rec := doJSON(t, engine, http.MethodGet, "/api/dashboard", nil)
	resp := decodeBody[dashboardResponse](t, rec)
	if !cardLocked(resp, "private") {
		t.Fatal("private card should be locked with no session")
	}

	// Simulate an active session and pass the cookie.
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptestNewRequest(t, http.MethodGet, "/api/dashboard", token)
	rec2 := serve(engine, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status = %d", rec2.Code)
	}
	resp2 := decodeBody[dashboardResponse](t, rec2)
	if cardLocked(resp2, "private") {
		t.Error("private card should be unlocked with a valid session")
	}
}

func cardLocked(resp dashboardResponse, id string) bool {
	for _, card := range resp.Cards {
		if string(card.ID) == id {
			return card.Locked
		}
	}
	return true
}
