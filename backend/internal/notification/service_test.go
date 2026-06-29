package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// fakeRepository is an in-memory Repository for unit tests. It enforces the same
// ownership scoping as the gorm implementation so the service's ownership checks
// and mark-read logic are exercised.
type fakeRepository struct {
	items     map[uuid.UUID]*Notification
	createErr error
}

func newFakeRepo() *fakeRepository {
	return &fakeRepository{items: map[uuid.UUID]*Notification{}}
}

func (f *fakeRepository) Create(_ context.Context, n *Notification) error {
	if f.createErr != nil {
		return f.createErr
	}
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	cp := *n
	f.items[n.ID] = &cp
	return nil
}

func (f *fakeRepository) UpsertByDedupKey(_ context.Context, n *Notification) (bool, *Notification, error) {
	if f.createErr != nil {
		return false, nil, f.createErr
	}
	if n.DedupKey != nil {
		for _, existing := range f.items {
			if existing.UserID == n.UserID && existing.DedupKey != nil && *existing.DedupKey == *n.DedupKey {
				cp := *existing
				return false, &cp, nil
			}
		}
	}
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	cp := *n
	f.items[n.ID] = &cp
	out := cp
	return true, &out, nil
}

func (f *fakeRepository) GetByID(_ context.Context, userID, id uuid.UUID) (*Notification, error) {
	n, ok := f.items[id]
	if !ok || n.UserID != userID {
		return nil, ErrNotFound
	}
	cp := *n
	return &cp, nil
}

func (f *fakeRepository) List(_ context.Context, userID uuid.UUID, fl ListFilter) ([]Notification, int64, error) {
	var matched []Notification
	for _, n := range f.items {
		if n.UserID != userID {
			continue
		}
		if fl.Status != nil && n.Status != *fl.Status {
			continue
		}
		matched = append(matched, *n)
	}
	total := int64(len(matched))
	return matched, total, nil
}

func (f *fakeRepository) MarkRead(_ context.Context, userID, id uuid.UUID) (*Notification, error) {
	n, ok := f.items[id]
	if !ok || n.UserID != userID {
		return nil, ErrNotFound
	}
	if n.Status == StatusUnread {
		n.Status = StatusRead
		now := time.Now()
		n.ReadAt = &now
	}
	cp := *n
	return &cp, nil
}

func (f *fakeRepository) MarkAllRead(_ context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	for _, n := range f.items {
		if n.UserID != userID || n.Status != StatusUnread {
			continue
		}
		n.Status = StatusRead
		now := time.Now()
		n.ReadAt = &now
		count++
	}
	return count, nil
}

func newService() (*Service, *fakeRepository) {
	repo := newFakeRepo()
	return NewService(ServiceConfig{Repo: repo}), repo
}

func validCreate(uid uuid.UUID) CreateInput {
	return CreateInput{
		UserID:  uid,
		Type:    TypeRevisionDue,
		Title:   "3 items due for revision today",
		Body:    "You have 3 spaced-repetition items due.",
		Payload: map[string]any{"due_count": 3},
	}
}

func TestService_SatisfiesNotifier(t *testing.T) {
	svc, _ := newService()
	var _ Notifier = svc
}

func TestCreate_Valid(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()
	got, err := svc.Create(context.Background(), validCreate(uid))
	require.NoError(t, err)
	require.Equal(t, uid, got.UserID)
	require.Equal(t, TypeRevisionDue, got.Type)
	// Channel defaults to in_app.
	require.Equal(t, ChannelInApp, got.Channel)
	// Status defaults to unread.
	require.Equal(t, StatusUnread, got.Status)
	require.NotNil(t, got.Body)
	require.Equal(t, 3, got.Payload["due_count"])
}

func TestCreate_DefaultsAndEmptyBody(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()
	got, err := svc.Create(context.Background(), CreateInput{
		UserID: uid,
		Type:   TypeSystem,
		Title:  "Welcome",
		// no body, no payload, no channel
	})
	require.NoError(t, err)
	require.Equal(t, ChannelInApp, got.Channel)
	require.Nil(t, got.Body)
	require.NotNil(t, got.Payload)
	require.Empty(t, got.Payload)
}

func TestCreate_ExplicitChannel(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()
	in := validCreate(uid)
	in.Channel = ChannelEmail
	got, err := svc.Create(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, ChannelEmail, got.Channel)
}

