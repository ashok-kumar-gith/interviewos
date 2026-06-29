package intake

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UpsertInput is the validated, domain-level payload for creating or updating a
// profile. It mirrors the UserProfileUpsert OpenAPI schema. Optional fields use
// pointers so "absent" is distinguishable from a zero value where it matters.
type UpsertInput struct {
	TrackID         uuid.UUID
	YearsExperience float64
	TargetCompanyID *uuid.UUID
	TargetRole      string
	TargetLevel     *string
	HoursPerWeek    int
	StartDate       time.Time
	TargetWeeks     *int
	PillarStrengths map[string]int
	Timezone        *string
	IntakeAnswers   json.RawMessage
}

// Service implements the intake use-cases. It depends only on the Repository
// interface so it is unit-testable with a fake.
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

// Defaults applied when the client omits a value.
const (
	defaultTargetWeeks = 12
	defaultTimezone    = "UTC"
	minHoursPerWeek    = 1
	maxHoursPerWeek    = 80
	minTargetWeeks     = 1
	maxTargetWeeks     = 52
	minConfidence      = 1
	maxConfidence      = 5
)

// Get returns the user's active intake profile or ErrProfileNotFound.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// Upsert validates the input and creates or updates the user's profile. On
// validation failure it returns a *ValidationError (wrapping ErrValidation).
func (s *Service) Upsert(ctx context.Context, userID uuid.UUID, in UpsertInput) (*Profile, error) {
	if err := validate(in); err != nil {
		return nil, err
	}

	pillars := []byte("{}")
	if in.PillarStrengths != nil {
		b, err := json.Marshal(in.PillarStrengths)
		if err != nil {
			return nil, &ValidationError{Violations: []FieldViolation{{
				Field: "pillar_strengths", Message: "must be a JSON object of pillar_type to 1-5",
			}}}
		}
		pillars = b
	}

	answers := []byte("{}")
	if len(in.IntakeAnswers) > 0 {
		answers = append([]byte(nil), in.IntakeAnswers...)
	}

	targetWeeks := int16(defaultTargetWeeks)
	if in.TargetWeeks != nil {
		targetWeeks = int16(*in.TargetWeeks)
	}
	timezone := defaultTimezone
	if in.Timezone != nil && *in.Timezone != "" {
		timezone = *in.Timezone
	}

	p := &Profile{
		UserID:          userID,
		TrackID:         in.TrackID,
		YearsExperience: in.YearsExperience,
		TargetCompanyID: in.TargetCompanyID,
		TargetRole:      in.TargetRole,
		TargetLevel:     in.TargetLevel,
		HoursPerWeek:    int16(in.HoursPerWeek),
		StartDate:       in.StartDate,
		TargetWeeks:     targetWeeks,
		PillarStrengths: pillars,
		Timezone:        timezone,
		IntakeAnswers:   answers,
	}

	if err := s.repo.Upsert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// validate enforces the domain rules from the OpenAPI UserProfileUpsert schema:
// required track/role/start_date, years_experience >= 0, hours_per_week 1..80,
// target_weeks 1..52, and pillar_strengths values 1..5 with known pillar keys.
func validate(in UpsertInput) error {
	var v []FieldViolation

	if in.TrackID == uuid.Nil {
		v = append(v, FieldViolation{Field: "track_id", Message: "is required"})
	}
	if in.TargetRole == "" {
		v = append(v, FieldViolation{Field: "target_role", Message: "is required"})
	}
	if in.StartDate.IsZero() {
		v = append(v, FieldViolation{Field: "start_date", Message: "is required"})
	}
	if in.YearsExperience < 0 {
		v = append(v, FieldViolation{Field: "years_experience", Message: "must be >= 0"})
	}
	if in.HoursPerWeek < minHoursPerWeek || in.HoursPerWeek > maxHoursPerWeek {
		v = append(v, FieldViolation{Field: "hours_per_week", Message: "must be between 1 and 80"})
	}
	if in.TargetWeeks != nil && (*in.TargetWeeks < minTargetWeeks || *in.TargetWeeks > maxTargetWeeks) {
		v = append(v, FieldViolation{Field: "target_weeks", Message: "must be between 1 and 52"})
	}
	for pillar, level := range in.PillarStrengths {
		if !IsValidPillarType(pillar) {
			v = append(v, FieldViolation{
				Field:   "pillar_strengths." + pillar,
				Message: "unknown pillar_type",
			})
			continue
		}
		if level < minConfidence || level > maxConfidence {
			v = append(v, FieldViolation{
				Field:   "pillar_strengths." + pillar,
				Message: "must be between 1 and 5",
			})
		}
	}

	if len(v) > 0 {
		return &ValidationError{Violations: v}
	}
	return nil
}
