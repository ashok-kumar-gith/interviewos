package resume

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository for unit tests.
type fakeRepository struct {
	profiles map[uuid.UUID]*Profile    // keyed by user_id
	projects map[uuid.UUID]*Project    // keyed by project id
	files    map[uuid.UUID]*ResumeFile // keyed by user_id
}

func newFakeRepo() *fakeRepository {
	return &fakeRepository{
		profiles: map[uuid.UUID]*Profile{},
		projects: map[uuid.UUID]*Project{},
	}
}

func (f *fakeRepository) GetProfileByUserID(_ context.Context, userID uuid.UUID) (*Profile, error) {
	p, ok := f.profiles[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	cp := *p
	cp.Projects = f.projectsForProfile(p.ID)
	return &cp, nil
}

func (f *fakeRepository) projectsForProfile(profileID uuid.UUID) []Project {
	var out []Project
	for _, pr := range f.projects {
		if pr.ResumeProfileID == profileID {
			out = append(out, *pr)
		}
	}
	return out
}

func (f *fakeRepository) CreateProfile(_ context.Context, p *Profile) error {
	if _, ok := f.profiles[p.UserID]; ok {
		return errors.New("duplicate profile for user")
	}
	cp := *p
	f.profiles[p.UserID] = &cp
	return nil
}

func (f *fakeRepository) UpdateProfile(_ context.Context, p *Profile) error {
	existing, ok := f.profiles[p.UserID]
	if !ok {
		return ErrProfileNotFound
	}
	existing.Headline = p.Headline
	existing.Summary = p.Summary
	existing.YearsExperience = p.YearsExperience
	existing.Skills = p.Skills
	existing.TargetKeywords = p.TargetKeywords
	return nil
}

func (f *fakeRepository) UpdateScore(_ context.Context, profileID uuid.UUID, atsScore float64, aiFeedback []byte) error {
	for _, p := range f.profiles {
		if p.ID == profileID {
			p.ATSScore = &atsScore
			p.AIFeedback = aiFeedback
			return nil
		}
	}
	return ErrProfileNotFound
}

func (f *fakeRepository) ListProjects(_ context.Context, profileID uuid.UUID) ([]Project, error) {
	return f.projectsForProfile(profileID), nil
}

func (f *fakeRepository) GetProject(_ context.Context, userID, projectID uuid.UUID) (*Project, error) {
	pr, ok := f.projects[projectID]
	if !ok || pr.UserID != userID {
		return nil, ErrProjectNotFound
	}
	cp := *pr
	return &cp, nil
}

func (f *fakeRepository) CreateProject(_ context.Context, p *Project) error {
	cp := *p
	f.projects[p.ID] = &cp
	return nil
}

func (f *fakeRepository) UpdateProject(_ context.Context, p *Project) error {
	existing, ok := f.projects[p.ID]
	if !ok || existing.UserID != p.UserID {
		return ErrProjectNotFound
	}
	cp := *p
	f.projects[p.ID] = &cp
	return nil
}

func (f *fakeRepository) DeleteProject(_ context.Context, userID, projectID uuid.UUID) error {
	pr, ok := f.projects[projectID]
	if !ok || pr.UserID != userID {
		return ErrProjectNotFound
	}
	delete(f.projects, projectID)
	return nil
}

func (f *fakeRepository) DeleteProfile(_ context.Context, userID uuid.UUID) error {
	p, ok := f.profiles[userID]
	if !ok {
		return ErrProfileNotFound
	}
	for id, pr := range f.projects {
		if pr.ResumeProfileID == p.ID {
			delete(f.projects, id)
		}
	}
	delete(f.profiles, userID)
	return nil
}

func (f *fakeRepository) UpsertFile(_ context.Context, file *ResumeFile) error {
	if f.files == nil {
		f.files = map[uuid.UUID]*ResumeFile{}
	}
	cp := *file
	f.files[file.UserID] = &cp
	return nil
}

func (f *fakeRepository) GetFile(_ context.Context, userID uuid.UUID) (*ResumeFile, error) {
	file, ok := f.files[userID]
	if !ok {
		return nil, ErrFileNotFound
	}
	cp := *file
	return &cp, nil
}

func (f *fakeRepository) DeleteFile(_ context.Context, userID uuid.UUID) error {
	if _, ok := f.files[userID]; !ok {
		return ErrFileNotFound
	}
	delete(f.files, userID)
	return nil
}

func strptr(s string) *string { return &s }

// ---- Profile upsert ----

func TestService_UpsertProfile_CreateThenUpdate(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	uid := uuid.New()

	created, err := svc.UpsertProfile(ctx, uid, ProfileInput{
		Headline: strptr("Senior Backend Engineer"),
		Skills:   []string{"Go", "Postgres"},
	})
	require.NoError(t, err)
	require.Equal(t, uid, created.UserID)
	require.Equal(t, "Senior Backend Engineer", *created.Headline)
	require.Equal(t, StringArray{"Go", "Postgres"}, created.Skills)

	updated, err := svc.UpsertProfile(ctx, uid, ProfileInput{
		Headline:       strptr("Staff Engineer"),
		Skills:         []string{"Go", "Kafka", "Kubernetes"},
		TargetKeywords: []string{"distributed systems"},
	})
	require.NoError(t, err)
	// Same row (single profile per user).
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, "Staff Engineer", *updated.Headline)
	require.Len(t, updated.Skills, 3)
	require.Equal(t, StringArray{"distributed systems"}, updated.TargetKeywords)
}

