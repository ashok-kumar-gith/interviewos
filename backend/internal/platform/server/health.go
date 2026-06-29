package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/interviewos/backend/internal/platform/database"
)

// HealthHandler serves liveness and readiness probes. It holds optional
// handles to the backing stores; nil handles are treated as "degraded" rather
// than fatal so the process can start even when a dependency is unavailable.
type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthHandler constructs a HealthHandler. Either dependency may be nil
// (graceful degradation); readiness will report it as unavailable.
func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb}
}

// Healthz is the liveness probe: it reports the process is up. It performs no
// dependency checks and always returns 200 while the process can serve.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz is the readiness probe: it verifies the DB and Redis are reachable.
// It returns 200 only when all configured dependencies are healthy, otherwise
// 503 with a per-dependency breakdown.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()
	checks := gin.H{}
	ready := true

	// Database check.
	switch {
	case h.db == nil:
		checks["database"] = "unavailable"
		ready = false
	default:
		if err := database.PingPostgres(ctx, h.db); err != nil {
			checks["database"] = "down"
			ready = false
		} else {
			checks["database"] = "ok"
		}
	}

	// Redis check.
	switch {
	case h.redis == nil:
		checks["redis"] = "unavailable"
		ready = false
	default:
		if err := database.PingRedis(ctx, h.redis); err != nil {
			checks["redis"] = "down"
			ready = false
		} else {
			checks["redis"] = "ok"
		}
	}

	status := http.StatusOK
	overall := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		overall = "not_ready"
	}
	c.JSON(status, gin.H{"status": overall, "checks": checks})
}
