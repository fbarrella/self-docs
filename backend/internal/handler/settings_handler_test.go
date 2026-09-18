package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

const settingsPassword = "initial-pass"

// newSettingsTestServer wires the settings + tag routes.
func newSettingsTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	engine, pool := newTestServer(t)

	settingsRepo := repository.NewSettingsRepository(pool)
	hash, err := private.HashPassword(settingsPassword)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := settingsRepo.SetMasterPasswordHash(t.Context(), hash); err != nil {
		t.Fatalf("seed hash: %v", err)
	}

	sessions := private.NewSessionStore(private.SessionTTL)
	settings := NewSettingsHandler(settingsRepo, sessions, "test", false)
	tags := NewTagHandler(repository.NewTagRepository(pool), repository.NewDocumentRepository(pool), nil)

	engine.GET("/api/settings", settings.Get)
	engine.PUT("/api/settings/master-password", settings.ChangeMasterPassword)
	engine.DELETE("/api/tags/:name", tags.Delete)
	return engine, pool
}

func TestSettingsGet(t *testing.T) {
	engine, _ := newSettingsTestServer(t)

	rec := doJSON(t, engine, http.MethodGet, "/api/settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decodeBody[struct {
		Data struct {
			Version           string `json:"version"`
			RedisEnabled      bool   `json:"redis_enabled"`
			MasterPasswordSet bool   `json:"master_password_set"`
		} `json:"data"`
	}](t, rec)
	if body.Data.Version != "test" {
		t.Errorf("version = %q", body.Data.Version)
	}
	if !body.Data.MasterPasswordSet {
		t.Error("master_password_set should be true")
	}
}

func TestChangeMasterPasswordWithCurrent(t *testing.T) {
	engine, pool := newSettingsTestServer(t)

	rec := doJSON(t, engine, http.MethodPut, "/api/settings/master-password", map[string]any{
		"current_password": settingsPassword,
		"new_password":     "brand-new-pass",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (%s)", rec.Code, rec.Body.String())
	}

	// The new password verifies against the stored hash.
	hash, err := repository.NewSettingsRepository(pool).GetMasterPasswordHash(t.Context())
	if err != nil {
		t.Fatalf("get hash: %v", err)
	}
	if !private.VerifyPassword(hash, "brand-new-pass") {
		t.Error("new password does not verify")
	}
	if private.VerifyPassword(hash, settingsPassword) {
		t.Error("old password still verifies")
	}
}

func TestChangeMasterPasswordWrongCurrent(t *testing.T) {
	engine, _ := newSettingsTestServer(t)

	rec := doJSON(t, engine, http.MethodPut, "/api/settings/master-password", map[string]any{
		"current_password": "wrong",
		"new_password":     "brand-new-pass",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestChangeMasterPasswordValidation(t *testing.T) {
	engine, _ := newSettingsTestServer(t)

	rec := doJSON(t, engine, http.MethodPut, "/api/settings/master-password", map[string]any{
		"current_password": settingsPassword,
		"new_password":     "short",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDeleteTag(t *testing.T) {
	engine, pool := newSettingsTestServer(t)

	// Create a document with a tag through the repository.
	if _, err := repository.NewDocumentRepository(pool).Create(t.Context(), repository.CreateDocumentInput{
		Title: "Tagged", Section: "workflow", Tags: []string{"removable"},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	rec := doJSON(t, engine, http.MethodDelete, "/api/tags/removable", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	rec = doJSON(t, engine, http.MethodDelete, "/api/tags/removable", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", rec.Code)
	}
}
