package notification

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

// Handler wires the notification HTTP endpoints to the service. Every route is
// protected by the auth.RequireAuth middleware and scoped to the caller.
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

// RegisterRoutes mounts the notification routes onto the /api/v1 group. All
// routes require authentication.
//
//	GET  /notifications            list (optional ?status=, paginated)
//	POST /notifications/:id/read   mark one read
//	POST /notifications/read-all   mark all read
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/notifications", auth.RequireAuth(h.tokens))
	{
		g.GET("", h.List)
		g.POST("/read-all", h.MarkAllRead)
		g.POST("/:notificationId/read", h.MarkRead)
	}
}

// ---- DTOs ----

type notificationResponse struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Channel   string         `json:"channel"`
	Status    string         `json:"status"`
	Title     string         `json:"title"`
	Body      *string        `json:"body"`
	Payload   map[string]any `json:"payload"`
	ReadAt    *string        `json:"read_at"`
	CreatedAt string         `json:"created_at"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type listResponse struct {
	Data []notificationResponse `json:"data"`
	Meta paginationMeta         `json:"meta"`
}

// ---- handlers ----

// List handles GET /notifications.
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
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	if s := c.Query("status"); s != "" {
		st := Status(s)
		f.Status = &st
	}

	items, total, err := h.svc.List(c.Request.Context(), uid, f)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	data := make([]notificationResponse, len(items))
	for i := range items {
		data[i] = toResponse(&items[i])
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

// MarkRead handles POST /notifications/:notificationId/read.
func (h *Handler) MarkRead(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	n, err := h.svc.MarkRead(c.Request.Context(), uid, id)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(n))
}

// MarkAllRead handles POST /notifications/read-all.
func (h *Handler) MarkAllRead(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	if _, err := h.svc.MarkAllRead(c.Request.Context(), uid); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- helpers ----

func (h *Handler) parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("notificationId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "notificationId must be a valid UUID", nil)
		return uuid.Nil, false
	}
	return id, true
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
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "notification not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toResponse(n *Notification) notificationResponse {
	payload := map[string]any(n.Payload)
	if payload == nil {
		payload = map[string]any{}
	}
	var readAt *string
	if n.ReadAt != nil {
		s := n.ReadAt.UTC().Format(time.RFC3339)
		readAt = &s
	}
	return notificationResponse{
		ID:        n.ID.String(),
		Type:      string(n.Type),
		Channel:   string(n.Channel),
		Status:    string(n.Status),
		Title:     n.Title,
		Body:      n.Body,
		Payload:   payload,
		ReadAt:    readAt,
		CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
	}
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
