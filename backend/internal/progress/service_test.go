package progress

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- fake repository ---

type fakeRepo struct {
	task   *PlanTaskRow
	day    *PlanDayRow
	aggs   []PillarAggregate
	streak Streak
	revDue int
	dayErr error

	completeCalled   bool
	completeIn       CompleteInput
	skipCalled       bool
	skipReason       string
	rescheduleCalled bool
	rescheduleDate   time.Time
}

func (f *fakeRepo) GetPlanDay(_ context.Context, _ uuid.UUID, _ time.Time) (*PlanDayRow, error) {
	if f.dayErr != nil {
		return nil, f.dayErr
	}
	return f.day, nil
}
func (f *fakeRepo) GetTask(_ context.Context, _, _ uuid.UUID) (*PlanTaskRow, error) {
	if f.task == nil {
		return nil, ErrTaskNotFound
	}
	return f.task, nil
}
func (f *fakeRepo) CompleteTask(_ context.Context, _, _ uuid.UUID, in CompleteInput, now time.Time) (*PlanTaskRow, error) {
	f.completeCalled = true
	f.completeIn = in
	conf := in.Confidence
	t := *f.task
	t.Status = "completed"
	t.Confidence = &conf
	t.TimeSpentMinutes = &in.TimeSpentMinutes
	t.CompletedAt = &now
	return &t, nil
}
func (f *fakeRepo) SkipTask(_ context.Context, _, _ uuid.UUID, reason string, _ time.Time) (*PlanTaskRow, error) {
	f.skipCalled = true
	f.skipReason = reason
	t := *f.task
	t.Status = "skipped"
	return &t, nil
}
func (f *fakeRepo) RescheduleTask(_ context.Context, _, _ uuid.UUID, toDate time.Time, _ time.Time) (*PlanTaskRow, error) {
	f.rescheduleCalled = true
	f.rescheduleDate = toDate
	t := *f.task
	t.ID = uuid.New()
	t.Status = "pending"
	return &t, nil
}
func (f *fakeRepo) ComputeStreak(_ context.Context, _ uuid.UUID, _ time.Time) (Streak, error) {
	return f.streak, nil
}
func (f *fakeRepo) PillarAggregates(_ context.Context, _ uuid.UUID) ([]PillarAggregate, error) {
	return f.aggs, nil
}
func (f *fakeRepo) RevisionDueCount(_ context.Context, _ uuid.UUID, _ time.Time) (int, error) {
	return f.revDue, nil
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
}

func newTask() *PlanTaskRow {
	return &PlanTaskRow{
		ID:               uuid.New(),
		PlanDayID:        uuid.New(),
		UserID:           uuid.New(),
		Kind:             "study",
		ItemType:         "topic",
		ItemID:           uuid.New(),
		PillarType:       "dsa",
		Title:            "Study: Arrays",
		EstimatedMinutes: 60,
		Priority:         "high",
		Status:           "pending",
	}
}

