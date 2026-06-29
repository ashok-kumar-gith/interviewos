package company

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service implements Company Mode: reading and setting the user's target
// company. Persisting target_company_id is sufficient to re-weight future
// roadmap generation, since the curriculum engine reads it (FR-CMP-002/005).
type Service struct {
	repo Repository
	now  func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo Repository
	Now  func() time.Time
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Service{repo: cfg.Repo, now: nowFn}
}

// GetTarget returns the user's profile (carrying the current target company).
func (s *Service) GetTarget(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.GetProfile(ctx, userID)
}

// SetTarget validates the company exists and sets it as the user's target,
// returning the updated profile. The roadmap re-weights on next generation.
func (s *Service) SetTarget(ctx context.Context, userID, companyID uuid.UUID) (*Profile, error) {
	exists, err := s.repo.CompanyExists(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCompanyNotFound
	}
	return s.repo.SetTargetCompany(ctx, userID, &companyID, s.now().UTC())
}
