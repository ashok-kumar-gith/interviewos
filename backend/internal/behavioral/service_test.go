package behavioral

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository for unit tests. It enforces the same
// ownership scoping as the gorm implementation so the service's ownership checks
// are exercised.
type fakeRepository struct {
	stories map[uuid.UUID]*Story
	createErr error
}

func newFakeRepo() *fakeRepository {
	return &fakeRepository{stories: map[uuid.UUID]*Story{}}
}

func (f *fakeRepository) Create(_ context.Context, s *Story) error {
	if f.createErr != nil {
		return f.createErr
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	cp := *s
	f.stories[s.ID] = &cp
	return nil
}

func (f *fakeRepository) GetByID(_ context.Context, userID, id uuid.UUID) (*Story, error) {
	s, ok := f.stories[id]
	if !ok || s.UserID != userID {
		return nil, ErrStoryNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepository) List(_ context.Context, userID uuid.UUID, fl ListFilter) ([]Story, int64, error) {
	var matched []Story
	for _, s := range f.stories {
		if s.UserID != userID {
			continue
		}
		if fl.Theme != nil && s.Theme != *fl.Theme {
			continue
		}
		if fl.Query != "" && !strings.Contains(strings.ToLower(s.Title), strings.ToLower(fl.Query)) {
			continue
		}
		matched = append(matched, *s)
	}
	total := int64(len(matched))
	// Apply offset/limit deterministically is not required for these tests.
	return matched, total, nil
}

func (f *fakeRepository) Update(_ context.Context, s *Story) error {
	existing, ok := f.stories[s.ID]
	if !ok || existing.UserID != s.UserID {
		return ErrStoryNotFound
	}
	cp := *s
	f.stories[s.ID] = &cp
	return nil
}

func (f *fakeRepository) Delete(_ context.Context, userID, id uuid.UUID) error {
	s, ok := f.stories[id]
	if !ok || s.UserID != userID {
		return ErrStoryNotFound
	}
	delete(f.stories, id)
	return nil
}

func newService() (*Service, *fakeRepository) {
	repo := newFakeRepo()
	return NewService(ServiceConfig{Repo: repo}), repo
}

func validCreate() CreateInput {
	return CreateInput{
		Title:     "Rescued the billing migration",
		Theme:     ThemeProjectRescue,
		Situation: "The billing service migration was stalled and at risk.",
		Task:      "I owned getting the migration back on track.",
		Action:    "I designed a phased cutover and automated the data backfill.",
		Result:    "We shipped on time and reduced failures by 80%.",
		Metrics:   "80% fewer failed invoices, 3 days ahead of schedule",
		Tags:      []string{"billing", "migration", "billing"},
	}
}

func TestCreate_Valid(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()
	got, err := svc.Create(context.Background(), uid, validCreate())
	require.NoError(t, err)
	require.Equal(t, uid, got.UserID)
	require.Equal(t, ThemeProjectRescue, got.Theme)
	require.NotNil(t, got.Situation)
	// Tags are de-duplicated.
	require.Equal(t, Tags{"billing", "migration"}, got.Tags)
}

func TestCreate_ValidationErrors(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()

	cases := map[string]CreateInput{
		"missing title":   {Theme: ThemeImpact},
		"missing theme":   {Title: "x"},
		"invalid theme":   {Title: "x", Theme: Theme("bogus")},
		"long title":      {Title: strings.Repeat("a", maxTitleLen+1), Theme: ThemeImpact},
		"long section":    {Title: "x", Theme: ThemeImpact, Action: strings.Repeat("a", maxSectionLen+1)},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), uid, in)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrValidation)
			var ve *ValidationError
			require.True(t, errors.As(err, &ve))
			require.NotEmpty(t, ve.Fields)
		})
	}
}

func TestGet_OwnershipEnforced(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()
	story, err := svc.Create(context.Background(), owner, validCreate())
	require.NoError(t, err)

	// Owner can read.
	got, err := svc.Get(context.Background(), owner, story.ID)
	require.NoError(t, err)
	require.Equal(t, story.ID, got.ID)

	// Another user gets not-found (no cross-user access, no existence leak).
	_, err = svc.Get(context.Background(), other, story.ID)
	require.ErrorIs(t, err, ErrStoryNotFound)
}

func TestUpdate_OwnershipAndAIReset(t *testing.T) {
	svc, repo := newService()
	owner := uuid.New()
	other := uuid.New()
	story, err := svc.Create(context.Background(), owner, validCreate())
	require.NoError(t, err)

	// Simulate prior AI improvement state.
	stored := repo.stories[story.ID]
	stored.AIImproved = true
	score := 50.0
	stored.StrengthScore = &score
	stored.AIFeedback = JSONMap{"suggestions": []any{"x"}}

	// Non-owner cannot update.
	_, err = svc.Update(context.Background(), other, story.ID, validCreate())
	require.ErrorIs(t, err, ErrStoryNotFound)

	// Owner update resets stale AI feedback.
	in := validCreate()
	in.Title = "Updated title"
	updated, err := svc.Update(context.Background(), owner, story.ID, in)
	require.NoError(t, err)
	require.Equal(t, "Updated title", updated.Title)
	require.False(t, updated.AIImproved)
	require.Nil(t, updated.StrengthScore)
	require.Nil(t, updated.AIFeedback)
}

