package content

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/platform/server"
)

// Handler exposes the content/DSA/company browsing endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs a content Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the content browsing routes onto the /api/v1 group.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/tracks", h.ListTracks)
	v1.GET("/pillars", h.ListPillars)
	v1.GET("/topics", h.ListTopics)
	v1.GET("/topics/:topicId", h.GetTopic)
	v1.GET("/resources", h.ListResources)

	v1.GET("/patterns", h.ListPatterns)
	v1.GET("/problems", h.ListProblems)
	v1.GET("/problems/:problemId", h.GetProblem)

	v1.GET("/companies", h.ListCompanies)
	v1.GET("/companies/:companyId", h.GetCompany)
}

// ListTracks handles GET /tracks.
func (h *Handler) ListTracks(c *gin.Context) {
	page, pageSize := parsePagination(c)
	sort := parseSort(c.Query("sort"), trackSortable)
	res, err := h.svc.ListTracks(c.Request.Context(), c.Query("q"), sort, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]trackResponse, 0, len(res.Items))
	for _, t := range res.Items {
		data = append(data, toTrackResponse(t))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// ListPillars handles GET /pillars.
func (h *Handler) ListPillars(c *gin.Context) {
	page, pageSize := parsePagination(c)
	sort := parseSort(c.Query("sort"), pillarSortable)
	res, err := h.svc.ListPillars(c.Request.Context(), queryUUID(c, "track_id"), sort, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]pillarResponse, 0, len(res.Items))
	for _, p := range res.Items {
		data = append(data, toPillarResponse(p))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// ListTopics handles GET /topics.
func (h *Handler) ListTopics(c *gin.Context) {
	page, pageSize := parsePagination(c)
	filters := parseFilter(c.Query("filter"))
	f := TopicFilter{
		TrackID:    queryUUID(c, "track_id"),
		PillarID:   queryUUID(c, "pillar_id"),
		PillarType: pillarTypeParam(c.Query("pillar_type")),
		Difficulty: difficultyParam(firstNonEmpty(c.Query("difficulty"), filters["difficulty"])),
		Priority:   priorityParam(firstNonEmpty(c.Query("priority"), filters["priority"])),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort"), topicSortable),
	}
	if f.PillarID == nil {
		f.PillarID = filterUUID(filters, "pillar_id")
	}
	res, err := h.svc.ListTopics(c.Request.Context(), f, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]topicResponse, 0, len(res.Items))
	for _, t := range res.Items {
		data = append(data, toTopicResponse(t))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// GetTopic handles GET /topics/:topicId.
func (h *Handler) GetTopic(c *gin.Context) {
	id, ok := h.pathUUID(c, "topicId")
	if !ok {
		return
	}
	b, err := h.svc.GetTopic(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTopicDetailResponse(b))
}

// ListResources handles GET /resources.
func (h *Handler) ListResources(c *gin.Context) {
	page, pageSize := parsePagination(c)
	f := ResourceFilter{
		Type:       resourceTypeParam(c.Query("type")),
		TopicID:    queryUUID(c, "topic_id"),
		Difficulty: difficultyParam(c.Query("difficulty")),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort"), resourceSortable),
	}
	res, err := h.svc.ListResources(c.Request.Context(), f, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]resourceResponse, 0, len(res.Items))
	for _, r := range res.Items {
		data = append(data, toResourceResponse(r))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// ListPatterns handles GET /patterns.
func (h *Handler) ListPatterns(c *gin.Context) {
	page, pageSize := parsePagination(c)
	sort := parseSort(c.Query("sort"), patternSortable)
	res, err := h.svc.ListPatterns(c.Request.Context(), queryUUID(c, "track_id"), c.Query("q"), sort, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]patternResponse, 0, len(res.Items))
	for _, p := range res.Items {
		data = append(data, toPatternResponse(p))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// ListProblems handles GET /problems.
func (h *Handler) ListProblems(c *gin.Context) {
	page, pageSize := parsePagination(c)
	filters := parseFilter(c.Query("filter"))
	f := ProblemFilter{
		Difficulty: difficultyParam(firstNonEmpty(c.Query("difficulty"), filters["difficulty"])),
		PatternID:  firstUUID(queryUUID(c, "pattern_id"), filterUUID(filters, "pattern")),
		TopicID:    queryUUID(c, "topic_id"),
		CompanyID:  firstUUID(queryUUID(c, "company_id"), filterUUID(filters, "company")),
		Source:     sourceParam(firstNonEmpty(c.Query("source"), filters["source"])),
		Query:      c.Query("q"),
		Sort:       parseSort(c.Query("sort"), problemSortable),
	}
	// A non-UUID filter value (e.g. company:amazon, pattern:two-pointers) is
	// treated as a slug.
	if f.CompanyID == nil {
		if v, ok := filters["company"]; ok {
			if _, err := uuid.Parse(v); err != nil {
				f.CompanySlug = v
			}
		}
	}
	if f.PatternID == nil {
		if v, ok := filters["pattern"]; ok {
			if _, err := uuid.Parse(v); err != nil {
				f.PatternSlug = v
			}
		}
	}
	res, err := h.svc.ListProblems(c.Request.Context(), f, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]problemResponse, 0, len(res.Items))
	for _, p := range res.Items {
		data = append(data, toProblemResponse(p))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// GetProblem handles GET /problems/:problemId.
func (h *Handler) GetProblem(c *gin.Context) {
	id, ok := h.pathUUID(c, "problemId")
	if !ok {
		return
	}
	b, err := h.svc.GetProblem(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProblemDetailResponse(b))
}

// ListCompanies handles GET /companies.
func (h *Handler) ListCompanies(c *gin.Context) {
	page, pageSize := parsePagination(c)
	sort := parseSort(c.Query("sort"), companySortable)
	res, err := h.svc.ListCompanies(c.Request.Context(), c.Query("q"), sort, page, pageSize)
	if err != nil {
		h.writeError(c, err)
		return
	}
	data := make([]companyResponse, 0, len(res.Items))
	for _, co := range res.Items {
		data = append(data, toCompanyResponse(co))
	}
	c.JSON(http.StatusOK, paginatedResponse{Data: data, Meta: metaFor(res)})
}

// GetCompany handles GET /companies/:companyId.
func (h *Handler) GetCompany(c *gin.Context) {
	id, ok := h.pathUUID(c, "companyId")
	if !ok {
		return
	}
	b, err := h.svc.GetCompany(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCompanyDetailResponse(b))
}

// ---- helpers ----

// pathUUID parses a required UUID path param, writing a 400 on a malformed id.
func (h *Handler) pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, name+" must be a valid uuid", nil)
		return uuid.Nil, false
	}
	return id, true
}

// writeError maps content domain errors to the HTTP error envelope.
func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstUUID(vals ...*uuid.UUID) *uuid.UUID {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func difficultyParam(s string) *Difficulty {
	switch Difficulty(s) {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		d := Difficulty(s)
		return &d
	}
	return nil
}

func priorityParam(s string) *Priority {
	switch s {
	case "low", "medium", "high", "critical":
		p := Priority(s)
		return &p
	}
	return nil
}

func pillarTypeParam(s string) *PillarType {
	switch PillarType(s) {
	case PillarDSA, PillarSystem, PillarLLD, PillarBackendEng, PillarBehavioral, PillarResume:
		p := PillarType(s)
		return &p
	}
	return nil
}

func resourceTypeParam(s string) *ResourceType {
	switch s {
	case "book", "video", "article", "course", "github", "practice", "documentation", "blog", "cheatsheet":
		t := ResourceType(s)
		return &t
	}
	return nil
}

func sourceParam(s string) *ProblemSourceName {
	switch s {
	case "blind75", "neetcode150", "grind75", "tech_interview_handbook", "leetcode_top", "striver_sde", "custom":
		n := ProblemSourceName(s)
		return &n
	}
	return nil
}
