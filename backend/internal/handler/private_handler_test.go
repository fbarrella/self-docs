package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

const testMasterPassword = "s3cret-pass"

// newPrivateTestServer wires the full private route surface plus public
// document/tag/search routes, and seeds the master password hash.
func newPrivateTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	engine, pool := newTestServer(t)

	documentRepo := repository.NewDocumentRepository(pool)
	activityRepo := repository.NewActivityRepository(pool)
	settingsRepo := repository.NewSettingsRepository(pool)
	tags := NewTagHandler(repository.NewTagRepository(pool), documentRepo, nil)
	search := NewSearchHandler(documentRepo)

	hash, err := private.HashPassword(testMasterPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := settingsRepo.SetMasterPasswordHash(t.Context(), hash); err != nil {
		t.Fatalf("seed master password: %v", err)
	}

	sessions := private.NewSessionStore(private.SessionTTL)
	auth := NewPrivateAuth(sessions, false)
	priv := NewPrivateHandler(settingsRepo, sessions,
		private.NewRateLimiter(3, private.UnlockWindow), auth, documentRepo, activityRepo)

	// Public routes beyond the document routes registered by newTestServer.
	engine.GET("/api/tags", tags.List)
	engine.GET("/api/tags/popular", tags.Popular)
	engine.GET("/api/search", search.Search)

	engine.POST("/api/private/unlock", priv.Unlock)
	engine.POST("/api/private/lock", auth.OptionalMiddleware(), priv.Lock)
	guarded := engine.Group("/api/private", auth.Middleware())
	{
		guarded.GET("/documents", priv.List)
		guarded.POST("/documents", priv.Create)
		guarded.GET("/documents/:id", priv.Get)
		guarded.PUT("/documents/:id", priv.Update)
		guarded.DELETE("/documents/:id", priv.Delete)
	}
	return engine, pool
}

