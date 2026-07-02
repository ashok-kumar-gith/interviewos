package content

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// AdminService implements the admin content-authoring use-cases (create/update/
// delete of DSA problems and topics). It validates input, resolves human-
// friendly references (pillar type, default track), and delegates persistence to
// the WriteRepository. It depends only on interfaces so it is unit-testable
// against a fake.
type AdminService struct {
	repo WriteRepository
}

// NewAdminService constructs an AdminService.
func NewAdminService(repo WriteRepository) *AdminService {
	return &AdminService{repo: repo}
}

// ProblemInput is the validated create/update payload for a DSA problem. TrackID
// and TopicID are optional (TrackID defaults to the primary track). Slug/Title/
// Difficulty are required.
type ProblemInput struct {
	TrackID          *uuid.UUID
	TopicID          *uuid.UUID
	Slug             string
	Title            string
	Difficulty       string
	Platform         string
	ExternalID       *string
	URL              *string
	PromptSummary    *string
	ApproachMD       *string
	CommonMistakes   *string
	EstimatedMinutes *int
	FrequencyScore   *float64
	IsPremium        *bool
	PatternSlugs     []string
	Sources          []string
	CompanyFrequency []CompanyFrequencyInput
}

// TopicInput is the validated create/update payload for a topic. Exactly one of
// PillarID/PillarType identifies the pillar. Name/Slug are required.
type TopicInput struct {
	PillarID          *uuid.UUID
	PillarType        *PillarType
	TrackID           *uuid.UUID
	Slug              string
	Name              string
	Summary           *string
	ConceptMD         *string
	Difficulty        string
	Priority          string
	EstimatedHours    *float64
	CommonMistakes    *string
	ExpectedQuestions []string
	Prerequisites     []string
	SortOrder         *int
}

func validDifficulty(s string) bool {
	switch Difficulty(s) {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	}
	return false
}

func validPriority(s string) bool {
	switch s {
	case "low", "medium", "high", "critical":
		return true
	}
	return false
}

func validPlatform(s string) bool {
	switch s {
	case "leetcode", "hackerrank", "codeforces", "interviewbit", "gfg", "custom":
		return true
	}
	return false
}

func validSource(s string) bool {
	switch s {
	case "blind75", "neetcode150", "grind75", "tech_interview_handbook", "leetcode_top", "striver_sde", "custom":
		return true
	}
	return false
}

// buildProblemWrite validates a ProblemInput and resolves it into a ProblemWrite.
func (s *AdminService) buildProblemWrite(ctx context.Context, in ProblemInput) (ProblemWrite, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Title = strings.TrimSpace(in.Title)
	if in.Slug == "" || in.Title == "" {
		return ProblemWrite{}, ErrValidation
	}
	if !validDifficulty(in.Difficulty) {
		return ProblemWrite{}, ErrValidation
	}
	platform := in.Platform
	if platform == "" {
		platform = "leetcode"
	}
	if !validPlatform(platform) {
		return ProblemWrite{}, ErrValidation
	}
	for _, src := range in.Sources {
		if !validSource(src) {
			return ProblemWrite{}, ErrValidation
		}
	}

	trackID := uuid.Nil
	if in.TrackID != nil {
		trackID = *in.TrackID
	} else {
		id, err := s.repo.DefaultTrackID(ctx)
		if err != nil {
			return ProblemWrite{}, err
		}
		trackID = id
	}

	prob := Problem{
		TrackID:        trackID,
		TopicID:        in.TopicID,
		Slug:           in.Slug,
		Title:          in.Title,
		Difficulty:     Difficulty(in.Difficulty),
		Platform:       ProblemPlatform(platform),
		ExternalID:     in.ExternalID,
		URL:            in.URL,
		PromptSummary:  in.PromptSummary,
		ApproachMD:     in.ApproachMD,
		CommonMistakes: in.CommonMistakes,
	}
	prob.EstimatedMinutes = 30
	if in.EstimatedMinutes != nil {
		if *in.EstimatedMinutes < 0 {
			return ProblemWrite{}, ErrValidation
		}
		prob.EstimatedMinutes = *in.EstimatedMinutes
	}
	if in.FrequencyScore != nil {
		if *in.FrequencyScore < 0 {
			return ProblemWrite{}, ErrValidation
		}
		prob.FrequencyScore = *in.FrequencyScore
	}
	if in.IsPremium != nil {
		prob.IsPremium = *in.IsPremium
	}

	sources := make([]ProblemSourceName, 0, len(in.Sources))
	for _, src := range in.Sources {
		sources = append(sources, ProblemSourceName(src))
	}
	return ProblemWrite{
		Problem:          prob,
		PatternSlugs:     in.PatternSlugs,
		Sources:          sources,
		CompanyFrequency: in.CompanyFrequency,
	}, nil
}

