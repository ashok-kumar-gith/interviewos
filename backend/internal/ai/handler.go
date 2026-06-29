package ai

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the /ai/* endpoints to the orchestrator Service. It reuses the
// auth module's RequireAuth middleware (all AI features are user-scoped).
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

// RegisterRoutes mounts the protected /ai routes onto /api/v1.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/ai")
	g.Use(auth.RequireAuth(h.tokens))
	{
		g.POST("/planner", h.Planner)
		g.POST("/coach", h.Coach)
		g.POST("/resume-review", h.ResumeReview)
		g.POST("/story-improve", h.StoryImprove)
		g.POST("/weakness-detect", h.WeaknessDetect)
		g.POST("/daily-plan", h.DailyPlan)
		g.POST("/sd-review", h.SDReview)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type aiResponse struct {
	Feature      string         `json:"feature"`
	Content      string         `json:"content"`
	Structured   map[string]any `json:"structured,omitempty"`
	Model        *string        `json:"model"`
	UsedFallback bool           `json:"used_fallback"`
	InvocationID string         `json:"invocation_id"`
}

type plannerRequest struct {
	RoadmapID    *string  `json:"roadmap_id"`
	FocusPillars []string `json:"focus_pillars"`
	Notes        string   `json:"notes"`
}

type coachRequest struct {
	Message        string  `json:"message"`
	ContextTopicID *string `json:"context_topic_id"`
}

type storyImproveRequest struct {
	StoryID   *string `json:"story_id"`
	Situation string  `json:"situation"`
	Task      string  `json:"task"`
	Action    string  `json:"action"`
	Result    string  `json:"result"`
}

type dailyPlanRequest struct {
	Date string `json:"date"`
}

type sdReviewRequest struct {
	DesignProblemID string `json:"design_problem_id"`
	AnswerMD        string `json:"answer_md"`
}

type improvedSTAR struct {
	Situation string `json:"situation,omitempty"`
	Task      string `json:"task,omitempty"`
	Action    string `json:"action,omitempty"`
	Result    string `json:"result,omitempty"`
	Metrics   string `json:"metrics,omitempty"`
}

type storyImproveResponse struct {
	StoryID       string       `json:"story_id"`
	Improved      improvedSTAR `json:"improved"`
	Suggestions   []string     `json:"suggestions"`
	StrengthScore float64      `json:"strength_score"`
	UsedFallback  bool         `json:"used_fallback"`
	InvocationID  string       `json:"invocation_id"`
}

type resumeReviewResponse struct {
	ATSScore        float64  `json:"ats_score"`
	KeywordMatches  []string `json:"keyword_matches"`
	MissingKeywords []string `json:"missing_keywords"`
	Suggestions     []string `json:"suggestions"`
	UsedFallback    bool     `json:"used_fallback"`
	InvocationID    string   `json:"invocation_id"`
}

type topicEntry struct {
	TopicID       string  `json:"topic_id"`
	TopicName     string  `json:"topic_name"`
	PillarType    string  `json:"pillar_type"`
	Confidence    *int    `json:"confidence"`
	CompletionPct float64 `json:"completion_pct"`
}

type weaknessResponse struct {
	WeakTopics       []topicEntry `json:"weak_topics"`
	RecommendedTasks []string     `json:"recommended_tasks"`
	UsedFallback     bool         `json:"used_fallback"`
	InvocationID     string       `json:"invocation_id"`
}

// ---- Handlers ----

// Planner handles POST /ai/planner.
func (h *Handler) Planner(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req plannerRequest
	_ = c.ShouldBindJSON(&req) // body optional
	res, err := h.svc.Planner(c.Request.Context(), uid, PlannerInput{FocusPillars: req.FocusPillars, Notes: req.Notes})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAIResponse(res))
}

// Coach handles POST /ai/coach.
func (h *Handler) Coach(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req coachRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "message is required", nil)
		return
	}
	res, err := h.svc.Coach(c.Request.Context(), uid, req.Message)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAIResponse(res))
}

// ResumeReview handles POST /ai/resume-review.
func (h *Handler) ResumeReview(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	res, err := h.svc.ResumeReview(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resumeReviewResponse{
		ATSScore:        res.ATSScore,
		KeywordMatches:  nonNilStrings(res.KeywordMatches),
		MissingKeywords: nonNilStrings(res.MissingKeywords),
		Suggestions:     nonNilStrings(res.Suggestions),
		UsedFallback:    res.UsedFallback,
		InvocationID:    res.InvocationID.String(),
	})
}

// StoryImprove handles POST /ai/story-improve.
func (h *Handler) StoryImprove(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req storyImproveRequest
	_ = c.ShouldBindJSON(&req)
	in := StoryImproveInput{Situation: req.Situation, Task: req.Task, Action: req.Action, Result: req.Result}
	if req.StoryID != nil && *req.StoryID != "" {
		sid, perr := uuid.Parse(*req.StoryID)
		if perr != nil {
			server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "story_id must be a uuid", nil)
			return
		}
		in.StoryID = &sid
	} else if req.Situation == "" && req.Task == "" && req.Action == "" && req.Result == "" {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "provide story_id or inline STAR text", nil)
		return
	}
	res, err := h.svc.StoryImprove(c.Request.Context(), uid, in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, storyImproveResponse{
		StoryID: res.StoryID.String(),
		Improved: improvedSTAR{
			Situation: res.Improved.Situation,
			Task:      res.Improved.Task,
			Action:    res.Improved.Action,
			Result:    res.Improved.Result,
			Metrics:   res.Improved.Metrics,
		},
		Suggestions:   nonNilStrings(res.Suggestions),
		StrengthScore: res.StrengthScore,
		UsedFallback:  res.UsedFallback,
		InvocationID:  res.InvocationID.String(),
	})
}

// WeaknessDetect handles POST /ai/weakness-detect.
func (h *Handler) WeaknessDetect(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	res, err := h.svc.WeaknessDetect(c.Request.Context(), uid)
	if err != nil {
		h.writeError(c, err)
		return
	}
	entries := make([]topicEntry, 0, len(res.WeakTopics))
	for _, w := range res.WeakTopics {
		entries = append(entries, topicEntry{
			TopicID:       w.TopicID.String(),
			TopicName:     w.TopicName,
			PillarType:    w.PillarType,
			Confidence:    w.Confidence,
			CompletionPct: w.CompletionPct,
		})
	}
	c.JSON(http.StatusOK, weaknessResponse{
		WeakTopics:       entries,
		RecommendedTasks: nonNilStrings(res.RecommendedTasks),
		UsedFallback:     res.UsedFallback,
		InvocationID:     res.InvocationID.String(),
	})
}

// DailyPlan handles POST /ai/daily-plan.
func (h *Handler) DailyPlan(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req dailyPlanRequest
	_ = c.ShouldBindJSON(&req) // body optional; date defaults to today
	res, err := h.svc.DailyPlan(c.Request.Context(), uid, req.Date)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAIResponse(res))
}

// SDReview handles POST /ai/sd-review.
func (h *Handler) SDReview(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req sdReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.DesignProblemID == "" || req.AnswerMD == "" {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "design_problem_id and answer_md are required", nil)
		return
	}
	pid, perr := uuid.Parse(req.DesignProblemID)
	if perr != nil {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "design_problem_id must be a uuid", nil)
		return
	}
	res, err := h.svc.SDReview(c.Request.Context(), uid, SDReviewInput{DesignProblemID: pid, AnswerMD: req.AnswerMD})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAIResponse(res))
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

func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "required record not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toAIResponse(r *TextResult) aiResponse {
	return aiResponse{
		Feature:      string(r.Feature),
		Content:      r.Content,
		Structured:   r.Structured,
		Model:        r.Model,
		UsedFallback: r.UsedFallback,
		InvocationID: r.InvocationID.String(),
	}
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
