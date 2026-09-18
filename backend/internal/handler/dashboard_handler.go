package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/self-docs/backend/internal/cache"
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
	cache    *cache.Cache
}

// NewDashboardHandler constructs a DashboardHandler. cache may be nil.
func NewDashboardHandler(
	docs *repository.DocumentRepository,
	tags *repository.TagRepository,
	activity *repository.ActivityRepository,
	sessions *private.SessionStore,
	auth *PrivateAuth,
	cache *cache.Cache,
) *DashboardHandler {
	return &DashboardHandler{docs: docs, tags: tags, activity: activity, sessions: sessions, auth: auth, cache: cache}
}

// cachedDashboard is the cacheable portion of the dashboard response. The
// private card's locked state is per-session and is not cached.
type cachedDashboard struct {
	Cards           []dashboardCard  `json:"cards"`
	RecentlyUpdated []recentItem     `json:"recently_updated"`
	PopularTags     []model.Tag      `json:"popular_tags"`
	Activity        []model.Activity `json:"activity"`
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
//
// The document-derived payload is cached in Redis when configured; the private
// card's locked state is always computed from the current session so an
// unlock/lock is reflected immediately.
func (h *DashboardHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	unlocked := h.isUnlocked(c)

	var payload cachedDashboard
	if !h.cache.GetJSON(ctx, cache.KeyDashboard, &payload) {
		built, err := h.buildDashboard(ctx)
		if err != nil {
			handleRepoError(c, err)
			return
		}
		payload = built
		h.cache.SetJSON(ctx, cache.KeyDashboard, payload)
	}

	// Override the per-session private card after cache retrieval.
	for i := range payload.Cards {
		if payload.Cards[i].ID == model.SectionPrivate {
			payload.Cards[i] = privateCard(unlocked)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cards":            payload.Cards,
		"recently_updated": payload.RecentlyUpdated,
		"popular_tags":     payload.PopularTags,
		"activity":         payload.Activity,
	})
}

// buildDashboard assembles the cacheable dashboard payload from the database.
func (h *DashboardHandler) buildDashboard(ctx context.Context) (cachedDashboard, error) {
	counts, err := h.docs.CountBySection(ctx, false)
	if err != nil {
		return cachedDashboard{}, err
	}

	cards := []dashboardCard{
		cardFor(model.SectionWorkflow, counts),
		cardFor(model.SectionProjectNote, counts),
		cardFor(model.SectionCheatSheet, counts),
		privateCard(false),
	}

	recent, err := h.docs.RecentlyUpdated(ctx, 8, false)
	if err != nil {
		return cachedDashboard{}, err
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
		return cachedDashboard{}, err
	}
	if popularTags == nil {
		popularTags = []model.Tag{}
	}

	activity, err := h.activity.Recent(ctx, 10)
	if err != nil {
		return cachedDashboard{}, err
	}
	if activity == nil {
		activity = []model.Activity{}
	}

	return cachedDashboard{
		Cards:           cards,
		RecentlyUpdated: recentItems,
		PopularTags:     popularTags,
		Activity:        activity,
	}, nil
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
