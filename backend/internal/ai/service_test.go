package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---- fakes ----

// fakeClient is an injectable ai.Client. It returns a canned text on success or
// a canned error to force the deterministic fallback. It never touches the
// network, so tests are hermetic.
type fakeClient struct {
	text  string
	usage Usage
	err   error
	calls int
}

func (f *fakeClient) Complete(_ context.Context, _, _ string, _ int) (string, Usage, error) {
	f.calls++
	if f.err != nil {
		return "", Usage{}, f.err
	}
	return f.text, f.usage, nil
}

// fakeRepo records invocations in memory.
type fakeRepo struct {
	rows []*Invocation
}

func (r *fakeRepo) Record(_ context.Context, inv *Invocation) error {
	inv.ID = uuid.New()
	r.rows = append(r.rows, inv)
	return nil
}

// stub readers: enough data for every feature to produce a useful result.
type stubReaders struct{}

func (stubReaders) Profile(_ context.Context, _ uuid.UUID) (*Profile, error) {
	return &Profile{TrackID: uuid.New(), TargetRole: "Backend SDE3", HoursPerWeek: 15, TargetWeeks: 12,
		PillarStrength: map[string]int{"dsa": 2, "system_design": 4, "behavioral": 3}}, nil
}
func (stubReaders) ActiveRoadmap(_ context.Context, _ uuid.UUID) (uuid.UUID, int, error) {
	return uuid.New(), 12, nil
}
func (stubReaders) TasksForDate(_ context.Context, _ uuid.UUID, _ string) ([]PlanTask, error) {
	return []PlanTask{
		{Title: "Two Sum", Kind: "solve", PillarType: "dsa", Priority: "high", EstimatedMinutes: 45, Status: "pending"},
		{Title: "Design a URL shortener", Kind: "study", PillarType: "system_design", Priority: "critical", EstimatedMinutes: 60, Status: "pending"},
	}, nil
}

var stubStoryID = uuid.New()

func (stubReaders) Story(_ context.Context, _ uuid.UUID, id uuid.UUID) (*Story, error) {
	if id != stubStoryID {
		return nil, ErrNotFound
	}
	return &Story{ID: id, Title: "Rescued a launch", Theme: "ownership",
		Situation: "Launch was at risk", Task: "I owned delivery", Action: "I worked with the team",
		Result: "We shipped", Metrics: ""}, nil
}
func (stubReaders) Resume(_ context.Context, _ uuid.UUID) (*ResumeData, error) {
	return &ResumeData{Headline: "Senior Engineer", Summary: "Backend specialist",
		Skills: []string{"go", "postgres"}, TargetKeywords: []string{"go", "kubernetes"},
		Bullets: []string{"Reduced p99 latency by 40% across the fleet"}, Projects: 1}, nil
}
func (stubReaders) Findings(_ context.Context, _ uuid.UUID) ([]MockFinding, error) {
	return []MockFinding{
		{PillarType: "system_design", Severity: "major", Category: "scalability", Detail: "missed sharding"},
		{PillarType: "dsa", Severity: "minor", Category: "communication", Detail: "did not state complexity"},
	}, nil
}
func (stubReaders) WeakTopics(_ context.Context, _ uuid.UUID, _ int) ([]WeakTopic, error) {
	return []WeakTopic{
		{TopicID: uuid.New(), TopicName: "Dynamic Programming", PillarType: "dsa", CompletionPct: 0},
	}, nil
}

var stubDesignID = uuid.New()

func (stubReaders) DesignProblem(_ context.Context, id uuid.UUID) (*DesignProblem, error) {
	if id != stubDesignID {
		return nil, ErrNotFound
	}
	return &DesignProblem{ID: id, Title: "Design Twitter", Difficulty: "hard", Slug: "design-twitter"}, nil
}

func newTestService(client Client, enabled bool, repo *fakeRepo) *Service {
	rd := stubReaders{}
	return NewService(ServiceConfig{
		Client:   client,
		Repo:     repo,
		Enabled:  enabled,
		Model:    "claude-sonnet-4-6",
		Profiles: rd, Plans: rd, Stories: rd, Resumes: rd, Mocks: rd, Topics: rd, Designs: rd,
	})
}

// ---- tests ----

// Success path: a working client returns model text; used_fallback is false and
// an invocation is recorded as succeeded with the model id.
func TestPlanner_SuccessPathUsesModel(t *testing.T) {
	repo := &fakeRepo{}
	fc := &fakeClient{text: "## Plan\nDo DSA first.", usage: Usage{PromptTokens: 10, CompletionTokens: 20}}
	svc := newTestService(fc, true, repo)

	res, err := svc.Planner(context.Background(), uuid.New(), PlannerInput{})
	if err != nil {
		t.Fatalf("Planner: %v", err)
	}
	if res.UsedFallback {
		t.Fatalf("expected used_fallback=false on success path")
	}
	if res.Content != fc.text {
		t.Fatalf("expected model text, got %q", res.Content)
	}
	if fc.calls != 1 {
		t.Fatalf("expected 1 client call, got %d", fc.calls)
	}
	if len(repo.rows) != 1 || repo.rows[0].Status != StatusSucceeded {
		t.Fatalf("expected 1 succeeded invocation, got %+v", repo.rows)
	}
	if repo.rows[0].UsedFallback {
		t.Fatalf("invocation should not be marked used_fallback on success")
	}
	if repo.rows[0].Model == nil || *repo.rows[0].Model != "claude-sonnet-4-6" {
		t.Fatalf("expected model recorded, got %+v", repo.rows[0].Model)
	}
	if repo.rows[0].PromptTokens == nil || *repo.rows[0].PromptTokens != 10 {
		t.Fatalf("expected prompt tokens recorded")
	}
}