func TestCreate_ValidationErrors(t *testing.T) {
	svc, _ := newService()
	uid := uuid.New()

	cases := map[string]CreateInput{
		"missing user":    {Type: TypeSystem, Title: "x"},
		"missing type":    {UserID: uid, Title: "x"},
		"invalid type":    {UserID: uid, Type: Type("bogus"), Title: "x"},
		"invalid channel": {UserID: uid, Type: TypeSystem, Channel: Channel("sms"), Title: "x"},
		"missing title":   {UserID: uid, Type: TypeSystem},
		"long title":      {UserID: uid, Type: TypeSystem, Title: longString(maxTitleLen + 1)},
		"long body":       {UserID: uid, Type: TypeSystem, Title: "x", Body: longString(maxBodyLen + 1)},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), in)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrValidation)
			var ve *ValidationError
			require.True(t, errors.As(err, &ve))
			require.NotEmpty(t, ve.Fields)
		})
	}
}

func TestCreate_RepoError(t *testing.T) {
	svc, repo := newService()
	repo.createErr = errors.New("db down")
	_, err := svc.Create(context.Background(), validCreate(uuid.New()))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrValidation)
}

func TestList_ScopedAndFiltered(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()

	// owner: one unread, one read.
	a, err := svc.Create(context.Background(), validCreate(owner))
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), validCreate(owner))
	require.NoError(t, err)
	_, err = svc.MarkRead(context.Background(), owner, a.ID)
	require.NoError(t, err)

	// other user's notification must not appear.
	_, err = svc.Create(context.Background(), validCreate(other))
	require.NoError(t, err)

	items, total, err := svc.List(context.Background(), owner, ListFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	unread := StatusUnread
	items, total, err = svc.List(context.Background(), owner, ListFilter{Status: &unread})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, StatusUnread, items[0].Status)

	// other user sees only their own.
	_, total, err = svc.List(context.Background(), other, ListFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
}

func TestList_InvalidStatus(t *testing.T) {
	svc, _ := newService()
	bogus := Status("archived")
	_, _, err := svc.List(context.Background(), uuid.New(), ListFilter{Status: &bogus})
	require.ErrorIs(t, err, ErrValidation)
}

func TestList_PaginationClamps(t *testing.T) {
	svc, repo := newService()
	uid := uuid.New()
	// Service should clamp Limit to 100 and floor Offset to 0; the fake captures
	// the filter it receives via a wrapper.
	captured := &capturingRepo{Repository: repo}
	svc = NewService(ServiceConfig{Repo: captured})
	_, _, err := svc.List(context.Background(), uid, ListFilter{Limit: 9999, Offset: -5})
	require.NoError(t, err)
	require.Equal(t, 100, captured.lastFilter.Limit)
	require.Equal(t, 0, captured.lastFilter.Offset)

	_, _, err = svc.List(context.Background(), uid, ListFilter{Limit: 0})
	require.NoError(t, err)
	require.Equal(t, 20, captured.lastFilter.Limit)
}

func TestMarkRead_OwnershipAndIdempotent(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()
	n, err := svc.Create(context.Background(), validCreate(owner))
	require.NoError(t, err)

	// Non-owner gets not-found (no existence leak).
	_, err = svc.MarkRead(context.Background(), other, n.ID)
	require.ErrorIs(t, err, ErrNotFound)

	// Owner marks read.
	got, err := svc.MarkRead(context.Background(), owner, n.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRead, got.Status)
	require.NotNil(t, got.ReadAt)

	// Idempotent: marking again succeeds and stays read.
	again, err := svc.MarkRead(context.Background(), owner, n.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRead, again.Status)
}

func TestMarkRead_NotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.MarkRead(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMarkAllRead(t *testing.T) {
	svc, _ := newService()
	owner := uuid.New()
	other := uuid.New()

	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), validCreate(owner))
		require.NoError(t, err)
	}
	// Another user's unread must not be touched.
	on, err := svc.Create(context.Background(), validCreate(other))
	require.NoError(t, err)

	count, err := svc.MarkAllRead(context.Background(), owner)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)

	// owner has zero unread now.
	unread := StatusUnread
	_, total, err := svc.List(context.Background(), owner, ListFilter{Status: &unread})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	// other user's notification is still unread.
	got, err := svc.MarkRead(context.Background(), other, on.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ReadAt) // confirms it was unread until now

	// Marking all again on owner is a no-op (zero rows).
	count, err = svc.MarkAllRead(context.Background(), owner)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

// --- helpers ---

func longString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// capturingRepo records the ListFilter passed through so pagination clamping in
// the service can be asserted.
type capturingRepo struct {
	Repository
	lastFilter ListFilter
}

func (c *capturingRepo) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Notification, int64, error) {
	c.lastFilter = f
	return c.Repository.List(ctx, userID, f)
}
