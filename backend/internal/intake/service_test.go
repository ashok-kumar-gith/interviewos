package intake

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository for unit tests.
type fakeRepository struct {
	byUser    map[uuid.UUID]*Profile
	upsertErr error
}

func newFakeRepo() *fakeRepository {
	return &fakeRepository{byUser: map[uuid.UUID]*Profile{}}
}

func (f *fakeRepository) GetByUserID(_ context.Context, userID uuid.UUID) (*Profile, error) {
	p, ok := f.byUser[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepository) Upsert(_ context.Context, p *Profile) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	now := time.Now()
	if existing, ok := f.byUser[p.UserID]; ok {
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
	} else {
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	cp := *p
	f.byUser[p.UserID] = &cp
	return nil
}

func validInput() UpsertInput {
	return UpsertInput{
		TrackID:         uuid.New(),
		YearsExperience: 6.5,
		TargetRole:      "SDE3 / Senior Backend",
		HoursPerWeek:    15,
		StartDate:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		PillarStrengths: map[string]int{"dsa": 3, "system_design": 2},
	}
}

func TestService_Upsert_Valid(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	uid := uuid.New()

	p, err := svc.Upsert(context.Background(), uid, validInput())
	require.NoError(t, err)
	require.Equal(t, uid, p.UserID)
	require.Equal(t, int16(15), p.HoursPerWeek)
	require.Equal(t, int16(defaultTargetWeeks), p.TargetWeeks, "target_weeks defaults to 12")
	require.Equal(t, defaultTimezone, p.Timezone, "timezone defaults to UTC")

	// pillar_strengths round-trips through JSONB.
	var ps map[string]int
	require.NoError(t, json.Unmarshal(p.PillarStrengths, &ps))
	require.Equal(t, map[string]int{"dsa": 3, "system_design": 2}, ps)

	// intake_answers defaults to an empty object.
	require.JSONEq(t, "{}", string(p.IntakeAnswers))
}

func TestService_Upsert_UpdatesExisting(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	uid := uuid.New()

	first, err := svc.Upsert(context.Background(), uid, validInput())
	require.NoError(t, err)

	in := validInput()
	in.HoursPerWeek = 40
	second, err := svc.Upsert(context.Background(), uid, in)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "upsert keeps the same row id")
	require.Equal(t, int16(40), second.HoursPerWeek)
}

func TestService_Upsert_Validation(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*UpsertInput)
		wantField string
	}{
		{"missing track_id", func(in *UpsertInput) { in.TrackID = uuid.Nil }, "track_id"},
		{"missing target_role", func(in *UpsertInput) { in.TargetRole = "" }, "target_role"},
		{"missing start_date", func(in *UpsertInput) { in.StartDate = time.Time{} }, "start_date"},
		{"negative years_experience", func(in *UpsertInput) { in.YearsExperience = -1 }, "years_experience"},
		{"hours_per_week zero", func(in *UpsertInput) { in.HoursPerWeek = 0 }, "hours_per_week"},
		{"hours_per_week over 80", func(in *UpsertInput) { in.HoursPerWeek = 81 }, "hours_per_week"},
		{"pillar level too high", func(in *UpsertInput) { in.PillarStrengths = map[string]int{"dsa": 6} }, "pillar_strengths.dsa"},
		{"pillar level too low", func(in *UpsertInput) { in.PillarStrengths = map[string]int{"dsa": 0} }, "pillar_strengths.dsa"},
		{"unknown pillar", func(in *UpsertInput) { in.PillarStrengths = map[string]int{"nope": 3} }, "pillar_strengths.nope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(ServiceConfig{Repo: newFakeRepo()})
			in := validInput()
			tc.mutate(&in)

			_, err := svc.Upsert(context.Background(), uuid.New(), in)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrValidation)

			var ve *ValidationError
			require.True(t, errors.As(err, &ve))
			found := false
			for _, v := range ve.Violations {
				if v.Field == tc.wantField {
					found = true
				}
			}
			require.True(t, found, "expected violation on field %q, got %+v", tc.wantField, ve.Violations)
		})
	}
}

func TestService_Upsert_BoundaryValuesAccepted(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})

	for _, hpw := range []int{minHoursPerWeek, maxHoursPerWeek} {
		in := validInput()
		in.HoursPerWeek = hpw
		in.PillarStrengths = map[string]int{"dsa": minConfidence, "lld": maxConfidence}
		_, err := svc.Upsert(context.Background(), uuid.New(), in)
		require.NoError(t, err, "hours_per_week=%d should be valid", hpw)
	}
}

func TestService_Upsert_TargetWeeksRange(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})

	over := 53
	in := validInput()
	in.TargetWeeks = &over
	_, err := svc.Upsert(context.Background(), uuid.New(), in)
	require.ErrorIs(t, err, ErrValidation)

	ok := 16
	in = validInput()
	in.TargetWeeks = &ok
	p, err := svc.Upsert(context.Background(), uuid.New(), in)
	require.NoError(t, err)
	require.Equal(t, int16(16), p.TargetWeeks)
}

func TestService_Get_NotFound(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	_, err := svc.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrProfileNotFound)
}
