package resume

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the resume HTTP endpoints to the service. All routes are
// protected by RequireAuth.
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

// RegisterRoutes mounts the resume routes onto the /api/v1 group, all behind
// RequireAuth.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	r := v1.Group("/resume")
	r.Use(auth.RequireAuth(h.tokens))
	{
		r.GET("/profile", h.GetProfile)
		r.PUT("/profile", h.UpsertProfile)
		r.GET("/projects", h.ListProjects)
		r.POST("/projects", h.CreateProject)
		r.PUT("/projects/:projectId", h.UpdateProject)
		r.DELETE("/projects/:projectId", h.DeleteProject)
		r.POST("/score", h.Score)
	}
}

// ---- Request DTOs (per openapi.yaml) ----

type profileUpsertRequest struct {
	Headline        *string  `json:"headline"`
	Summary         *string  `json:"summary"`
	YearsExperience *float64 `json:"years_experience"`
	Skills          []string `json:"skills"`
	TargetKeywords  []string `json:"target_keywords"`
}

type projectUpsertRequest struct {
	Name        string   `json:"name"`
	Role        *string  `json:"role"`
	Description *string  `json:"description"`
	Impact      *string  `json:"impact"`
	Metrics     []string `json:"metrics"`
	TechStack   []string `json:"tech_stack"`
	StartDate   *string  `json:"start_date"`
	EndDate     *string  `json:"end_date"`
	SortOrder   int      `json:"sort_order"`
}

// ---- Response DTOs (per openapi.yaml) ----

type profileResponse struct {
	ID              string            `json:"id"`
	UserID          string            `json:"user_id"`
	Headline        *string           `json:"headline"`
	Summary         *string           `json:"summary"`
	YearsExperience *float64          `json:"years_experience"`
	Skills          []string          `json:"skills"`
	TargetKeywords  []string          `json:"target_keywords"`
	ATSScore        *float64          `json:"ats_score"`
	LastScoredAt    *string           `json:"last_scored_at"`
	Projects        []projectResponse `json:"projects"`
}

type projectResponse struct {
	ID              string   `json:"id"`
	ResumeProfileID string   `json:"resume_profile_id"`
	Name            string   `json:"name"`
	Role            *string  `json:"role"`
	Description     *string  `json:"description"`
	Impact          *string  `json:"impact"`
	Metrics         []string `json:"metrics"`
	TechStack       []string `json:"tech_stack"`
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	SortOrder       int      `json:"sort_order"`
}

type scoreResponse struct {
	ATSScore        float64          `json:"ats_score"`
	KeywordMatches  []string         `json:"keyword_matches"`
	MissingKeywords []string         `json:"missing_keywords"`
	Suggestions     []string         `json:"suggestions"`
	Breakdown       []ScoreBreakdown `json:"breakdown"`
	UsedFallback    bool             `json:"used_fallback"`
}

// ---- Handlers ----

// GetProfile handles GET /resume/profile.
func (h *Handler) GetProfile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetProfile(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// UpsertProfile handles PUT /resume/profile.
func (h *Handler) UpsertProfile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req profileUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.UpsertProfile(c.Request.Context(), uid, ProfileInput{
		Headline:        req.Headline,
		Summary:         req.Summary,
		YearsExperience: req.YearsExperience,
		Skills:          req.Skills,
		TargetKeywords:  req.TargetKeywords,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// ListProjects handles GET /resume/projects.
func (h *Handler) ListProjects(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	projects, err := h.svc.ListProjects(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	out := make([]projectResponse, 0, len(projects))
	for i := range projects {
		out = append(out, toProjectResponse(&projects[i]))
	}
	c.JSON(http.StatusOK, out)
}

// CreateProject handles POST /resume/projects.
func (h *Handler) CreateProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req projectUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.CreateProject(c.Request.Context(), uid, toProjectInput(req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProjectResponse(p))
}

// UpdateProject handles PUT /resume/projects/:projectId.
func (h *Handler) UpdateProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	pid, ok := h.projectID(c)
	if !ok {
		return
	}
	var req projectUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.UpdateProject(c.Request.Context(), uid, pid, toProjectInput(req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(p))
}

// DeleteProject handles DELETE /resume/projects/:projectId.
func (h *Handler) DeleteProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	pid, ok := h.projectID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProject(c.Request.Context(), uid, pid); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Score handles POST /resume/score.
func (h *Handler) Score(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	result, err := h.svc.Score(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, scoreResponse{
		ATSScore:        result.ATSScore,
		KeywordMatches:  nonNil(result.KeywordMatches),
		MissingKeywords: nonNil(result.MissingKeywords),
		Suggestions:     nonNil(result.Suggestions),
		Breakdown:       result.Breakdown,
		UsedFallback:    result.UsedFallback,
	})
}

// ---- helpers ----

func (h *Handler) userID(c *gin.Context) (uuid.UUID, bool) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return uuid.Nil, false
	}
	return uid, true
}

func (h *Handler) projectID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid project id", nil)
		return uuid.Nil, false
	}
	return id, true
}

// bindJSON binds the request body, writing a 400 envelope on malformed JSON.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return false
	}
	return true
}

// writeServiceError maps domain errors to HTTP status + error envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resume profile not found", nil)
	case errors.Is(err, ErrProjectNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resume project not found", nil)
	case errors.Is(err, ErrForbidden):
		server.AbortError(c, http.StatusForbidden, server.CodeForbidden, "not permitted", nil)
	case errors.Is(err, ErrValidation):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, err.Error(), nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toProjectInput(req projectUpsertRequest) ProjectInput {
	return ProjectInput{
		Name:        req.Name,
		Role:        req.Role,
		Description: req.Description,
		Impact:      req.Impact,
		Metrics:     req.Metrics,
		TechStack:   req.TechStack,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		SortOrder:   req.SortOrder,
	}
}

func toProfileResponse(p *Profile) profileResponse {
	r := profileResponse{
		ID:              p.ID.String(),
		UserID:          p.UserID.String(),
		Headline:        p.Headline,
		Summary:         p.Summary,
		YearsExperience: p.YearsExperience,
		Skills:          nonNil([]string(p.Skills)),
		TargetKeywords:  nonNil([]string(p.TargetKeywords)),
		ATSScore:        p.ATSScore,
		Projects:        make([]projectResponse, 0, len(p.Projects)),
	}
	if p.LastScoredAt != nil {
		s := p.LastScoredAt.UTC().Format(time.RFC3339)
		r.LastScoredAt = &s
	}
	for i := range p.Projects {
		r.Projects = append(r.Projects, toProjectResponse(&p.Projects[i]))
	}
	return r
}

func toProjectResponse(p *Project) projectResponse {
	r := projectResponse{
		ID:              p.ID.String(),
		ResumeProfileID: p.ResumeProfileID.String(),
		Name:            p.Name,
		Role:            p.Role,
		Description:     p.Description,
		Impact:          p.Impact,
		Metrics:         nonNil([]string(p.Metrics)),
		TechStack:       nonNil([]string(p.TechStack)),
		SortOrder:       p.SortOrder,
	}
	if p.StartDate != nil {
		s := p.StartDate.UTC().Format("2006-01-02")
		r.StartDate = &s
	}
	if p.EndDate != nil {
		s := p.EndDate.UTC().Format("2006-01-02")
		r.EndDate = &s
	}
	return r
}

// nonNil returns an empty slice instead of nil so JSON renders "[]" not "null".
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
