package mock

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateInput is the validated payload for creating a mock interview.
type CreateInput struct {
	Type            Type
	TopicID         *uuid.UUID
	DesignProblemID *uuid.UUID
	CompanyID       *uuid.UUID
	ScheduledAt     *time.Time
	ConductedAt     *time.Time
	DurationMinutes *int
	Outcome         Outcome
	OverallScore    *float64
	Interviewer     string
	TranscriptMD    string
	Summary         string
}

// UpdateInput is the validated payload for replacing a mock's mutable fields
// (PUT semantics: all upsert fields are set from the request).
type UpdateInput = CreateInput

// FindingInput is the validated payload for adding a finding to a mock.
type FindingInput struct {
	PillarType            *Pillar
	TopicID               *uuid.UUID
	Severity              Severity
	Category              string
	Detail                string
	CreateRemediationTask bool
}

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

// RemediationRequest is the input to scheduling a follow-up study task from a
// mock finding. The planner places it on the user's next available plan day.
type RemediationRequest struct {
	UserID     uuid.UUID
	PillarType string     // e.g. "dsa" (falls back to a generic study task when empty)
	TopicID    *uuid.UUID // optional content item to study
	Title      string
	Detail     string
}

// RemediationPlanner schedules a remediation study task on the user's plan in
// response to a mock weakness. It is an optional cross-module port (satisfied by
// the progress/roadmap layer) and is nil-safe: when absent, findings are still
// recorded, just without an auto-scheduled follow-up task. Returns the created
// task id so it can be linked back to the finding.
type RemediationPlanner interface {
	ScheduleRemediation(ctx context.Context, req RemediationRequest) (uuid.UUID, error)
}

// Service implements the mock interview use-cases. It depends only on interfaces
// so it is unit-testable with fakes.
type Service struct {
	repo      Repository
	detector  WeaknessDetector
	remediate RemediationPlanner
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo     Repository
	Detector WeaknessDetector
	// Remediation is the optional planner used to schedule a follow-up study
	// task when a finding sets create_remediation_task. nil disables scheduling.
	Remediation RemediationPlanner
}

// NewService constructs a Service. If no WeaknessDetector is supplied it
// defaults to the deterministic implementation so the weaknesses endpoint is
// always available.
func NewService(cfg ServiceConfig) *Service {
	det := cfg.Detector
	if det == nil {
		det = NewDeterministicWeaknessDetector()
	}
	return &Service{repo: cfg.Repo, detector: det, remediate: cfg.Remediation}
}

const (
	maxInterviewerLen = 200
	maxCategoryLen    = 200
	maxDetailLen      = 5000
	maxSummaryLen     = 10000
	maxTranscriptLen  = 200000
	maxDurationMin    = 1440 // 24h
)

// Create validates and persists a new mock interview owned by userID.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (*Interview, error) {
	if err := validateMock(in); err != nil {
		return nil, err
	}
	m := &Interview{UserID: userID}
	applyMock(m, in)
	if m.Outcome == "" {
		m.Outcome = OutcomeNotRated
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get returns a single mock interview with findings owned by userID, or
// ErrMockNotFound.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Interview, error) {
	return s.repo.GetByIDWithFindings(ctx, userID, id)
}

// List returns a page of the user's mock interviews plus the total count.
func (s *Service) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Interview, int64, error) {
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

// Update validates and replaces the mutable fields of a mock owned by userID.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (*Interview, error) {
	if err := validateMock(in); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	applyMock(existing, in)
	if existing.Outcome == "" {
		existing.Outcome = OutcomeNotRated
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	// Reload with findings for a consistent detail response.
	return s.repo.GetByIDWithFindings(ctx, userID, id)
}

// Delete soft-deletes a mock interview owned by userID.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

// AddFinding validates and adds a finding to a mock owned by userID. Ownership
// is enforced by loading the mock first (mismatched owner => ErrMockNotFound).
func (s *Service) AddFinding(ctx context.Context, userID, mockID uuid.UUID, in FindingInput) (*Finding, error) {
	if err := validateFinding(in); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, userID, mockID); err != nil {
		return nil, err
	}
	f := &Finding{
		MockInterviewID: mockID,
		UserID:          userID,
		PillarType:      in.PillarType,
		TopicID:         in.TopicID,
		Severity:        in.Severity,
		Category:        strings.TrimSpace(in.Category),
		Detail:          strings.TrimSpace(in.Detail),
	}
	if err := s.repo.AddFinding(ctx, f); err != nil {
		return nil, err
	}

	// When requested, schedule a follow-up study task on the user's plan and link
	// it back to the finding. Best-effort + nil-safe: a scheduling failure (or no
	// planner / no active roadmap) must not fail the finding that already saved.
	if in.CreateRemediationTask && s.remediate != nil {
		pillar := ""
		if in.PillarType != nil {
			pillar = string(*in.PillarType)
		}
		title := strings.TrimSpace(in.Category)
		if title == "" {
			title = "Mock follow-up: review weakness"
		}
		taskID, rerr := s.remediate.ScheduleRemediation(ctx, RemediationRequest{
			UserID:     userID,
			PillarType: pillar,
			TopicID:    in.TopicID,
			Title:      "Remediate: " + title,
			Detail:     f.Detail,
		})
		if rerr == nil && taskID != uuid.Nil {
			f.RemediationTaskID = &taskID
			_ = s.repo.UpdateFinding(ctx, f)
		}
	}
	return f, nil
}

