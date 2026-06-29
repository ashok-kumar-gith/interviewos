package progress

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the progress / Today / dashboard HTTP endpoints to the service.
// It reuses the auth module's RequireAuth middleware and authenticated-principal
// accessor.
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

// RegisterRoutes mounts the protected progress routes onto the /api/v1 group,
// matching the OpenAPI Curriculum (today/tasks) and Analytics (dashboard) paths.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	authed := v1.Group("")
	authed.Use(auth.RequireAuth(h.tokens))
	{
		authed.GET("/today", h.GetToday)
		authed.POST("/tasks/:taskId/complete", h.CompleteTask)
		authed.POST("/tasks/:taskId/start", h.StartTask)
		authed.POST("/tasks/:taskId/reopen", h.ReopenTask)
		authed.POST("/tasks/:taskId/skip", h.SkipTask)
		authed.POST("/tasks/:taskId/reschedule", h.RescheduleTask)
		authed.GET("/dashboard", h.GetDashboard)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type planDayResponse struct {
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

type completeTaskRequest struct {
	Confidence       int    `json:"confidence"`
	TimeSpentMinutes int    `json:"time_spent_minutes"`
	Notes            string `json:"notes"`
}

type streakResponse struct {
	CurrentStreak int   `json:"current_streak"`
	LongestStreak int   `json:"longest_streak"`
	Days          []any `json:"days,omitempty"`
}

type completeTaskResponse struct {
	Task         taskResponse   `json:"task"`
	RevisionItem any            `json:"revision_item"`
	Streak       streakResponse `json:"streak"`
}

type skipTaskRequest struct {
	Reason string `json:"reason"`
}

type rescheduleTaskRequest struct {
	ToDate string `json:"to_date"`
}

type pillarReadinessResponse struct {
	Pillar         string  `json:"pillar"`
	Readiness      float64 `json:"readiness"`
	Coverage       float64 `json:"coverage"`
	AvgConfidence  float64 `json:"avg_confidence"`
	RevisionHealth float64 `json:"revision_health"`
}

type dashboardTodayResponse struct {
	Date           string  `json:"date"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	EstimatedHours float64 `json:"estimated_hours"`
	RemainingHours float64 `json:"remaining_hours"`
}

type dashboardResponse struct {
	OverallReadiness       float64                   `json:"overall_readiness"`
	EstimatedReadinessDate *string                   `json:"estimated_readiness_date"`
	PillarReadiness        []pillarReadinessResponse `json:"pillar_readiness"`
	StudyStreak            struct {
		Current int `json:"current"`
		Longest int `json:"longest"`
	} `json:"study_streak"`
	Today            dashboardTodayResponse `json:"today"`
	RevisionDueCount int                    `json:"revision_due_count"`
	GeneratedAt      string                 `json:"generated_at"`
}

// GetToday handles GET /today.
func (h *Handler) GetToday(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	day, err := h.svc.GetToday(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPlanDayResponse(day))
}

// CompleteTask handles POST /tasks/{taskId}/complete.
func (h *Handler) CompleteTask(c *gin.Context) {
	uid, taskID, ok := h.authAndTask(c)
	if !ok {
		return
	}
	var req completeTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}
	task, streak, err := h.svc.CompleteTask(c.Request.Context(), uid, taskID, CompleteParams{
		Confidence:       req.Confidence,
		TimeSpentMinutes: req.TimeSpentMinutes,
		Notes:            req.Notes,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, completeTaskResponse{
		Task:         toTaskResponse(task),
		RevisionItem: nil,
		Streak:       toStreakResponse(streak),
	})
}

// SkipTask handles POST /tasks/{taskId}/skip.
func (h *Handler) SkipTask(c *gin.Context) {
	uid, taskID, ok := h.authAndTask(c)
	if !ok {
		return
	}
	var req skipTaskRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
			return
		}
	}
	task, err := h.svc.SkipTask(c.Request.Context(), uid, taskID, req.Reason)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

// StartTask handles POST /tasks/{taskId}/start — mark a task in progress.
func (h *Handler) StartTask(c *gin.Context) {
	uid, taskID, ok := h.authAndTask(c)
	if !ok {
		return
	}
	task, err := h.svc.StartTask(c.Request.Context(), uid, taskID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

// ReopenTask handles POST /tasks/{taskId}/reopen — revert a completed/skipped
// task back to pending, undoing its completion side effects.
func (h *Handler) ReopenTask(c *gin.Context) {
	uid, taskID, ok := h.authAndTask(c)
	if !ok {
		return
	}
	task, err := h.svc.ReopenTask(c.Request.Context(), uid, taskID)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

// RescheduleTask handles POST /tasks/{taskId}/reschedule.
func (h *Handler) RescheduleTask(c *gin.Context) {
	uid, taskID, ok := h.authAndTask(c)
	if !ok {
		return
	}
	var req rescheduleTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}
	toDate, err := time.Parse("2006-01-02", req.ToDate)
	if err != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "to_date must be YYYY-MM-DD", nil)
		return
	}
	task, err := h.svc.RescheduleTask(c.Request.Context(), uid, taskID, toDate)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

// GetDashboard handles GET /dashboard.
func (h *Handler) GetDashboard(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	dash, err := h.svc.GetDashboard(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDashboardResponse(dash))
}

// authAndTask resolves the authenticated user and the :taskId path param.
func (h *Handler) authAndTask(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return uuid.Nil, uuid.Nil, false
	}
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid task id", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return uid, taskID, true
}

// writeServiceError maps domain errors to HTTP status + envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTaskNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "task not found", nil)
	case errors.Is(err, ErrPlanDayNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "no plan day for date", nil)
	case errors.Is(err, ErrTaskAlreadyResolved):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "task is already completed or skipped", nil)
	case errors.Is(err, ErrInvalidConfidence):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "confidence must be between 1 and 5 and time must be non-negative", nil)
	case errors.Is(err, ErrInvalidReschedule):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "a valid to_date is required", nil)
	case errors.Is(err, ErrNoTargetPlanDay):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "no plan day exists for the target date", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

// ---- mappers ----

func toPlanDayResponse(d *PlanDayRow) planDayResponse {
	out := planDayResponse{
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

func toTaskResponse(t *PlanTaskRow) taskResponse {
	out := taskResponse{
		ID:               t.ID.String(),
		PlanDayID:        t.PlanDayID.String(),
		Kind:             t.Kind,
		ItemType:         t.ItemType,
		ItemID:           t.ItemID.String(),
		PillarType:       t.PillarType,
		Title:            t.Title,
		Description:      t.Description,
		Objectives:       []string{},
		EstimatedMinutes: t.EstimatedMinutes,
		Priority:         t.Priority,
		Difficulty:       t.Difficulty,
		Status:           t.Status,
		SortOrder:        t.SortOrder,
		CompletionNotes:  t.CompletionNotes,
		TimeSpentMinutes: t.TimeSpentMinutes,
	}
	if t.Confidence != nil {
		v := int(*t.Confidence)
		out.Confidence = &v
	}
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

func toStreakResponse(s Streak) streakResponse {
	return streakResponse{CurrentStreak: s.Current, LongestStreak: s.Longest}
}

func toDashboardResponse(d *Dashboard) dashboardResponse {
	out := dashboardResponse{
		OverallReadiness: d.OverallReadiness,
		PillarReadiness:  make([]pillarReadinessResponse, 0, len(d.PillarReadiness)),
		RevisionDueCount: d.RevisionDueCount,
		GeneratedAt:      d.GeneratedAt.UTC().Format(time.RFC3339),
	}
	out.StudyStreak.Current = d.Streak.Current
	out.StudyStreak.Longest = d.Streak.Longest
	if d.EstimatedReadinessDate != nil {
		s := d.EstimatedReadinessDate.UTC().Format("2006-01-02")
		out.EstimatedReadinessDate = &s
	}
	for _, p := range d.PillarReadiness {
		out.PillarReadiness = append(out.PillarReadiness, pillarReadinessResponse{
			Pillar:         p.Pillar,
			Readiness:      p.Readiness,
			Coverage:       p.Coverage,
			AvgConfidence:  p.AvgConfidence,
			RevisionHealth: p.RevisionHealth,
		})
	}
	out.Today = dashboardTodayResponse{
		Date:           d.Today.Date.UTC().Format("2006-01-02"),
		TotalTasks:     d.Today.TotalTasks,
		CompletedTasks: d.Today.CompletedTasks,
		EstimatedHours: d.Today.EstimatedHours,
		RemainingHours: d.Today.RemainingHours,
	}
	return out
}
