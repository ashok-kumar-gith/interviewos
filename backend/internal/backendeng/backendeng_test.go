package backendeng

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/content"
)

// fakeRepo is a minimal content.Repository fake exercising only the methods the
// backendeng service uses (ListTopics, GetTopicBundle).
type fakeRepo struct {
	content.Repository // embed so unused methods panic if ever called
	topics             []content.Topic
	lastFilter         content.TopicFilter
}

func (f *fakeRepo) ListTopics(_ context.Context, filter content.TopicFilter, p content.Page) ([]content.Topic, int64, error) {
	f.lastFilter = filter
	start := p.Offset()
	if start > len(f.topics) {
		start = len(f.topics)
	}
	end := start + p.PageSize
	if end > len(f.topics) {
		end = len(f.topics)
	}
	return f.topics[start:end], int64(len(f.topics)), nil
}

func (f *fakeRepo) GetTopicBundle(_ context.Context, id uuid.UUID) (*content.TopicBundle, error) {
	for _, t := range f.topics {
		if t.ID == id {
			return &content.TopicBundle{Topic: t}, nil
		}
	}
	return nil, content.ErrNotFound
}

func TestListTopicsPinsBackendEngineeringPillar(t *testing.T) {
	repo := &fakeRepo{topics: []content.Topic{
		{ID: uuid.New(), Slug: "be-mvcc", Name: "MVCC", Difficulty: content.DifficultyHard},
		{ID: uuid.New(), Slug: "be-kafka", Name: "Kafka", Difficulty: content.DifficultyMedium},
	}}
	svc := NewService(repo)

	res, err := svc.ListTopics(context.Background(), TopicQuery{})
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if repo.lastFilter.PillarType == nil || *repo.lastFilter.PillarType != content.PillarBackendEng {
		t.Fatalf("expected pillar filter pinned to backend_engineering, got %v", repo.lastFilter.PillarType)
	}
	if res.Total != 2 || len(res.Items) != 2 {
		t.Fatalf("expected 2 topics, got total=%d items=%d", res.Total, len(res.Items))
	}
	if res.Page != defaultPage || res.PageSize != defaultPageSize {
		t.Errorf("pagination defaults not applied: %+v", res)
	}
}

func TestGetTopicGuardsForeignPillar(t *testing.T) {
	beID := uuid.New()
	repo := &fakeRepo{topics: []content.Topic{{ID: beID, Slug: "be-mvcc", Name: "MVCC"}}}
	svc := NewService(repo)

	if _, err := svc.GetTopic(context.Background(), beID); err != nil {
		t.Fatalf("expected backend_engineering topic to resolve, got %v", err)
	}
	// A topic id not in the backend_engineering set must 404.
	if _, err := svc.GetTopic(context.Background(), uuid.New()); err != content.ErrNotFound {
		t.Fatalf("expected ErrNotFound for foreign topic, got %v", err)
	}
}

func TestNormalizePageAndParseSort(t *testing.T) {
	if p, ps := normalizePage(0, 0); p != defaultPage || ps != defaultPageSize {
		t.Errorf("defaults not applied: page=%d size=%d", p, ps)
	}
	if _, ps := normalizePage(1, 1000); ps != maxPageSize {
		t.Errorf("page size not clamped, got %d", ps)
	}
	got := parseSort("-difficulty,name,evil")
	if len(got) != 2 {
		t.Fatalf("expected 2 valid sort fields, got %d (%+v)", len(got), got)
	}
	if got[0].Column != "difficulty" || !got[0].Desc {
		t.Errorf("expected -difficulty first, got %+v", got[0])
	}
	if d := difficultyParam("nope"); d != nil {
		t.Errorf("expected nil for invalid difficulty")
	}
}
