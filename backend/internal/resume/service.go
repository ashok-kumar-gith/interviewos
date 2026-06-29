package resume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service implements the resume use-cases. It depends only on interfaces
// (Repository, Scorer) so it is unit-testable with fakes.
type Service struct {
	repo   Repository
	scorer Scorer
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo   Repository
	Scorer Scorer
}

// NewService constructs a Service. A nil Scorer defaults to the deterministic
// RuleScorer.
func NewService(cfg ServiceConfig) *Service {
	sc := cfg.Scorer
	if sc == nil {
		sc = NewRuleScorer()
	}
	return &Service{repo: cfg.Repo, scorer: sc}
}

// ProfileInput is the validated upsert payload for a resume profile.
type ProfileInput struct {
	Headline        *string
	Summary         *string
	YearsExperience *float64
	Skills          []string
	TargetKeywords  []string
}

// ProjectInput is the validated upsert payload for a resume project.
type ProjectInput struct {
	Name        string
	Role        *string
	Description *string
	Impact      *string
	Metrics     []string
	TechStack   []string
	StartDate   *string // ISO date (YYYY-MM-DD)
	EndDate     *string
	SortOrder   int
}

// GetProfile returns the user's resume profile (with projects) or ErrProfileNotFound.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
}

// UpsertProfile creates or updates the single resume profile for a user.
func (s *Service) UpsertProfile(ctx context.Context, userID uuid.UUID, in ProfileInput) (*Profile, error) {
	if err := validateProfile(in); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetProfileByUserID(ctx, userID)
	switch {
	case err == nil:
		existing.Headline = in.Headline
		existing.Summary = in.Summary
		existing.YearsExperience = in.YearsExperience
		existing.Skills = StringArray(in.Skills)
		existing.TargetKeywords = StringArray(in.TargetKeywords)
		if uerr := s.repo.UpdateProfile(ctx, existing); uerr != nil {
			return nil, uerr
		}
		return s.repo.GetProfileByUserID(ctx, userID)
	case isNotFound(err):
		p := &Profile{
			ID:              uuid.New(),
			UserID:          userID,
			Headline:        in.Headline,
			Summary:         in.Summary,
			YearsExperience: in.YearsExperience,
			Skills:          StringArray(in.Skills),
			TargetKeywords:  StringArray(in.TargetKeywords),
		}
		if cerr := s.repo.CreateProfile(ctx, p); cerr != nil {
			return nil, cerr
		}
		return s.repo.GetProfileByUserID(ctx, userID)
	default:
		return nil, err
	}
}

// ListProjects returns the user's projects (ownership enforced via the profile).
func (s *Service) ListProjects(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListProjects(ctx, profile.ID)
}

// CreateProject adds a project to the user's profile.
func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, in ProjectInput) (*Project, error) {
	if err := validateProject(in); err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	p, err := buildProject(in)
	if err != nil {
		return nil, err
	}
	p.ID = uuid.New()
	p.ResumeProfileID = profile.ID
	p.UserID = userID
	if cerr := s.repo.CreateProject(ctx, p); cerr != nil {
		return nil, cerr
	}
	return s.repo.GetProject(ctx, userID, p.ID)
}

// UpdateProject updates a project the user owns.
func (s *Service) UpdateProject(ctx context.Context, userID, projectID uuid.UUID, in ProjectInput) (*Project, error) {
	if err := validateProject(in); err != nil {
		return nil, err
	}
	// Ownership check: GetProject is scoped by user_id.
	existing, err := s.repo.GetProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	updated, err := buildProject(in)
	if err != nil {
		return nil, err
	}
	updated.ID = existing.ID
	updated.ResumeProfileID = existing.ResumeProfileID
	updated.UserID = userID
	if uerr := s.repo.UpdateProject(ctx, updated); uerr != nil {
		return nil, uerr
	}
	return s.repo.GetProject(ctx, userID, projectID)
}

// DeleteProject soft-deletes a project the user owns.
func (s *Service) DeleteProject(ctx context.Context, userID, projectID uuid.UUID) error {
	// Ownership check first so a missing/foreign project returns 404.
	if _, err := s.repo.GetProject(ctx, userID, projectID); err != nil {
		return err
	}
	return s.repo.DeleteProject(ctx, userID, projectID)
}

// Score runs the deterministic ATS scorer over the user's profile + projects,
// persists the resulting ats_score / ai_feedback, and returns the result.
func (s *Service) Score(ctx context.Context, userID uuid.UUID) (ScoreResult, error) {
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return ScoreResult{}, err
	}
	projects := profile.Projects
	if len(projects) == 0 {
		// Preload may be empty if the caller fetched without projects; reload.
		projects, err = s.repo.ListProjects(ctx, profile.ID)
		if err != nil {
			return ScoreResult{}, err
		}
	}

	in := buildScoreInput(profile, projects)
	result, err := s.scorer.Score(ctx, in)
	if err != nil {
		return ScoreResult{}, err
	}

	feedback, _ := json.Marshal(result)
	if perr := s.repo.UpdateScore(ctx, profile.ID, result.ATSScore, feedback); perr != nil {
		return ScoreResult{}, perr
	}
	return result, nil
}

// --- builders & validation ---

func buildScoreInput(p *Profile, projects []Project) ScoreInput {
	in := ScoreInput{
		Skills:         []string(p.Skills),
		TargetKeywords: []string(p.TargetKeywords),
		Projects:       len(projects),
	}
	if p.Headline != nil {
		in.Headline = *p.Headline
	}
	if p.Summary != nil {
		in.Summary = *p.Summary
	}
	for _, proj := range projects {
		if proj.Description != nil && strings.TrimSpace(*proj.Description) != "" {
			in.Bullets = append(in.Bullets, *proj.Description)
		}
		if proj.Impact != nil && strings.TrimSpace(*proj.Impact) != "" {
			in.Bullets = append(in.Bullets, *proj.Impact)
		}
		for _, m := range proj.Metrics {
			if strings.TrimSpace(m) != "" {
				in.Bullets = append(in.Bullets, m)
			}
		}
	}
	return in
}

func buildProject(in ProjectInput) (*Project, error) {
	p := &Project{
		Name:        strings.TrimSpace(in.Name),
		Role:        in.Role,
		Description: in.Description,
		Impact:      in.Impact,
		Metrics:     StringArray(in.Metrics),
		TechStack:   StringArray(in.TechStack),
		SortOrder:   in.SortOrder,
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w: start_date must be YYYY-MM-DD", ErrValidation)
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w: end_date must be YYYY-MM-DD", ErrValidation)
	}
	p.StartDate = start
	p.EndDate = end
	return p, nil
}

func validateProfile(in ProfileInput) error {
	if in.YearsExperience != nil && *in.YearsExperience < 0 {
		return fmt.Errorf("%w: years_experience must be >= 0", ErrValidation)
	}
	if in.Headline != nil && len(*in.Headline) > 300 {
		return fmt.Errorf("%w: headline too long (max 300)", ErrValidation)
	}
	if in.Summary != nil && len(*in.Summary) > 5000 {
		return fmt.Errorf("%w: summary too long (max 5000)", ErrValidation)
	}
	return nil
}

func validateProject(in ProjectInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(in.Name) > 300 {
		return fmt.Errorf("%w: name too long (max 300)", ErrValidation)
	}
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, ErrProfileNotFound)
}

// parseDate parses an optional ISO date (YYYY-MM-DD). A nil or empty pointer
// yields a nil time (no date set).
func parseDate(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, err
	}
	return &t, nil
}