func TestDelete_OwnershipEnforced(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()
	story, err := svc.Create(context.Background(), owner, validCreate())
	require.NoError(t, err)

	require.ErrorIs(t, svc.Delete(context.Background(), other, story.ID), ErrStoryNotFound)
	require.NoError(t, svc.Delete(context.Background(), owner, story.ID))
	_, err = svc.Get(context.Background(), owner, story.ID)
	require.ErrorIs(t, err, ErrStoryNotFound)
}

func TestList_ScopedAndFiltered(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()

	a := validCreate()
	a.Title = "Leadership during outage"
	a.Theme = ThemeLeadership
	_, err := svc.Create(context.Background(), owner, a)
	require.NoError(t, err)

	b := validCreate()
	b.Title = "Owned the rescue"
	b.Theme = ThemeProjectRescue
	_, err = svc.Create(context.Background(), owner, b)
	require.NoError(t, err)

	// Another user's story must not appear.
	_, err = svc.Create(context.Background(), other, validCreate())
	require.NoError(t, err)

	items, total, err := svc.List(context.Background(), owner, ListFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	theme := ThemeLeadership
	items, total, err = svc.List(context.Background(), owner, ListFilter{Theme: &theme})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, ThemeLeadership, items[0].Theme)
}

func TestImprove_PersistsFeedbackAndScore(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()
	story, err := svc.Create(context.Background(), owner, validCreate())
	require.NoError(t, err)

	// Non-owner cannot improve.
	_, _, err = svc.Improve(context.Background(), other, story.ID)
	require.ErrorIs(t, err, ErrStoryNotFound)

	updated, res, err := svc.Improve(context.Background(), owner, story.ID)
	require.NoError(t, err)
	require.True(t, updated.AIImproved)
	require.NotNil(t, updated.StrengthScore)
	require.NotNil(t, updated.AIFeedback)
	require.True(t, res.UsedFallback)
	require.Equal(t, *updated.StrengthScore, res.StrengthScore)
	// A complete, quantified story scores well.
	require.GreaterOrEqual(t, res.StrengthScore, 80.0)
}

func TestDeterministicImprover_FlagsWeaknesses(t *testing.T) {
	imp := NewDeterministicImprover()

	// A weak story: missing sections, no metrics, weak verbs, no first person.
	weak := ImproveInput{
		Title:  "Did stuff",
		Theme:  ThemeOwnership,
		Action: "Helped the team and assisted with the work.",
	}
	res, err := imp.Improve(context.Background(), weak)
	require.NoError(t, err)
	require.True(t, res.UsedFallback)
	require.Less(t, res.StrengthScore, 50.0)

	joined := strings.ToLower(strings.Join(res.Suggestions, " | "))
	require.Contains(t, joined, "situation")
	require.Contains(t, joined, "task")
	require.Contains(t, joined, "result")
	require.Contains(t, joined, "quantify")
	require.Contains(t, joined, "weak action verbs")
	// Names the specific weak verbs.
	require.Contains(t, joined, "helped")
	require.Contains(t, joined, "assisted")

	// When a Result is present but unquantified, the improver offers a concrete
	// metrics scaffold in the improved Result/Metrics.
	withResult := ImproveInput{
		Title:  "Shipped the feature",
		Theme:  ThemeImpact,
		Result: "We launched and users were happier.",
	}
	res2, err := imp.Improve(context.Background(), withResult)
	require.NoError(t, err)
	require.NotEmpty(t, res2.Improved.Metrics)
	require.NotEmpty(t, res2.Improved.Result)
}

func TestDeterministicImprover_Deterministic(t *testing.T) {
	imp := NewDeterministicImprover()
	in := validCreate()
	a, err := imp.Improve(context.Background(), ImproveInput{
		Title: in.Title, Theme: in.Theme, Situation: in.Situation,
		Task: in.Task, Action: in.Action, Result: in.Result, Metrics: in.Metrics,
	})
	require.NoError(t, err)
	b, err := imp.Improve(context.Background(), ImproveInput{
		Title: in.Title, Theme: in.Theme, Situation: in.Situation,
		Task: in.Task, Action: in.Action, Result: in.Result, Metrics: in.Metrics,
	})
	require.NoError(t, err)
	require.Equal(t, a.StrengthScore, b.StrengthScore)
	require.Equal(t, a.Suggestions, b.Suggestions)
}

func TestImproveInline(t *testing.T) {
	svc, _ := newService()
	res, err := svc.ImproveInline(context.Background(), ImproveInput{
		Situation: "ctx", Task: "goal", Action: "I built it", Result: "won, 50% faster",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.UsedFallback)
}

func TestCreate_RepoError(t *testing.T) {
	svc, repo := newService()
	repo.createErr = errors.New("db down")
	_, err := svc.Create(context.Background(), uuid.New(), validCreate())
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrValidation)
}
