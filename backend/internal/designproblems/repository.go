package designproblems

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Page describes pagination input (1-based page, bounded size).
type Page struct {
	Page     int
	PageSize int
}

// Offset returns the SQL OFFSET for the page.
func (p Page) Offset() int { return (p.Page - 1) * p.PageSize }

// SortField is a single normalized ordering instruction.
type SortField struct {
	Column string
	Desc   bool
}

// Filter constrains a design-problem listing.
type Filter struct {
	TrackID    *uuid.UUID
	Difficulty *Difficulty
	Query      string
	Sort       []SortField
}

// Repository abstracts read access to the design-problems catalog so the service
// can be tested against a fake. The GORM implementation is gormRepository.
type Repository interface {
	List(ctx context.Context, f Filter, p Page) ([]DesignProblem, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*DesignProblem, error)
	GetBySlug(ctx context.Context, slug string) (*DesignProblem, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// applySort appends ORDER BY clauses from a validated sort list, falling back to
// the supplied default ordering when no sort is given.
func applySort(q *gorm.DB, sort []SortField, def string) *gorm.DB {
	if len(sort) == 0 {
		return q.Order(def)
	}
	for _, s := range sort {
		dir := " ASC"
		if s.Desc {
			dir = " DESC"
		}
		q = q.Order(s.Column + dir)
	}
	return q
}

func (r *gormRepository) List(ctx context.Context, f Filter, p Page) ([]DesignProblem, int64, error) {
	tx := r.db.WithContext(ctx).Model(&DesignProblem{})
	if f.TrackID != nil {
		tx = tx.Where("track_id = ?", *f.TrackID)
	}
	if f.Difficulty != nil {
		tx = tx.Where("difficulty = ?", *f.Difficulty)
	}
	if f.Query != "" {
		tx = tx.Where("title ILIKE ?", "%"+f.Query+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []DesignProblem
	err := applySort(tx, f.Sort, "order_index ASC, title ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) GetByID(ctx context.Context, id uuid.UUID) (*DesignProblem, error) {
	var dp DesignProblem
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&dp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &dp, nil
}

func (r *gormRepository) GetBySlug(ctx context.Context, slug string) (*DesignProblem, error) {
	var dp DesignProblem
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&dp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &dp, nil
}
