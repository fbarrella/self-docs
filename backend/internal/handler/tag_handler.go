package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/cache"
	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

// TagHandler serves the tags API (docs/api.md section 5).
type TagHandler struct {
	tags  *repository.TagRepository
	docs  *repository.DocumentRepository
	cache *cache.Cache
}

// NewTagHandler constructs a TagHandler. cache may be nil.
func NewTagHandler(tags *repository.TagRepository, docs *repository.DocumentRepository, cache *cache.Cache) *TagHandler {
	return &TagHandler{tags: tags, docs: docs, cache: cache}
}

// List handles GET /api/tags.
func (h *TagHandler) List(c *gin.Context) {
	opts := repository.ListTagsOptions{
		Query:    c.Query("q"),
		Sort:     c.DefaultQuery("sort", "name"),
		Order:    c.DefaultQuery("order", "asc"),
		Page:     1,
		PageSize: model.DefaultPageSize,
	}

	if opts.Sort != "name" && opts.Sort != "count" {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid value for sort")
		return
	}
	if opts.Order != "asc" && opts.Order != "desc" {
		respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid value for order")
		return
	}

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
	opts.Page = page
	opts.PageSize = pageSize

	tags, pagination, err := h.tags.List(c.Request.Context(), opts)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tags, "pagination": pagination})
}

// Popular handles GET /api/tags/popular.
func (h *TagHandler) Popular(c *gin.Context) {
	limit := 12
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			respondError(c, http.StatusBadRequest, CodeBadRequest, "invalid value for limit")
			return
		}
		limit = n
	}

	// Popular tags are read-mostly; serve from cache when available.
	var tags []model.Tag
	var err error
	if h.cache.GetJSON(c.Request.Context(), cache.KeyPopularTags, &tags) {
		c.JSON(http.StatusOK, gin.H{"data": tags})
		return
	}

	tags, err = h.tags.Popular(c.Request.Context(), limit)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	h.cache.SetJSON(c.Request.Context(), cache.KeyPopularTags, tags)
	c.JSON(http.StatusOK, gin.H{"data": tags})
}

// Delete handles DELETE /api/tags/:name. Removing a tag detaches it from all
// documents (document_tags cascades).
func (h *TagHandler) Delete(c *gin.Context) {
	if err := h.tags.Delete(c.Request.Context(), c.Param("name")); err != nil {
		handleRepoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Documents handles GET /api/tags/:name/documents.
func (h *TagHandler) Documents(c *gin.Context) {
	name := c.Param("name")

	// A tag attached only to private documents must behave as unknown.
	if _, err := h.tags.GetVisibleByNormalized(c.Request.Context(), name); err != nil {
		handleRepoError(c, err)
		return
	}

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
		Tags:     []string{name},
		Sort:     "updated_at",
		Order:    "desc",
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs, "pagination": pagination})
}
