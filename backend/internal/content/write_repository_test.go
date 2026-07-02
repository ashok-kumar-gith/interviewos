package content

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestWriteProblemRoundtrip exercises create -> read -> update -> delete for a
// DSA problem including its pattern/source/company-frequency join tables. It is
// DATABASE_URL-guarded (skips without a seeded DB) via openTestDB.
func TestWriteProblemRoundtrip(t *testing.T) {
	db := openTestDB(t)
	wr := NewWriteRepository(db)
	rd := NewRepository(db)
	ctx := context.Background()

	trackID, err := wr.DefaultTrackID(ctx)
	if err != nil {
		t.Fatalf("DefaultTrackID: %v", err)
	}

	slug := "admin-test-" + uuid.NewString()[:8]
	freq := "2026-Q1"
	created, err := wr.CreateProblem(ctx, ProblemWrite{
		Problem: Problem{
			TrackID:       trackID,
			Slug:          slug,
			Title:         "Admin Test Problem",
			Difficulty:    DifficultyMedium,
			Platform:      ProblemPlatform("leetcode"),
			PromptSummary: ptrStr("summarize"),
		},
		PatternSlugs:     []string{"arrays-hashing"},
		Sources:          []ProblemSourceName{"blind75", "neetcode150"},
		CompanyFrequency: []CompanyFrequencyInput{{CompanySlug: "amazon", Frequency: 42, LastSeenPeriod: &freq}},
	})
	if err != nil {
		t.Fatalf("CreateProblem: %v", err)
	}
	defer func() { _ = wr.DeleteProblem(ctx, created.Problem.ID) }()

	if len(created.Patterns) != 1 || created.Patterns[0].Slug != "arrays-hashing" {
		t.Fatalf("expected 1 arrays-hashing pattern, got %+v", created.Patterns)
	}
	if len(created.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(created.Sources))
	}
	if len(created.CompanyFrequency) != 1 || created.CompanyFrequency[0].Frequency != 42 {
		t.Fatalf("expected amazon frequency 42, got %+v", created.CompanyFrequency)
	}

	// Read back via the read repository.
	got, err := rd.GetProblemBundle(ctx, created.Problem.ID)
	if err != nil {
		t.Fatalf("GetProblemBundle: %v", err)
	}
	if got.Problem.Slug != slug {
		t.Fatalf("slug mismatch: %s", got.Problem.Slug)
	}

	// Update: change title + swap associations.
	updated, err := wr.UpdateProblem(ctx, created.Problem.ID, ProblemWrite{
		Problem: Problem{
			TrackID:    trackID,
			Slug:       slug,
			Title:      "Admin Test Problem (updated)",
			Difficulty: DifficultyHard,
			Platform:   ProblemPlatform("leetcode"),
		},
		PatternSlugs: []string{"two-pointers"},
		Sources:      []ProblemSourceName{"grind75"},
	})
	if err != nil {
		t.Fatalf("UpdateProblem: %v", err)
	}
	if updated.Problem.Title != "Admin Test Problem (updated)" || updated.Problem.Difficulty != DifficultyHard {
		t.Fatalf("update did not persist: %+v", updated.Problem)
	}
	if len(updated.Patterns) != 1 || updated.Patterns[0].Slug != "two-pointers" {
		t.Fatalf("patterns not re-set on update: %+v", updated.Patterns)
	}
	if len(updated.Sources) != 1 || len(updated.CompanyFrequency) != 0 {
		t.Fatalf("sources/frequencies not re-set on update: %+v / %+v", updated.Sources, updated.CompanyFrequency)
	}

	// Delete (soft) and confirm it is gone from reads.
	if err := wr.DeleteProblem(ctx, created.Problem.ID); err != nil {
		t.Fatalf("DeleteProblem: %v", err)
	}
	if _, err := rd.GetProblemBundle(ctx, created.Problem.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

// TestWriteTopicRoundtrip exercises create -> update -> delete for a topic under
// the backend_engineering pillar.
func TestWriteTopicRoundtrip(t *testing.T) {
	db := openTestDB(t)
	wr := NewWriteRepository(db)
	ctx := context.Background()

	pt := PillarBackendEng
	pillar, err := wr.ResolvePillar(ctx, nil, &pt)
	if err != nil {
		t.Fatalf("ResolvePillar(backend_engineering): %v", err)
	}

	slug := "admin-topic-" + uuid.NewString()[:8]
	created, err := wr.CreateTopic(ctx, TopicWrite{Topic: Topic{
		PillarID:          pillar.ID,
		TrackID:           pillar.TrackID,
		Slug:              slug,
		Name:              "Admin Topic",
		Difficulty:        DifficultyMedium,
		Priority:          Priority("high"),
		EstimatedHours:    3,
		ExpectedQuestions: JSONArray{"q1"},
		Prerequisites:     JSONArray{},
	}})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	defer func() { _ = wr.DeleteTopic(ctx, created.Topic.ID) }()

	if created.Topic.PillarID != pillar.ID {
		t.Fatalf("pillar mismatch")
	}

	updated, err := wr.UpdateTopic(ctx, created.Topic.ID, TopicWrite{Topic: Topic{
		PillarID:          pillar.ID,
		TrackID:           pillar.TrackID,
		Slug:              slug,
		Name:              "Admin Topic (updated)",
		Difficulty:        DifficultyHard,
		Priority:          Priority("critical"),
		EstimatedHours:    5,
		ExpectedQuestions: JSONArray{},
		Prerequisites:     JSONArray{},
	}})
	if err != nil {
		t.Fatalf("UpdateTopic: %v", err)
	}
	if updated.Topic.Name != "Admin Topic (updated)" || updated.Topic.Difficulty != DifficultyHard {
		t.Fatalf("topic update did not persist: %+v", updated.Topic)
	}

	if err := wr.DeleteTopic(ctx, created.Topic.ID); err != nil {
		t.Fatalf("DeleteTopic: %v", err)
	}
	if _, err := wr.(*gormRepository).GetTopicBundle(ctx, created.Topic.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after topic delete, got %v", err)
	}
}

func ptrStr(s string) *string { return &s }
