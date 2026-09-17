package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/markdown"
	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

// ImportHandler serves POST /api/documents/import (docs/api.md 4.7).
type ImportHandler struct {
	docs           *repository.DocumentRepository
	activity       *repository.ActivityRepository
	maxImportBytes int64
}

// NewImportHandler constructs an ImportHandler. maxImportBytes caps each file.
func NewImportHandler(docs *repository.DocumentRepository, activity *repository.ActivityRepository, maxImportBytes int64) *ImportHandler {
	if maxImportBytes <= 0 {
		maxImportBytes = 2 << 20
	}
	return &ImportHandler{docs: docs, activity: activity, maxImportBytes: maxImportBytes}
}

// importResult is one entry in the import response (api.md 4.7).
type importResult struct {
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	DocumentID string `json:"document_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// Import handles POST /api/documents/import (multipart/form-data).
func (h *ImportHandler) Import(c *gin.Context) {
	// Bound the whole request body: files + fields must fit.
	if err := c.Request.ParseMultipartForm(h.maxImportBytes); err != nil {
		respondError(c, http.StatusRequestEntityTooLarge, CodePayloadTooLarge,
			"upload exceeds the configured size limit")
		return
	}

	section := model.Section(strings.TrimSpace(c.PostForm("section")))
	if !section.Valid() {
		respondValidation(c, "invalid or missing section",
			fieldError{Field: "section", Issue: "must be a valid section"})
		return
	}
	if section.IsPrivate() {
		respondError(c, http.StatusForbidden, CodeForbidden,
			"use /api/private/documents/import to import private documents")
		return
	}

	parentID := strings.TrimSpace(c.PostForm("parent_id"))
	var parentPtr *string
	if parentID != "" {
		if section != model.SectionProjectNote {
			respondValidation(c, "parent_id is only allowed for project_note imports",
				fieldError{Field: "parent_id", Issue: "only valid for project_note"})
			return
		}
		parentPtr = &parentID
	}

	defaultTags := markdown.SplitTags(c.PostForm("tags"))

	form, err := c.MultipartForm()
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid multipart form")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		respondValidation(c, "at least one file is required",
			fieldError{Field: "files", Issue: "required"})
		return
	}

	results := make([]importResult, 0, len(files))
	summary := gin.H{"created": 0, "skipped": 0, "failed": 0}

	for _, header := range files {
		result := h.importFile(c, header, section, parentPtr, defaultTags)
		results = append(results, result)
		switch result.Status {
		case "created":
			summary["created"] = summary["created"].(int) + 1
		case "skipped":
			summary["skipped"] = summary["skipped"].(int) + 1
		default:
			summary["failed"] = summary["failed"].(int) + 1
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "summary": summary})
}

// importFile reads, parses, and persists a single uploaded file.
func (h *ImportHandler) importFile(
	c *gin.Context,
	header *multipart.FileHeader,
	section model.Section,
	parentID *string,
	defaultTags []string,
) importResult {
	filename := filepath.Base(header.Filename)
	base := importResult{Filename: filename}

	if !strings.EqualFold(filepath.Ext(filename), ".md") {
		base.Status = "skipped"
		base.Reason = "not_markdown"
		return base
	}
	if header.Size > h.maxImportBytes {
		base.Status = "failed"
		base.Reason = "file_too_large"
		return base
	}

	content, err := readUpload(header, h.maxImportBytes)
	if err != nil {
		if errors.Is(err, errTooLarge) {
			base.Status = "failed"
			base.Reason = "file_too_large"
			return base
		}
		base.Status = "failed"
		base.Reason = "read_error"
		return base
	}
	if strings.TrimSpace(content) == "" {
		base.Status = "skipped"
		base.Reason = "empty_file"
		return base
	}

	parsed := markdown.Parse(content)
	title := parsed.Title
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	if strings.TrimSpace(title) == "" {
		base.Status = "skipped"
		base.Reason = "no_title"
		return base
	}

	tags := markdown.MergeTags(parsed.Tags, defaultTags)
	doc, err := h.docs.Create(c.Request.Context(), repository.CreateDocumentInput{
		Title:    title,
		Content:  parsed.Body,
		Section:  section,
		ParentID: parentID,
		Tags:     tags,
	})
	if err != nil {
		base.Status = "failed"
		base.Reason = importFailureReason(err)
		return base
	}

	base.Status = "created"
	base.DocumentID = doc.ID

	if h.activity != nil {
		s := doc.Section
		title := doc.Title
		// Imported documents are non-private here, so the activity is recorded.
		_ = h.activity.Log(c.Request.Context(), repository.LogActivityInput{
			Action:        model.ActionImported,
			DocumentID:    &doc.ID,
			DocumentTitle: &title,
			Section:       &s,
		})
	}
	return base
}

var errTooLarge = errors.New("file too large")

// readUpload reads at most max bytes plus one, so oversized files are detected
// even when the multipart header lies about the size.
func readUpload(header *multipart.FileHeader, max int64) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > max {
		return "", errTooLarge
	}
	return string(data), nil
}

func importFailureReason(err error) string {
	switch {
	case errors.Is(err, repository.ErrConflict):
		return "slug_conflict"
	case errors.Is(err, repository.ErrInvalidHierarchy):
		return "invalid_parent"
	default:
		return "create_failed"
	}
}
