package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler reports liveness of the server and its dependencies.
type HealthHandler struct {
	pool         *pgxpool.Pool
	redisEnabled bool
	version      string
}

// NewHealthHandler builds the health handler. pool may be nil in tests that
// only exercise routing.
func NewHealthHandler(pool *pgxpool.Pool, redisEnabled bool, version string) *HealthHandler {
	return &HealthHandler{pool: pool, redisEnabled: redisEnabled, version: version}
}

// healthResponse mirrors the contract in docs/api.md section 10.
type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Redis    string `json:"redis"`
	Version  string `json:"version"`
}

// Health handles GET /healthz and never fails: it reports degraded state in the
// body so an orchestrator can distinguish liveness from dependency health.
func (h *HealthHandler) Health(c *gin.Context) {
	resp := healthResponse{Status: "ok", Database: "ok", Redis: "disabled", Version: h.version}
	status := http.StatusOK

	if h.redisEnabled {
		resp.Redis = "enabled"
	}

	if h.pool != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.pool.Ping(ctx); err != nil {
			resp.Status = "degraded"
			resp.Database = "error"
			status = http.StatusServiceUnavailable
		}
	}

	c.JSON(status, resp)
}
