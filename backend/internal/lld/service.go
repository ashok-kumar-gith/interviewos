package lld

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

// Service implements the LLD read use-cases. It normalizes pagination and
// delegates to the repository; seed/write logic lives in the migration seed.
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

// ListResult is a paginated result envelope.
type ListResult struct {
	Items      []Problem
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// ListProblems lists LLD problems with optional difficulty filter, sort, and search.
func (s *Service) ListProblems(ctx context.Context, f ProblemFilter, page, pageSize int) (ListResult, error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListProblems(ctx, f, p)
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

// GetProblem returns a single LLD problem (all sections) by id.
func (s *Service) GetProblem(ctx context.Context, id uuid.UUID) (*Problem, error) {
	return s.repo.GetProblem(ctx, id)
}

// GetProblemBySlug returns a single LLD problem (all sections) by slug.
func (s *Service) GetProblemBySlug(ctx context.Context, slug string) (*Problem, error) {
	return s.repo.GetProblemBySlug(ctx, slug)
}
