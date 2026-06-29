package revision

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

// Handler wires the revision HTTP endpoints to the service. Every route is
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

// RegisterRoutes mounts the revision routes onto the /api/v1 group, matching the
// OpenAPI Revision paths. All routes require authentication.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/revision", auth.RequireAuth(h.tokens))
	{
		g.GET("/due", h.ListDue)
		g.POST("/:revisionItemId/recall", h.Recall)
	}
}

// ---- DTOs (mirror openapi.yaml schemas) ----

type revisionItemResponse struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	ItemType       string  `json:"item_type"`
	ItemID         string  `json:"item_id"`
	PillarType     string  `json:"pillar_type"`
	Title          string  `json:"title,omitempty"`
	IntervalDays   int     `json:"interval_days"`
	Stage          int     `json:"stage"`
	Ease           float64 `json:"ease"`
	DueAt          string  `json:"due_at"`
	LastReviewedAt *string `json:"last_reviewed_at"`
	LastRecall     *string `json:"last_recall"`
	ReviewCount    int     `json:"review_count"`
	LapseCount     int     `json:"lapse_count"`
	IsActive       bool    `json:"is_active"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type dueListResponse struct {
	Data []revisionItemResponse `json:"data"`
	Meta paginationMeta         `json:"meta"`
}

type recallRequest struct {
	Recall           string `json:"recall"`
	TimeSpentMinutes int    `json:"time_spent_minutes"`
	Notes            string `json:"notes"`
}

// ---- handlers ----

// ListDue handles GET /revision/due.
func (h *Handler) ListDue(c *gin.Context) {
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

	p := DueParams{Limit: pageSize, Offset: (page - 1) * pageSize}
	if d := c.Query("on_date"); d != "" {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil {
			server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "on_date must be YYYY-MM-DD", nil)
			return
		}
		p.OnDate = parsed
	}

	items, total, err := h.svc.Due(c.Request.Context(), uid, p)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	data := make([]revisionItemResponse, len(items))
	for i := range items {
		data[i] = toItemResponse(&items[i])
	}
	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	c.JSON(http.StatusOK, dueListResponse{
		Data: data,
		Meta: paginationMeta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

// Recall handles POST /revision/{revisionItemId}/recall.
func (h *Handler) Recall(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, err := uuid.Parse(c.Param("revisionItemId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid revision item id", nil)
		return
	}
	var req recallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return
	}
	item, err := h.svc.Recall(c.Request.Context(), uid, id, RecallResult(req.Recall))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toItemResponse(item))
}

// ---- helpers ----

func (h *Handler) unauth(c *gin.Context) {
	server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
}

func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrItemNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "revision item not found", nil)
	case errors.Is(err, ErrInvalidRecall):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "recall must be 'correct' or 'incorrect'", nil)
	case errors.Is(err, ErrItemGraduated):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "revision item has graduated; no further recalls", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// ---- mappers ----

func toItemResponse(it *Item) revisionItemResponse {
	out := revisionItemResponse{
		ID:           it.ID.String(),
		UserID:       it.UserID.String(),
		ItemType:     string(it.ItemType),
		ItemID:       it.ItemID.String(),
		PillarType:   it.PillarType,
		Title:        it.Title,
		IntervalDays: it.IntervalDays,
		Stage:        it.Stage,
		Ease:         it.Ease,
		DueAt:        it.DueAt.UTC().Format("2006-01-02"),
		ReviewCount:  it.ReviewCount,
		LapseCount:   it.LapseCount,
		IsActive:     it.IsActive,
	}
	if it.LastReviewedAt != nil {
		s := it.LastReviewedAt.UTC().Format(time.RFC3339)
		out.LastReviewedAt = &s
	}
	if it.LastRecall != nil {
		s := string(*it.LastRecall)
		out.LastRecall = &s
	}
	return out
}
