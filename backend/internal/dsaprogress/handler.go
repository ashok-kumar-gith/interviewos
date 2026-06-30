package dsaprogress

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the authenticated DSA problem progress + solution endpoints.
type Handler struct {
	svc    *Service
	tokens *auth.TokenManager
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service, tokens *auth.TokenManager) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// RegisterRoutes mounts the routes onto /api/v1. All require authentication.
//
//	GET    /problems/solved                  the user's solved/attempted log
//	GET    /problems/{problemId}/progress    progress + stored solution
//	PUT    /problems/{problemId}/progress    record solve state + solution
//	DELETE /problems/{problemId}/progress    clear it
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("", auth.RequireAuth(h.tokens))
	g.GET("/problems/solved", h.List)
	g.GET("/problems/:problemId/progress", h.Get)
	g.PUT("/problems/:problemId/progress", h.Save)
	g.DELETE("/problems/:problemId/progress", h.Delete)
}

// ---- DTOs ----

type progressResponse struct {
	ProblemID         string  `json:"problem_id"`
	Status            string  `json:"status"`
	Solved            bool    `json:"solved"`
	Confidence        *int16  `json:"confidence"`
	Attempts          int     `json:"attempts"`
	TimeSpentMinutes  int     `json:"time_spent_minutes"`
	SolvedAt          *string `json:"solved_at"`
	SolutionCode      *string `json:"solution_code"`
	SolutionLanguage  *string `json:"solution_language"`
	SolutionNotes     *string `json:"solution_notes"`
	SolutionUpdatedAt *string `json:"solution_updated_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type saveRequest struct {
	Solved           bool    `json:"solved"`
	Confidence       *int16  `json:"confidence"`
	TimeSpentMinutes int     `json:"time_spent_minutes"`
	SolutionCode     *string `json:"solution_code"`
	SolutionLanguage *string `json:"solution_language"`
	SolutionNotes    *string `json:"solution_notes"`
}

func tsPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func toResponse(problemID uuid.UUID, p *Progress) progressResponse {
	if p == nil {
		return progressResponse{ProblemID: problemID.String(), Status: "not_started"}
	}
	return progressResponse{
		ProblemID:         p.ProblemID.String(),
		Status:            p.Status,
		Solved:            p.Solved,
		Confidence:        p.Confidence,
		Attempts:          p.Attempts,
		TimeSpentMinutes:  p.TimeSpentMinutes,
		SolvedAt:          tsPtr(p.SolvedAt),
		SolutionCode:      p.SolutionCode,
		SolutionLanguage:  p.SolutionLanguage,
		SolutionNotes:     p.SolutionNotes,
		SolutionUpdatedAt: tsPtr(p.SolutionUpdatedAt),
		UpdatedAt:         p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ---- handlers ----

func (h *Handler) List(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	rows, err := h.svc.List(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]progressResponse, 0, len(rows))
	for i := range rows {
		data = append(data, toResponse(rows[i].ProblemID, &rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) Get(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(c.Request.Context(), uid, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(id, p))
}

func (h *Handler) Save(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req saveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	p, err := h.svc.Save(c.Request.Context(), uid, id, Input{
		Solved:           req.Solved,
		Confidence:       req.Confidence,
		TimeSpentMinutes: req.TimeSpentMinutes,
		SolutionCode:     req.SolutionCode,
		SolutionLanguage: req.SolutionLanguage,
		SolutionNotes:    req.SolutionNotes,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(id, p))
}

func (h *Handler) Delete(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uid, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- helpers ----

func (h *Handler) pathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("problemId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "problemId must be a valid uuid", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) unauth(c *gin.Context) {
	server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
}

func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProblemNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "problem not found", nil)
	case errors.Is(err, ErrValidation):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "confidence must be 1–5 and fields within size limits", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}
