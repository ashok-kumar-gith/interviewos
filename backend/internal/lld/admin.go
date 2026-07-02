package lld

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/platform/server"
)

// WriteRepository abstracts admin write access to the LLD catalog. The GORM
// implementation is gormRepository (which also satisfies the read Repository).
type WriteRepository interface {
	Create(ctx context.Context, p *Problem) (*Problem, error)
	Update(ctx context.Context, id uuid.UUID, p *Problem) (*Problem, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DefaultTrackID(ctx context.Context) (uuid.UUID, error)
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)
}

// NewWriteRepository returns a gorm-backed WriteRepository sharing the same
// underlying gormRepository as NewRepository.
func NewWriteRepository(db *gorm.DB) WriteRepository {
	return &gormRepository{db: db}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "sqlstate 23505")
}

func (r *gormRepository) DefaultTrackID(ctx context.Context) (uuid.UUID, error) {
	type track struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var t track
	err := r.db.WithContext(ctx).Table("tracks").
		Select("id").Order("sort_order ASC, created_at ASC").First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return t.ID, nil
}

func (r *gormRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	tx := r.db.WithContext(ctx).Model(&Problem{}).Where("slug = ?", slug)
	if excludeID != nil {
		tx = tx.Where("id <> ?", *excludeID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *gormRepository) Create(ctx context.Context, p *Problem) (*Problem, error) {
	p.ID = uuid.Nil
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return r.GetProblem(ctx, p.ID)
}

func (r *gormRepository) Update(ctx context.Context, id uuid.UUID, p *Problem) (*Problem, error) {
	res := r.db.WithContext(ctx).Model(&Problem{}).Where("id = ?", id).Updates(map[string]any{
		"track_id":            p.TrackID,
		"pillar_id":           p.PillarID,
		"slug":                p.Slug,
		"title":               p.Title,
		"difficulty":          p.Difficulty,
		"order_index":         p.OrderIndex,
		"requirements_md":     p.RequirementsMD,
		"entities_md":         p.EntitiesMD,
		"class_diagram_md":    p.ClassDiagramMD,
		"design_patterns":     p.DesignPatterns,
		"solid_notes_md":      p.SolidNotesMD,
		"api_or_interface_md": p.APIOrInterfaceMD,
		"tradeoffs_md":        p.TradeoffsMD,
		"follow_up_questions": p.FollowUpQuestions,
		"updated_at":          gorm.Expr("now()"),
	})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return nil, ErrConflict
		}
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		var n int64
		if err := r.db.WithContext(ctx).Model(&Problem{}).Where("id = ?", id).Count(&n).Error; err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrNotFound
		}
	}
	return r.GetProblem(ctx, id)
}

func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Problem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- admin service ----

// AdminService implements the LLD admin write use-cases.
type AdminService struct {
	repo WriteRepository
}

// NewAdminService constructs an AdminService.
func NewAdminService(repo WriteRepository) *AdminService {
	return &AdminService{repo: repo}
}

// ProblemInput is the validated create/update payload for an LLD problem.
type ProblemInput struct {
	TrackID           *uuid.UUID
	PillarID          *uuid.UUID
	Slug              string
	Title             string
	Difficulty        string
	OrderIndex        *int
	RequirementsMD    *string
	EntitiesMD        *string
	ClassDiagramMD    *string
	DesignPatterns    []string
	SolidNotesMD      *string
	APIOrInterfaceMD  *string
	TradeoffsMD       *string
	FollowUpQuestions []string
}

func validDifficulty(s string) bool {
	switch Difficulty(s) {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	}
	return false
}

func (s *AdminService) build(ctx context.Context, in ProblemInput) (*Problem, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Title = strings.TrimSpace(in.Title)
	if in.Slug == "" || in.Title == "" || !validDifficulty(in.Difficulty) {
		return nil, ErrValidation
	}
	trackID := uuid.Nil
	if in.TrackID != nil {
		trackID = *in.TrackID
	} else {
		id, err := s.repo.DefaultTrackID(ctx)
		if err != nil {
			return nil, err
		}
		trackID = id
	}
	p := &Problem{
		TrackID:           trackID,
		PillarID:          in.PillarID,
		Slug:              in.Slug,
		Title:             in.Title,
		Difficulty:        Difficulty(in.Difficulty),
		RequirementsMD:    in.RequirementsMD,
		EntitiesMD:        in.EntitiesMD,
		ClassDiagramMD:    in.ClassDiagramMD,
		DesignPatterns:    JSONArray(in.DesignPatterns),
		SolidNotesMD:      in.SolidNotesMD,
		APIOrInterfaceMD:  in.APIOrInterfaceMD,
		TradeoffsMD:       in.TradeoffsMD,
		FollowUpQuestions: JSONArray(in.FollowUpQuestions),
	}
	if in.OrderIndex != nil {
		p.OrderIndex = *in.OrderIndex
	}
	if p.DesignPatterns == nil {
		p.DesignPatterns = JSONArray{}
	}
	if p.FollowUpQuestions == nil {
		p.FollowUpQuestions = JSONArray{}
	}
	return p, nil
}