func TestService_UpsertProfile_Validation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	neg := -2.0
	_, err := svc.UpsertProfile(context.Background(), uuid.New(), ProfileInput{YearsExperience: &neg})
	require.ErrorIs(t, err, ErrValidation)
}

func TestService_GetProfile_NotFound(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	_, err := svc.GetProfile(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrProfileNotFound)
}

// ---- Project ownership & validation ----

func TestService_Project_OwnershipEnforced(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()

	owner := uuid.New()
	intruder := uuid.New()
	_, err := svc.UpsertProfile(ctx, owner, ProfileInput{Headline: strptr("Eng")})
	require.NoError(t, err)

	proj, err := svc.CreateProject(ctx, owner, ProjectInput{Name: "Payments platform"})
	require.NoError(t, err)
	require.Equal(t, owner, proj.UserID)

	// Intruder cannot read, update, or delete the owner's project.
	_, err = svc.UpdateProject(ctx, intruder, proj.ID, ProjectInput{Name: "Hijacked"})
	require.ErrorIs(t, err, ErrProjectNotFound)

	err = svc.DeleteProject(ctx, intruder, proj.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)

	// Owner can update and delete.
	updated, err := svc.UpdateProject(ctx, owner, proj.ID, ProjectInput{Name: "Payments v2", SortOrder: 1})
	require.NoError(t, err)
	require.Equal(t, "Payments v2", updated.Name)
	require.Equal(t, 1, updated.SortOrder)

	require.NoError(t, svc.DeleteProject(ctx, owner, proj.ID))
	_, err = repo.GetProject(ctx, owner, proj.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)
}

func TestService_CreateProject_Validation(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(ServiceConfig{Repo: repo})
	ctx := context.Background()
	uid := uuid.New()
	_, _ = svc.UpsertProfile(ctx, uid, ProfileInput{})

	_, err := svc.CreateProject(ctx, uid, ProjectInput{Name: "  "})
	require.ErrorIs(t, err, ErrValidation)

	badDate := "2020/01/01"
	_, err = svc.CreateProject(ctx, uid, ProjectInput{Name: "X", StartDate: &badDate})
	require.ErrorIs(t, err, ErrValidation)
}

func TestService_CreateProject_RequiresProfile(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	_, err := svc.CreateProject(context.Background(), uuid.New(), ProjectInput{Name: "X"})
	require.ErrorIs(t, err, ErrProfileNotFound)
}

// ---- Scoring: good vs weak resume ----

func seedResume(t *testing.T, svc *Service, repo *fakeRepository, uid uuid.UUID, p ProfileInput, projects []ProjectInput) {
	t.Helper()
	ctx := context.Background()
	_, err := svc.UpsertProfile(ctx, uid, p)
	require.NoError(t, err)
	for _, proj := range projects {
		_, err := svc.CreateProject(ctx, uid, proj)
		require.NoError(t, err)
	}
}