// Weaknesses aggregates the user's findings into a ranked weakness summary via
// the configured detector.
func (s *Service) Weaknesses(ctx context.Context, userID uuid.UUID) (*WeaknessSummary, error) {
	findings, err := s.repo.ListFindings(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.detector.Detect(ctx, findings)
}

// --- helpers ---

func validateMock(in CreateInput) error {
	var fields []FieldError
	if in.Type == "" {
		fields = append(fields, FieldError{Field: "type", Message: "is required"})
	} else if !in.Type.Valid() {
		fields = append(fields, FieldError{Field: "type", Message: "must be a valid mock type"})
	}
	if in.Outcome != "" && !in.Outcome.Valid() {
		fields = append(fields, FieldError{Field: "outcome", Message: "must be a valid outcome"})
	}
	if in.OverallScore != nil && (*in.OverallScore < 0 || *in.OverallScore > 100) {
		fields = append(fields, FieldError{Field: "overall_score", Message: "must be between 0 and 100"})
	}
	if in.DurationMinutes != nil && (*in.DurationMinutes < 0 || *in.DurationMinutes > maxDurationMin) {
		fields = append(fields, FieldError{Field: "duration_minutes", Message: "must be between 0 and 1440"})
	}
	if len(in.Interviewer) > maxInterviewerLen {
		fields = append(fields, FieldError{Field: "interviewer", Message: "must be at most 200 characters"})
	}
	if len(in.Summary) > maxSummaryLen {
		fields = append(fields, FieldError{Field: "summary", Message: "must be at most 10000 characters"})
	}
	if len(in.TranscriptMD) > maxTranscriptLen {
		fields = append(fields, FieldError{Field: "transcript_md", Message: "must be at most 200000 characters"})
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateFinding(in FindingInput) error {
	var fields []FieldError
	if in.Severity == "" {
		fields = append(fields, FieldError{Field: "severity", Message: "is required"})
	} else if !in.Severity.Valid() {
		fields = append(fields, FieldError{Field: "severity", Message: "must be a valid severity"})
	}
	if strings.TrimSpace(in.Category) == "" {
		fields = append(fields, FieldError{Field: "category", Message: "is required"})
	} else if len(in.Category) > maxCategoryLen {
		fields = append(fields, FieldError{Field: "category", Message: "must be at most 200 characters"})
	}
	if strings.TrimSpace(in.Detail) == "" {
		fields = append(fields, FieldError{Field: "detail", Message: "is required"})
	} else if len(in.Detail) > maxDetailLen {
		fields = append(fields, FieldError{Field: "detail", Message: "must be at most 5000 characters"})
	}
	if in.PillarType != nil && !in.PillarType.Valid() {
		fields = append(fields, FieldError{Field: "pillar_type", Message: "must be a valid pillar type"})
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// applyMock copies the validated input onto the model, trimming and nilling
// empty text fields.
func applyMock(m *Interview, in CreateInput) {
	m.Type = in.Type
	m.TopicID = in.TopicID
	m.DesignProblemID = in.DesignProblemID
	m.CompanyID = in.CompanyID
	m.ScheduledAt = in.ScheduledAt
	m.ConductedAt = in.ConductedAt
	m.DurationMinutes = in.DurationMinutes
	if in.Outcome != "" {
		m.Outcome = in.Outcome
	}
	m.OverallScore = in.OverallScore
	m.Interviewer = strOrNil(in.Interviewer)
	m.TranscriptMD = strOrNil(in.TranscriptMD)
	m.Summary = strOrNil(in.Summary)
}

func strOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