func TestCompleteTask_UpsertsAndReturnsStreak(t *testing.T) {
	repo := &fakeRepo{task: newTask(), streak: Streak{Current: 3, Longest: 5}}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})

	task, streak, err := svc.CompleteTask(context.Background(), repo.task.UserID, repo.task.ID, CompleteParams{
		Confidence: 4, TimeSpentMinutes: 45, Notes: "done",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.completeCalled {
		t.Fatal("expected CompleteTask to be delegated to repo")
	}
	if repo.completeIn.Confidence != 4 || repo.completeIn.TimeSpentMinutes != 45 {
		t.Fatalf("repo got wrong input: %+v", repo.completeIn)
	}
	if repo.completeIn.Notes == nil || *repo.completeIn.Notes != "done" {
		t.Fatalf("notes not forwarded: %+v", repo.completeIn.Notes)
	}
	if task.Status != "completed" {
		t.Fatalf("expected completed, got %q", task.Status)
	}
	if streak.Current != 3 || streak.Longest != 5 {
		t.Fatalf("expected streak 3/5, got %d/%d", streak.Current, streak.Longest)
	}
}

func TestCompleteTask_RejectsBadConfidence(t *testing.T) {
	repo := &fakeRepo{task: newTask()}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	for _, c := range []int{0, 6, -1} {
		if _, _, err := svc.CompleteTask(context.Background(), uuid.New(), uuid.New(), CompleteParams{Confidence: c, TimeSpentMinutes: 10}); err != ErrInvalidConfidence {
			t.Fatalf("confidence %d: expected ErrInvalidConfidence, got %v", c, err)
		}
	}
	if repo.completeCalled {
		t.Fatal("repo should not be called on invalid confidence")
	}
}

func TestSkipTask(t *testing.T) {
	repo := &fakeRepo{task: newTask()}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	task, err := svc.SkipTask(context.Background(), uuid.New(), uuid.New(), "no time")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.skipCalled || repo.skipReason != "no time" {
		t.Fatalf("skip not delegated correctly: called=%v reason=%q", repo.skipCalled, repo.skipReason)
	}
	if task.Status != "skipped" {
		t.Fatalf("expected skipped, got %q", task.Status)
	}
}

func TestRescheduleTask(t *testing.T) {
	repo := &fakeRepo{task: newTask()}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	to := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	task, err := svc.RescheduleTask(context.Background(), uuid.New(), uuid.New(), to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.rescheduleCalled || !repo.rescheduleDate.Equal(to) {
		t.Fatalf("reschedule not delegated: called=%v date=%v", repo.rescheduleCalled, repo.rescheduleDate)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending clone, got %q", task.Status)
	}
}

func TestRescheduleTask_RejectsZeroDate(t *testing.T) {
	repo := &fakeRepo{task: newTask()}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	if _, err := svc.RescheduleTask(context.Background(), uuid.New(), uuid.New(), time.Time{}); err != ErrInvalidReschedule {
		t.Fatalf("expected ErrInvalidReschedule, got %v", err)
	}
}

func TestGetDashboard_ReadinessFormula(t *testing.T) {
	// dsa: coverage 0.5 (60 of 120 min), avg confidence 4 → conf=(4-1)/4=0.75
	// readiness = 100 * 0.5 * (0.6*0.75 + 0.4*1.0) = 100*0.5*0.85 = 42.5
	repo := &fakeRepo{
		aggs: []PillarAggregate{
			{Pillar: "dsa", Weight: 1.5, PlannedMinutes: 120, CompletedMinutes: 60, ConfidenceSum: 4, ConfidenceCount: 1},
		},
		streak: Streak{Current: 1, Longest: 2},
		revDue: 0,
		day: &PlanDayRow{
			ID:   uuid.New(),
			Date: fixedNow(),
			Tasks: []PlanTaskRow{
				{Status: "completed", EstimatedMinutes: 60},
				{Status: "pending", EstimatedMinutes: 30},
			},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	dash, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dash.PillarReadiness) != 1 {
		t.Fatalf("expected 1 pillar, got %d", len(dash.PillarReadiness))
	}
	p := dash.PillarReadiness[0]
	if p.Readiness != 42.5 {
		t.Fatalf("expected readiness 42.5, got %v", p.Readiness)
	}
	if p.Coverage != 0.5 {
		t.Fatalf("expected coverage 0.5, got %v", p.Coverage)
	}
	if p.RevisionHealth != 1.0 {
		t.Fatalf("expected revhealth 1.0, got %v", p.RevisionHealth)
	}
	// Single pillar ⇒ overall == pillar readiness.
	if dash.OverallReadiness != 42.5 {
		t.Fatalf("expected overall 42.5, got %v", dash.OverallReadiness)
	}
	if dash.Today.TotalTasks != 2 || dash.Today.CompletedTasks != 1 {
		t.Fatalf("today counts wrong: %+v", dash.Today)
	}
	if dash.Today.EstimatedHours != 1.5 || dash.Today.RemainingHours != 0.5 {
		t.Fatalf("today hours wrong: est=%v remain=%v", dash.Today.EstimatedHours, dash.Today.RemainingHours)
	}
	if dash.Streak.Current != 1 {
		t.Fatalf("expected current streak 1, got %d", dash.Streak.Current)
	}
	if dash.EstimatedReadinessDate != nil {
		t.Fatalf("expected nil estimated date below threshold, got %v", *dash.EstimatedReadinessDate)
	}
}

func TestGetDashboard_ZeroCoverageGatesReadiness(t *testing.T) {
	// No completed minutes ⇒ coverage 0 ⇒ readiness 0 (multiplicative gate).
	repo := &fakeRepo{
		aggs: []PillarAggregate{
			{Pillar: "dsa", Weight: 1.5, PlannedMinutes: 120, CompletedMinutes: 0, ConfidenceSum: 0, ConfidenceCount: 0},
		},
		dayErr: ErrPlanDayNotFound,
	}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	dash, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dash.OverallReadiness != 0 {
		t.Fatalf("expected 0 readiness with 0 coverage, got %v", dash.OverallReadiness)
	}
	if dash.Today.TotalTasks != 0 {
		t.Fatalf("expected 0 today tasks when no plan-day, got %d", dash.Today.TotalTasks)
	}
}

func TestComputeStreakFromDates(t *testing.T) {
	asOf := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	d := func(day int) time.Time { return time.Date(2026, 6, day, 0, 0, 0, 0, time.UTC) }

	// Consecutive 27,28,29 ending today ⇒ current 3.
	s := computeStreakFromDates([]time.Time{d(27), d(28), d(29)}, asOf)
	if s.Current != 3 || s.Longest != 3 {
		t.Fatalf("expected 3/3, got %d/%d", s.Current, s.Longest)
	}

	// Gap then today ⇒ current 1; longest from the older run of 2.
	s = computeStreakFromDates([]time.Time{d(20), d(21), d(29)}, asOf)
	if s.Current != 1 || s.Longest != 2 {
		t.Fatalf("expected 1/2, got %d/%d", s.Current, s.Longest)
	}

	// Ends yesterday (not today) ⇒ still counts current via yesterday anchor.
	s = computeStreakFromDates([]time.Time{d(27), d(28)}, asOf)
	if s.Current != 2 {
		t.Fatalf("expected current 2 via yesterday anchor, got %d", s.Current)
	}

	// Stale (ended 3 days ago) ⇒ current 0.
	s = computeStreakFromDates([]time.Time{d(25), d(26)}, asOf)
	if s.Current != 0 || s.Longest != 2 {
		t.Fatalf("expected 0/2, got %d/%d", s.Current, s.Longest)
	}

	// Empty.
	s = computeStreakFromDates(nil, asOf)
	if s.Current != 0 || s.Longest != 0 {
		t.Fatalf("expected 0/0 for empty, got %d/%d", s.Current, s.Longest)
	}
}
