package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo is an in-memory Repository for service unit tests.
type fakeRepo struct {
	inputs    []PillarInputs
	topics    []TopicEntry
	streak    []StreakDay
	snapshots []Snapshot
	roadmapID *uuid.UUID
	upserted  *Snapshot
}

func (f *fakeRepo) PillarInputs(_ context.Context, _ uuid.UUID) ([]PillarInputs, error) {
	return f.inputs, nil
}
func (f *fakeRepo) TopicEntries(_ context.Context, _ uuid.UUID) ([]TopicEntry, error) {
	return f.topics, nil
}
func (f *fakeRepo) TimeSpent(_ context.Context, _ uuid.UUID, _, _ time.Time, groupBy string) (TimeSpent, error) {
	return TimeSpent{GroupBy: groupBy}, nil
}
func (f *fakeRepo) StreakDays(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]StreakDay, error) {
	return f.streak, nil
}
func (f *fakeRepo) ActiveRoadmapID(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return f.roadmapID, nil
}
func (f *fakeRepo) UpsertSnapshot(_ context.Context, s Snapshot) (Snapshot, error) {
	f.upserted = &s
	return s, nil
}
func (f *fakeRepo) ListSnapshots(_ context.Context, _ uuid.UUID, _, _ time.Time, _, _ int) ([]Snapshot, int64, error) {
	return f.snapshots, int64(len(f.snapshots)), nil
}

func fixedNow(d time.Time) func() time.Time { return func() time.Time { return d } }

// TestService_Readiness exercises the end-to-end readiness assembly through the
// service against a fake repo, confirming the SRS overall matches the calculator.
func TestService_Readiness(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		inputs: []PillarInputs{
			{Pillar: "dsa", Weight: 1, CompletedItems: 1, TotalItems: 1, AvgRating: 5, RevHealth: 1}, // 100
			{Pillar: "lld", Weight: 1, CompletedItems: 0, TotalItems: 1, AvgRating: 5, RevHealth: 1}, // 0
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow(today)})
	r, err := svc.Readiness(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if r.OverallReadiness != 50 {
		t.Fatalf("overall: got %v want 50", r.OverallReadiness)
	}
	if r.PillarReadiness["dsa"] != 100 || r.PillarReadiness["lld"] != 0 {
		t.Fatalf("pillar readiness map = %v", r.PillarReadiness)
	}
	// No history ⇒ estimated date undefined (rate<=0) and not at threshold.
	if r.EstimatedReadyDate != nil {
		t.Fatalf("expected nil estimated date, got %v", *r.EstimatedReadyDate)
	}
}

// TestService_RecordSnapshot confirms the snapshot is built from live readiness.
func TestService_RecordSnapshot(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	rid := uuid.New()
	repo := &fakeRepo{
		roadmapID: &rid,
		inputs: []PillarInputs{
			{Pillar: "dsa", Weight: 1, CompletedItems: 1, TotalItems: 2, AvgRating: 5, RevHealth: 1},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow(today)})
	snap, err := svc.RecordSnapshot(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.SnapshotDate.Equal(today) {
		t.Fatalf("snapshot date = %v", snap.SnapshotDate)
	}
	if repo.upserted == nil || repo.upserted.RoadmapID == nil || *repo.upserted.RoadmapID != rid {
		t.Fatalf("expected roadmap id persisted")
	}
	// coverage 0.5, conf 1, rev 1 ⇒ 100*0.5*(0.6+0.4)=50.
	if snap.OverallReadiness != 50 {
		t.Fatalf("overall = %v want 50", snap.OverallReadiness)
	}
}

// TestService_Streak verifies current/longest streak counting over study days.
func TestService_Streak(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{streak: []StreakDay{
		{Date: today.AddDate(0, 0, -2), TasksCompleted: 1},
		{Date: today.AddDate(0, 0, -1), TasksCompleted: 1},
		{Date: today, TasksCompleted: 1},
	}}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow(today)})
	s, err := svc.Streak(context.Background(), uuid.New(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if s.Current != 3 || s.Longest != 3 {
		t.Fatalf("streak current=%d longest=%d want 3/3", s.Current, s.Longest)
	}
}

// TestService_Topics verifies weak/strong bucketing by composite score.
func TestService_Topics(t *testing.T) {
	weakID, strongID := uuid.New(), uuid.New()
	c5 := 5
	repo := &fakeRepo{topics: []TopicEntry{
		{TopicID: weakID, TopicName: "Weak", PillarType: "dsa", CompletionPct: 0, Score: 0},
		{TopicID: strongID, TopicName: "Strong", PillarType: "dsa", Confidence: &c5, CompletionPct: 100, Score: 1.0},
	}}
	svc := NewService(ServiceConfig{Repo: repo, Now: fixedNow(time.Now())})
	ta, err := svc.Topics(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(ta.Weak) == 0 || ta.Weak[0].TopicID != weakID {
		t.Fatalf("weakest topic mismatch: %+v", ta.Weak)
	}
	if len(ta.Strong) == 0 || ta.Strong[0].TopicID != strongID {
		t.Fatalf("strongest topic mismatch: %+v", ta.Strong)
	}
}
