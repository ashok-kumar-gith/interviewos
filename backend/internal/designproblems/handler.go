package designproblems

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the design-problems (HLD) browsing endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs a design-problems Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the design-problems browsing routes onto the /api/v1
// group. These endpoints are public (read-only catalog), mirroring content.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/design-problems", h.List)
	v1.GET("/design-problems/:designProblemId", h.Get)
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
