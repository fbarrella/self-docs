package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

// PrivateHandler serves unlock/lock and the private document endpoints
// (docs/api.md section 7).
type PrivateHandler struct {
	settings *repository.SettingsRepository
	sessions *private.SessionStore
	limiter  *private.RateLimiter
	auth     *PrivateAuth
	docs     *repository.DocumentRepository
	activity *repository.ActivityRepository
}

// NewPrivateHandler constructs a PrivateHandler.
func NewPrivateHandler(
	settings *repository.SettingsRepository,
	sessions *private.SessionStore,
	limiter *private.RateLimiter,
	auth *PrivateAuth,
	docs *repository.DocumentRepository,
	activity *repository.ActivityRepository,
) *PrivateHandler {
	return &PrivateHandler{
		settings: settings,
		sessions: sessions,
		limiter:  limiter,
		auth:     auth,
		docs:     docs,
		activity: activity,
	}
}

type unlockRequest struct {
	Password string `json:"password"`
}

// Unlock handles POST /api/private/unlock.
func (h *PrivateHandler) Unlock(c *gin.Context) {
	var req unlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	key := c.ClientIP()
	if !h.limiter.Allow(key) {
		respondError(c, http.StatusTooManyRequests, CodeRateLimited,
			"too many unlock attempts; try again later")
		return
	}

	hash, err := h.settings.GetMasterPasswordHash(c.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrMasterPasswordUnset) {
			respondError(c, http.StatusUnauthorized, CodeUnauthorized,
				"master password is not configured")
			return
		}
		respondInternal(c, err)
		return
	}

	if req.Password == "" || !private.VerifyPassword(hash, req.Password) {
		remaining := h.limiter.RecordFailure(key)
		if remaining == 0 {
			respondError(c, http.StatusTooManyRequests, CodeRateLimited,
				"too many unlock attempts; try again later")
			return
		}
		respondError(c, http.StatusUnauthorized, CodeUnauthorized, "incorrect password")
		return
	}

	h.limiter.Reset(key)
	token, expiresAt, err := h.sessions.Create()
	if err != nil {
		respondInternal(c, err)
		return
	}
	h.auth.SetCookie(c, token, expiresAt)
	h.logUnlocked(c)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"token":      token,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		},
	})
}

// Lock handles POST /api/private/lock.
func (h *PrivateHandler) Lock(c *gin.Context) {
	if token, ok := c.Get("private_token"); ok {
		if s, ok := token.(string); ok {
			h.sessions.Invalidate(s)
		}
	}
	h.auth.ClearCookie(c)
	c.Status(http.StatusNoContent)
}

// List handles GET /api/private/documents.
func (h *PrivateHandler) List(c *gin.Context) {
	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid value for page")
		return
	}
	pageSize, err := parsePositiveInt(c.Query("page_size"), model.DefaultPageSize)
	if err != nil {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid value for page_size")
		return
	}

	docs, pagination, err := h.docs.List(c.Request.Context(), repository.ListDocumentsOptions{
		Sections:       []model.Section{model.SectionPrivate},
		IncludeContent: c.Query("include") == "content",
		Sort:           "updated_at",
		Order:          "desc",
		Page:           page,
		PageSize:       pageSize,
		IncludePrivate: true,
	})
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs, "pagination": pagination})
}

// Create handles POST /api/private/documents.
func (h *PrivateHandler) Create(c *gin.Context) {
	var req createDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
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
		Section:  model.SectionPrivate,
		Position: position,
		Tags:     req.Tags,
	})
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusCreated, doc)
}

// Get handles GET /api/private/documents/:id.
func (h *PrivateHandler) Get(c *gin.Context) {
	doc, err := h.getPrivate(c)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, doc)
}

// Update handles PUT /api/private/documents/:id.
func (h *PrivateHandler) Update(c *gin.Context) {
	if _, err := h.getPrivate(c); err != nil {
		handleRepoError(c, err)
		return
	}

	var req updateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	in, validation := buildUpdateInput(req)
	if validation != nil {
		respondValidation(c, validation.message, validation.details...)
		return
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		respondValidation(c, "title must not be empty", fieldError{Field: "title", Issue: "required"})
		return
	}
	// A private document must stay private; moving sections is not allowed here.
	section := model.SectionPrivate
	in.Section = &section
	in.SetParent = false
	in.ParentID = nil

	doc, err := h.docs.Update(c.Request.Context(), c.Param("id"), in, true)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, doc)
}

// Delete handles DELETE /api/private/documents/:id.
func (h *PrivateHandler) Delete(c *gin.Context) {
	if _, err := h.getPrivate(c); err != nil {
		handleRepoError(c, err)
		return
	}
	if err := h.docs.Delete(c.Request.Context(), c.Param("id"), true); err != nil {
		handleRepoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// getPrivate loads a document and verifies it belongs to the Private Archive.
func (h *PrivateHandler) getPrivate(c *gin.Context) (model.Document, error) {
	doc, err := h.docs.GetByID(c.Request.Context(), c.Param("id"), true)
	if err != nil {
		return model.Document{}, err
	}
	if doc.Section != model.SectionPrivate {
		// Do not reveal public documents through private routes.
		return model.Document{}, repository.ErrNotFound
	}
	return doc, nil
}

// logUnlocked records an unlock action. It carries no document reference and is
// allowed in the feed (the action itself does not leak private titles).
func (h *PrivateHandler) logUnlocked(c *gin.Context) {
	if h.activity == nil {
		return
	}
	actor := "user"
	_ = h.activity.Log(c.Request.Context(), repository.LogActivityInput{
		Action: model.ActionUnlocked,
		Actor:  actor,
	})
}
