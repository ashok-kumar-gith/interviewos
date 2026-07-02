package content

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// AdminHandler exposes the admin-gated content-authoring endpoints for DSA
// problems and topics. All routes require a valid admin bearer token.
type AdminHandler struct {
	svc    *AdminService
	tokens *auth.TokenManager
}

// NewAdminHandler constructs an AdminHandler. Written entities are returned via
// the same detail DTOs (toProblemDetailResponse/toTopicDetailResponse) as GET.
func NewAdminHandler(svc *AdminService, tokens *auth.TokenManager) *AdminHandler {
	return &AdminHandler{svc: svc, tokens: tokens}
}

// RegisterRoutes mounts the admin content routes onto the /api/v1 group, all
// gated by RequireAdmin (401 unauth, 403 non-admin).
func (h *AdminHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	if h.tokens == nil {
		return
	}
	admin := v1.Group("", auth.RequireAdmin(h.tokens))
	admin.POST("/problems", h.CreateProblem)
	admin.PUT("/problems/:problemId", h.UpdateProblem)
	admin.DELETE("/problems/:problemId", h.DeleteProblem)

	admin.POST("/topics", h.CreateTopic)
	admin.PUT("/topics/:topicId", h.UpdateTopic)
	admin.DELETE("/topics/:topicId", h.DeleteTopic)
}

// ---- request DTOs ----

type companyFrequencyRequest struct {
	CompanyID      *string `json:"company_id"`
	CompanySlug    string  `json:"company_slug"`
	Frequency      float64 `json:"frequency"`
	LastSeenPeriod *string `json:"last_seen_period"`
}

type problemRequest struct {
	TrackID          *string                   `json:"track_id"`
	TopicID          *string                   `json:"topic_id"`
	Slug             string                    `json:"slug"`
	Title            string                    `json:"title"`
	Difficulty       string                    `json:"difficulty"`
	Platform         string                    `json:"platform"`
	ExternalID       *string                   `json:"external_id"`
	URL              *string                   `json:"url"`
	PromptSummary    *string                   `json:"prompt_summary"`
	ApproachMD       *string                   `json:"approach_md"`
	CommonMistakes   *string                   `json:"common_mistakes"`
	EstimatedMinutes *int                      `json:"estimated_minutes"`
	FrequencyScore   *float64                  `json:"frequency_score"`
	IsPremium        *bool                     `json:"is_premium"`
	PatternSlugs     []string                  `json:"pattern_slugs"`
	Sources          []string                  `json:"sources"`
	CompanyFrequency []companyFrequencyRequest `json:"company_frequency"`
}

func (r problemRequest) toInput() (ProblemInput, bool) {
	in := ProblemInput{
		Slug:             r.Slug,
		Title:            r.Title,
		Difficulty:       r.Difficulty,
		Platform:         r.Platform,
		ExternalID:       r.ExternalID,
		URL:              r.URL,
		PromptSummary:    r.PromptSummary,
		ApproachMD:       r.ApproachMD,
		CommonMistakes:   r.CommonMistakes,
		EstimatedMinutes: r.EstimatedMinutes,
		FrequencyScore:   r.FrequencyScore,
		IsPremium:        r.IsPremium,
		PatternSlugs:     r.PatternSlugs,
		Sources:          r.Sources,
	}
	if r.TrackID != nil {
		id, err := uuid.Parse(*r.TrackID)
		if err != nil {
			return ProblemInput{}, false
		}
		in.TrackID = &id
	}
	if r.TopicID != nil {
		id, err := uuid.Parse(*r.TopicID)
		if err != nil {
			return ProblemInput{}, false
		}
		in.TopicID = &id
	}
	freqs := make([]CompanyFrequencyInput, 0, len(r.CompanyFrequency))
	for _, cf := range r.CompanyFrequency {
		item := CompanyFrequencyInput{
			CompanySlug:    cf.CompanySlug,
			Frequency:      cf.Frequency,
			LastSeenPeriod: cf.LastSeenPeriod,
		}
		if cf.CompanyID != nil {
			id, err := uuid.Parse(*cf.CompanyID)
			if err != nil {
				return ProblemInput{}, false
			}
			item.CompanyID = &id
		}
		freqs = append(freqs, item)
	}
	in.CompanyFrequency = freqs
	return in, true
}

