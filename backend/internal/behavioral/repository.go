package behavioral

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListFilter narrows and paginates a user's stories.
type ListFilter struct {
	Theme    *Theme // optional theme filter
	Query    string // optional case-insensitive title/content search
	SortDesc bool   // sort by created_at; desc when true (default true)
	Limit    int    // page size (>0)
	Offset   int    // page offset (>=0)
}

// Repository abstracts persistence for the behavioral domain so the service can
// be unit-tested against a fake. The gorm implementation is gormRepository. All
// reads/writes are scoped to a user_id and are soft-delete aware (GORM filters
// deleted_at IS NULL via gorm.DeletedAt on the model).
type Repository interface {
	Create(ctx context.Context, s *Story) error
	GetByID(ctx context.Context, userID, id uuid.UUID) (*Story, error)
	List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Story, int64, error)
	Update(ctx context.Context, s *Story) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, s *Story) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *gormRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Story, error) {
	var s Story
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *gormRepository) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Story, int64, error) {
	q := r.db.WithContext(ctx).Model(&Story{}).Where("user_id = ?", userID)
	if f.Theme != nil {
		q = q.Where("theme = ?", *f.Theme)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where(
			"title ILIKE ? OR situation ILIKE ? OR task ILIKE ? OR action ILIKE ? OR result ILIKE ?",
			like, like, like, like, like,
		)
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

	var out []Story
	if err := q.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *gormRepository) Update(ctx context.Context, s *Story) error {
	// Scope the update to the owner; Save would otherwise update by PK alone.
	res := r.db.WithContext(ctx).Model(&Story{}).
		Where("id = ? AND user_id = ?", s.ID, s.UserID).
		Select("title", "theme", "situation", "task", "action", "result",
			"metrics", "tags", "ai_improved", "ai_feedback", "strength_score").
		Updates(s)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrStoryNotFound
	}
	return nil
}

func (r *gormRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	// Soft delete (gorm.DeletedAt) scoped to the owner.
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&Story{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrStoryNotFound
	}
	return nil
}
