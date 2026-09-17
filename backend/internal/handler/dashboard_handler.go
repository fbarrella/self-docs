package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/model"
	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

// DashboardHandler serves GET /api/dashboard (docs/api.md section 8).
type DashboardHandler struct {
	docs     *repository.DocumentRepository
	tags     *repository.TagRepository
	activity *repository.ActivityRepository
	sessions *private.SessionStore
	auth     *PrivateAuth
}

// NewDashboardHandler constructs a DashboardHandler.
func NewDashboardHandler(
	docs *repository.DocumentRepository,
	tags *repository.TagRepository,
	activity *repository.ActivityRepository,
	sessions *private.SessionStore,
	auth *PrivateAuth,
) *DashboardHandler {
	return &DashboardHandler{docs: docs, tags: tags, activity: activity, sessions: sessions, auth: auth}
}

// dashboardCard is one navigation card.
type dashboardCard struct {
	ID     model.Section `json:"id"`
	Title  string        `json:"title"`
	Count  *int          `json:"count"`
	Route  string        `json:"route"`
	Locked bool          `json:"locked"`
}

// recentItem is a recently-updated row.
type recentItem struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Section   model.Section `json:"section"`
	UpdatedAt string        `json:"updated_at"`
}

// Dashboard handles GET /api/dashboard.
func (h *DashboardHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	counts, err := h.docs.CountBySection(ctx, false)
	if err != nil {
		handleRepoError(c, err)
		return
	}

	unlocked := h.isUnlocked(c)

	cards := []dashboardCard{
		cardFor(model.SectionWorkflow, counts),
		cardFor(model.SectionProjectNote, counts),
		cardFor(model.SectionCheatSheet, counts),
		privateCard(unlocked),
	}

	recent, err := h.docs.RecentlyUpdated(ctx, 8, false)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	recentItems := make([]recentItem, 0, len(recent))
	for _, doc := range recent {
		recentItems = append(recentItems, recentItem{
			ID:        doc.ID,
			Title:     doc.Title,
			Section:   doc.Section,
			UpdatedAt: doc.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	popularTags, err := h.tags.Popular(ctx, 12)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	if popularTags == nil {
		popularTags = []model.Tag{}
	}

	activity, err := h.activity.Recent(ctx, 10)
	if err != nil {
		handleRepoError(c, err)
		return
	}
	if activity == nil {
		activity = []model.Activity{}
	}

	c.JSON(http.StatusOK, gin.H{
		"cards":            cards,
		"recently_updated": recentItems,
		"popular_tags":     popularTags,
		"activity":         activity,
	})
}

// isUnlocked reports whether the request carries a valid private session,
// without extending it.
func (h *DashboardHandler) isUnlocked(c *gin.Context) bool {
	if h.sessions == nil || h.auth == nil {
		return false
	}
	token := tokenFromRequest(c)
	if token == "" {
		return false
	}
	return h.sessions.Valid(token)
}

// sectionRoutes maps each public section to its SPA route.
var sectionRoutes = map[model.Section]string{
	model.SectionWorkflow:    "/workflows",
	model.SectionProjectNote: "/knowledge-base",
	model.SectionCheatSheet:  "/cheat-sheets",
	model.SectionPrivate:     "/private",
}

func cardFor(section model.Section, counts map[model.Section]int) dashboardCard {
	count := counts[section]
	return dashboardCard{
		ID:     section,
		Title:  section.Label(),
		Count:  &count,
		Route:  sectionRoutes[section],
		Locked: false,
	}
}

// privateCard omits the count and reports the session's locked state.
func privateCard(unlocked bool) dashboardCard {
	return dashboardCard{
		ID:     model.SectionPrivate,
		Title:  model.SectionPrivate.Label(),
		Count:  nil,
		Route:  sectionRoutes[model.SectionPrivate],
		Locked: !unlocked,
	}
}
