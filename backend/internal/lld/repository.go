package lld

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

// ProblemFilter constrains an LLD problem listing.
type ProblemFilter struct {
	Difficulty *Difficulty
	Query      string
	Sort       []SortField
}

// Repository abstracts read access to the LLD catalog so the service can be
// tested against a fake. The GORM implementation is gormRepository.
type Repository interface {
	ListProblems(ctx context.Context, f ProblemFilter, p Page) ([]Problem, int64, error)
	GetProblem(ctx context.Context, id uuid.UUID) (*Problem, error)
	GetProblemBySlug(ctx context.Context, slug string) (*Problem, error)
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
// the supplied default when no sort is given.
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

func (r *gormRepository) ListProblems(ctx context.Context, f ProblemFilter, p Page) ([]Problem, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Problem{})
	if f.Difficulty != nil {
		tx = tx.Where("lld_problems.difficulty = ?", *f.Difficulty)
	}
	if f.Query != "" {
		tx = tx.Where("lld_problems.title ILIKE ?", "%"+f.Query+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Problem
	err := applySort(tx, f.Sort, "order_index ASC, title ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) GetProblem(ctx context.Context, id uuid.UUID) (*Problem, error) {
	var prob Problem
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&prob).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &prob, nil
}

func (r *gormRepository) GetProblemBySlug(ctx context.Context, slug string) (*Problem, error) {
	var prob Problem
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&prob).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &prob, nil
}
