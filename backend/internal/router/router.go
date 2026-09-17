// Package router wires HTTP routes to handlers.
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/self-docs/backend/internal/handler"
	"github.com/self-docs/backend/internal/repository"
)

// Deps are the dependencies required to build the router.
type Deps struct {
	Pool         *pgxpool.Pool
	RedisEnabled bool
	Version      string
	CORSOrigins  []string
}

// New builds the Gin engine with middleware and all registered routes.
func New(deps Deps) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     deps.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	health := handler.NewHealthHandler(deps.Pool, deps.RedisEnabled, deps.Version)
	engine.GET("/healthz", health.Health)

	docs := handler.NewDocumentHandler(
		repository.NewDocumentRepository(deps.Pool),
		repository.NewActivityRepository(deps.Pool),
	)

	api := engine.Group("/api")
	{
		api.GET("/healthz", health.Health)

		api.POST("/documents", docs.Create)
		api.GET("/documents", docs.List)
		api.GET("/documents/tree", docs.Tree)
		api.GET("/documents/:id", docs.Get)
		api.PUT("/documents/:id", docs.Update)
		api.DELETE("/documents/:id", docs.Delete)
	}

	return engine
}
