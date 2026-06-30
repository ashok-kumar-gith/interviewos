package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- fake repository ---

type fakeRepo struct {
	task   *PlanTaskRow
	day    *PlanDayRow
	aggs   []PillarAggregate
	streak     Streak
	revDue     int
	dayErr     error
	hasRoadmap bool

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
func (f *fakeRepo) HasActiveRoadmap(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.hasRoadmap, nil
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
func (f *fakeRepo) StartTask(_ context.Context, _, _ uuid.UUID, _ time.Time) (*PlanTaskRow, error) {
	t := *f.task
	t.Status = "in_progress"
	return &t, nil
}
func (f *fakeRepo) ReopenTask(_ context.Context, _, _ uuid.UUID, _ time.Time) (*PlanTaskRow, error) {
	t := *f.task
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

// TestBlendCoverage exercises the coverage blend directly: each signal counts
// only when its denominator is positive, and coverage is the mean of present
// signals (capped at 1).
func TestBlendCoverage(t *testing.T) {
	cases := []struct {
		name string
		agg  PillarAggregate
		want float64
	}{
		{"plan-only behaves as before", PillarAggregate{PlannedMinutes: 120, CompletedMinutes: 60}, 0.5},
		{"items-only drives coverage", PillarAggregate{ItemsTotal: 10, ItemsCompleted: 4}, 0.4},
		{"both signals are meaned", PillarAggregate{PlannedMinutes: 100, CompletedMinutes: 50, ItemsTotal: 10, ItemsCompleted: 1}, 0.3},
		{"neither signal ⇒ 0", PillarAggregate{}, 0},
		{"capped at 1", PillarAggregate{PlannedMinutes: 10, CompletedMinutes: 30, ItemsTotal: 1, ItemsCompleted: 5}, 1},
	}
	for _, c := range cases {
		if got := blendCoverage(c.agg); got != c.want {
			t.Fatalf("%s: blendCoverage = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestGetDashboard_SolvedProblemsMoveReadiness asserts that an items-completed
// signal (e.g. solved DSA problems) with no plan tasks at all produces non-zero
// readiness — the core linkage fix: solving a problem moves the needle.
func TestGetDashboard_SolvedProblemsMoveReadiness(t *testing.T) {
	// Before: dsa pillar with no progress ⇒ coverage 0 ⇒ readiness 0.
	before := &fakeRepo{
		aggs:   []PillarAggregate{{Pillar: "dsa", Weight: 1.5, ItemsTotal: 10, ItemsCompleted: 0}},
		dayErr: ErrPlanDayNotFound,
	}
	svcBefore := NewService(ServiceConfig{Repo: before, Now: fixedNow})
	dashBefore, err := svcBefore.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("before: %v", err)
	}
	if dashBefore.PillarReadiness[0].Readiness != 0 {
		t.Fatalf("expected 0 readiness before solving, got %v", dashBefore.PillarReadiness[0].Readiness)
	}

	// After: 4 of 10 solved with confidence avg 4 ⇒ coverage 0.4,
	// readiness = 100*0.4*(0.6*0.75 + 0.4*1.0) = 100*0.4*0.85 = 34.
	after := &fakeRepo{
		aggs:   []PillarAggregate{{Pillar: "dsa", Weight: 1.5, ItemsTotal: 10, ItemsCompleted: 4, ConfidenceSum: 16, ConfidenceCount: 4}},
		dayErr: ErrPlanDayNotFound,
	}
	svcAfter := NewService(ServiceConfig{Repo: after, Now: fixedNow})
	dashAfter, err := svcAfter.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("after: %v", err)
	}
	got := dashAfter.PillarReadiness[0]
	if got.Coverage != 0.4 {
		t.Fatalf("expected coverage 0.4 after solving, got %v", got.Coverage)
	}
	if got.Readiness != 34 {
		t.Fatalf("expected readiness 34 after solving, got %v", got.Readiness)
	}
	if !(dashAfter.OverallReadiness > dashBefore.OverallReadiness) {
		t.Fatalf("solving problems must increase overall readiness: before %v, after %v",
			dashBefore.OverallReadiness, dashAfter.OverallReadiness)
	}
}

// TestGetDashboard_AllPillarsPresent asserts every pillar the repository returns
// surfaces on the dashboard, including zeroed lld/behavioral/resume that have no
// plan tasks or items yet.
func TestGetDashboard_AllPillarsPresent(t *testing.T) {
	repo := &fakeRepo{
		aggs: []PillarAggregate{
			{Pillar: "dsa", Weight: 1.5, PlannedMinutes: 100, CompletedMinutes: 50},
			{Pillar: "system_design", Weight: 1.5, ItemsTotal: 5, ItemsCompleted: 1},
			{Pillar: "lld", Weight: 1.0},
			{Pillar: "backend_engineering", Weight: 1.0, PlannedMinutes: 60, CompletedMinutes: 0},
			{Pillar: "behavioral", Weight: 0.75},
			{Pillar: "resume", Weight: 0.5},
		},
		dayErr: ErrPlanDayNotFound,
	}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow})
	dash, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]bool{
		"dsa": false, "system_design": false, "lld": false,
		"backend_engineering": false, "behavioral": false, "resume": false,
	}
	for _, p := range dash.PillarReadiness {
		want[p.Pillar] = true
	}
	for pillar, present := range want {
		if !present {
			t.Fatalf("pillar %q missing from dashboard", pillar)
		}
	}
	if len(dash.PillarReadiness) != 6 {
		t.Fatalf("expected 6 pillars, got %d", len(dash.PillarReadiness))
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

// --- revision-hook tests ---

type fakeScheduler struct {
	calls    int
	itemType string
	itemID   string
	pillar   string
	err      error
}

func (f *fakeScheduler) ScheduleForCompletion(_ context.Context, _ uuid.UUID, itemType, itemID, pillarType string) error {
	f.calls++
	f.itemType, f.itemID, f.pillar = itemType, itemID, pillarType
	return f.err
}

// TestCompleteTask_SchedulesRevisionForLearningTask verifies a completed STUDY
// task on a topic triggers the revision scheduler with the task's poly fields.
func TestCompleteTask_SchedulesRevisionForLearningTask(t *testing.T) {
	task := newTask() // kind=study, item_type=topic
	repo := &fakeRepo{task: task, streak: Streak{Current: 1, Longest: 1}}
	sched := &fakeScheduler{}
	svc := NewService(ServiceConfig{Repo: repo, Revision: sched, Now: fixedNow})

	if _, _, err := svc.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 4, TimeSpentMinutes: 30}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if sched.calls != 1 {
		t.Fatalf("scheduler calls = %d, want 1", sched.calls)
	}
	if sched.itemType != "topic" || sched.itemID != task.ItemID.String() || sched.pillar != "dsa" {
		t.Fatalf("scheduler got %s/%s/%s, want topic/%s/dsa", sched.itemType, sched.itemID, sched.pillar, task.ItemID)
	}
}

// TestCompleteTask_SchedulesRevisionForSolveProblem verifies that completing a
// SOLVE task on a problem schedules a revision item (FR-ROAD-011): solving a
// problem is itself a revisable event.
func TestCompleteTask_SchedulesRevisionForSolveProblem(t *testing.T) {
	task := newTask()
	task.Kind = "solve"
	task.ItemType = "problem"
	repo := &fakeRepo{task: task}
	sched := &fakeScheduler{}
	svc := NewService(ServiceConfig{Repo: repo, Revision: sched, Now: fixedNow})

	if _, _, err := svc.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 3, TimeSpentMinutes: 20}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if sched.calls != 1 {
		t.Fatalf("scheduler calls = %d, want 1 for solve/problem", sched.calls)
	}
	if sched.itemType != "problem" || sched.itemID != task.ItemID.String() {
		t.Fatalf("scheduler got %s/%s, want problem/%s", sched.itemType, sched.itemID, task.ItemID)
	}
}

// TestCompleteTask_NoRevisionForMockKind ensures a non-revisable kind (mock) on a
// topic does not schedule a revision.
func TestCompleteTask_NoRevisionForMockKind(t *testing.T) {
	task := newTask()
	task.Kind = "mock"
	task.ItemType = "topic"
	repo := &fakeRepo{task: task}
	sched := &fakeScheduler{}
	svc := NewService(ServiceConfig{Repo: repo, Revision: sched, Now: fixedNow})

	if _, _, err := svc.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 3, TimeSpentMinutes: 20}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if sched.calls != 0 {
		t.Fatalf("scheduler calls = %d, want 0 for mock kind", sched.calls)
	}
}

// fakeSnapshotter records RecordSnapshot calls.
type fakeSnapshotter struct {
	calls int
	err   error
}

func (f *fakeSnapshotter) RecordSnapshot(_ context.Context, _ uuid.UUID) (SnapshotStub, error) {
	f.calls++
	return SnapshotStub{}, f.err
}

// TestCompleteTask_RecordsSnapshot verifies the daily readiness snapshot is
// recorded (best-effort) on completion, and that a snapshotter error never fails
// the completion that already committed.
func TestCompleteTask_RecordsSnapshot(t *testing.T) {
	task := newTask()
	repo := &fakeRepo{task: task, streak: Streak{Current: 1, Longest: 1}}
	snap := &fakeSnapshotter{}
	svc := NewService(ServiceConfig{Repo: repo, Snapshot: snap, Now: fixedNow})

	if _, _, err := svc.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 4, TimeSpentMinutes: 30}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if snap.calls != 1 {
		t.Fatalf("snapshotter calls = %d, want 1", snap.calls)
	}

	// A snapshotter failure must not fail the completion.
	snap2 := &fakeSnapshotter{err: errors.New("boom")}
	svc2 := NewService(ServiceConfig{Repo: &fakeRepo{task: newTask()}, Snapshot: snap2, Now: fixedNow})
	if _, _, err := svc2.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 4, TimeSpentMinutes: 30}); err != nil {
		t.Fatalf("complete should not fail on snapshotter error: %v", err)
	}
}

