package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

// DocumentHandler serves the public documents API (docs/api.md section 4).
type DocumentHandler struct {
	docs     *repository.DocumentRepository
	activity *repository.ActivityRepository
}

// NewDocumentHandler constructs a DocumentHandler.
func NewDocumentHandler(docs *repository.DocumentRepository, activity *repository.ActivityRepository) *DocumentHandler {
	return &DocumentHandler{docs: docs, activity: activity}
}

// createDocumentRequest is the POST /api/documents body (api.md 4.1).
type createDocumentRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Excerpt  *string  `json:"excerpt"`
	Section  string   `json:"section"`
	ParentID *string  `json:"parent_id"`
	Position *int     `json:"position"`
	Tags     []string `json:"tags"`
}

// updateDocumentRequest is the PUT /api/documents/:id body (api.md 4.4). All
// fields are optional; nil pointers leave the stored value unchanged, except
// for ParentID, which tracks presence separately so it can be cleared with an
// explicit null.
type updateDocumentRequest struct {
	Title    *string        `json:"title"`
	Content  *string        `json:"content"`
	Excerpt  *string        `json:"excerpt"`
	Section  *string        `json:"section"`
	ParentID nullableString `json:"parent_id"`
	Position *int           `json:"position"`
	Tags     *[]string      `json:"tags"`
}

// Create handles POST /api/documents.
func (h *DocumentHandler) Create(c *gin.Context) {
	var req createDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "malformed JSON body")
		return
	}

	section := model.Section(req.Section)
	if !section.Valid() {
		respondValidation(c, "invalid or missing section",
			fieldError{Field: "section", Issue: "must be one of workflow, project_note, cheat_sheet, private"})
		return
	}
	if section.IsPrivate() {
		respondError(c, http.StatusForbidden, CodeForbidden,
			"use /api/private/documents to create private documents")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		respondValidation(c, "title is required", fieldError{Field: "title", Issue: "required"})
		return
	}

	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	doc, err := h.docs.Create(c.Request.Context(), repository.CreateDocumentInput{
		Title:    strings.TrimSpace(req.Title),
		Content:  req.Content,
		Excerpt:  req.Excerpt,
		Section:  section,
		ParentID: req.ParentID,
		Position: position,
		Tags:     req.Tags,
	})
	if err != nil {
		handleRepoError(c, err)
		return
	}

	h.logActivity(c, model.ActionCreated, &doc)
	c.JSON(http.StatusCreated, doc)
}

// List handles GET /api/documents.
func (h *DocumentHandler) List(c *gin.Context) {
	opts, err := parseListOptions(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	if opts.IncludePrivate {
		// Public listing never surfaces private documents (api.md 4.2).
		opts.IncludePrivate = false
	}
	for _, s := range opts.Sections {
		if s.IsPrivate() {
			respondError(c, http.StatusForbidden, CodeForbidden, "private documents are not listed here")
			return
		}
	}

	docs, pagination, err := h.docs.List(c.Request.Context(), opts)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs, "pagination": pagination})
}

// Get handles GET /api/documents/:id.
func (h *DocumentHandler) Get(c *gin.Context) {
	doc, err := h.docs.GetByID(c.Request.Context(), c.Param("id"), false)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, doc)
}

// Update handles PUT /api/documents/:id.
func (h *DocumentHandler) Update(c *gin.Context) {
	var req updateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "malformed JSON body")
		return
	}

	if _, err := h.docs.GetByID(c.Request.Context(), c.Param("id"), false); err != nil {
		handleRepoError(c, err)
		return
	}

	in, validation := buildUpdateInput(req)
	if validation != nil {
		respondValidation(c, validation.message, validation.details...)
		return
	}
	if in.Section != nil && in.Section.IsPrivate() {
		respondError(c, http.StatusForbidden, CodeForbidden,
			"moving a document to the private section is not permitted here")
		return
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		respondValidation(c, "title must not be empty", fieldError{Field: "title", Issue: "required"})
		return
	}

	doc, err := h.docs.Update(c.Request.Context(), c.Param("id"), in, false)
	if err != nil {
		handleRepoError(c, err)
		return
	}

	h.logActivity(c, model.ActionUpdated, &doc)
	c.JSON(http.StatusOK, doc)
}

