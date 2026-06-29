package mock

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

// Handler wires the mock interview HTTP endpoints to the service. Every route is
// protected by the auth.RequireAuth middleware (mock data belongs to a user).
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

// RegisterRoutes mounts the mock routes onto the /api/v1 group. All routes
// require authentication.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group("/mock-interviews", auth.RequireAuth(h.tokens))
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		// Weaknesses summary is a fixed sub-path; register before /:mockId so it
		// is not captured by the param route.
		g.GET("/weaknesses", h.Weaknesses)
		g.GET("/:mockId", h.Get)
		g.PUT("/:mockId", h.Update)
		g.DELETE("/:mockId", h.Delete)
		g.POST("/:mockId/findings", h.AddFinding)
	}
}

// ---- DTOs ----

type upsertRequest struct {
	Type            string   `json:"type"`
	TopicID         *string  `json:"topic_id"`
	DesignProblemID *string  `json:"design_problem_id"`
	CompanyID       *string  `json:"company_id"`
	ScheduledAt     *string  `json:"scheduled_at"`
	ConductedAt     *string  `json:"conducted_at"`
	DurationMinutes *int     `json:"duration_minutes"`
	Outcome         string   `json:"outcome"`
	OverallScore    *float64 `json:"overall_score"`
	Interviewer     string   `json:"interviewer"`
	TranscriptMD    string   `json:"transcript_md"`
	Summary         string   `json:"summary"`
}

type findingRequest struct {
	PillarType            *string `json:"pillar_type"`
	TopicID               *string `json:"topic_id"`
	Severity              string  `json:"severity"`
	Category              string  `json:"category"`
	Detail                string  `json:"detail"`
	CreateRemediationTask bool    `json:"create_remediation_task"`
}

type mockResponse struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	Type            string   `json:"type"`
	TopicID         *string  `json:"topic_id"`
	DesignProblemID *string  `json:"design_problem_id"`
	CompanyID       *string  `json:"company_id"`
	ScheduledAt     *string  `json:"scheduled_at"`
	ConductedAt     *string  `json:"conducted_at"`
	DurationMinutes *int     `json:"duration_minutes"`
	Outcome         string   `json:"outcome"`
	OverallScore    *float64 `json:"overall_score"`
	Interviewer     *string  `json:"interviewer"`
	Summary         *string  `json:"summary"`
	CreatedAt       string   `json:"created_at"`
}

type mockDetailResponse struct {
	mockResponse
	TranscriptMD *string           `json:"transcript_md"`
	Findings     []findingResponse `json:"findings"`
}

