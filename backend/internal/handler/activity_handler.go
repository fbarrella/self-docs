package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/repository"
)

// ActivityHandler serves GET /api/activity (docs/api.md section 9).
type ActivityHandler struct {
	activity *repository.ActivityRepository
}

// NewActivityHandler constructs an ActivityHandler.
func NewActivityHandler(activity *repository.ActivityRepository) *ActivityHandler {
	return &ActivityHandler{activity: activity}
}

// List handles GET /api/activity?page=&page_size=.
func (h *ActivityHandler) List(c *gin.Context) {
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

	entries, pagination, err := h.activity.List(c.Request.Context(), page, pageSize)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entries, "pagination": pagination})
}
