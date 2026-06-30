package resume

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// Handler wires the resume HTTP endpoints to the service. All routes are
// protected by RequireAuth.
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

// RegisterRoutes mounts the resume routes onto the /api/v1 group, all behind
// RequireAuth.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	r := v1.Group("/resume")
	r.Use(auth.RequireAuth(h.tokens))
	{
		r.GET("/profile", h.GetProfile)
		r.PUT("/profile", h.UpsertProfile)
		r.DELETE("/profile", h.DeleteProfile)
		r.GET("/projects", h.ListProjects)
		r.POST("/projects", h.CreateProject)
		r.PUT("/projects/:projectId", h.UpdateProject)
		r.DELETE("/projects/:projectId", h.DeleteProject)
		r.POST("/score", h.Score)
		r.POST("/file", h.UploadFile)
		r.GET("/file", h.DownloadFile)
		r.GET("/file/meta", h.GetFileMeta)
		r.DELETE("/file", h.DeleteFile)
	}
}

// ---- Request DTOs (per openapi.yaml) ----

type profileUpsertRequest struct {
	Headline        *string  `json:"headline"`
	Summary         *string  `json:"summary"`
	YearsExperience *float64 `json:"years_experience"`
	Skills          []string `json:"skills"`
	TargetKeywords  []string `json:"target_keywords"`
}

type projectUpsertRequest struct {
	Name        string   `json:"name"`
	Role        *string  `json:"role"`
	Description *string  `json:"description"`
	Impact      *string  `json:"impact"`
	Metrics     []string `json:"metrics"`
	TechStack   []string `json:"tech_stack"`
	StartDate   *string  `json:"start_date"`
	EndDate     *string  `json:"end_date"`
	SortOrder   int      `json:"sort_order"`
}

// ---- Response DTOs (per openapi.yaml) ----

type profileResponse struct {
	ID              string            `json:"id"`
	UserID          string            `json:"user_id"`
	Headline        *string           `json:"headline"`
	Summary         *string           `json:"summary"`
	YearsExperience *float64          `json:"years_experience"`
	Skills          []string          `json:"skills"`
	TargetKeywords  []string          `json:"target_keywords"`
	ATSScore        *float64          `json:"ats_score"`
	LastScoredAt    *string           `json:"last_scored_at"`
	Projects        []projectResponse `json:"projects"`
}

type projectResponse struct {
	ID              string   `json:"id"`
	ResumeProfileID string   `json:"resume_profile_id"`
	Name            string   `json:"name"`
	Role            *string  `json:"role"`
	Description     *string  `json:"description"`
	Impact          *string  `json:"impact"`
	Metrics         []string `json:"metrics"`
	TechStack       []string `json:"tech_stack"`
	StartDate       *string  `json:"start_date"`
	EndDate         *string  `json:"end_date"`
	SortOrder       int      `json:"sort_order"`
}

type scoreResponse struct {
	ATSScore        float64          `json:"ats_score"`
	KeywordMatches  []string         `json:"keyword_matches"`
	MissingKeywords []string         `json:"missing_keywords"`
	Suggestions     []string         `json:"suggestions"`
	Breakdown       []ScoreBreakdown `json:"breakdown"`
	UsedFallback    bool             `json:"used_fallback"`
}

type fileMetaResponse struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int    `json:"size_bytes"`
	UploadedAt  string `json:"uploaded_at"`
}

const maxUploadBytes = 5 * 1024 * 1024

// ---- Handlers ----

// GetProfile handles GET /resume/profile.
func (h *Handler) GetProfile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetProfile(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// UpsertProfile handles PUT /resume/profile.
func (h *Handler) UpsertProfile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req profileUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.UpsertProfile(c.Request.Context(), uid, ProfileInput{
		Headline:        req.Headline,
		Summary:         req.Summary,
		YearsExperience: req.YearsExperience,
		Skills:          req.Skills,
		TargetKeywords:  req.TargetKeywords,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(p))
}

// DeleteProfile handles DELETE /resume/profile, soft-deleting the user's
// resume profile.
func (h *Handler) DeleteProfile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProfile(c.Request.Context(), uid); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListProjects handles GET /resume/projects.
func (h *Handler) ListProjects(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	projects, err := h.svc.ListProjects(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	out := make([]projectResponse, 0, len(projects))
	for i := range projects {
		out = append(out, toProjectResponse(&projects[i]))
	}
	c.JSON(http.StatusOK, out)
}

// CreateProject handles POST /resume/projects.
func (h *Handler) CreateProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	var req projectUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.CreateProject(c.Request.Context(), uid, toProjectInput(req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProjectResponse(p))
}

// UpdateProject handles PUT /resume/projects/:projectId.
func (h *Handler) UpdateProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	pid, ok := h.projectID(c)
	if !ok {
		return
	}
	var req projectUpsertRequest
	if !bindJSON(c, &req) {
		return
	}
	p, err := h.svc.UpdateProject(c.Request.Context(), uid, pid, toProjectInput(req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(p))
}

// DeleteProject handles DELETE /resume/projects/:projectId.
func (h *Handler) DeleteProject(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	pid, ok := h.projectID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProject(c.Request.Context(), uid, pid); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Score handles POST /resume/score.
func (h *Handler) Score(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	result, err := h.svc.Score(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, scoreResponse{
		ATSScore:        result.ATSScore,
		KeywordMatches:  nonNil(result.KeywordMatches),
		MissingKeywords: nonNil(result.MissingKeywords),
		Suggestions:     nonNil(result.Suggestions),
		Breakdown:       result.Breakdown,
		UsedFallback:    result.UsedFallback,
	})
}

// UploadFile handles POST /resume/file (multipart/form-data, field "file").
func (h *Handler) UploadFile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "missing 'file' upload field", nil)
		return
	}
	if fh.Size > maxUploadBytes {
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "file too large (max 5MB)", nil)
		return
	}
	src, err := fh.Open()
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "could not read uploaded file", nil)
		return
	}
	defer src.Close()
	content, err := io.ReadAll(io.LimitReader(src, maxUploadBytes+1))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "could not read uploaded file", nil)
		return
	}

	f, err := h.svc.UploadFile(c.Request.Context(), uid, FileInput{
		FileName:    filepath.Base(fh.Filename),
		ContentType: fh.Header.Get("Content-Type"),
		Content:     content,
	})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toFileMetaResponse(f))
}

