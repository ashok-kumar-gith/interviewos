package content

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// fakeWriteRepo is an in-memory WriteRepository for service unit tests.
type fakeWriteRepo struct {
	problems    map[uuid.UUID]*ProblemBundle
	topics      map[uuid.UUID]*TopicBundle
	problemSlug map[string]uuid.UUID
	topicSlug   map[string]uuid.UUID
	pillars     map[PillarType]*Pillar
	trackID     uuid.UUID
	lastWrite   ProblemWrite
	lastTopic   TopicWrite
}

func newFakeWriteRepo() *fakeWriteRepo {
	track := uuid.New()
	pillar := &Pillar{ID: uuid.New(), TrackID: track, Type: PillarBackendEng, Name: "Backend Engineering"}
	return &fakeWriteRepo{
		problems:    map[uuid.UUID]*ProblemBundle{},
		topics:      map[uuid.UUID]*TopicBundle{},
		problemSlug: map[string]uuid.UUID{},
		topicSlug:   map[string]uuid.UUID{},
		pillars:     map[PillarType]*Pillar{PillarBackendEng: pillar},
		trackID:     track,
	}
}

func (f *fakeWriteRepo) CreateProblem(_ context.Context, w ProblemWrite) (*ProblemBundle, error) {
	f.lastWrite = w
	id := uuid.New()
	w.Problem.ID = id
	f.problems[id] = &ProblemBundle{Problem: w.Problem}
	f.problemSlug[w.Problem.Slug] = id
	return f.problems[id], nil
}

func (f *fakeWriteRepo) UpdateProblem(_ context.Context, id uuid.UUID, w ProblemWrite) (*ProblemBundle, error) {
	if _, ok := f.problems[id]; !ok {
		return nil, ErrNotFound
	}
	f.lastWrite = w
	w.Problem.ID = id
	f.problems[id] = &ProblemBundle{Problem: w.Problem}
	f.problemSlug[w.Problem.Slug] = id
	return f.problems[id], nil
}

func (f *fakeWriteRepo) DeleteProblem(_ context.Context, id uuid.UUID) error {
	if _, ok := f.problems[id]; !ok {
		return ErrNotFound
	}
	delete(f.problems, id)
	return nil
}

func (f *fakeWriteRepo) CreateTopic(_ context.Context, w TopicWrite) (*TopicBundle, error) {
	f.lastTopic = w
	id := uuid.New()
	w.Topic.ID = id
	f.topics[id] = &TopicBundle{Topic: w.Topic}
	f.topicSlug[w.Topic.Slug] = id
	return f.topics[id], nil
}

func (f *fakeWriteRepo) UpdateTopic(_ context.Context, id uuid.UUID, w TopicWrite) (*TopicBundle, error) {
	if _, ok := f.topics[id]; !ok {
		return nil, ErrNotFound
	}
	w.Topic.ID = id
	f.topics[id] = &TopicBundle{Topic: w.Topic}
	return f.topics[id], nil
}

func (f *fakeWriteRepo) DeleteTopic(_ context.Context, id uuid.UUID) error {
	if _, ok := f.topics[id]; !ok {
		return ErrNotFound
	}
	delete(f.topics, id)
	return nil
}

func (f *fakeWriteRepo) DefaultTrackID(_ context.Context) (uuid.UUID, error) {
	return f.trackID, nil
}

func (f *fakeWriteRepo) ResolvePillar(_ context.Context, pillarID *uuid.UUID, pillarType *PillarType) (*Pillar, error) {
	if pillarType != nil {
		if p, ok := f.pillars[*pillarType]; ok {
			return p, nil
		}
		return nil, ErrNotFound
	}
	if pillarID != nil {
		for _, p := range f.pillars {
			if p.ID == *pillarID {
				return p, nil
			}
		}
		return nil, ErrNotFound
	}
	return nil, ErrValidation
}

func (f *fakeWriteRepo) ProblemSlugExists(_ context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	id, ok := f.problemSlug[slug]
	if !ok {
		return false, nil
	}
	if excludeID != nil && *excludeID == id {
		return false, nil
	}
	return true, nil
}

