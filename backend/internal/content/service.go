package content

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

// Service implements the content read use-cases. It is intentionally thin: it
// normalizes pagination, then delegates to the repository. Dedup/write logic
// lives in the seeder, not here.
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

// ListResult is a generic paginated result envelope.
type ListResult[T any] struct {
	Items      []T
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

func makeResult[T any](items []T, total int64, p Page) ListResult[T] {
	return ListResult[T]{
		Items:      items,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: totalPages(total, p.PageSize),
	}
}

// ListTracks lists tracks with optional text search.
func (s *Service) ListTracks(ctx context.Context, q string, sort []SortField, page, pageSize int) (ListResult[Track], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListTracks(ctx, q, sort, p)
	if err != nil {
		return ListResult[Track]{}, err
	}
	return makeResult(items, total, p), nil
}

// ListPillars lists pillars optionally scoped to a track.
func (s *Service) ListPillars(ctx context.Context, trackID *uuid.UUID, sort []SortField, page, pageSize int) (ListResult[Pillar], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListPillars(ctx, trackID, sort, p)
	if err != nil {
		return ListResult[Pillar]{}, err
	}
	return makeResult(items, total, p), nil
}

// ListTopics lists topics with filtering, sorting, and search.
func (s *Service) ListTopics(ctx context.Context, f TopicFilter, page, pageSize int) (ListResult[Topic], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListTopics(ctx, f, p)
	if err != nil {
		return ListResult[Topic]{}, err
	}
	return makeResult(items, total, p), nil
}

// GetTopic returns a topic with subtopics and resources.
func (s *Service) GetTopic(ctx context.Context, id uuid.UUID) (*TopicBundle, error) {
	return s.repo.GetTopicBundle(ctx, id)
}

// ListResources lists resources with filtering and search.
func (s *Service) ListResources(ctx context.Context, f ResourceFilter, page, pageSize int) (ListResult[Resource], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListResources(ctx, f, p)
	if err != nil {
		return ListResult[Resource]{}, err
	}
	return makeResult(items, total, p), nil
}

// ListPatterns lists DSA patterns optionally scoped to a track.
func (s *Service) ListPatterns(ctx context.Context, trackID *uuid.UUID, q string, sort []SortField, page, pageSize int) (ListResult[Pattern], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListPatterns(ctx, trackID, q, sort, p)
	if err != nil {
		return ListResult[Pattern]{}, err
	}
	return makeResult(items, total, p), nil
}

// ListProblems lists DSA problems with filtering and search.
func (s *Service) ListProblems(ctx context.Context, f ProblemFilter, page, pageSize int) (ListResult[Problem], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListProblems(ctx, f, p)
	if err != nil {
		return ListResult[Problem]{}, err
	}
	return makeResult(items, total, p), nil
}

// GetProblem returns a problem with patterns, sources, and company frequency.
func (s *Service) GetProblem(ctx context.Context, id uuid.UUID) (*ProblemBundle, error) {
	return s.repo.GetProblemBundle(ctx, id)
}

// ListCompanies lists companies with optional text search.
func (s *Service) ListCompanies(ctx context.Context, q string, sort []SortField, page, pageSize int) (ListResult[Company], error) {
	p := normalizePage(page, pageSize)
	items, total, err := s.repo.ListCompanies(ctx, q, sort, p)
	if err != nil {
		return ListResult[Company]{}, err
	}
	return makeResult(items, total, p), nil
}

// GetCompany returns a company with its weights.
func (s *Service) GetCompany(ctx context.Context, id uuid.UUID) (*CompanyBundle, error) {
	return s.repo.GetCompanyBundle(ctx, id)
}
