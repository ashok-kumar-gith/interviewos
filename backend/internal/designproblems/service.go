package designproblems

import (
	"context"

	"github.com/google/uuid"
)

const (
	// defaultPageSize and maxPageSize bound pagination per the OpenAPI contract.
	defaultPageSize = 20
	maxPageSize     = 100
	defaultPage     = 1
)

// Service implements the design-problems read use-cases. It normalizes
// pagination and delegates to the repository; seeding lives in migrations.
type Service struct {
	repo Repository
}

// NewService constructs a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// normalizePage clamps page/page_size to the allowed bounds with sane defaults.
func normalizePage(page, pageSize int) Page {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return Page{Page: page, PageSize: pageSize}
}

// totalPages computes the page count for a total and page size.
func totalPages(total int64, pageSize int) int {
	if pageSize <= 0 || total <= 0 {
		return 0
	}
	pages := int(total / int64(pageSize))
	if total%int64(pageSize) != 0 {
		pages++
	}
	return pages
}

// ListResult is the paginated result envelope.
type ListResult struct {
	Items      []DesignProblem
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// List lists design problems with filtering, sorting, and search.
func (s *Service) List(ctx context.Context, f Filter, page, pageSize int) (ListResult, error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.List(ctx, f, p)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{
		Items:      items,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages(total, p.PageSize),
	}, nil
}

// Get returns a single design problem with all its sections.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*DesignProblem, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySlug returns a single design problem by its slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*DesignProblem, error) {
	return s.repo.GetBySlug(ctx, slug)
}