func (f *fakeWriteRepo) TopicSlugExists(_ context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	id, ok := f.topicSlug[slug]
	if !ok {
		return false, nil
	}
	if excludeID != nil && *excludeID == id {
		return false, nil
	}
	return true, nil
}

func TestAdminCreateProblem_Defaults(t *testing.T) {
	repo := newFakeWriteRepo()
	svc := NewAdminService(repo)
	b, err := svc.CreateProblem(context.Background(), ProblemInput{
		Slug:       "two-sum",
		Title:      "Two Sum",
		Difficulty: "easy",
		Sources:    []string{"blind75"},
	})
	if err != nil {
		t.Fatalf("CreateProblem: %v", err)
	}
	if b.Problem.TrackID != repo.trackID {
		t.Fatalf("expected default track %s, got %s", repo.trackID, b.Problem.TrackID)
	}
	if b.Problem.Platform != ProblemPlatform("leetcode") {
		t.Fatalf("expected default platform leetcode, got %s", b.Problem.Platform)
	}
	if b.Problem.EstimatedMinutes != 30 {
		t.Fatalf("expected default estimated_minutes 30, got %d", b.Problem.EstimatedMinutes)
	}
}

func TestAdminCreateProblem_DuplicateSlugConflict(t *testing.T) {
	repo := newFakeWriteRepo()
	svc := NewAdminService(repo)
	in := ProblemInput{Slug: "dup", Title: "Dup", Difficulty: "medium"}
	if _, err := svc.CreateProblem(context.Background(), in); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.CreateProblem(context.Background(), in); err != ErrConflict {
		t.Fatalf("expected ErrConflict on duplicate slug, got %v", err)
	}
}

func TestAdminCreateProblem_InvalidDifficulty(t *testing.T) {
	svc := NewAdminService(newFakeWriteRepo())
	_, err := svc.CreateProblem(context.Background(), ProblemInput{Slug: "s", Title: "t", Difficulty: "impossible"})
	if err != ErrValidation {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAdminCreateProblem_InvalidSource(t *testing.T) {
	svc := NewAdminService(newFakeWriteRepo())
	_, err := svc.CreateProblem(context.Background(), ProblemInput{
		Slug: "s", Title: "t", Difficulty: "easy", Sources: []string{"nope"},
	})
	if err != ErrValidation {
		t.Fatalf("expected ErrValidation for bad source, got %v", err)
	}
}

func TestAdminUpdateProblem_SameSlugAllowed(t *testing.T) {
	repo := newFakeWriteRepo()
	svc := NewAdminService(repo)
	b, err := svc.CreateProblem(context.Background(), ProblemInput{Slug: "keep", Title: "Keep", Difficulty: "easy"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.UpdateProblem(context.Background(), b.Problem.ID, ProblemInput{
		Slug: "keep", Title: "Keep Updated", Difficulty: "hard",
	}); err != nil {
		t.Fatalf("update with same slug should be allowed, got %v", err)
	}
}

func TestAdminDeleteProblem_NotFound(t *testing.T) {
	svc := NewAdminService(newFakeWriteRepo())
	if err := svc.DeleteProblem(context.Background(), uuid.New()); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAdminCreateTopic_ResolvesPillarAndTrack(t *testing.T) {
	repo := newFakeWriteRepo()
	svc := NewAdminService(repo)
	pt := PillarBackendEng
	b, err := svc.CreateTopic(context.Background(), TopicInput{
		PillarType: &pt,
		Slug:       "connection-pooling",
		Name:       "Connection Pooling",
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if b.Topic.PillarID != repo.pillars[PillarBackendEng].ID {
		t.Fatalf("topic pillar not resolved")
	}
	if b.Topic.TrackID != repo.trackID {
		t.Fatalf("topic track not derived from pillar")
	}
	if b.Topic.Difficulty != DifficultyMedium || b.Topic.Priority != Priority("medium") {
		t.Fatalf("expected default difficulty/priority medium")
	}
}

func TestAdminCreateTopic_MissingPillar(t *testing.T) {
	svc := NewAdminService(newFakeWriteRepo())
	_, err := svc.CreateTopic(context.Background(), TopicInput{Slug: "s", Name: "n"})
	if err != ErrValidation {
		t.Fatalf("expected ErrValidation when no pillar given, got %v", err)
	}
}
