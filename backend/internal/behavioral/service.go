package behavioral

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// CreateInput is the validated payload for creating a story.
type CreateInput struct {
	Title     string
	Theme     Theme
	Situation string
	Task      string
	Action    string
	Result    string
	Metrics   string
	Tags      []string
}

// UpdateInput is the validated payload for replacing a story's mutable fields
// (PUT semantics: all upsert fields are set from the request).
type UpdateInput = CreateInput

// FieldError is a single field-level validation detail surfaced to the handler.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError carries field-level details and wraps ErrValidation so callers
// can branch with errors.Is.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return ErrValidation.Error() }
func (e *ValidationError) Unwrap() error { return ErrValidation }

// Service implements the behavioral use-cases. It depends only on interfaces so
// it is unit-testable with fakes.
type Service struct {
	repo     Repository
	improver Improver
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo     Repository
	Improver Improver
}

// NewService constructs a Service. If no Improver is supplied it defaults to the
// deterministic stub so the improve endpoint is always available.
func NewService(cfg ServiceConfig) *Service {
	imp := cfg.Improver
	if imp == nil {
		imp = NewDeterministicImprover()
	}
	return &Service{repo: cfg.Repo, improver: imp}
}

// maxLen caps free-text fields to keep payloads sane and the DB tidy.
const (
	maxTitleLen   = 200
	maxSectionLen = 5000
	maxTags       = 30
	maxTagLen     = 50
)

// Create validates and persists a new story owned by userID.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (*Story, error) {
	if err := validate(in); err != nil {
		return nil, err
	}
	story := &Story{
		UserID: userID,
		Title:  strings.TrimSpace(in.Title),
		Theme:  in.Theme,
		Tags:   normalizeTags(in.Tags),
	}
	applyText(story, in)
	if err := s.repo.Create(ctx, story); err != nil {
		return nil, err
	}
	return story, nil
}

// Get returns a single story owned by userID, or ErrStoryNotFound.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Story, error) {
	return s.repo.GetByID(ctx, userID, id)
}

// List returns a page of the user's stories plus the total count.
func (s *Service) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Story, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(ctx, userID, f)
}

// Update validates and replaces the mutable fields of a story owned by userID.
// Editing the STAR content invalidates any prior AI feedback/score.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (*Story, error) {
	if err := validate(in); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	existing.Title = strings.TrimSpace(in.Title)
	existing.Theme = in.Theme
	existing.Tags = normalizeTags(in.Tags)
	applyText(existing, in)
	// Content changed: stale AI feedback no longer reflects the story.
	existing.AIImproved = false
	existing.AIFeedback = nil
	existing.StrengthScore = nil

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Delete soft-deletes a story owned by userID.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

// Improve runs the AI improver against a stored story (ownership enforced),
// persists the resulting feedback/score on the story, and returns the result.
func (s *Service) Improve(ctx context.Context, userID, id uuid.UUID) (*Story, *ImproveResult, error) {
	story, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	res, err := s.improver.Improve(ctx, storyToImproveInput(story))
	if err != nil {
		return nil, nil, err
	}

	story.AIImproved = true
	score := res.StrengthScore
	story.StrengthScore = &score
	story.AIFeedback = improveResultToJSON(res)
	if err := s.repo.Update(ctx, story); err != nil {
		return nil, nil, err
	}
	return story, res, nil
}

// ImproveInline runs the improver against unsaved STAR text (no persistence).
// Used by POST /ai/story-improve when called with inline content rather than a
// stored story id.
func (s *Service) ImproveInline(ctx context.Context, in ImproveInput) (*ImproveResult, error) {
	return s.improver.Improve(ctx, in)
}

// --- helpers ---

func validate(in CreateInput) error {
	var fields []FieldError
	title := strings.TrimSpace(in.Title)
	if title == "" {
		fields = append(fields, FieldError{Field: "title", Message: "is required"})
	} else if len(title) > maxTitleLen {
		fields = append(fields, FieldError{Field: "title", Message: "must be at most 200 characters"})
	}
	if in.Theme == "" {
		fields = append(fields, FieldError{Field: "theme", Message: "is required"})
	} else if !in.Theme.Valid() {
		fields = append(fields, FieldError{Field: "theme", Message: "must be a valid story theme"})
	}
	for name, v := range map[string]string{
		"situation": in.Situation,
		"task":      in.Task,
		"action":    in.Action,
		"result":    in.Result,
		"metrics":   in.Metrics,
	} {
		if len(v) > maxSectionLen {
			fields = append(fields, FieldError{Field: name, Message: "must be at most 5000 characters"})
		}
	}
	if len(in.Tags) > maxTags {
		fields = append(fields, FieldError{Field: "tags", Message: "must have at most 30 entries"})
	}
	for _, t := range in.Tags {
		if len(t) > maxTagLen {
			fields = append(fields, FieldError{Field: "tags", Message: "each tag must be at most 50 characters"})
			break
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// applyText sets the nullable STAR/metrics fields, storing trimmed non-empty
// values and NULL for empty ones.
func applyText(s *Story, in CreateInput) {
	s.Situation = strOrNil(in.Situation)
	s.Task = strOrNil(in.Task)
	s.Action = strOrNil(in.Action)
	s.Result = strOrNil(in.Result)
	s.Metrics = strOrNil(in.Metrics)
}

func strOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// normalizeTags trims, drops empties, and de-duplicates tags (stable order).
func normalizeTags(in []string) Tags {
	seen := map[string]struct{}{}
	out := Tags{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func storyToImproveInput(s *Story) ImproveInput {
	return ImproveInput{
		Title:     s.Title,
		Theme:     s.Theme,
		Situation: deref(s.Situation),
		Task:      deref(s.Task),
		Action:    deref(s.Action),
		Result:    deref(s.Result),
		Metrics:   deref(s.Metrics),
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func improveResultToJSON(r *ImproveResult) JSONMap {
	improved := map[string]any{}
	if r.Improved.Situation != "" {
		improved["situation"] = r.Improved.Situation
	}
	if r.Improved.Task != "" {
		improved["task"] = r.Improved.Task
	}
	if r.Improved.Action != "" {
		improved["action"] = r.Improved.Action
	}
	if r.Improved.Result != "" {
		improved["result"] = r.Improved.Result
	}
	if r.Improved.Metrics != "" {
		improved["metrics"] = r.Improved.Metrics
	}
	suggestions := make([]any, len(r.Suggestions))
	for i, s := range r.Suggestions {
		suggestions[i] = s
	}
	return JSONMap{
		"improved":       improved,
		"suggestions":    suggestions,
		"strength_score": r.StrengthScore,
		"used_fallback":  r.UsedFallback,
	}
}
