package roadmap

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the roadmap/curriculum HTTP endpoints to the service. It reuses
// the auth module's RequireAuth middleware and authenticated-principal accessor.
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

// RegisterRoutes mounts the protected roadmap routes onto the /api/v1 group,
// matching the OpenAPI Curriculum paths.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	authed := v1.Group("")
	authed.Use(auth.RequireAuth(h.tokens))
	{
		authed.POST("/roadmaps/generate", h.Generate)
		authed.GET("/roadmaps/active", h.GetActive)
		authed.GET("/roadmaps/:roadmapId/weeks/:weekNumber", h.GetWeek)
		authed.GET("/plan-days/:date", h.GetPlanDay)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type generateRequest struct {
	Regenerate bool `json:"regenerate"`
	UseAI      bool `json:"use_ai"`
}

type roadmapResponse struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	TrackID         string             `json:"track_id"`
	ProfileID       string             `json:"profile_id"`
	TargetCompanyID *string            `json:"target_company_id"`
	StartDate       string             `json:"start_date"`
	EndDate         string             `json:"end_date"`
	TotalWeeks      int                `json:"total_weeks"`
	HoursPerWeek    int                `json:"hours_per_week"`
	Status          string             `json:"status"`
	IsActive        bool               `json:"is_active"`
	GeneratedBy     string             `json:"generated_by"`
	Weeks           []weekResponse     `json:"weeks"`
	CreatedAt       string             `json:"created_at"`
	UpdatedAt       string             `json:"updated_at"`
}

type weekResponse struct {
	ID           string        `json:"id"`
	RoadmapID    string        `json:"roadmap_id"`
	WeekNumber   int           `json:"week_number"`
	StartDate    string        `json:"start_date"`
	EndDate      string        `json:"end_date"`
	Theme        *string       `json:"theme"`
	FocusPillars []string      `json:"focus_pillars"`
	PlannedHours float64       `json:"planned_hours"`
	Days         []dayResponse `json:"days,omitempty"`
}

type dayResponse struct {
	ID               string         `json:"id"`
	RoadmapWeekID    string         `json:"roadmap_week_id"`
	UserID           string         `json:"user_id"`
	Date             string         `json:"date"`
	PlannedMinutes   int            `json:"planned_minutes"`
	CompletedMinutes int            `json:"completed_minutes"`
	IsRestDay        bool           `json:"is_rest_day"`
	Summary          *string        `json:"summary"`
	Tasks            []taskResponse `json:"tasks"`
}

