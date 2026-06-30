package coderun

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the code-run HTTP endpoint to the service. The route is
// protected by the auth.RequireAuth middleware (mirrors revision/handler.go).
type Handler struct {
	svc    *Service
	tokens *auth.TokenManager
}

// HandlerConfig configures a Handler.
type HandlerConfig struct {
	Service *Service
	Tokens  *auth.TokenManager
}

// NewHandler constructs a Handler.
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{svc: cfg.Service, tokens: cfg.Tokens}
}

// RegisterRoutes mounts the code-run route onto the /api/v1 group, matching the
// OpenAPI Code paths. The route requires authentication.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/code", auth.RequireAuth(h.tokens))
	{
		g.POST("/run", h.Run)
	}
}

// Run handles POST /code/run. It executes a code snippet via the executor and
// returns the normalized stdout/stderr/exit envelope. Execution failures
// (timeout, no network, upstream error) are surfaced as a clear error envelope
// — never a 500 panic.
func (h *Handler) Run(c *gin.Context) {
	if _, ok := auth.UserIDFromContext(c); !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}

	var req runRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}

	out, err := h.svc.Run(c.Request.Context(), req.Language, req.Source, req.Stdin)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// writeServiceError maps service errors to the standard error envelope. Upstream
// failures are reported as a clear 502/504 (resilient: no panic, actionable
// message) rather than a generic 500.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedLanguage):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError,
			"unsupported language; allowed: "+strings.Join(sortedSupportedLanguages(), ", "), nil)
	case errors.Is(err, ErrEmptySource):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "source must not be empty", nil)
	case errors.Is(err, ErrSourceTooLarge):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "source exceeds 64KB limit", nil)
	case errors.Is(err, ErrStdinTooLarge):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "stdin exceeds 16KB limit", nil)
	case errors.Is(err, ErrRuntimeUnavailable):
		server.AbortError(c, http.StatusBadGateway, server.CodeInternal, "no runtime available for the requested language", nil)
	case errors.Is(err, context.DeadlineExceeded):
		server.AbortError(c, http.StatusGatewayTimeout, server.CodeInternal, "code execution timed out", nil)
	case errors.Is(err, ErrExecutorUnavailable):
		// Includes wrapped context deadline / network failures. Resilient: clear,
		// non-panicking upstream error.
		if errors.Is(err, context.DeadlineExceeded) {
			server.AbortError(c, http.StatusGatewayTimeout, server.CodeInternal, "code execution timed out", nil)
			return
		}
		server.AbortError(c, http.StatusBadGateway, server.CodeInternal, "code executor is currently unavailable; please retry", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}
