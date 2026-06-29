package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// --- fake read ports ---

type fakePlanReader struct {
	days map[string]*PlanDaySummary // keyed by YYYY-MM-DD
	err  error
}

func (f *fakePlanReader) PlanDay(_ context.Context, _ uuid.UUID, date time.Time) (*PlanDaySummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.days[date.UTC().Format("2006-01-02")], nil
}

type fakeRevReader struct {
	due int
	err error
}

func (f *fakeRevReader) DueCount(_ context.Context, _ uuid.UUID, _ time.Time) (int, error) {
	return f.due, f.err
}

type fakeStreakReader struct {
	current     int
	loggedToday bool
	err         error
}

func (f *fakeStreakReader) Streak(_ context.Context, _ uuid.UUID, _ time.Time) (int, bool, error) {
	return f.current, f.loggedToday, f.err
}

type fakeReadinessReader struct {
	latest, previous float64
	hasAny           bool
	err              error
}

func (f *fakeReadinessReader) LatestReadiness(_ context.Context, _ uuid.UUID) (float64, float64, bool, error) {
	return f.latest, f.previous, f.hasAny, f.err
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func typesOf(ns []Notification) map[Type]int {
	m := map[Type]int{}
	for _, n := range ns {
		m[n.Type]++
	}
	return m
}

func TestGenerate_TodayPlan(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:  repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{"2026-06-29": {TotalTasks: 3, CompletedTasks: 0, EstimatedMins: 150}}},
		Now:   fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeTodayPlan, out[0].Type)
	require.Contains(t, out[0].Title, "3 tasks today")
	require.Contains(t, out[0].Title, "2.5h")
	require.NotNil(t, out[0].DedupKey)
}

func TestGenerate_NoTodayPlanWhenCompleted(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:  repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{"2026-06-29": {TotalTasks: 3, CompletedTasks: 1, EstimatedMins: 150}}},
		Now:   fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, out, "today_plan should not fire once a task is completed")
}

func TestGenerate_RevisionDue(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:      repo,
		Revisions: &fakeRevReader{due: 4},
		Now:       fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeRevisionDue, out[0].Type)
	require.Contains(t, out[0].Title, "4 items due for revision")
	require.Equal(t, 4, out[0].Payload["due_count"])
}

func TestGenerate_RevisionDueZeroSkipped(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{Repo: repo, Revisions: &fakeRevReader{due: 0}, Now: fixedNow()})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestGenerate_MissedGoal(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo: repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{
			// yesterday had 2 planned, none completed -> missed_goal.
			"2026-06-28": {TotalTasks: 2, CompletedTasks: 0, EstimatedMins: 90},
			// no plan today -> no today_plan.
		}},
		Now: fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeMissedGoal, out[0].Type)
	require.Contains(t, out[0].Title, "missed 2 planned tasks yesterday")
}

func TestGenerate_NoMissedGoalWhenYesterdayCompleted(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo: repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{
			"2026-06-28": {TotalTasks: 2, CompletedTasks: 2, EstimatedMins: 90},
		}},
		Now: fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestGenerate_StreakReminder(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:    repo,
		Streaks: &fakeStreakReader{current: 5, loggedToday: false},
		Now:     fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeStreakReminder, out[0].Type)
	require.Contains(t, out[0].Title, "5-day streak")
}

func TestGenerate_NoStreakReminderWhenLoggedToday(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:    repo,
		Streaks: &fakeStreakReader{current: 5, loggedToday: true},
		Now:     fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestGenerate_ReadinessMilestone(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:      repo,
		Readiness: &fakeReadinessReader{latest: 62.5, previous: 58.0, hasAny: true},
		Now:       fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, TypeReadinessMilestone, out[0].Type)
	require.Contains(t, out[0].Title, "60%")
}

func TestGenerate_NoMilestoneWhenNoCrossing(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo:      repo,
		Readiness: &fakeReadinessReader{latest: 64.0, previous: 61.0, hasAny: true},
		Now:       fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, out, "no new 10-point multiple crossed")
}

func TestGenerate_AllTypesTogether(t *testing.T) {
	repo := newFakeRepo()
	gen := NewGenerator(GeneratorConfig{
		Repo: repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{
			"2026-06-29": {TotalTasks: 2, CompletedTasks: 0, EstimatedMins: 60},
			"2026-06-28": {TotalTasks: 1, CompletedTasks: 0, EstimatedMins: 30},
		}},
		Revisions: &fakeRevReader{due: 3},
		Streaks:   &fakeStreakReader{current: 2, loggedToday: false},
		Readiness: &fakeReadinessReader{latest: 70.0, previous: 59.0, hasAny: true},
		Now:       fixedNow(),
	})
	out, err := gen.Generate(context.Background(), uuid.New())
	require.NoError(t, err)
	counts := typesOf(out)
	require.Equal(t, 1, counts[TypeTodayPlan])
	require.Equal(t, 1, counts[TypeMissedGoal])
	require.Equal(t, 1, counts[TypeRevisionDue])
	require.Equal(t, 1, counts[TypeStreakReminder])
	require.Equal(t, 1, counts[TypeReadinessMilestone])
	require.Len(t, out, 5)
}

func TestGenerate_Idempotent_NoDuplicates(t *testing.T) {
	repo := newFakeRepo()
	uid := uuid.New()
	cfg := GeneratorConfig{
		Repo: repo,
		Plans: &fakePlanReader{days: map[string]*PlanDaySummary{
			"2026-06-29": {TotalTasks: 2, CompletedTasks: 0, EstimatedMins: 60},
		}},
		Revisions: &fakeRevReader{due: 3},
		Now:       fixedNow(),
	}
	gen := NewGenerator(cfg)

	first, err := gen.Generate(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.Len(t, repo.items, 2)

	// Re-run for the same day: same set, NO new rows.
	second, err := gen.Generate(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, second, 2)
	require.Len(t, repo.items, 2, "second run must not create duplicates")

	// IDs are stable (same persisted rows returned).
	firstIDs := map[uuid.UUID]struct{}{}
	for _, n := range first {
		firstIDs[n.ID] = struct{}{}
	}
	for _, n := range second {
		_, ok := firstIDs[n.ID]
		require.True(t, ok, "re-run returned a different row id")
	}
}

func TestGenerate_RequiresUserID(t *testing.T) {
	gen := NewGenerator(GeneratorConfig{Repo: newFakeRepo()})
	_, err := gen.Generate(context.Background(), uuid.Nil)
	require.ErrorIs(t, err, ErrValidation)
}

func TestService_GenerateUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	_, err := svc.Generate(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrGeneratorUnavailable)
}

func TestCrossedMilestone(t *testing.T) {
	cases := []struct {
		prev, latest float64
		want         int
		crossed      bool
	}{
		{58, 62.5, 60, true},
		{59, 70, 70, true},
		{61, 64, 0, false},
		{0, 9, 0, false},
		{0, 10, 10, true},
		{95, 92, 0, false}, // regression doesn't fire
	}
	for _, c := range cases {
		got, crossed := crossedMilestone(c.prev, c.latest)
		require.Equal(t, c.crossed, crossed, "prev=%v latest=%v", c.prev, c.latest)
		if crossed {
			require.Equal(t, c.want, got)
		}
	}
}
