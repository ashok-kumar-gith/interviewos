// Package server assembles the Gin HTTP engine: it wires the middleware chain
// (request-id, security headers, metrics, structured logging, recovery, CORS)
// and registers the route groups (health/readiness probes, Prometheus /metrics,
// the Swagger UI, and the versioned /api/v1 API group). Composition of concrete
// dependencies happens in cmd/api.
package server

import (
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
	// Registrars are feature modules that mount their routes onto /api/v1.
	Registrars []RouteRegistrar
}

// RouteRegistrar attaches a feature module's routes to the versioned /api/v1
// group. Feature modules (auth, profile, …) implement this and are passed to
// the server at composition time so wiring stays dependency-injected.
type RouteRegistrar interface {
	RegisterRoutes(v1 *gin.RouterGroup)
}

// New builds and returns a fully-wired *gin.Engine with the middleware chain
// applied and all routes registered.
func New(opts Options) *gin.Engine {
	if opts.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// gin.New() (not Default) so we install our own logger/recovery middleware.
	engine := gin.New()

	// Middleware chain order (outermost first): request-id establishes the
	// correlation ID; security headers are set early so they apply even to error
	// responses; metrics observes the full handler latency (via c.FullPath());
	// logging emits the structured access line; recovery converts panics to a
	// clean 500; CORS applies the origin allowlist last before routing.
	var metrics *Metrics
	engine.Use(RequestID())
	engine.Use(SecurityHeaders(opts.Config.IsProduction()))
	if opts.Config.MetricsEnabled {
		metrics = NewMetrics()
		engine.Use(metrics.Middleware())
	}
	engine.Use(
		Logger(opts.Logger),
		Recovery(opts.Logger),
		CORS(opts.Config.CORSOrigins),
	)

	RegisterRoutes(engine, opts, metrics)
	return engine
}

// RegisterRoutes mounts the liveness/readiness probes, the Prometheus /metrics
// endpoint, the Swagger UI, and the versioned /api/v1 group onto the engine.
// /healthz, /readyz, /metrics and the docs are unauthenticated and cheap.
func RegisterRoutes(engine *gin.Engine, opts Options, metrics *Metrics) {
	health := NewHealthHandler(opts.DB, opts.Redis)

	// Liveness and readiness probes (unversioned, used by orchestrators).
	engine.GET("/healthz", health.Healthz)
	engine.GET("/readyz", health.Readyz)

	// Prometheus exposition — mounted outside any auth group, no auth.
	if metrics != nil {
		engine.GET("/metrics", metrics.Handler())
	}

	// Interactive API docs backed by the embedded OpenAPI spec.
	RegisterSwagger(engine)

	// Versioned API group. Feature modules mount their routes here.
	v1 := engine.Group("/api/v1")
	for _, r := range opts.Registrars {
		if r != nil {
			r.RegisterRoutes(v1)
		}
	}
}
