// Package router wires HTTP routes to handlers.
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/cache"
	"github.com/self-docs/backend/internal/handler"
	"github.com/self-docs/backend/internal/middleware"
	"github.com/self-docs/backend/internal/private"
	"github.com/self-docs/backend/internal/repository"
)

// maxJSONBodyBytes caps non-upload JSON request bodies (1 MiB).
const maxJSONBodyBytes = 1 << 20

// Deps are the dependencies required to build the router.
type Deps struct {
	Pool           *pgxpool.Pool
	RedisEnabled   bool
	Version        string
	CORSOrigins    []string
	MaxImportBytes int64
	// SecureCookies sets the Secure flag on the private session cookie. Leave
	// false for local HTTP; set true behind TLS.
	SecureCookies bool
	// Cache is the optional Redis cache. A nil or disabled cache is a no-op.
	Cache *cache.Cache
	// TrustedProxies lists CIDRs whose forwarded headers are honored. When
	// empty, no proxies are trusted and ClientIP uses the socket address.
	TrustedProxies []string
}

// New builds the Gin engine with middleware and all registered routes.
func New(deps Deps) *gin.Engine {
	origins := deps.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	engine := gin.New()
	// Only trust X-Forwarded-For from configured proxies; otherwise ClientIP is
	// the socket address, preventing spoofed rate-limit keys.
	if len(deps.TrustedProxies) > 0 {
		if err := engine.SetTrustedProxies(deps.TrustedProxies); err != nil {
			// Invalid CIDRs fall back to the safe default (no proxies trusted).
			_ = engine.SetTrustedProxies(nil)
		}
	} else {
		_ = engine.SetTrustedProxies(nil)
	}
	engine.Use(middleware.RequestID(), middleware.Logger(), gin.Recovery(), middleware.SecurityHeaders())
	// Strip same-origin Origin headers before CORS so proxied same-origin POSTs
	// are not rejected.
	engine.Use(middleware.NormalizeSameOriginOrigin())
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	health := handler.NewHealthHandler(deps.Pool, deps.RedisEnabled, deps.Version)
	engine.GET("/healthz", health.Health)

	c := deps.Cache
	documentRepo := repository.NewDocumentRepository(deps.Pool)
	activityRepo := repository.NewActivityRepository(deps.Pool)
	docs := handler.NewDocumentHandler(documentRepo, activityRepo, c)
	tags := handler.NewTagHandler(repository.NewTagRepository(deps.Pool), documentRepo, c)
	importer := handler.NewImportHandler(documentRepo, activityRepo, c, deps.MaxImportBytes)
	search := handler.NewSearchHandler(documentRepo)
	activity := handler.NewActivityHandler(activityRepo)

	sessions := private.NewSessionStore(private.SessionTTL)
	privateAuth := handler.NewPrivateAuth(sessions, deps.SecureCookies)
	dashboard := handler.NewDashboardHandler(
		documentRepo,
		repository.NewTagRepository(deps.Pool),
		activityRepo,
		sessions,
		privateAuth,
		c,
	)
	settingsRepo := repository.NewSettingsRepository(deps.Pool)
	privateHandler := handler.NewPrivateHandler(
		settingsRepo,
		sessions,
		private.NewRateLimiter(private.MaxUnlockAttempts, private.UnlockWindow),
		privateAuth,
		documentRepo,
		activityRepo,
	)
	settings := handler.NewSettingsHandler(settingsRepo, sessions, deps.Version, deps.RedisEnabled)

	// limitJSON caps JSON bodies; upload routes enforce their own limits.
	limitJSON := middleware.MaxBodyBytes(maxJSONBodyBytes)

	api := engine.Group("/api")
	{
		api.GET("/healthz", health.Health)

		api.POST("/documents", limitJSON, docs.Create)
		api.GET("/documents", docs.List)
		api.GET("/documents/tree", docs.Tree)
		api.POST("/documents/import", importer.Import)
		api.GET("/documents/:id", docs.Get)
		api.PUT("/documents/:id", limitJSON, docs.Update)
		api.DELETE("/documents/:id", docs.Delete)

		api.GET("/tags", tags.List)
		api.GET("/tags/popular", tags.Popular)
		api.GET("/tags/:name/documents", tags.Documents)
		api.DELETE("/tags/:name", limitJSON, tags.Delete)

		api.GET("/search", search.Search)
		api.GET("/dashboard", dashboard.Dashboard)
		api.GET("/activity", activity.List)

		api.GET("/settings", settings.Get)
		api.PUT("/settings/master-password", limitJSON, settings.ChangeMasterPassword)

		api.POST("/private/unlock", limitJSON, privateHandler.Unlock)
		// Lock is unguarded so an already-expired session can still clear its
		// cookie and always receives a 204.
		api.POST("/private/lock", privateAuth.OptionalMiddleware(), privateHandler.Lock)
		priv := api.Group("/private", privateAuth.Middleware())
		{
			priv.GET("/documents", privateHandler.List)
			priv.POST("/documents", limitJSON, privateHandler.Create)
			priv.GET("/documents/:id", privateHandler.Get)
			priv.PUT("/documents/:id", limitJSON, privateHandler.Update)
			priv.DELETE("/documents/:id", privateHandler.Delete)
		}
	}

	return engine
}