func TestService_Score_GoodVsWeak(t *testing.T) {
	ctx := context.Background()

	// Strong resume: headline, summary, skills, target keywords all covered,
	// bullets start with action verbs and carry quantified impact at good length.
	goodRepo := newFakeRepo()
	goodSvc := NewService(ServiceConfig{Repo: goodRepo})
	goodUID := uuid.New()
	seedResume(t, goodSvc, goodRepo, goodUID, ProfileInput{
		Headline:       strptr("Senior Backend Engineer"),
		Summary:        strptr("Backend engineer with deep experience in distributed systems, Go, and Kafka."),
		Skills:         []string{"Go", "Kafka", "Postgres", "Kubernetes"},
		TargetKeywords: []string{"Go", "Kafka", "Postgres", "Kubernetes"},
	}, []ProjectInput{
		{
			Name:        "Order Service",
			Description: strptr("Built a Go order service handling 50k requests per second across 12 regions reliably"),
			Impact:      strptr("Reduced p99 latency by 40% and cut infra cost by $200k annually for the team"),
			Metrics:     []string{"Scaled throughput from 5k to 50k rps over two quarters with zero downtime"},
		},
	})

	good, err := goodSvc.Score(ctx, goodUID)
	require.NoError(t, err)
	require.False(t, good.UsedFallback)
	require.NotEmpty(t, good.Breakdown)
	require.ElementsMatch(t, []string{"go", "kafka", "postgres", "kubernetes"}, good.KeywordMatches)
	require.Empty(t, good.MissingKeywords)

	// Weak resume: no summary/skills, no target keywords matched, short bullet,
	// no action verb, no metrics.
	weakRepo := newFakeRepo()
	weakSvc := NewService(ServiceConfig{Repo: weakRepo})
	weakUID := uuid.New()
	seedResume(t, weakSvc, weakRepo, weakUID, ProfileInput{
		TargetKeywords: []string{"Go", "Kafka", "Rust", "GraphQL"},
	}, []ProjectInput{
		{
			Name:        "Thing",
			Description: strptr("did stuff"),
		},
	})

	weak, err := weakSvc.Score(ctx, weakUID)
	require.NoError(t, err)
	require.NotEmpty(t, weak.MissingKeywords)
	require.NotEmpty(t, weak.Suggestions)

	// The strong resume must score materially higher than the weak one.
	require.Greater(t, good.ATSScore, weak.ATSScore)
	require.GreaterOrEqual(t, good.ATSScore, 70.0)
	require.Less(t, weak.ATSScore, 40.0)

	// Score is persisted on the profile.
	reloaded, err := goodRepo.GetProfileByUserID(ctx, goodUID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.ATSScore)
	require.Equal(t, good.ATSScore, *reloaded.ATSScore)
	require.NotEmpty(t, reloaded.AIFeedback)
}

func TestService_Score_NoProfile(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: newFakeRepo()})
	_, err := svc.Score(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrProfileNotFound)
}

// ---- Scorer directly: breakdown bounds and keyword math ----

func TestRuleScorer_BreakdownBounded(t *testing.T) {
	sc := NewRuleScorer()
	res, err := sc.Score(context.Background(), ScoreInput{
		Headline:       "Engineer",
		Summary:        "Summary text here that is reasonably descriptive of the work.",
		Skills:         []string{"Go"},
		TargetKeywords: []string{"Go", "Rust"},
		Projects:       1,
		Bullets: []string{
			"Led the redesign of the billing pipeline reducing failures by 30% across all tenants",
		},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, res.ATSScore, 0.0)
	require.LessOrEqual(t, res.ATSScore, 100.0)
	for _, b := range res.Breakdown {
		require.GreaterOrEqual(t, b.Score, 0.0)
		require.LessOrEqual(t, b.Score, b.Max)
	}
	require.Contains(t, res.KeywordMatches, "go")
	require.Contains(t, res.MissingKeywords, "rust")
}
