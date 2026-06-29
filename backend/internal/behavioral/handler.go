package behavioral

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the behavioral HTTP endpoints to the service. Every route is
// protected by the auth.RequireAuth middleware.
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

// RegisterRoutes mounts the behavioral routes onto the /api/v1 group. All
// routes require authentication.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/behavioral-stories", auth.RequireAuth(h.tokens))
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:storyId", h.Get)
		g.PUT("/:storyId", h.Update)
		g.DELETE("/:storyId", h.Delete)
		g.POST("/:storyId/improve", h.Improve)
	}
	// Note: POST /api/v1/ai/story-improve is owned by the internal/ai module
	// (M4), which adds graceful fallback + ai_invocations logging. This module
	// owns only the resource-scoped POST /behavioral-stories/{id}/improve.
}

// ---- DTOs ----

type upsertRequest struct {
	Title     string   `json:"title"`
	Theme     string   `json:"theme"`
	Situation string   `json:"situation"`
	Task      string   `json:"task"`
	Action    string   `json:"action"`
	Result    string   `json:"result"`
	Metrics   string   `json:"metrics"`
	Tags      []string `json:"tags"`
}

func (r upsertRequest) toInput() CreateInput {
	return CreateInput{
		Title:     r.Title,
		Theme:     Theme(r.Theme),
		Situation: r.Situation,
		Task:      r.Task,
		Action:    r.Action,
		Result:    r.Result,
		Metrics:   r.Metrics,
		Tags:      r.Tags,
	}
}

type storyResponse struct {
	ID            string   `json:"id"`
	UserID        string   `json:"user_id"`
	Title         string   `json:"title"`
	Theme         string   `json:"theme"`
	Situation     *string  `json:"situation"`
	Task          *string  `json:"task"`
	Action        *string  `json:"action"`
	Result        *string  `json:"result"`
	Metrics       *string  `json:"metrics"`
	Tags          []string `json:"tags"`
	AIImproved    bool     `json:"ai_improved"`
	StrengthScore *float64 `json:"strength_score"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type listResponse struct {
	Data []storyResponse `json:"data"`
	Meta paginationMeta  `json:"meta"`
}

type improveResponse struct {
	StoryID       string        `json:"story_id,omitempty"`
	Improved      *ImprovedSTAR `json:"improved,omitempty"`
	Suggestions   []string      `json:"suggestions"`
	StrengthScore float64       `json:"strength_score"`
	UsedFallback  bool          `json:"used_fallback"`
}

// ---- handlers ----

// List handles GET /behavioral-stories.
func (h *Handler) List(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}

	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(c.Query("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	f := ListFilter{
		Query:    strings.TrimSpace(c.Query("q")),
		SortDesc: sortDesc(c.Query("sort")),
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}
	if t := c.Query("theme"); t != "" {
		th := Theme(t)
		f.Theme = &th
	}

	items, total, err := h.svc.List(c.Request.Context(), uid, f)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	data := make([]storyResponse, len(items))
	for i := range items {
		data[i] = toStoryResponse(&items[i])
	}
	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	c.JSON(http.StatusOK, listResponse{
		Data: data,
		Meta: paginationMeta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

// Create handles POST /behavioral-stories.
func (h *Handler) Create(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	var req upsertRequest
	if !h.bindJSON(c, &req) {
		return
	}
	story, err := h.svc.Create(c.Request.Context(), uid, req.toInput())
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toStoryResponse(story))
}

// Get handles GET /behavioral-stories/:storyId.
func (h *Handler) Get(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	story, err := h.svc.Get(c.Request.Context(), uid, id)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toStoryResponse(story))
}

// Update handles PUT /behavioral-stories/:storyId.
func (h *Handler) Update(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	var req upsertRequest
	if !h.bindJSON(c, &req) {
		return
	}
	story, err := h.svc.Update(c.Request.Context(), uid, id, req.toInput())
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toStoryResponse(story))
}

// Delete handles DELETE /behavioral-stories/:storyId.
func (h *Handler) Delete(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uid, id); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Improve handles POST /behavioral-stories/:storyId/improve.
func (h *Handler) Improve(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	story, res, err := h.svc.Improve(c.Request.Context(), uid, id)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toImproveResponse(story.ID.String(), res))
}

// ---- helpers ----

func (h *Handler) parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("storyId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "storyId must be a valid UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return false
	}
	return true
}

func (h *Handler) unauth(c *gin.Context) {
	server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
}

// writeServiceError maps domain errors to HTTP status + error envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		details := make([]server.FieldError, len(ve.Fields))
		for i, f := range ve.Fields {
			details[i] = server.FieldError{Field: f.Field, Message: f.Message}
		}
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed", details)
	case errors.Is(err, ErrStoryNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "behavioral story not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toStoryResponse(s *Story) storyResponse {
	tags := []string(s.Tags)
	if tags == nil {
		tags = []string{}
	}
	return storyResponse{
		ID:            s.ID.String(),
		UserID:        s.UserID.String(),
		Title:         s.Title,
		Theme:         string(s.Theme),
		Situation:     s.Situation,
		Task:          s.Task,
		Action:        s.Action,
		Result:        s.Result,
		Metrics:       s.Metrics,
		Tags:          tags,
		AIImproved:    s.AIImproved,
		StrengthScore: s.StrengthScore,
		CreatedAt:     s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toImproveResponse(storyID string, r *ImproveResult) improveResponse {
	resp := improveResponse{
		StoryID:       storyID,
		Suggestions:   r.Suggestions,
		StrengthScore: r.StrengthScore,
		UsedFallback:  r.UsedFallback,
	}
	if r.Improved != (ImprovedSTAR{}) {
		imp := r.Improved
		resp.Improved = &imp
	}
	if resp.Suggestions == nil {
		resp.Suggestions = []string{}
	}
	return resp
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// sortDesc interprets the openapi sort param: a leading "-" on created_at means
// descending. Default (empty) is descending (newest first).
func sortDesc(sort string) bool {
	sort = strings.TrimSpace(sort)
	if sort == "" {
		return true
	}
	for _, field := range strings.Split(sort, ",") {
		field = strings.TrimSpace(field)
		if field == "created_at" {
			return false
		}
		if field == "-created_at" {
			return true
		}
	}
	return true
}
