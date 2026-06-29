package lld

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the public LLD catalog browsing endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs an LLD Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the LLD browsing routes onto the /api/v1 group. Browsing
// is public (no auth), mirroring the content catalog.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/lld-problems", h.ListProblems)
	v1.GET("/lld-problems/:lldProblemId", h.GetProblem)
}

// ListProblems handles GET /lld-problems.
func (h *Handler) ListProblems(c *gin.Context) {
	page, pageSize := parsePagination(c)
	f := ProblemFilter{
		Difficulty: difficultyParam(c.Query("difficulty")),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort"), problemSortable),
	}
	res, err := h.svc.ListProblems(c.Request.Context(), f, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]problemResponse, 0, len(res.Items))
	for _, p := range res.Items {
		data = append(data, toProblemResponse(p))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// GetProblem handles GET /lld-problems/:lldProblemId. The path param is a UUID
// per the contract; for convenience a slug is also accepted (looked up when the
// value is not a valid UUID).
func (h *Handler) GetProblem(c *gin.Context) {
	raw := c.Param("lldProblemId")
	var (
		prob *Problem
		err  error
	)
	if id, perr := uuid.Parse(raw); perr == nil {
		prob, err = h.svc.GetProblem(c.Request.Context(), id)
	} else {
		prob, err = h.svc.GetProblemBySlug(c.Request.Context(), raw)
	}
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProblemDetailResponse(prob))
}

// writeError maps LLD domain errors to the HTTP error envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}