// CreateProblem validates and creates a DSA problem with its associations.
func (s *AdminService) CreateProblem(ctx context.Context, in ProblemInput) (*ProblemBundle, error) {
	w, err := s.buildProblemWrite(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ProblemSlugExists(ctx, w.Problem.Slug, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.CreateProblem(ctx, w)
}

// UpdateProblem validates and updates a DSA problem, re-setting its associations.
func (s *AdminService) UpdateProblem(ctx context.Context, id uuid.UUID, in ProblemInput) (*ProblemBundle, error) {
	w, err := s.buildProblemWrite(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.ProblemSlugExists(ctx, w.Problem.Slug, &id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.UpdateProblem(ctx, id, w)
}

// DeleteProblem soft-deletes a DSA problem.
func (s *AdminService) DeleteProblem(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProblem(ctx, id)
}

// buildTopicWrite validates a TopicInput and resolves it into a TopicWrite.
func (s *AdminService) buildTopicWrite(ctx context.Context, in TopicInput) (TopicWrite, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	if in.Slug == "" || in.Name == "" {
		return TopicWrite{}, ErrValidation
	}
	if in.PillarID == nil && in.PillarType == nil {
		return TopicWrite{}, ErrValidation
	}
	difficulty := in.Difficulty
	if difficulty == "" {
		difficulty = string(DifficultyMedium)
	}
	if !validDifficulty(difficulty) {
		return TopicWrite{}, ErrValidation
	}
	priority := in.Priority
	if priority == "" {
		priority = "medium"
	}
	if !validPriority(priority) {
		return TopicWrite{}, ErrValidation
	}

	pillar, err := s.repo.ResolvePillar(ctx, in.PillarID, in.PillarType)
	if err != nil {
		return TopicWrite{}, err
	}
	// Topic's track is derived from the pillar unless explicitly overridden.
	trackID := pillar.TrackID
	if in.TrackID != nil {
		trackID = *in.TrackID
	}

	t := Topic{
		PillarID:          pillar.ID,
		TrackID:           trackID,
		Slug:              in.Slug,
		Name:              in.Name,
		Summary:           in.Summary,
		ConceptMD:         in.ConceptMD,
		Difficulty:        Difficulty(difficulty),
		Priority:          Priority(priority),
		EstimatedHours:    2.0,
		CommonMistakes:    in.CommonMistakes,
		ExpectedQuestions: JSONArray(in.ExpectedQuestions),
		Prerequisites:     JSONArray(in.Prerequisites),
	}
	if in.EstimatedHours != nil {
		if *in.EstimatedHours < 0 {
			return TopicWrite{}, ErrValidation
		}
		t.EstimatedHours = *in.EstimatedHours
	}
	if in.SortOrder != nil {
		t.SortOrder = *in.SortOrder
	}
	if t.ExpectedQuestions == nil {
		t.ExpectedQuestions = JSONArray{}
	}
	if t.Prerequisites == nil {
		t.Prerequisites = JSONArray{}
	}
	return TopicWrite{Topic: t}, nil
}

// CreateTopic validates and creates a topic under the resolved pillar.
func (s *AdminService) CreateTopic(ctx context.Context, in TopicInput) (*TopicBundle, error) {
	w, err := s.buildTopicWrite(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.TopicSlugExists(ctx, w.Topic.Slug, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.CreateTopic(ctx, w)
}

// UpdateTopic validates and updates a topic.
func (s *AdminService) UpdateTopic(ctx context.Context, id uuid.UUID, in TopicInput) (*TopicBundle, error) {
	w, err := s.buildTopicWrite(ctx, in)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.TopicSlugExists(ctx, w.Topic.Slug, &id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	return s.repo.UpdateTopic(ctx, id, w)
}

// DeleteTopic soft-deletes a topic.
func (s *AdminService) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTopic(ctx, id)
}
