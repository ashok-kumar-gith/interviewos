package company

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the Company Mode HTTP endpoints (GET/PUT /company/target) to the
// service. It reuses the auth module's RequireAuth middleware.
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

// RegisterRoutes mounts the protected /company/target routes onto /api/v1.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/company/target")
	g.Use(auth.RequireAuth(h.tokens))
	{
		g.GET("", h.GetTarget)
		g.PUT("", h.SetTarget)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type setTargetRequest struct {
	CompanyID       string `json:"company_id"`
	ReweightRoadmap *bool  `json:"reweight_roadmap"`
}

// profileResponse mirrors the UserProfile OpenAPI schema (the PUT/GET responses).
type profileResponse struct {
	ID                    string         `json:"id"`
	UserID                string         `json:"user_id"`
	TrackID               string         `json:"track_id"`
	YearsExperience       float64        `json:"years_experience"`
	TargetCompanyID       *string        `json:"target_company_id"`
	TargetRole            string         `json:"target_role"`
	TargetLevel           *string        `json:"target_level"`
	HoursPerWeek          int            `json:"hours_per_week"`
	StartDate             string         `json:"start_date"`
	TargetWeeks           int            `json:"target_weeks"`
	PillarStrengths       map[string]int `json:"pillar_strengths"`
	Timezone              string         `json:"timezone"`
	OnboardingCompletedAt *string        `json:"onboarding_completed_at"`
	CreatedAt             string         `json:"created_at"`
	UpdatedAt             string         `json:"updated_at"`
}

// GetTarget handles GET /company/target.
func (h *Handler) GetTarget(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	p, err := h.svc.GetTarget(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// SetTarget handles PUT /company/target.
func (h *Handler) SetTarget(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	var req setTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}
	companyID, err := uuid.Parse(req.CompanyID)
	if err != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "company_id must be a valid uuid", nil)
		return
	}
	p, err := h.svc.SetTarget(c.Request.Context(), uid, companyID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// writeError maps domain errors to HTTP status + envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "profile not found", nil)
	case errors.Is(err, ErrCompanyNotFound):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "target company does not exist", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toProfileResponse(p *Profile) profileResponse {
	r := profileResponse{
		ID:              p.ID.String(),
		UserID:          p.UserID.String(),
		TrackID:         p.TrackID.String(),
		YearsExperience: p.YearsExperience,
		TargetRole:      p.TargetRole,
		TargetLevel:     p.TargetLevel,
		HoursPerWeek:    int(p.HoursPerWeek),
		StartDate:       p.StartDate.UTC().Format("2006-01-02"),
		TargetWeeks:     int(p.TargetWeeks),
		PillarStrengths: map[string]int{},
		Timezone:        p.Timezone,
		CreatedAt:       p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.TargetCompanyID != nil {
		s := p.TargetCompanyID.String()
		r.TargetCompanyID = &s
	}
	if p.OnboardingCompletedAt != nil {
		s := p.OnboardingCompletedAt.UTC().Format(time.RFC3339)
		r.OnboardingCompletedAt = &s
	}
	if len(p.PillarStrengths) > 0 {
		_ = json.Unmarshal(p.PillarStrengths, &r.PillarStrengths)
	}
	return r
}
