package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

// SearchHandler serves GET /api/search (docs/api.md section 6).
type SearchHandler struct {
	docs *repository.DocumentRepository
}

// NewSearchHandler constructs a SearchHandler.
func NewSearchHandler(docs *repository.DocumentRepository) *SearchHandler {
	return &SearchHandler{docs: docs}
}

// Search handles GET /api/search?q=&section=&page=&page_size=.
func (h *SearchHandler) Search(c *gin.Context) {
	query := c.Query("q")

	opts := repository.SearchOptions{
		Query:    query,
		Sections: parseSections(c.QueryArray("section")),
		Page:     1,
		PageSize: model.DefaultPageSize,
	}
	for _, s := range opts.Sections {
		if !s.Valid() || s.IsPrivate() {
			// Private documents are never searchable (PRD 3.3), regardless of
			// session state; querying that section is simply invalid here.
			respondError(c, http.StatusForbidden, CodeForbidden,
				"private documents are never included in search")
			return
		}
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

	results, pagination, err := h.docs.Search(c.Request.Context(), opts)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"pagination": pagination,
		"query":      query,
	})
}