// Delete handles DELETE /api/documents/:id.
func (h *DocumentHandler) Delete(c *gin.Context) {
	doc, err := h.docs.GetByID(c.Request.Context(), c.Param("id"), false)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	// Record before deleting so the FK reference is valid; the delete then sets
	// activity_logs.document_id to NULL while keeping the title snapshot.
	h.logActivity(c, model.ActionDeleted, &doc)
	if err := h.docs.Delete(c.Request.Context(), c.Param("id"), false); err != nil {
		handleRepoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Tree handles GET /api/documents/tree?section=project_note.
func (h *DocumentHandler) Tree(c *gin.Context) {
	section := model.Section(c.DefaultQuery("section", string(model.SectionProjectNote)))
	if !section.Valid() {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid section")
		return
	}
	if section.IsPrivate() {
		respondError(c, http.StatusForbidden, CodeForbidden, "private documents are not listed here")
		return
	}

	nodes, err := h.docs.Tree(c.Request.Context(), section, false)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

// logActivity records a feed entry; a failure here must not fail the request.
func (h *DocumentHandler) logActivity(c *gin.Context, action model.Action, doc *model.Document) {
	if h.activity == nil {
		return
	}
	section := doc.Section
	titled := doc.Title
	if err := h.activity.Log(c.Request.Context(), repository.LogActivityInput{
		Action:        action,
		DocumentID:    &doc.ID,
		DocumentTitle: &titled,
		Section:       &section,
	}); err != nil {
		// Logged, but not surfaced: the mutation already succeeded.
		_ = err
	}
}

type validationFailure struct {
	message string
	details []fieldError
}

// buildUpdateInput converts the request DTO into a repository update.
func buildUpdateInput(req updateDocumentRequest) (repository.UpdateDocumentInput, *validationFailure) {
	in := repository.UpdateDocumentInput{
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		Position:  req.Position,
		SetParent: req.ParentID.Present,
	}
	if req.ParentID.Present {
		in.ParentID = req.ParentID.Value
	}
	if req.Excerpt != nil {
		in.Excerpt = req.Excerpt
		in.SetExcerpt = true
	}
	if req.Section != nil {
		s := model.Section(*req.Section)
		if !s.Valid() {
			return in, &validationFailure{message: "invalid section",
				details: []fieldError{{Field: "section", Issue: "unknown section"}}}
		}
		in.Section = &s
	}
	return in, nil
}

// parseListOptions reads and validates the GET /api/documents query string.
func parseListOptions(c *gin.Context) (repository.ListDocumentsOptions, error) {
	opts := repository.ListDocumentsOptions{
		Sections:       parseSections(c.QueryArray("section")),
		Tags:           c.QueryArray("tag"),
		Query:          c.Query("q"),
		IncludeContent: c.Query("include") == "content",
		Sort:           c.DefaultQuery("sort", "updated_at"),
		Order:          c.DefaultQuery("order", "desc"),
		Page:           1,
		PageSize:       model.DefaultPageSize,
	}

	for _, raw := range opts.Sections {
		if !raw.Valid() {
			return opts, errInvalidParam("section", string(raw))
		}
	}

	if pid := c.Query("parent_id"); pid != "" {
		opts.ParentID = &pid
	}

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		return opts, errInvalidParam("page", c.Query("page"))
	}
	pageSize, err := parsePositiveInt(c.Query("page_size"), model.DefaultPageSize)
	if err != nil {
		return opts, errInvalidParam("page_size", c.Query("page_size"))
	}
	opts.Page = page
	opts.PageSize = pageSize

	switch opts.Sort {
	case "updated_at", "created_at", "title":
	default:
		return opts, errInvalidParam("sort", opts.Sort)
	}
	if opts.Order != "asc" && opts.Order != "desc" {
		return opts, errInvalidParam("order", opts.Order)
	}
	return opts, nil
}

func parseSections(values []string) []model.Section {
	out := make([]model.Section, 0, len(values))
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, model.Section(part))
			}
		}
	}
	return out
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, errInvalidParam("", raw)
	}
	return n, nil
}

type paramError struct {
	field string
	value string
}

func (e paramError) Error() string {
	if e.field == "" {
		return "invalid value " + strconv.Quote(e.value)
	}
	return "invalid value for " + e.field
}

func errInvalidParam(field, value string) error {
	return paramError{field: field, value: value}
}
