package resume

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository abstracts persistence for the resume domain so the service can be
// unit-tested against a fake. The gorm implementation is gormRepository. All
// queries are scoped by user_id; gorm.DeletedAt auto-filters soft-deleted rows.
type Repository interface {
	// Profile.
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	CreateProfile(ctx context.Context, p *Profile) error
	UpdateProfile(ctx context.Context, p *Profile) error
	UpdateScore(ctx context.Context, profileID uuid.UUID, atsScore float64, aiFeedback []byte) error
	DeleteProfile(ctx context.Context, userID uuid.UUID) error

	// Projects.
	ListProjects(ctx context.Context, profileID uuid.UUID) ([]Project, error)
	GetProject(ctx context.Context, userID, projectID uuid.UUID) (*Project, error)
	CreateProject(ctx context.Context, p *Project) error
	UpdateProject(ctx context.Context, p *Project) error
	DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error

	// Resume file (uploaded bytes).
	UpsertFile(ctx context.Context, f *ResumeFile) error
	GetFile(ctx context.Context, userID uuid.UUID) (*ResumeFile, error)
	DeleteFile(ctx context.Context, userID uuid.UUID) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).
		Preload("Projects", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).
		Where("user_id = ?", userID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) CreateProfile(ctx context.Context, p *Profile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *gormRepository) UpdateProfile(ctx context.Context, p *Profile) error {
	// Update only the user-editable columns; scope by id AND user_id for safety.
	return r.db.WithContext(ctx).Model(&Profile{}).
		Where("id = ? AND user_id = ?", p.ID, p.UserID).
		Updates(map[string]any{
			"headline":         p.Headline,
			"summary":          p.Summary,
			"years_experience": p.YearsExperience,
			"skills":           p.Skills,
			"target_keywords":  p.TargetKeywords,
		}).Error
}

func (r *gormRepository) UpdateScore(ctx context.Context, profileID uuid.UUID, atsScore float64, aiFeedback []byte) error {
	return r.db.WithContext(ctx).Model(&Profile{}).
		Where("id = ?", profileID).
		Updates(map[string]any{
			"ats_score":      atsScore,
			"last_scored_at": gorm.Expr("now()"),
			"ai_feedback":    aiFeedback,
		}).Error
}

func (r *gormRepository) ListProjects(ctx context.Context, profileID uuid.UUID) ([]Project, error) {
	var out []Project
	err := r.db.WithContext(ctx).
		Where("resume_profile_id = ?", profileID).
		Order("sort_order ASC, created_at ASC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *gormRepository) GetProject(ctx context.Context, userID, projectID uuid.UUID) (*Project, error) {
	var p Project
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", projectID, userID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) CreateProject(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *gormRepository) UpdateProject(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Model(&Project{}).
		Where("id = ? AND user_id = ?", p.ID, p.UserID).
		Updates(map[string]any{
			"name":        p.Name,
			"role":        p.Role,
			"description": p.Description,
			"impact":      p.Impact,
			"metrics":     p.Metrics,
			"tech_stack":  p.TechStack,
			"start_date":  p.StartDate,
			"end_date":    p.EndDate,
			"sort_order":  p.SortOrder,
		}).Error
}

func (r *gormRepository) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", projectID, userID).
		Delete(&Project{}).Error
}

// DeleteProfile soft-deletes the user's resume profile.
func (r *gormRepository) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&Profile{}).Error
}

// UpsertFile replaces the user's current resume file: it soft-deletes any
// existing live row then inserts the new one, all in a transaction so the
// partial unique index on (user_id) WHERE deleted_at IS NULL is never violated.
func (r *gormRepository) UpsertFile(ctx context.Context, f *ResumeFile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", f.UserID).Delete(&ResumeFile{}).Error; err != nil {
			return err
		}
		return tx.Create(f).Error
	})
}

// GetFile returns the user's current (non-deleted) resume file or ErrFileNotFound.
func (r *gormRepository) GetFile(ctx context.Context, userID uuid.UUID) (*ResumeFile, error) {
	var f ResumeFile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// DeleteFile soft-deletes the user's current resume file.
func (r *gormRepository) DeleteFile(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&ResumeFile{}).Error
}
