// Package backendeng owns the Backend Engineering pillar depth topics (maps to
// enum backend_engineering). Browsing is public and read-only.
//
// The Backend Engineering catalog is just the backend_engineering slice of the
// shared topics table, so this module is intentionally thin: it reuses
// internal/content's Repository (the single source of read logic) and only fixes
// the pillar filter to backend_engineering, exposing the dedicated
// GET /backend-engineering/topics path from the OpenAPI contract. No read logic
// is duplicated.
package backendeng

import (
	"context"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/content"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	defaultPage     = 1

	// pillarBackendEng is the fixed pillar this module scopes every query to.
	pillarBackendEng = content.PillarBackendEng
)

// Service implements the Backend Engineering read use-cases. It normalizes
// pagination and delegates to the content repository with the pillar pinned to
// backend_engineering.
type Service struct {
	repo content.Repository
}

// NewService constructs a Service over the shared content repository.
func NewService(repo content.Repository) *Service {
	return &Service{repo: repo}
}

// TopicQuery is the validated browse input for backend-engineering topics.
type TopicQuery struct {
	Difficulty *content.Difficulty
	Query      string
	Sort       []content.SortField
	Page       int
	PageSize   int
}

// ListResult is a paginated topic result envelope.
type ListResult struct {
	Items      []content.Topic
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// ListTopics lists backend_engineering topics with optional difficulty filter,
// sort, and search.
func (s *Service) ListTopics(ctx context.Context, q TopicQuery) (ListResult, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	pillar := pillarBackendEng
	f := content.TopicFilter{
		PillarType: &pillar,
		Difficulty: q.Difficulty,
		Query:      q.Query,
		Sort:       q.Sort,
	}
	items, total, err := s.repo.ListTopics(ctx, f, content.Page{Page: page, PageSize: pageSize})
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages(total, pageSize),
	}, nil
}

// GetTopic returns a single backend_engineering topic bundle (with subtopics and
// resources) by id. It returns content.ErrNotFound when the id is unknown or the
// topic belongs to a different pillar.
func (s *Service) GetTopic(ctx context.Context, id uuid.UUID) (*content.TopicBundle, error) {
	b, err := s.repo.GetTopicBundle(ctx, id)
	if err != nil {
		return nil, err
	}
	// Guard: only serve backend_engineering topics from this module.
	pillar := pillarBackendEng
	owned, _, err := s.repo.ListTopics(ctx, content.TopicFilter{PillarType: &pillar}, content.Page{Page: 1, PageSize: maxPageSize})
	if err != nil {
		return nil, err
	}
	for _, t := range owned {
		if t.ID == id {
			return b, nil
		}
	}
	return nil, content.ErrNotFound
}

// normalizePage clamps page/page_size to the allowed bounds with sane defaults.
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
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
