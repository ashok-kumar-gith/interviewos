package dsaprogress

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service implements DSA problem progress + solution use-cases over a Repository.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService constructs a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Get returns the caller's progress for a problem (nil when none). The problem
// must exist (404 otherwise).
func (s *Service) Get(ctx context.Context, userID, problemID uuid.UUID) (*Progress, error) {
	ok, err := s.repo.ProblemExists(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrProblemNotFound
	}
	return s.repo.Get(ctx, userID, problemID)
}

// List returns the caller's recorded problem progress (the "solved" log).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Progress, error) {
	return s.repo.List(ctx, userID)
}

// Save validates and upserts solve state + an optional stored solution.
func (s *Service) Save(ctx context.Context, userID, problemID uuid.UUID, in Input) (*Progress, error) {
	ok, err := s.repo.ProblemExists(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrProblemNotFound
	}
	if in.Confidence != nil && (*in.Confidence < 1 || *in.Confidence > 5) {
		return nil, ErrValidation
	}
	if in.TimeSpentMinutes < 0 {
		return nil, ErrValidation
	}
	in.SolutionCode = trimPtr(in.SolutionCode, maxCodeLen)
	in.SolutionLanguage = trimPtr(in.SolutionLanguage, maxLangLen)
	in.SolutionNotes = trimPtr(in.SolutionNotes, maxNotesLen)
	if in.SolutionCode != nil && len(*in.SolutionCode) > maxCodeLen {
		return nil, ErrValidation
	}
	return s.repo.Upsert(ctx, userID, problemID, in, s.now().UTC())
}

// Delete clears the caller's progress on a problem.
func (s *Service) Delete(ctx context.Context, userID, problemID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, problemID)
}

// trimPtr trims a string pointer; returns nil when it trims to empty so an empty
// field doesn't overwrite stored content. Caps to max as a guard.
func trimPtr(p *string, max int) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	if len(v) > max {
		v = v[:max]
	}
	return &v
}
