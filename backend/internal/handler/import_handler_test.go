package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

type importResponse struct {
	Data    []importResult `json:"data"`
	Summary struct {
		Created int `json:"created"`
		Skipped int `json:"skipped"`
		Failed  int `json:"failed"`
	} `json:"summary"`
}

// newImportTestServer adds the import route to the shared test engine, which
// already registers the document routes, and returns the database pool.
func newImportTestServer(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	engine, pool := newTestServer(t)
	importer := NewImportHandler(
		repository.NewDocumentRepository(pool),
		repository.NewActivityRepository(pool),
		nil,
		1<<20,
	)
	engine.POST("/api/documents/import", importer.Import)
	return engine, pool
}

// upload builds a multipart request from named files and extra form fields.
func upload(t *testing.T, engine *gin.Engine, fields map[string]string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for name, content := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("create file part: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/documents/import", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestImportFrontMatterDocument(t *testing.T) {
	engine, _ := newImportTestServer(t)

	rec := upload(t, engine, map[string]string{"section": "cheat_sheet"}, map[string]string{
		"git.md": "---\ntitle: Git Commands\ntags: [Git, CLI]\n---\n\n# Git\n\nBody.",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[importResponse](t, rec)
	if resp.Summary.Created != 1 || len(resp.Data) != 1 {
		t.Fatalf("summary=%+v data=%+v", resp.Summary, resp.Data)
	}
	if resp.Data[0].Status != "created" || resp.Data[0].DocumentID == "" {
		t.Errorf("unexpected result: %+v", resp.Data[0])
	}
}

func TestImportMultipleFilesWithDefaultTags(t *testing.T) {
	engine, _ := newImportTestServer(t)

	rec := upload(t, engine,
		map[string]string{"section": "project_note", "tags": "imported, shared"},
		map[string]string{
			"a.md":     "# Alpha\n\nContent A.",
			"b.md":     "# Beta\n\nContent B.",
			"skip.txt": "not markdown",
			"empty.md": "   \n",
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[importResponse](t, rec)
	if resp.Summary.Created != 2 || resp.Summary.Skipped != 2 || resp.Summary.Failed != 0 {
		t.Fatalf("summary = %+v, want created 2 skipped 2 failed 0", resp.Summary)
	}

	statuses := map[string]string{}
	for _, r := range resp.Data {
		statuses[r.Filename] = r.Status
	}
	if statuses["skip.txt"] != "skipped" || statuses["empty.md"] != "skipped" {
		t.Errorf("statuses = %v", statuses)
	}
}

func TestImportTitleFromFilename(t *testing.T) {
	engine, pool := newImportTestServer(t)
	documentRepo := repository.NewDocumentRepository(pool)

	rec := upload(t, engine, map[string]string{"section": "workflow"}, map[string]string{
		"my-notes.md": "No heading here, just text.",
	})
	resp := decodeBody[importResponse](t, rec)
	if len(resp.Data) != 1 || resp.Data[0].Status != "created" {
		t.Fatalf("unexpected: %+v", resp.Data)
	}

	doc, err := documentRepo.GetByID(t.Context(), resp.Data[0].DocumentID, false)
	if err != nil {
		t.Fatalf("get imported: %v", err)
	}
	if doc.Title != "my-notes" {
		t.Errorf("Title = %q, want my-notes", doc.Title)
	}
	if doc.Section != model.SectionWorkflow {
		t.Errorf("Section = %q", doc.Section)
	}
}

func TestImportValidation(t *testing.T) {
	engine, _ := newImportTestServer(t)

	tests := []struct {
		name   string
		fields map[string]string
		want   int
	}{
		{"missing section", map[string]string{}, http.StatusBadRequest},
		{"invalid section", map[string]string{"section": "bogus"}, http.StatusBadRequest},
		{"private forbidden", map[string]string{"section": "private"}, http.StatusForbidden},
		{"parent on non-project-note", map[string]string{"section": "workflow", "parent_id": "00000000-0000-0000-0000-000000000000"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := upload(t, engine, tt.fields, map[string]string{"a.md": "# A\ncontent"})
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestImportRequiresFiles(t *testing.T) {
	engine, _ := newImportTestServer(t)
	rec := upload(t, engine, map[string]string{"section": "workflow"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
	}
}

func TestImportIntoProjectNoteFolder(t *testing.T) {
	engine, _ := newImportTestServer(t)

	// Create a parent project note through the API.
	rec := doJSON(t, engine, http.MethodPost, "/api/documents", map[string]any{
		"title": "Parent", "section": "project_note",
	})
	parent := decodeBody[model.Document](t, rec)

	rec = upload(t, engine,
		map[string]string{"section": "project_note", "parent_id": parent.ID},
		map[string]string{"child.md": "# Child\n\nNested."})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	resp := decodeBody[importResponse](t, rec)
	if resp.Summary.Created != 1 {
		t.Fatalf("summary = %+v", resp.Summary)
	}
}