// fakeProfiles returns a fixed timezone for the timezone-resolution test.
type fakeProfiles struct {
	tz  string
	err error
}

func (f *fakeProfiles) Timezone(_ context.Context, _ uuid.UUID) (string, error) {
	return f.tz, f.err
}

// TestGetToday_ResolvesUserTimezone verifies the Today plan-day lookup resolves
// "today" in the user's profile timezone (FR-DASH-008), not server UTC.
func TestGetToday_ResolvesUserTimezone(t *testing.T) {
	// 2026-06-29 22:30 UTC. In Asia/Kolkata (+05:30) that is 2026-06-30 04:00.
	clock := func() time.Time { return time.Date(2026, 6, 29, 22, 30, 0, 0, time.UTC) }

	// capturing repo records the date GetPlanDay was queried with.
	cr := &captureRepo{}
	svc := NewService(ServiceConfig{Repo: cr, Profiles: &fakeProfiles{tz: "Asia/Kolkata"}, Now: clock})
	if _, err := svc.GetToday(context.Background(), uuid.New()); err != nil && err != ErrPlanDayNotFound {
		t.Fatalf("get today: %v", err)
	}
	if got := cr.lastDate.Format("2006-01-02"); got != "2026-06-30" {
		t.Fatalf("Kolkata today = %s, want 2026-06-30", got)
	}

	// No profile reader ⇒ UTC fallback (still 2026-06-29).
	cr2 := &captureRepo{}
	svc2 := NewService(ServiceConfig{Repo: cr2, Now: clock})
	if _, err := svc2.GetToday(context.Background(), uuid.New()); err != nil && err != ErrPlanDayNotFound {
		t.Fatalf("get today: %v", err)
	}
	if got := cr2.lastDate.Format("2006-01-02"); got != "2026-06-29" {
		t.Fatalf("UTC today = %s, want 2026-06-29", got)
	}

	// Unknown timezone ⇒ UTC fallback.
	cr3 := &captureRepo{}
	svc3 := NewService(ServiceConfig{Repo: cr3, Profiles: &fakeProfiles{tz: "Not/AZone"}, Now: clock})
	if _, err := svc3.GetToday(context.Background(), uuid.New()); err != nil && err != ErrPlanDayNotFound {
		t.Fatalf("get today: %v", err)
	}
	if got := cr3.lastDate.Format("2006-01-02"); got != "2026-06-29" {
		t.Fatalf("unknown-tz today = %s, want 2026-06-29 (UTC fallback)", got)
	}
}