// unlock obtains a session token for authenticated private requests.
func unlock(t *testing.T, engine *gin.Engine) string {
	t.Helper()
	rec := doJSON(t, engine, http.MethodPost, "/api/private/unlock",
		map[string]any{"password": testMasterPassword})
	if rec.Code != http.StatusOK {
		t.Fatalf("unlock status = %d (%s)", rec.Code, rec.Body.String())
	}
	body := decodeBody[struct {
		Data struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"data"`
	}](t, rec)
	if body.Data.Token == "" {
		t.Fatal("unlock returned an empty token")
	}
	return body.Data.Token
}

// doAuth issues a request with a bearer token.
func doAuth(t *testing.T, engine *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestPrivateRequiresUnlock(t *testing.T) {
	engine, _ := newPrivateTestServer(t)

	rec := doAuth(t, engine, http.MethodGet, "/api/private/documents", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	env := decodeBody[struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}](t, rec)
	if env.Error.Code != CodeUnauthorized {
		t.Errorf("code = %q, want %q", env.Error.Code, CodeUnauthorized)
	}
}

func TestPrivateWrongPassword(t *testing.T) {
	engine, _ := newPrivateTestServer(t)

	rec := doJSON(t, engine, http.MethodPost, "/api/private/unlock",
		map[string]any{"password": "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestPrivateUnlockAndCrud(t *testing.T) {
	engine, _ := newPrivateTestServer(t)
	token := unlock(t, engine)

	// Create a private document.
	rec := doAuth(t, engine, http.MethodPost, "/api/private/documents", token,
		map[string]any{"title": "Secret Notes", "content": "classified", "tags": []string{"confidential"}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d (%s)", rec.Code, rec.Body.String())
	}
	doc := decodeBody[model.Document](t, rec)
	if doc.Section != model.SectionPrivate {
		t.Errorf("section = %q, want private", doc.Section)
	}

	// Read it back.
	rec = doAuth(t, engine, http.MethodGet, "/api/private/documents/"+doc.ID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	// Update it.
	rec = doAuth(t, engine, http.MethodPut, "/api/private/documents/"+doc.ID, token,
		map[string]any{"content": "updated secret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d (%s)", rec.Code, rec.Body.String())
	}

	// List.
	rec = doAuth(t, engine, http.MethodGet, "/api/private/documents", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	list := decodeBody[listResponse](t, rec)
	if list.Pagination.Total != 1 {
		t.Errorf("private total = %d, want 1", list.Pagination.Total)
	}

	// Delete.
	rec = doAuth(t, engine, http.MethodDelete, "/api/private/documents/"+doc.ID, token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
}

func TestPrivateIsolationAcrossPublicEndpoints(t *testing.T) {
	engine, _ := newPrivateTestServer(t)
	token := unlock(t, engine)

	rec := doAuth(t, engine, http.MethodPost, "/api/private/documents", token,
		map[string]any{"title": "Hidden Rebase", "content": "hiddenkeyword content", "tags": []string{"hidden-tag"}})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d", rec.Code)
	}

	// Public document list.
	rec = doJSON(t, engine, http.MethodGet, "/api/documents", nil)
	if list := decodeBody[listResponse](t, rec); list.Pagination.Total != 0 {
		t.Errorf("public list leaked %d private docs", list.Pagination.Total)
	}

	// Public search.
	rec = doJSON(t, engine, http.MethodGet, "/api/search?q=hiddenkeyword", nil)
	if resp := decodeBody[searchResponse](t, rec); resp.Pagination.Total != 0 {
		t.Errorf("public search leaked %d private docs", resp.Pagination.Total)
	}

	// Tag list and popular.
	rec = doJSON(t, engine, http.MethodGet, "/api/tags", nil)
	if resp := decodeBody[tagsResponse](t, rec); resp.Pagination.Total != 0 {
		t.Errorf("tag list leaked: %+v", resp.Data)
	}
	rec = doJSON(t, engine, http.MethodGet, "/api/tags/popular", nil)
	if resp := decodeBody[tagsResponse](t, rec); len(resp.Data) != 0 {
		t.Errorf("popular tags leaked: %+v", resp.Data)
	}
}

func TestPrivateLockInvalidatesSession(t *testing.T) {
	engine, _ := newPrivateTestServer(t)
	token := unlock(t, engine)

	rec := doAuth(t, engine, http.MethodPost, "/api/private/lock", token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("lock status = %d", rec.Code)
	}
	rec = doAuth(t, engine, http.MethodGet, "/api/private/documents", token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status after lock = %d, want 401", rec.Code)
	}
}

func TestPrivateRateLimit(t *testing.T) {
	engine, _ := newPrivateTestServer(t)

	// The limiter is configured with max=3: the first two failures return 401
	// and the third trips the lockout (429).
	for i := 0; i < 2; i++ {
		rec := doJSON(t, engine, http.MethodPost, "/api/private/unlock",
			map[string]any{"password": "wrong"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := doJSON(t, engine, http.MethodPost, "/api/private/unlock",
		map[string]any{"password": "wrong"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	// Even the correct password is refused while locked out.
	rec = doJSON(t, engine, http.MethodPost, "/api/private/unlock",
		map[string]any{"password": testMasterPassword})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("correct password during lockout status = %d, want 429", rec.Code)
	}
}

func TestPrivateCookieTransport(t *testing.T) {
	engine, _ := newPrivateTestServer(t)

	rec := doJSON(t, engine, http.MethodPost, "/api/private/unlock",
		map[string]any{"password": testMasterPassword})
	if rec.Code != http.StatusOK {
		t.Fatalf("unlock status = %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	var session string
	for _, ck := range cookies {
		if ck.Name == PrivateCookieName {
			session = ck.Value
		}
	}
	if session == "" {
		t.Fatal("unlock did not set the private cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/private/documents", nil)
	req.AddCookie(&http.Cookie{Name: PrivateCookieName, Value: session})
	rec2 := httptest.NewRecorder()
	engine.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("cookie-authenticated status = %d, want 200", rec2.Code)
	}
}

func TestPublicDocumentCreateRejectsPrivate(t *testing.T) {
	engine, _ := newPrivateTestServer(t)
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Nope", "section": "private",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestPrivateCannotReadPublicDocument(t *testing.T) {
	engine, _ := newPrivateTestServer(t)
	token := unlock(t, engine)

	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Public Doc", "section": "workflow",
	})
	pub := decodeBody[model.Document](t, rec)

	rec = doAuth(t, engine, http.MethodGet, "/api/private/documents/"+pub.ID, token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
