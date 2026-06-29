package designproblems

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the design-problems (HLD) browsing endpoints plus the
// authenticated per-user progress endpoints.
type Handler struct {
	svc    *Service
	tokens *auth.TokenManager
}

// NewHandler constructs a read-only design-problems Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// NewHandlerWithAuth constructs a Handler that also serves the authenticated
// progress endpoints (requires a shared token manager).
func NewHandlerWithAuth(svc *Service, tokens *auth.TokenManager) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// RegisterRoutes mounts the design-problems routes onto the /api/v1 group. The
// catalog (list/get) is public; progress endpoints require authentication.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/design-problems", h.List)
	v1.GET("/design-problems/:designProblemId", h.Get)

	if h.tokens != nil {
		authed := v1.Group("", auth.RequireAuth(h.tokens))
		authed.GET("/design-problems/:designProblemId/progress", h.GetProgress)
		authed.PUT("/design-problems/:designProblemId/progress", h.SaveProgress)
	}
}

// List handles GET /design-problems.
func (h *Handler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	f := Filter{
		TrackID:    queryUUID(c, "track_id"),
		Difficulty: difficultyParam(c.Query("difficulty")),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort"), designProblemSortable),
	}
	res, err := h.svc.List(c.Request.Context(), f, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]designProblemResponse, 0, len(res.Items))
	for _, d := range res.Items {
		data = append(data, toDesignProblemResponse(d))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// Get handles GET /design-problems/:designProblemId.
func (h *Handler) Get(c *gin.Context) {
	id, ok := h.pathUUID(c, "designProblemId")
	if !ok {
		return
	}
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDesignProblemDetailResponse(d))
}

// ---- progress DTOs + handlers ----

type progressResponse struct {
	DesignProblemID  string  `json:"design_problem_id"`
	Status           string  `json:"status"`
	Confidence       *int16  `json:"confidence"`
	Attempts         int     `json:"attempts"`
	TimeSpentMinutes int     `json:"time_spent_minutes"`
	Notes            *string `json:"notes"`
	FirstCompletedAt *string `json:"first_completed_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type saveProgressRequest struct {
	Status           string  `json:"status"`
	Confidence       *int16  `json:"confidence"`
	TimeSpentMinutes int     `json:"time_spent_minutes"`
	Notes            *string `json:"notes"`
}

func toProgressResponse(dpID uuid.UUID, p *Progress) progressResponse {
	if p == nil {
		// Not started yet — return a zero-value progress so the UI has a shape.
		return progressResponse{DesignProblemID: dpID.String(), Status: "not_started"}
	}
	var completed *string
	if p.FirstCompletedAt != nil {
		s := p.FirstCompletedAt.UTC().Format(time.RFC3339)
		completed = &s
	}
	return progressResponse{
		DesignProblemID:  p.DesignProblemID.String(),
		Status:           p.Status,
		Confidence:       p.Confidence,
		Attempts:         p.Attempts,
		TimeSpentMinutes: p.TimeSpentMinutes,
		Notes:            p.Notes,
		FirstCompletedAt: completed,
		UpdatedAt:        p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// GetProgress handles GET /design-problems/:id/progress.
func (h *Handler) GetProgress(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	id, ok := h.pathUUID(c, "designProblemId")
	if !ok {
		return
	}
	p, err := h.svc.GetProgress(c.Request.Context(), uid, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProgressResponse(id, p))
}

// SaveProgress handles PUT /design-problems/:id/progress.
func (h *Handler) SaveProgress(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	id, ok := h.pathUUID(c, "designProblemId")
	if !ok {
		return
	}
	var req saveProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	if req.Status == "" {
		req.Status = "completed"
	}
	p, err := h.svc.SaveProgress(c.Request.Context(), uid, id, ProgressInput{
		Status:           req.Status,
		Confidence:       req.Confidence,
		TimeSpentMinutes: req.TimeSpentMinutes,
		Notes:            req.Notes,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProgressResponse(id, p))
}

// ---- helpers ----

// pathUUID parses a required UUID path param, writing a 400 on a malformed id.
func (h *Handler) pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, name+" must be a valid uuid", nil)
		return uuid.Nil, false
	}
	return id, true
}

// writeError maps design-problems domain errors to the HTTP error envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	case errors.Is(err, ErrInvalidProgress):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "status must be valid and confidence 1–5", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

// queryUUID parses an optional UUID query param. Returns nil when absent or
// malformed so a bad value simply drops the filter rather than erroring.
func queryUUID(c *gin.Context, key string) *uuid.UUID {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