// Error path: a failing client falls back deterministically; used_fallback is
// true and the invocation is recorded as fallback with the error captured.
func TestPlanner_ErrorPathFallsBack(t *testing.T) {
	repo := &fakeRepo{}
	fc := &fakeClient{err: errors.New("boom")}
	svc := newTestService(fc, true, repo)

	res, err := svc.Planner(context.Background(), uuid.New(), PlannerInput{})
	if err != nil {
		t.Fatalf("Planner: %v", err)
	}
	if !res.UsedFallback {
		t.Fatalf("expected used_fallback=true on error path")
	}
	if !strings.Contains(res.Content, "Study plan") {
		t.Fatalf("expected deterministic plan content, got %q", res.Content)
	}
	if len(repo.rows) != 1 || repo.rows[0].Status != StatusFallback {
		t.Fatalf("expected 1 fallback invocation, got %+v", repo.rows)
	}
	if !repo.rows[0].UsedFallback {
		t.Fatalf("invocation should be marked used_fallback")
	}
}

// No-key path: a nil client (no AI configured) always falls back, never panics,
// and records a fallback invocation — this is the live-environment path.
func TestFeatures_NoKeyFallback(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(nil, false, repo) // nil client + disabled
	uid := uuid.New()
	ctx := context.Background()

	planner, err := svc.Planner(ctx, uid, PlannerInput{})
	if err != nil || !planner.UsedFallback {
		t.Fatalf("planner fallback: %v %+v", err, planner)
	}
	coach, err := svc.Coach(ctx, uid, "How do I prep for system design?")
	if err != nil || !coach.UsedFallback || !strings.Contains(coach.Content, "system-design framework") {
		t.Fatalf("coach fallback: %v %+v", err, coach)
	}
	daily, err := svc.DailyPlan(ctx, uid, "2026-06-29")
	if err != nil || !daily.UsedFallback || !strings.Contains(daily.Content, "Recommended plan") {
		t.Fatalf("daily fallback: %v %+v", err, daily)
	}
	sd, err := svc.SDReview(ctx, uid, SDReviewInput{DesignProblemID: stubDesignID, AnswerMD: "We use a load balancer and cache and sharding."})
	if err != nil || !sd.UsedFallback {
		t.Fatalf("sd fallback: %v %+v", err, sd)
	}
	story, err := svc.StoryImprove(ctx, uid, StoryImproveInput{StoryID: &stubStoryID})
	if err != nil || !story.UsedFallback || len(story.Suggestions) == 0 {
		t.Fatalf("story fallback: %v %+v", err, story)
	}
	res, err := svc.ResumeReview(ctx, uid)
	if err != nil || !res.UsedFallback {
		t.Fatalf("resume fallback: %v %+v", err, res)
	}
	weak, err := svc.WeaknessDetect(ctx, uid)
	if err != nil || !weak.UsedFallback || len(weak.RecommendedTasks) == 0 {
		t.Fatalf("weakness fallback: %v %+v", err, weak)
	}

	if len(repo.rows) != 7 {
		t.Fatalf("expected 7 invocations recorded, got %d", len(repo.rows))
	}
	for _, r := range repo.rows {
		if !r.UsedFallback || r.Status != StatusFallback {
			t.Fatalf("expected every invocation to be fallback, got %+v", r)
		}
	}
}

// StoryImprove on a missing story id returns ErrNotFound (mapped to 404).
func TestStoryImprove_NotFound(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(nil, false, repo)
	missing := uuid.New()
	_, err := svc.StoryImprove(context.Background(), uuid.New(), StoryImproveInput{StoryID: &missing})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Success story-improve still layers the deterministic strength score so the
// response shape is stable regardless of provider.
func TestStoryImprove_SuccessLayersScore(t *testing.T) {
	repo := &fakeRepo{}
	fc := &fakeClient{text: "Quantify the result with concrete metrics.\nLead the action with \"I\"."}
	svc := newTestService(fc, true, repo)
	res, err := svc.StoryImprove(context.Background(), uuid.New(), StoryImproveInput{StoryID: &stubStoryID})
	if err != nil {
		t.Fatalf("StoryImprove: %v", err)
	}
	if res.UsedFallback {
		t.Fatalf("expected success path")
	}
	if len(res.Suggestions) != 2 {
		t.Fatalf("expected 2 parsed suggestions, got %+v", res.Suggestions)
	}
	if res.StrengthScore <= 0 {
		t.Fatalf("expected deterministic strength score to be layered in, got %v", res.StrengthScore)
	}
}
