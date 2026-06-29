package designproblems

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	// defaultPageSize and maxPageSize bound pagination per the OpenAPI contract.
	defaultPageSize = 20
	maxPageSize     = 100
	defaultPage     = 1
)

// Service implements the design-problems read use-cases plus per-user progress.
// It normalizes pagination and delegates to the repositories; seeding lives in
// migrations. The progress repo is optional (nil disables progress endpoints).
type Service struct {
	repo     Repository
	progress ProgressRepository
}

// NewService constructs a read-only Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithProgress constructs a Service that also records per-user progress.
func NewServiceWithProgress(repo Repository, progress ProgressRepository) *Service {
	return &Service{repo: repo, progress: progress}
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

// GetProgress returns the caller's progress on a design problem, or (nil, nil)
// when none recorded yet. The problem must exist (404 otherwise).
func (s *Service) GetProgress(ctx context.Context, userID, dpID uuid.UUID) (*Progress, error) {
	if s.progress == nil {
		return nil, ErrNotFound
	}
	if _, err := s.repo.GetByID(ctx, dpID); err != nil {
		return nil, err
	}
	return s.progress.Get(ctx, userID, dpID)
}

// SaveProgress validates and upserts the caller's progress on a design problem.
func (s *Service) SaveProgress(ctx context.Context, userID, dpID uuid.UUID, in ProgressInput) (*Progress, error) {
	if s.progress == nil {
		return nil, ErrNotFound
	}
	// The design problem must exist (surfaces 404 for a bad id).
	if _, err := s.repo.GetByID(ctx, dpID); err != nil {
		return nil, err
	}
	if _, ok := validProgressStatus[in.Status]; !ok {
		return nil, ErrInvalidProgress
	}
	if in.Confidence != nil && (*in.Confidence < 1 || *in.Confidence > 5) {
		return nil, ErrInvalidProgress
	}
	if in.TimeSpentMinutes < 0 {
		return nil, ErrInvalidProgress
	}
	return s.progress.Upsert(ctx, userID, dpID, in, time.Now().UTC())
}