// captureRepo is a minimal Repository that records the date passed to GetPlanDay.
type captureRepo struct {
	fakeRepo
	lastDate time.Time
}

func (c *captureRepo) GetPlanDay(_ context.Context, _ uuid.UUID, date time.Time) (*PlanDayRow, error) {
	c.lastDate = date
	return nil, ErrPlanDayNotFound
}

// TestCompleteTask_NilSchedulerIsSafe ensures progress works without a scheduler.
func TestCompleteTask_NilSchedulerIsSafe(t *testing.T) {
	task := newTask()
	repo := &fakeRepo{task: task}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow}) // no Revision
	if _, _, err := svc.CompleteTask(context.Background(), task.UserID, task.ID, CompleteParams{Confidence: 5, TimeSpentMinutes: 10}); err != nil {
		t.Fatalf("complete with nil scheduler: %v", err)
	}
}

func TestIsRevisableTask(t *testing.T) {
	// Learning kinds on content items schedule a revision...
	revisable := [][2]string{
		{"study", "topic"}, {"read", "subtopic"}, {"watch", "design_problem"},
		{"study", "lld_problem"}, {"study", "problem"},
		// ...and solving a problem is itself revisable (FR-ROAD-011).
		{"solve", "problem"}, {"solve", "lld_problem"}, {"solve", "design_problem"},
	}
	for _, c := range revisable {
		if !isRevisableTask(c[0], c[1]) {
			t.Fatalf("expected revisable: %s/%s", c[0], c[1])
		}
	}
	notRevisable := [][2]string{
		{"solve", "topic"}, // solving a "topic" is not a thing
		{"mock", "topic"}, {"revise", "topic"},
		{"study", "resource"}, {"study", "behavioral_story"},
	}
	for _, c := range notRevisable {
		if isRevisableTask(c[0], c[1]) {
			t.Fatalf("expected non-revisable: %s/%s", c[0], c[1])
		}
	}
}