// Create validates and creates an LLD problem.
func (s *AdminService) Create(ctx context.Context, in ProblemInput) (*Problem, error) {
	p, err := s.build(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.SlugExists(ctx, p.Slug, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.Create(ctx, p)
}

// Update validates and updates an LLD problem.
func (s *AdminService) Update(ctx context.Context, id uuid.UUID, in ProblemInput) (*Problem, error) {
	p, err := s.build(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.SlugExists(ctx, p.Slug, &id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.Update(ctx, id, p)
}

// Delete soft-deletes an LLD problem.
func (s *AdminService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// ---- admin handler ----

// AdminHandler exposes the admin-gated LLD CRUD endpoints.
type AdminHandler struct {
	svc    *AdminService
	tokens *auth.TokenManager
}

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(svc *AdminService, tokens *auth.TokenManager) *AdminHandler {
	return &AdminHandler{svc: svc, tokens: tokens}
}

// RegisterRoutes mounts the admin LLD routes, gated by RequireAdmin.
func (h *AdminHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	if h.tokens == nil {
		return
	}
	admin := v1.Group("", auth.RequireAdmin(h.tokens))
	admin.POST("/lld-problems", h.Create)
	admin.PUT("/lld-problems/:lldProblemId", h.Update)
	admin.DELETE("/lld-problems/:lldProblemId", h.Delete)
}

type problemRequest struct {
	TrackID           *string  `json:"track_id"`
	PillarID          *string  `json:"pillar_id"`
	Slug              string   `json:"slug"`
	Title             string   `json:"title"`
	Difficulty        string   `json:"difficulty"`
	OrderIndex        *int     `json:"order_index"`
	RequirementsMD    *string  `json:"requirements_md"`
	EntitiesMD        *string  `json:"entities_md"`
	ClassDiagramMD    *string  `json:"class_diagram_md"`
	DesignPatterns    []string `json:"design_patterns"`
	SolidNotesMD      *string  `json:"solid_notes_md"`
	APIOrInterfaceMD  *string  `json:"api_or_interface_md"`
	TradeoffsMD       *string  `json:"tradeoffs_md"`
	FollowUpQuestions []string `json:"follow_up_questions"`
}

func (r problemRequest) toInput() (ProblemInput, bool) {
	in := ProblemInput{
		Slug:              r.Slug,
		Title:             r.Title,
		Difficulty:        r.Difficulty,
		OrderIndex:        r.OrderIndex,
		RequirementsMD:    r.RequirementsMD,
		EntitiesMD:        r.EntitiesMD,
		ClassDiagramMD:    r.ClassDiagramMD,
		DesignPatterns:    r.DesignPatterns,
		SolidNotesMD:      r.SolidNotesMD,
		APIOrInterfaceMD:  r.APIOrInterfaceMD,
		TradeoffsMD:       r.TradeoffsMD,
		FollowUpQuestions: r.FollowUpQuestions,
	}
	if r.TrackID != nil {
		id, err := uuid.Parse(*r.TrackID)
		if err != nil {
			return ProblemInput{}, false
		}
		in.TrackID = &id
	}
	if r.PillarID != nil {
		id, err := uuid.Parse(*r.PillarID)
		if err != nil {
			return ProblemInput{}, false
		}
		in.PillarID = &id
	}
	return in, true
}

// Create handles POST /lld-problems (admin).
func (h *AdminHandler) Create(c *gin.Context) {
	var req problemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, ok := req.toInput()
	if !ok {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid uuid in request body", nil)
		return
	}
	p, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProblemDetailResponse(p))
}

// Update handles PUT /lld-problems/:lldProblemId (admin).
func (h *AdminHandler) Update(c *gin.Context) {
	id, ok := h.pathUUID(c, "lldProblemId")
	if !ok {
		return
	}
	var req problemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid request body", nil)
		return
	}
	in, valid := req.toInput()
	if !valid {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid uuid in request body", nil)
		return
	}
	p, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProblemDetailResponse(p))
}

// Delete handles DELETE /lld-problems/:lldProblemId (admin).
func (h *AdminHandler) Delete(c *gin.Context) {
	id, ok := h.pathUUID(c, "lldProblemId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, name+" must be a valid uuid", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *AdminHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "resource not found", nil)
	case errors.Is(err, ErrConflict):
		server.AbortError(c, http.StatusConflict, server.CodeConflict, "slug already exists", nil)
	case errors.Is(err, ErrValidation):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "invalid or missing required fields", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}
