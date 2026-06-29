package intake

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

// Handler wires the intake/profile HTTP endpoints to the service. It reuses the
// auth module's RequireAuth middleware and authenticated-principal accessor.
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

// RegisterRoutes mounts the protected /profile routes onto the /api/v1 group.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	p := v1.Group("/profile")
	p.Use(auth.RequireAuth(h.tokens))
	{
		p.GET("", h.Get)
		p.PUT("", h.Upsert)
	}
}

// ---- DTOs ----

// upsertRequest mirrors the UserProfileUpsert OpenAPI schema. Optional fields
// are pointers so an omitted value is distinguishable from a zero value.
type upsertRequest struct {
	TrackID         string          `json:"track_id"`
	YearsExperience float64         `json:"years_experience"`
	TargetCompanyID *string         `json:"target_company_id"`
	TargetRole      string          `json:"target_role"`
	TargetLevel     *string         `json:"target_level"`
	HoursPerWeek    int             `json:"hours_per_week"`
	StartDate       string          `json:"start_date"`
	TargetWeeks     *int            `json:"target_weeks"`
	PillarStrengths map[string]int  `json:"pillar_strengths"`
	Timezone        *string         `json:"timezone"`
	IntakeAnswers   json.RawMessage `json:"intake_answers"`
}

// profileResponse mirrors the UserProfile OpenAPI schema.
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

// Get handles GET /profile.
func (h *Handler) Get(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	p, err := h.svc.Get(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// Upsert handles PUT /profile.
func (h *Handler) Upsert(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}

	var req upsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}

	in, parseErrs := req.toInput()
	if len(parseErrs) > 0 {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed", parseErrs)
		return
	}

	p, err := h.svc.Upsert(c.Request.Context(), uid, in)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// toInput parses/coerces the wire request into a domain UpsertInput, collecting
// any field-level parse errors (malformed UUIDs/dates) as validation details.
func (r *upsertRequest) toInput() (UpsertInput, []server.FieldError) {
	var errs []server.FieldError
	in := UpsertInput{
		YearsExperience: r.YearsExperience,
		TargetRole:      r.TargetRole,
		TargetLevel:     r.TargetLevel,
		HoursPerWeek:    r.HoursPerWeek,
		TargetWeeks:     r.TargetWeeks,
		PillarStrengths: r.PillarStrengths,
		Timezone:        r.Timezone,
		IntakeAnswers:   r.IntakeAnswers,
	}

	if r.TrackID == "" {
		errs = append(errs, server.FieldError{Field: "track_id", Message: "is required"})
	} else if tid, err := uuid.Parse(r.TrackID); err != nil {
		errs = append(errs, server.FieldError{Field: "track_id", Message: "must be a valid uuid"})
	} else {
		in.TrackID = tid
	}

	if r.TargetCompanyID != nil && *r.TargetCompanyID != "" {
		if cid, err := uuid.Parse(*r.TargetCompanyID); err != nil {
			errs = append(errs, server.FieldError{Field: "target_company_id", Message: "must be a valid uuid"})
		} else {
			in.TargetCompanyID = &cid
		}
	}

	if r.StartDate == "" {
		errs = append(errs, server.FieldError{Field: "start_date", Message: "is required"})
	} else if d, err := time.Parse("2006-01-02", r.StartDate); err != nil {
		errs = append(errs, server.FieldError{Field: "start_date", Message: "must be a date (YYYY-MM-DD)"})
	} else {
		in.StartDate = d
	}

	return in, errs
}

// writeServiceError maps domain errors to HTTP status + error envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		details := make([]server.FieldError, 0, len(ve.Violations))
		for _, fv := range ve.Violations {
			details = append(details, server.FieldError{Field: fv.Field, Message: fv.Message})
		}
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed", details)
	case errors.Is(err, ErrProfileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "profile not found", nil)
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
	if len(p.PillarStrengths) > 0 {
		_ = json.Unmarshal(p.PillarStrengths, &r.PillarStrengths)
	}
	if p.OnboardingCompletedAt != nil {
		s := p.OnboardingCompletedAt.UTC().Format(time.RFC3339)
		r.OnboardingCompletedAt = &s
	}
	return r
}
