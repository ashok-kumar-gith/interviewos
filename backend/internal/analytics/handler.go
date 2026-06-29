package analytics

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the Analytics Engine HTTP endpoints (GET /analytics/*) to the
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

// RegisterRoutes mounts the protected /analytics routes onto /api/v1, matching
// the OpenAPI Analytics paths. (GET /dashboard is owned by the progress module.)
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/analytics")
	g.Use(auth.RequireAuth(h.tokens))
	{
		g.GET("/readiness", h.GetReadiness)
		g.GET("/snapshots", h.GetSnapshots)
		g.GET("/streak", h.GetStreak)
		g.GET("/topics", h.GetTopics)
		g.GET("/time-spent", h.GetTimeSpent)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type readinessSnapshotResponse struct {
	ID                 *string            `json:"id,omitempty"`
	UserID             *string            `json:"user_id,omitempty"`
	RoadmapID          *string            `json:"roadmap_id"`
	SnapshotDate       string             `json:"snapshot_date"`
	OverallReadiness   float64            `json:"overall_readiness"`
	PillarReadiness    map[string]float64 `json:"pillar_readiness"`
	CompletionPct      float64            `json:"completion_pct"`
	AvgConfidence      *float64           `json:"avg_confidence"`
	RevisionHealth     *float64           `json:"revision_health"`
	EstimatedReadyDate *string            `json:"estimated_ready_date"`
	WeakTopics         []string           `json:"weak_topics"`
	StrongTopics       []string           `json:"strong_topics"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type snapshotsResponse struct {
	Data []readinessSnapshotResponse `json:"data"`
	Meta paginationMeta              `json:"meta"`
}

type streakDayResponse struct {
	Date           string `json:"date"`
	TasksCompleted int    `json:"tasks_completed"`
	MinutesStudied int    `json:"minutes_studied"`
	GoalMet        bool   `json:"goal_met"`
}

type streakResponse struct {
	CurrentStreak int                 `json:"current_streak"`
	LongestStreak int                 `json:"longest_streak"`
	Days          []streakDayResponse `json:"days"`
}

type topicEntryResponse struct {
	TopicID          string   `json:"topic_id"`
	TopicName        string   `json:"topic_name"`
	PillarType       string   `json:"pillar_type"`
	Confidence       *int     `json:"confidence"`
	CompletionPct    float64  `json:"completion_pct"`
	RevisionAccuracy *float64 `json:"revision_accuracy"`
}

type topicAnalyticsResponse struct {
	Weak   []topicEntryResponse `json:"weak"`
	Strong []topicEntryResponse `json:"strong"`
}

type timeBucketResponse struct {
	Key     string `json:"key"`
	Minutes int    `json:"minutes"`
}

type timeSpentResponse struct {
	TotalMinutes int                  `json:"total_minutes"`
	GroupBy      string               `json:"group_by"`
	Buckets      []timeBucketResponse `json:"buckets"`
}

// GetReadiness handles GET /analytics/readiness.
func (h *Handler) GetReadiness(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	r, err := h.svc.Readiness(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toReadinessResponse(r))
}

// GetSnapshots handles GET /analytics/snapshots.
func (h *Handler) GetSnapshots(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	from, to, err := parseDateRange(c)
	if err != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "from/to must be YYYY-MM-DD with from<=to", nil)
		return
	}
	page, pageSize := parsePaging(c)
	offset := (page - 1) * pageSize

	snaps, total, err := h.svc.Snapshots(c.Request.Context(), uid, from, to, pageSize, offset)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]readinessSnapshotResponse, 0, len(snaps))
	for i := range snaps {
		data = append(data, toSnapshotResponse(&snaps[i]))
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	c.JSON(http.StatusOK, snapshotsResponse{
		Data: data,
		Meta: paginationMeta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

// GetStreak handles GET /analytics/streak.
func (h *Handler) GetStreak(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	from, to, err := parseDateRange(c)
	if err != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "from/to must be YYYY-MM-DD with from<=to", nil)
		return
	}
	s, err := h.svc.Streak(c.Request.Context(), uid, from, to)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toStreakResponse(s))
}

// GetTopics handles GET /analytics/topics.
func (h *Handler) GetTopics(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	ta, err := h.svc.Topics(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	bucket := c.DefaultQuery("bucket", "all")
	resp := topicAnalyticsResponse{Weak: []topicEntryResponse{}, Strong: []topicEntryResponse{}}
	if bucket == "weak" || bucket == "all" {
		resp.Weak = toTopicEntries(ta.Weak)
	}
	if bucket == "strong" || bucket == "all" {
		resp.Strong = toTopicEntries(ta.Strong)
	}
	c.JSON(http.StatusOK, resp)
}

// GetTimeSpent handles GET /analytics/time-spent.
func (h *Handler) GetTimeSpent(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	from, to, err := parseDateRange(c)
	if err != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "from/to must be YYYY-MM-DD with from<=to", nil)
		return
	}
	groupBy := c.DefaultQuery("group_by", "day")
	if groupBy != "day" && groupBy != "pillar" {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "group_by must be day or pillar", nil)
		return
	}
	ts, err := h.svc.TimeSpent(c.Request.Context(), uid, from, to, groupBy)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTimeSpentResponse(ts))
}

// writeError maps domain errors to HTTP status + envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "no profile; complete intake first", nil)
	case errors.Is(err, ErrInvalidDateRange):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "invalid date range", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

// ---- query parsing ----

func parseDateRange(c *gin.Context) (from, to time.Time, err error) {
	if v := c.Query("from"); v != "" {
		from, err = time.Parse("2006-01-02", v)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidDateRange
		}
	}
	if v := c.Query("to"); v != "" {
		to, err = time.Parse("2006-01-02", v)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidDateRange
		}
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}
	return from, to, nil
}

const (
	defaultPageSize = 30
	maxPageSize     = 200
)

func parsePaging(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = defaultPageSize
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v >= 1 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("page_size")); err == nil && v >= 1 {
		pageSize = v
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
	}
	return page, pageSize
}

// ---- mappers ----

func toReadinessResponse(r *Readiness) readinessSnapshotResponse {
	out := readinessSnapshotResponse{
		SnapshotDate:     r.SnapshotDate.UTC().Format("2006-01-02"),
		OverallReadiness: r.OverallReadiness,
		PillarReadiness:  r.PillarReadiness,
		CompletionPct:    r.CompletionPct,
		AvgConfidence:    r.AvgConfidence,
		RevisionHealth:   r.RevisionHealth,
		WeakTopics:       uuidStrings(r.WeakTopics),
		StrongTopics:     uuidStrings(r.StrongTopics),
	}
	if out.PillarReadiness == nil {
		out.PillarReadiness = map[string]float64{}
	}
	if r.EstimatedReadyDate != nil {
		s := r.EstimatedReadyDate.UTC().Format("2006-01-02")
		out.EstimatedReadyDate = &s
	}
	return out
}

func toSnapshotResponse(s *Snapshot) readinessSnapshotResponse {
	id := s.ID.String()
	uid := s.UserID.String()
	out := readinessSnapshotResponse{
		ID:               &id,
		UserID:           &uid,
		SnapshotDate:     s.SnapshotDate.UTC().Format("2006-01-02"),
		OverallReadiness: s.OverallReadiness,
		PillarReadiness:  s.PillarReadiness,
		CompletionPct:    s.CompletionPct,
		AvgConfidence:    s.AvgConfidence,
		RevisionHealth:   s.RevisionHealth,
		WeakTopics:       uuidStrings(s.WeakTopics),
		StrongTopics:     uuidStrings(s.StrongTopics),
	}
	if out.PillarReadiness == nil {
		out.PillarReadiness = map[string]float64{}
	}
	if s.RoadmapID != nil {
		rid := s.RoadmapID.String()
		out.RoadmapID = &rid
	}
	if s.EstimatedReadyDate != nil {
		d := s.EstimatedReadyDate.UTC().Format("2006-01-02")
		out.EstimatedReadyDate = &d
	}
	return out
}

func toStreakResponse(s *Streak) streakResponse {
	out := streakResponse{
		CurrentStreak: s.Current,
		LongestStreak: s.Longest,
		Days:          make([]streakDayResponse, 0, len(s.Days)),
	}
	for _, d := range s.Days {
		out.Days = append(out.Days, streakDayResponse{
			Date:           d.Date.UTC().Format("2006-01-02"),
			TasksCompleted: d.TasksCompleted,
			MinutesStudied: d.MinutesStudied,
			GoalMet:        d.GoalMet,
		})
	}
	return out
}

func toTopicEntries(entries []TopicEntry) []topicEntryResponse {
	out := make([]topicEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, topicEntryResponse{
			TopicID:          e.TopicID.String(),
			TopicName:        e.TopicName,
			PillarType:       e.PillarType,
			Confidence:       e.Confidence,
			CompletionPct:    round2(e.CompletionPct),
			RevisionAccuracy: e.RevisionAccuracy,
		})
	}
	return out
}

func toTimeSpentResponse(ts *TimeSpent) timeSpentResponse {
	out := timeSpentResponse{
		TotalMinutes: ts.TotalMinutes,
		GroupBy:      ts.GroupBy,
		Buckets:      make([]timeBucketResponse, 0, len(ts.Buckets)),
	}
	for _, b := range ts.Buckets {
		out.Buckets = append(out.Buckets, timeBucketResponse{Key: b.Key, Minutes: b.Minutes})
	}
	return out
}
