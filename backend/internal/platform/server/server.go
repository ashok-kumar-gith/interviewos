// Package server assembles the Gin HTTP engine: it wires the middleware chain
// (request-id, structured logging, recovery, CORS) and registers the route
// groups (health/readiness probes, the versioned /api/v1 API group, and the
// Swagger placeholder). Composition of concrete dependencies happens in cmd/api.
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/interviewos/backend/internal/platform/config"
)

// Options carries the dependencies needed to build the router. DB and Redis may
// be nil when those backing services are unavailable (graceful degradation):
// liveness still serves, readiness reports them down.
type Options struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *gorm.DB
	Redis  *redis.Client
}

// New builds and returns a fully-wired *gin.Engine with the middleware chain
// applied and all routes registered.
func New(opts Options) *gin.Engine {
	if opts.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// gin.New() (not Default) so we install our own logger/recovery middleware.
	engine := gin.New()
	engine.Use(
		RequestID(),
		Logger(opts.Logger),
		Recovery(opts.Logger),
		CORS(opts.Config.CORSOrigins),
	)

	RegisterRoutes(engine, opts)
	return engine
}

// RegisterRoutes mounts the liveness/readiness probes, the Swagger placeholder,
// and the versioned /api/v1 group onto the engine. Feature modules will attach
// their routes to the v1 group at composition time in later milestones.
func RegisterRoutes(engine *gin.Engine, opts Options) {
	health := NewHealthHandler(opts.DB, opts.Redis)

	// Liveness and readiness probes (unversioned, used by orchestrators).
	engine.GET("/healthz", health.Healthz)
	engine.GET("/readyz", health.Readyz)

	// Swagger placeholder. The OpenAPI source of truth lives at api/openapi.yaml;
	// interactive docs will be served here in a later milestone.
	engine.GET("/swagger", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Swagger UI not yet wired; see backend/api/openapi.yaml",
		})
	})

	// Versioned API group. Domain module routes mount here later.
	v1 := engine.Group("/api/v1")
	_ = v1 // reserved for feature-module route registration (M1+).
}
