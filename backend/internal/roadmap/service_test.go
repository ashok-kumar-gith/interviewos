package roadmap

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/curriculum"
)

// --- fakes ---

type fakeRepo struct {
	active    *Roadmap
	created   *Roadmap
	replaceAt bool
}

func (f *fakeRepo) GetActive(_ context.Context, _ uuid.UUID) (*Roadmap, error) {
	if f.active == nil {
		return nil, ErrNoActiveRoadmap
	}
	return f.active, nil
}
func (f *fakeRepo) GetActiveWithWeeks(ctx context.Context, u uuid.UUID) (*Roadmap, error) {
	return f.GetActive(ctx, u)
}
func (f *fakeRepo) GetWeek(_ context.Context, _, _ uuid.UUID, _ int) (*RoadmapWeek, error) {
	return nil, ErrNotFound
}
func (f *fakeRepo) GetPlanDay(_ context.Context, _ uuid.UUID, _ time.Time) (*PlanDay, error) {
	return nil, ErrNotFound
}
func (f *fakeRepo) CreateGraph(_ context.Context, rm *Roadmap, replaceActive bool) error {
	rm.ID = uuid.New()
	f.created = rm
	f.replaceAt = replaceActive
	f.active = rm
	return nil
}

type fakeProfiles struct{ prof *Profile }

func (f *fakeProfiles) GetProfile(_ context.Context, _ uuid.UUID) (*Profile, error) {
	if f.prof == nil {
		return nil, ErrProfileRequired
	}
	return f.prof, nil
}

type fakeContent struct{ in curriculum.Input }

func (f *fakeContent) LoadEngineInput(_ context.Context, _ uuid.UUID, _ *uuid.UUID, p curriculum.Profile) (curriculum.Input, error) {
	in := f.in
	in.Profile = p
	return in, nil
}

func fixtureContent() curriculum.Input {
	t1 := uuid.New()
	return curriculum.Input{
		Pillars: []curriculum.PillarMeta{{Type: curriculum.PillarDSA, Weight: 1.5}},
		Topics: []curriculum.Topic{
			{ID: t1, Pillar: curriculum.PillarDSA, Name: "Arrays", Difficulty: curriculum.DifficultyEasy, Priority: curriculum.PriorityHigh, EstimatedHours: 4},
		},
		Problems: []curriculum.Problem{
			{ID: uuid.New(), TopicID: t1, Title: "Two Sum", Difficulty: curriculum.DifficultyEasy, EstimatedMinutes: 30, FrequencyScore: 90},
		},
		CompanyMul: map[curriculum.PillarType]float64{},
	}
}

func newProfile() *Profile {
	return &Profile{
		ID:           uuid.New(),
		TrackID:      uuid.New(),
		HoursPerWeek: 10,
		StartDate:    time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC),
		TargetWeeks:  12,
		PillarStrengths: map[string]int{"dsa": 3},
	}
}

func TestService_Generate_PersistsGraph(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(ServiceConfig{
		Repo:     repo,
		Profiles: &fakeProfiles{prof: newProfile()},
		Content:  &fakeContent{in: fixtureContent()},
	})
	rm, err := svc.GenerateRoadmap(context.Background(), uuid.New(), false)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if rm == nil || len(rm.Weeks) != 12 {
		t.Fatalf("expected 12 weeks, got %v", rm)
	}
	if repo.created == nil {
		t.Fatal("CreateGraph not called")
	}
	if !rm.IsActive || rm.Status != "active" || rm.GeneratedBy != "engine" {
		t.Fatalf("roadmap metadata wrong: %+v", rm)
	}
}

func TestService_Generate_ConflictWhenActiveExists(t *testing.T) {
	repo := &fakeRepo{active: &Roadmap{ID: uuid.New(), IsActive: true}}
	svc := NewService(ServiceConfig{
		Repo:     repo,
		Profiles: &fakeProfiles{prof: newProfile()},
		Content:  &fakeContent{in: fixtureContent()},
	})
	_, err := svc.GenerateRoadmap(context.Background(), uuid.New(), false)
	if err != ErrActiveRoadmapExists {
		t.Fatalf("want ErrActiveRoadmapExists, got %v", err)
	}
}

func TestService_Generate_RegenerateReplacesActive(t *testing.T) {
	repo := &fakeRepo{active: &Roadmap{ID: uuid.New(), IsActive: true}}
	svc := NewService(ServiceConfig{
		Repo:     repo,
		Profiles: &fakeProfiles{prof: newProfile()},
		Content:  &fakeContent{in: fixtureContent()},
	})
	_, err := svc.GenerateRoadmap(context.Background(), uuid.New(), true)
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if !repo.replaceAt {
		t.Fatal("expected replaceActive=true to be passed to CreateGraph")
	}
}

func TestService_Generate_RequiresProfile(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo:     &fakeRepo{},
		Profiles: &fakeProfiles{}, // no profile
		Content:  &fakeContent{in: fixtureContent()},
	})
	_, err := svc.GenerateRoadmap(context.Background(), uuid.New(), false)
	if err != ErrProfileRequired {
		t.Fatalf("want ErrProfileRequired, got %v", err)
	}
}