// DownloadFile handles GET /resume/file — streams the stored bytes as an attachment.
func (h *Handler) DownloadFile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	f, err := h.svc.GetFile(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", f.FileName))
	c.Data(http.StatusOK, f.ContentType, f.Content)
}

// GetFileMeta handles GET /resume/file/meta — JSON metadata only.
func (h *Handler) GetFileMeta(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	f, err := h.svc.GetFile(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toFileMetaResponse(f))
}

// DeleteFile handles DELETE /resume/file.
func (h *Handler) DeleteFile(c *gin.Context) {
	uid, ok := h.userID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteFile(c.Request.Context(), uid); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toFileMetaResponse(f *ResumeFile) fileMetaResponse {
	return fileMetaResponse{
		FileName:    f.FileName,
		ContentType: f.ContentType,
		SizeBytes:   f.SizeBytes,
		UploadedAt:  f.CreatedAt.UTC().Format(time.RFC3339),
	}
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

func (h *Handler) projectID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid project id", nil)
		return uuid.Nil, false
	}
	return id, true
}

// bindJSON binds the request body, writing a 400 envelope on malformed JSON.
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return false
	}
	return true
}

// writeServiceError maps domain errors to HTTP status + error envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resume profile not found", nil)
	case errors.Is(err, ErrProjectNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resume project not found", nil)
	case errors.Is(err, ErrFileNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resume file not found", nil)
	case errors.Is(err, ErrForbidden):
		server.AbortError(c, http.StatusForbidden, server.CodeForbidden, "not permitted", nil)
	case errors.Is(err, ErrValidation):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, err.Error(), nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func toProjectInput(req projectUpsertRequest) ProjectInput {
	return ProjectInput{
		Name:        req.Name,
		Role:        req.Role,
		Description: req.Description,
		Impact:      req.Impact,
		Metrics:     req.Metrics,
		TechStack:   req.TechStack,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		SortOrder:   req.SortOrder,
	}
}

func toProfileResponse(p *Profile) profileResponse {
	r := profileResponse{
		ID:              p.ID.String(),
		UserID:          p.UserID.String(),
		Headline:        p.Headline,
		Summary:         p.Summary,
		YearsExperience: p.YearsExperience,
		Skills:          nonNil([]string(p.Skills)),
		TargetKeywords:  nonNil([]string(p.TargetKeywords)),
		ATSScore:        p.ATSScore,
		Projects:        make([]projectResponse, 0, len(p.Projects)),
	}
	if p.LastScoredAt != nil {
		s := p.LastScoredAt.UTC().Format(time.RFC3339)
		r.LastScoredAt = &s
	}
	for i := range p.Projects {
		r.Projects = append(r.Projects, toProjectResponse(&p.Projects[i]))
	}
	return r
}

func toProjectResponse(p *Project) projectResponse {
	r := projectResponse{
		ID:              p.ID.String(),
		ResumeProfileID: p.ResumeProfileID.String(),
		Name:            p.Name,
		Role:            p.Role,
		Description:     p.Description,
		Impact:          p.Impact,
		Metrics:         nonNil([]string(p.Metrics)),
		TechStack:       nonNil([]string(p.TechStack)),
		SortOrder:       p.SortOrder,
	}
	if p.StartDate != nil {
		s := p.StartDate.UTC().Format("2006-01-02")
		r.StartDate = &s
	}
	if p.EndDate != nil {
		s := p.EndDate.UTC().Format("2006-01-02")
		r.EndDate = &s
	}
	return r
}

// nonNil returns an empty slice instead of nil so JSON renders "[]" not "null".
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