type taskResponse struct {
	ID               string   `json:"id"`
	PlanDayID        string   `json:"plan_day_id"`
	Kind             string   `json:"kind"`
	ItemType         string   `json:"item_type"`
	ItemID           string   `json:"item_id"`
	PillarType       string   `json:"pillar_type"`
	Title            string   `json:"title"`
	Description      *string  `json:"description"`
	Objectives       []string `json:"objectives"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Priority         string   `json:"priority"`
	Difficulty       *string  `json:"difficulty"`
	Status           string   `json:"status"`
	SortOrder        int      `json:"sort_order"`
	Confidence       *int     `json:"confidence"`
	TimeSpentMinutes *int     `json:"time_spent_minutes"`
	CompletionNotes  *string  `json:"completion_notes"`
	RevisionItemID   *string  `json:"revision_item_id"`
	CompletedAt      *string  `json:"completed_at"`
}

// Generate handles POST /roadmaps/generate.
func (h *Handler) Generate(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	var req generateRequest
	// Body is optional; ignore bind errors on empty body, reject malformed JSON.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
			return
		}
	}

	rm, err := h.svc.GenerateRoadmap(c.Request.Context(), uid, req.Regenerate)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRoadmapResponse(rm, true))
}

// GetActive handles GET /roadmaps/active.
func (h *Handler) GetActive(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	rm, err := h.svc.GetActive(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toRoadmapResponse(rm, false))
}

// GetWeek handles GET /roadmaps/{roadmapId}/weeks/{weekNumber}.
func (h *Handler) GetWeek(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	roadmapID, err := uuid.Parse(c.Param("roadmapId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid roadmap id", nil)
		return
	}
	weekNumber, err := strconv.Atoi(c.Param("weekNumber"))
	if err != nil || weekNumber < 1 {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid week number", nil)
		return
	}
	week, err := h.svc.GetWeek(c.Request.Context(), uid, roadmapID, weekNumber)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWeekResponse(week, true))
}

// GetPlanDay handles GET /plan-days/{date}.
func (h *Handler) GetPlanDay(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "date must be YYYY-MM-DD", nil)
		return
	}
	day, err := h.svc.GetPlanDay(c.Request.Context(), uid, date)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDayResponse(day))
}

// writeServiceError maps domain errors to HTTP status + envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrActiveRoadmapExists):
		server.AbortError(c, http.StatusConflict, server.CodeConflict, "an active roadmap already exists; pass regenerate=true to replace it", nil)
	case errors.Is(err, ErrProfileRequired):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "complete your intake profile before generating a roadmap", nil)
	case errors.Is(err, ErrNoActiveRoadmap):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "no active roadmap", nil)
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

// ---- mappers ----

func toRoadmapResponse(rm *Roadmap, includeDays bool) roadmapResponse {
	out := roadmapResponse{
		ID:           rm.ID.String(),
		UserID:       rm.UserID.String(),
		TrackID:      rm.TrackID.String(),
		ProfileID:    rm.ProfileID.String(),
		StartDate:    rm.StartDate.UTC().Format("2006-01-02"),
		EndDate:      rm.EndDate.UTC().Format("2006-01-02"),
		TotalWeeks:   int(rm.TotalWeeks),
		HoursPerWeek: int(rm.HoursPerWeek),
		Status:       rm.Status,
		IsActive:     rm.IsActive,
		GeneratedBy:  rm.GeneratedBy,
		Weeks:        make([]weekResponse, 0, len(rm.Weeks)),
		CreatedAt:    rm.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    rm.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if rm.TargetCompanyID != nil {
		s := rm.TargetCompanyID.String()
		out.TargetCompanyID = &s
	}
	for i := range rm.Weeks {
		out.Weeks = append(out.Weeks, toWeekResponse(&rm.Weeks[i], includeDays))
	}
	return out
}

func toWeekResponse(w *RoadmapWeek, includeDays bool) weekResponse {
	out := weekResponse{
		ID:           w.ID.String(),
		RoadmapID:    w.RoadmapID.String(),
		WeekNumber:   int(w.WeekNumber),
		StartDate:    w.StartDate.UTC().Format("2006-01-02"),
		EndDate:      w.EndDate.UTC().Format("2006-01-02"),
		Theme:        w.Theme,
		FocusPillars: []string(w.FocusPillars),
		PlannedHours: w.PlannedHours,
	}
	if out.FocusPillars == nil {
		out.FocusPillars = []string{}
	}
	if includeDays {
		out.Days = make([]dayResponse, 0, len(w.Days))
		for i := range w.Days {
			out.Days = append(out.Days, toDayResponse(&w.Days[i]))
		}
	}
	return out
}

func toDayResponse(d *PlanDay) dayResponse {
	out := dayResponse{
		ID:               d.ID.String(),
		RoadmapWeekID:    d.RoadmapWeekID.String(),
		UserID:           d.UserID.String(),
		Date:             d.Date.UTC().Format("2006-01-02"),
		PlannedMinutes:   d.PlannedMinutes,
		CompletedMinutes: d.CompletedMinutes,
		IsRestDay:        d.IsRestDay,
		Summary:          d.Summary,
		Tasks:            make([]taskResponse, 0, len(d.Tasks)),
	}
	for i := range d.Tasks {
		out.Tasks = append(out.Tasks, toTaskResponse(&d.Tasks[i]))
	}
	return out
}

func toTaskResponse(t *PlanTask) taskResponse {
	out := taskResponse{
		ID:               t.ID.String(),
		PlanDayID:        t.PlanDayID.String(),
		Kind:             t.Kind,
		ItemType:         t.ItemType,
		ItemID:           t.ItemID.String(),
		PillarType:       t.PillarType,
		Title:            t.Title,
		Description:      t.Description,
		Objectives:       []string(t.Objectives),
		EstimatedMinutes: t.EstimatedMinutes,
		Priority:         t.Priority,
		Difficulty:       t.Difficulty,
		Status:           t.Status,
		SortOrder:        t.SortOrder,
		CompletionNotes:  t.CompletionNotes,
	}
	if out.Objectives == nil {
		out.Objectives = []string{}
	}
	if t.Confidence != nil {
		v := int(*t.Confidence)
		out.Confidence = &v
	}
	out.TimeSpentMinutes = t.TimeSpentMinutes
	if t.RevisionItemID != nil {
		s := t.RevisionItemID.String()
		out.RevisionItemID = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &s
	}
	return out
}
