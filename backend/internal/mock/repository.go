package mock

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListFilter narrows and paginates a user's mock interviews.
type ListFilter struct {
	Type     *Type // optional mock_type filter
	SortDesc bool  // sort by created_at; desc when true (default true)
	Limit    int   // page size (>0)
	Offset   int   // page offset (>=0)
}

// Repository abstracts persistence for the mock domain so the service can be
// unit-tested against a fake. The gorm implementation is gormRepository. All
// reads/writes are scoped to a user_id and are soft-delete aware (GORM filters
// deleted_at IS NULL via gorm.DeletedAt on the models).
type Repository interface {
	Create(ctx context.Context, m *Interview) error
	// GetByID returns the mock without its findings.
	GetByID(ctx context.Context, userID, id uuid.UUID) (*Interview, error)
	// GetByIDWithFindings returns the mock with Findings preloaded (detail view).
	GetByIDWithFindings(ctx context.Context, userID, id uuid.UUID) (*Interview, error)
	List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Interview, int64, error)
	Update(ctx context.Context, m *Interview) error
	Delete(ctx context.Context, userID, id uuid.UUID) error

	// AddFinding inserts a finding (the caller has verified mock ownership).
	AddFinding(ctx context.Context, f *Finding) error
	// ListFindings returns all of a user's findings (across mocks), newest first.
	// Used by the weakness aggregator.
	ListFindings(ctx context.Context, userID uuid.UUID) ([]Finding, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, m *Interview) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *gormRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Interview, error) {
	var m Interview
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMockNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormRepository) GetByIDWithFindings(ctx context.Context, userID, id uuid.UUID) (*Interview, error) {
	var m Interview
	err := r.db.WithContext(ctx).
		Preload("Findings", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("id = ? AND user_id = ?", id, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMockNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormRepository) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Interview, int64, error) {
	q := r.db.WithContext(ctx).Model(&Interview{}).Where("user_id = ?", userID)
	if f.Type != nil {
		q = q.Where("type = ?", *f.Type)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "created_at DESC"
	if !f.SortDesc {
		order = "created_at ASC"
	}
	q = q.Order(order)
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var out []Interview
	if err := q.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *gormRepository) Update(ctx context.Context, m *Interview) error {
	// Scope the update to the owner; Save would otherwise update by PK alone.
	res := r.db.WithContext(ctx).Model(&Interview{}).
		Where("id = ? AND user_id = ?", m.ID, m.UserID).
		Select("type", "topic_id", "design_problem_id", "company_id",
			"scheduled_at", "conducted_at", "duration_minutes", "outcome",
			"overall_score", "interviewer", "transcript_md", "summary").
		Updates(m)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMockNotFound
	}
	return nil
}

func (r *gormRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	// Soft delete (gorm.DeletedAt) scoped to the owner.
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&Interview{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMockNotFound
	}
	return nil
}

func (r *gormRepository) AddFinding(ctx context.Context, f *Finding) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *gormRepository) ListFindings(ctx context.Context, userID uuid.UUID) ([]Finding, error) {
	var out []Finding
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
