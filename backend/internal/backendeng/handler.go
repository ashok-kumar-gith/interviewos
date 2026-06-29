package backendeng

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/content"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the public Backend Engineering catalog browsing endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Backend Engineering Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the backend-engineering routes onto the /api/v1 group.
// Browsing is public (no auth), mirroring the content catalog.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/backend-engineering/topics", h.ListTopics)
	v1.GET("/backend-engineering/topics/:topicId", h.GetTopic)
}

// ListTopics handles GET /backend-engineering/topics. It returns the
// backend_engineering pillar topics with pagination, difficulty filter, sort,
// and full-text-ish name search.
func (h *Handler) ListTopics(c *gin.Context) {
	page := atoiDefault(c.Query("page"), defaultPage)
	pageSize := atoiDefault(c.Query("page_size"), defaultPageSize)
	res, err := h.svc.ListTopics(c.Request.Context(), TopicQuery{
		Difficulty: difficultyParam(c.Query("difficulty")),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]topicResponse, 0, len(res.Items))
	for _, t := range res.Items {
		data = append(data, toTopicResponse(t))
	}
	c.JSON(http.StatusOK, paginatedResponse{
		Data: data,
		Meta: paginationMeta{Page: res.Page, PageSize: res.PageSize, Total: res.Total, TotalPages: res.TotalPages},
	})
}

// GetTopic handles GET /backend-engineering/topics/:topicId, returning the topic
// with its subtopics and linked resources.
func (h *Handler) GetTopic(c *gin.Context) {
	id, err := uuid.Parse(c.Param("topicId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "topicId must be a valid uuid", nil)
		return
	}
	b, err := h.svc.GetTopic(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTopicDetailResponse(b))
}

// writeError maps content domain errors to the HTTP error envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, content.ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

// ---- param parsing ----

// topicSortable is the ORDER BY allowlist for backend-engineering topics. Only
// these columns may appear in a sort clause, guarding against SQL injection.
var topicSortable = map[string]struct{}{
	"name": {}, "slug": {}, "difficulty": {}, "priority": {},
	"sort_order": {}, "estimated_hours": {}, "created_at": {},
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

// parseSort parses the `sort` query param ("-difficulty,name") into validated
// content.SortField instructions, dropping any column not in the allowlist.
func parseSort(raw string) []content.SortField {
	if raw == "" {
		return nil
	}
	var out []content.SortField
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(part, "-") {
			desc = true
			part = part[1:]
		} else if strings.HasPrefix(part, "+") {
			part = part[1:]
		}
		col := strings.ToLower(strings.TrimSpace(part))
		if _, ok := topicSortable[col]; !ok {
			continue
		}
		out = append(out, content.SortField{Column: col, Desc: desc})
	}
	return out
}

// difficultyParam validates an optional difficulty query value, returning nil
// (drop the filter) when absent or malformed.
func difficultyParam(s string) *content.Difficulty {
	switch content.Difficulty(s) {
	case content.DifficultyEasy, content.DifficultyMedium, content.DifficultyHard:
		d := content.Difficulty(s)
		return &d
	}
	return nil
}