type findingResponse struct {
	ID                string  `json:"id"`
	MockInterviewID   string  `json:"mock_interview_id"`
	PillarType        *string `json:"pillar_type"`
	TopicID           *string `json:"topic_id"`
	Severity          string  `json:"severity"`
	Category          string  `json:"category"`
	Detail            string  `json:"detail"`
	RemediationTaskID *string `json:"remediation_task_id"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type listResponse struct {
	Data []mockResponse `json:"data"`
	Meta paginationMeta `json:"meta"`
}

type weaknessItemResponse struct {
	Area           string         `json:"area"`
	Pillar         *string        `json:"pillar,omitempty"`
	Count          int            `json:"count"`
	Score          int            `json:"score"`
	MaxSeverity    string         `json:"max_severity"`
	SeverityCounts map[string]int `json:"severity_counts"`
}

type weaknessResponse struct {
	Items         []weaknessItemResponse `json:"items"`
	TotalFindings int                    `json:"total_findings"`
	GeneratedBy   string                 `json:"generated_by"`
}

// ---- handlers ----

// List handles GET /mock-interviews.
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
		SortDesc: sortDesc(c.Query("sort")),
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}
	if t := c.Query("type"); t != "" {
		mt := Type(t)
		f.Type = &mt
	}

	items, total, err := h.svc.List(c.Request.Context(), uid, f)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}

	data := make([]mockResponse, len(items))
	for i := range items {
		data[i] = toMockResponse(&items[i])
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

// Create handles POST /mock-interviews.
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
	in, verr := req.toInput()
	if verr != nil {
		h.writeServiceError(c, verr)
		return
	}
	m, err := h.svc.Create(c.Request.Context(), uid, in)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toMockResponse(m))
}

// Get handles GET /mock-interviews/:mockId.
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
	m, err := h.svc.Get(c.Request.Context(), uid, id)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toMockDetailResponse(m))
}

// Update handles PUT /mock-interviews/:mockId.
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
	in, verr := req.toInput()
	if verr != nil {
		h.writeServiceError(c, verr)
		return
	}
	m, err := h.svc.Update(c.Request.Context(), uid, id, in)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toMockDetailResponse(m))
}

// Delete handles DELETE /mock-interviews/:mockId.
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

// AddFinding handles POST /mock-interviews/:mockId/findings.
func (h *Handler) AddFinding(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	var req findingRequest
	if !h.bindJSON(c, &req) {
		return
	}
	in, verr := req.toInput()
	if verr != nil {
		h.writeServiceError(c, verr)
		return
	}
	f, err := h.svc.AddFinding(c.Request.Context(), uid, id, in)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toFindingResponse(f))
}

// Weaknesses handles GET /mock-interviews/weaknesses: a ranked weakness summary
// aggregated deterministically from the user's findings.
func (h *Handler) Weaknesses(c *gin.Context) {
	uid, ok := auth.UserIDFromContext(c)
	if !ok {
		h.unauth(c)
		return
	}
	sum, err := h.svc.Weaknesses(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWeaknessResponse(sum))
}

// ---- request -> input mapping ----

func (r upsertRequest) toInput() (CreateInput, error) {
	var fields []FieldError
	in := CreateInput{
		Type:            Type(strings.TrimSpace(r.Type)),
		Outcome:         Outcome(strings.TrimSpace(r.Outcome)),
		DurationMinutes: r.DurationMinutes,
		OverallScore:    r.OverallScore,
		Interviewer:     r.Interviewer,
		TranscriptMD:    r.TranscriptMD,
		Summary:         r.Summary,
	}
	in.TopicID = parseUUIDPtr(r.TopicID, "topic_id", &fields)
	in.DesignProblemID = parseUUIDPtr(r.DesignProblemID, "design_problem_id", &fields)
	in.CompanyID = parseUUIDPtr(r.CompanyID, "company_id", &fields)
	in.ScheduledAt = parseTimePtr(r.ScheduledAt, "scheduled_at", &fields)
	in.ConductedAt = parseTimePtr(r.ConductedAt, "conducted_at", &fields)
	if len(fields) > 0 {
		return CreateInput{}, &ValidationError{Fields: fields}
	}
	return in, nil
}

func (r findingRequest) toInput() (FindingInput, error) {
	var fields []FieldError
	in := FindingInput{
		Severity:              Severity(strings.TrimSpace(r.Severity)),
		Category:              r.Category,
		Detail:                r.Detail,
		CreateRemediationTask: r.CreateRemediationTask,
	}
	in.TopicID = parseUUIDPtr(r.TopicID, "topic_id", &fields)
	if r.PillarType != nil && strings.TrimSpace(*r.PillarType) != "" {
		p := Pillar(strings.TrimSpace(*r.PillarType))
		in.PillarType = &p
	}
	if len(fields) > 0 {
		return FindingInput{}, &ValidationError{Fields: fields}
	}
	return in, nil
}

// ---- helpers ----

func (h *Handler) parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("mockId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "mockId must be a valid UUID", nil)
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
	case errors.Is(err, ErrMockNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "mock interview not found", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func parseUUIDPtr(s *string, field string, fields *[]FieldError) *uuid.UUID {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		*fields = append(*fields, FieldError{Field: field, Message: "must be a valid UUID"})
		return nil
	}
	return &id
}

func parseTimePtr(s *string, field string, fields *[]FieldError) *time.Time {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		*fields = append(*fields, FieldError{Field: field, Message: "must be an RFC3339 date-time"})
		return nil
	}
	return &t
}

func toMockResponse(m *Interview) mockResponse {
	return mockResponse{
		ID:              m.ID.String(),
		UserID:          m.UserID.String(),
		Type:            string(m.Type),
		TopicID:         uuidPtrStr(m.TopicID),
		DesignProblemID: uuidPtrStr(m.DesignProblemID),
		CompanyID:       uuidPtrStr(m.CompanyID),
		ScheduledAt:     timePtrStr(m.ScheduledAt),
		ConductedAt:     timePtrStr(m.ConductedAt),
		DurationMinutes: m.DurationMinutes,
		Outcome:         string(m.Outcome),
		OverallScore:    m.OverallScore,
		Interviewer:     m.Interviewer,
		Summary:         m.Summary,
		CreatedAt:       m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toMockDetailResponse(m *Interview) mockDetailResponse {
	findings := make([]findingResponse, len(m.Findings))
	for i := range m.Findings {
		findings[i] = toFindingResponse(&m.Findings[i])
	}
	return mockDetailResponse{
		mockResponse: toMockResponse(m),
		TranscriptMD: m.TranscriptMD,
		Findings:     findings,
	}
}

func toFindingResponse(f *Finding) findingResponse {
	var pillar *string
	if f.PillarType != nil {
		s := string(*f.PillarType)
		pillar = &s
	}
	return findingResponse{
		ID:                f.ID.String(),
		MockInterviewID:   f.MockInterviewID.String(),
		PillarType:        pillar,
		TopicID:           uuidPtrStr(f.TopicID),
		Severity:          string(f.Severity),
		Category:          f.Category,
		Detail:            f.Detail,
		RemediationTaskID: uuidPtrStr(f.RemediationTaskID),
	}
}

func toWeaknessResponse(s *WeaknessSummary) weaknessResponse {
	items := make([]weaknessItemResponse, len(s.Items))
	for i, it := range s.Items {
		var pillar *string
		if it.Pillar != nil {
			p := string(*it.Pillar)
			pillar = &p
		}
		counts := make(map[string]int, len(it.SeverityCounts))
		for sev, n := range it.SeverityCounts {
			counts[string(sev)] = n
		}
		items[i] = weaknessItemResponse{
			Area:           it.Area,
			Pillar:         pillar,
			Count:          it.Count,
			Score:          it.Score,
			MaxSeverity:    string(it.MaxSeverity),
			SeverityCounts: counts,
		}
	}
	return weaknessResponse{
		Items:         items,
		TotalFindings: s.TotalFindings,
		GeneratedBy:   s.GeneratedBy,
	}
}

func uuidPtrStr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func timePtrStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
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