type topicRequest struct {
	PillarID          *string  `json:"pillar_id"`
	PillarType        string   `json:"pillar_type"`
	TrackID           *string  `json:"track_id"`
	Slug              string   `json:"slug"`
	Name              string   `json:"name"`
	Summary           *string  `json:"summary"`
	ConceptMD         *string  `json:"concept_md"`
	Difficulty        string   `json:"difficulty"`
	Priority          string   `json:"priority"`
	EstimatedHours    *float64 `json:"estimated_hours"`
	CommonMistakes    *string  `json:"common_mistakes"`
	ExpectedQuestions []string `json:"expected_questions"`
	Prerequisites     []string `json:"prerequisites"`
	SortOrder         *int     `json:"sort_order"`
}

func (r topicRequest) toInput() (TopicInput, bool) {
	in := TopicInput{
		Slug:              r.Slug,
		Name:              r.Name,
		Summary:           r.Summary,
		ConceptMD:         r.ConceptMD,
		Difficulty:        r.Difficulty,
		Priority:          r.Priority,
		EstimatedHours:    r.EstimatedHours,
		CommonMistakes:    r.CommonMistakes,
		ExpectedQuestions: r.ExpectedQuestions,
		Prerequisites:     r.Prerequisites,
		SortOrder:         r.SortOrder,
	}
	if r.PillarID != nil {
		id, err := uuid.Parse(*r.PillarID)
		if err != nil {
			return TopicInput{}, false
		}
		in.PillarID = &id
	}
	if r.PillarType != "" {
		pt := pillarTypeParam(r.PillarType)
		if pt == nil {
			return TopicInput{}, false
		}
		in.PillarType = pt
	}
	if r.TrackID != nil {
		id, err := uuid.Parse(*r.TrackID)
		if err != nil {
			return TopicInput{}, false
		}
		in.TrackID = &id
	}
	return in, true
}

// ---- handlers ----

// CreateProblem handles POST /problems (admin).
func (h *AdminHandler) CreateProblem(c *gin.Context) {
	var req problemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, ok := req.toInput()
	if !ok {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid uuid in request body", nil)
		return
	}
	b, err := h.svc.CreateProblem(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProblemDetailResponse(b))
}

// UpdateProblem handles PUT /problems/:problemId (admin).
func (h *AdminHandler) UpdateProblem(c *gin.Context) {
	id, ok := h.pathUUID(c, "problemId")
	if !ok {
		return
	}
	var req problemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, valid := req.toInput()
	if !valid {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid uuid in request body", nil)
		return
	}
	b, err := h.svc.UpdateProblem(c.Request.Context(), id, in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProblemDetailResponse(b))
}

// DeleteProblem handles DELETE /problems/:problemId (admin).
func (h *AdminHandler) DeleteProblem(c *gin.Context) {
	id, ok := h.pathUUID(c, "problemId")
	if !ok {
		return
	}
	if err := h.svc.DeleteProblem(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateTopic handles POST /topics (admin).
func (h *AdminHandler) CreateTopic(c *gin.Context) {
	var req topicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, ok := req.toInput()
	if !ok {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid pillar_type or uuid in request body", nil)
		return
	}
	b, err := h.svc.CreateTopic(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTopicDetailResponse(b))
}

// UpdateTopic handles PUT /topics/:topicId (admin).
func (h *AdminHandler) UpdateTopic(c *gin.Context) {
	id, ok := h.pathUUID(c, "topicId")
	if !ok {
		return
	}
	var req topicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, valid := req.toInput()
	if !valid {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid pillar_type or uuid in request body", nil)
		return
	}
	b, err := h.svc.UpdateTopic(c.Request.Context(), id, in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTopicDetailResponse(b))
}

// DeleteTopic handles DELETE /topics/:topicId (admin).
func (h *AdminHandler) DeleteTopic(c *gin.Context) {
	id, ok := h.pathUUID(c, "topicId")
	if !ok {
		return
	}
	if err := h.svc.DeleteTopic(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- helpers ----

func (h *AdminHandler) pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, name+" must be a valid uuid", nil)
		return uuid.Nil, false
	}
	return id, true
}

// writeError maps content admin domain errors to the HTTP error envelope.
func (h *AdminHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	case errors.Is(err, ErrConflict):
		server.AbortError(c, http.StatusConflict, server.CodeConflict, "slug already exists", nil)
	case errors.Is(err, ErrUnknownReference):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "a referenced pattern, company, or pillar does not exist", nil)
	case errors.Is(err, ErrValidation):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "invalid or missing required fields", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}
