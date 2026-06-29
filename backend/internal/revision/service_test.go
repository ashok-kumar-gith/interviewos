package revision

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepo is an in-memory Repository for service unit tests. It keys items by
// id and enforces the active (user,item_type,item_id) dedupe of Create.
type fakeRepo struct {
	items      map[uuid.UUID]*Item
	createErr  error
	createdNum int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{items: map[uuid.UUID]*Item{}} }

func (f *fakeRepo) Create(_ context.Context, it *Item) (bool, error) {
	if f.createErr != nil {
		return false, f.createErr
	}
	for _, ex := range f.items {
		if ex.DeletedAt.Time.IsZero() && ex.UserID == it.UserID && ex.ItemType == it.ItemType && ex.ItemID == it.ItemID {
			return false, nil // active dupe — no-op
		}
	}
	if it.ID == uuid.Nil {
		it.ID = uuid.New()
	}
	cp := *it
	f.items[it.ID] = &cp
	f.createdNum++
	return true, nil
}

func (f *fakeRepo) GetByID(_ context.Context, userID, id uuid.UUID) (*Item, error) {
	it, ok := f.items[id]
	if !ok || it.UserID != userID {
		return nil, ErrItemNotFound
	}
	cp := *it
	return &cp, nil
}

func (f *fakeRepo) ListDue(_ context.Context, userID uuid.UUID, fl DueFilter) ([]Item, int64, error) {
	on := dayFloor(fl.OnDate)
	var out []Item
	for _, it := range f.items {
		if it.UserID == userID && it.IsActive && !dayFloor(it.DueAt).After(on) {
			out = append(out, *it)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) Update(_ context.Context, it *Item) error {
	ex, ok := f.items[it.ID]
	if !ok || ex.UserID != it.UserID {
		return ErrItemNotFound
	}
	cp := *it
	f.items[it.ID] = &cp
	return nil
}

func fixedNow() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) }

func newSvc(repo Repository) *Service {
	return NewService(ServiceConfig{Repo: repo, Now: fixedNow})
}

// TestScheduleForCompletion_CreatesStageZeroDuePlusOne verifies a learning
// completion creates a stage-0, interval-1, due-tomorrow active item.
func TestScheduleForCompletion_CreatesStageZeroDuePlusOne(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	uid := uuid.New()
	itemID := uuid.New()

	if err := svc.ScheduleForCompletion(context.Background(), uid, "topic", itemID.String(), "dsa"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if repo.createdNum != 1 {
		t.Fatalf("created = %d, want 1", repo.createdNum)
	}
	var got *Item
	for _, it := range repo.items {
		got = it
	}
	if got.Stage != 0 || got.IntervalDays != 1 || !got.IsActive {
		t.Fatalf("unexpected initial state: %+v", got)
	}
	wantDue := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if !got.DueAt.Equal(wantDue) {
		t.Fatalf("due = %v, want %v", got.DueAt, wantDue)
	}
	if got.PillarType != "dsa" || got.ItemType != ItemTopic {
		t.Fatalf("unexpected poly fields: %+v", got)
	}
}

// TestScheduleForCompletion_Idempotent ensures re-completion does not duplicate.
func TestScheduleForCompletion_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	uid := uuid.New()
	itemID := uuid.New().String()

	for i := 0; i < 3; i++ {
		if err := svc.ScheduleForCompletion(context.Background(), uid, "subtopic", itemID, "dsa"); err != nil {
			t.Fatalf("schedule %d: %v", i, err)
		}
	}
	if repo.createdNum != 1 {
		t.Fatalf("created = %d, want 1 (dedupe)", repo.createdNum)
	}
}

// TestScheduleForCompletion_NonSchedulableNoop ensures solve/mock/revise content
// types (resource/behavioral_story) are silently skipped.
func TestScheduleForCompletion_NonSchedulableNoop(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	if err := svc.ScheduleForCompletion(context.Background(), uuid.New(), "behavioral_story", uuid.New().String(), "behavioral"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if repo.createdNum != 0 {
		t.Fatalf("created = %d, want 0 (non-schedulable)", repo.createdNum)
	}
}

// TestRecall_AdvanceAndReset drives a full ladder via the service: correct
// advances 0->1->2->3->4, the next correct graduates; an incorrect on a fresh
// item resets and bumps lapse_count.
func TestRecall_AdvanceAndReset(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	uid := uuid.New()
	itemID := uuid.New().String()
	if err := svc.ScheduleForCompletion(context.Background(), uid, "topic", itemID, "dsa"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	var id uuid.UUID
	for k := range repo.items {
		id = k
	}

	wantIntervals := []int{3, 7, 15, 30}
	for i, wantInterval := range wantIntervals {
		it, err := svc.Recall(context.Background(), uid, id, RecallCorrect)
		if err != nil {
			t.Fatalf("recall %d: %v", i, err)
		}
		if it.Stage != i+1 {
			t.Fatalf("recall %d: stage = %d, want %d", i, it.Stage, i+1)
		}
		if it.IntervalDays != wantInterval {
			t.Fatalf("recall %d: interval = %d, want %d", i, it.IntervalDays, wantInterval)
		}
		if it.ReviewCount != i+1 {
			t.Fatalf("recall %d: review_count = %d, want %d", i, it.ReviewCount, i+1)
		}
		if !it.IsActive {
			t.Fatalf("recall %d: should still be active", i)
		}
	}

	// Final correct at stage 4 graduates.
	grad, err := svc.Recall(context.Background(), uid, id, RecallCorrect)
	if err != nil {
		t.Fatalf("graduate recall: %v", err)
	}
	if grad.IsActive {
		t.Fatal("expected graduated (is_active=false)")
	}
	if grad.LastRecall == nil || *grad.LastRecall != RecallCorrect {
		t.Fatalf("graduate last_recall = %v, want correct", grad.LastRecall)
	}

	// A recall against a graduated item is rejected.
	if _, err := svc.Recall(context.Background(), uid, id, RecallCorrect); err != ErrItemGraduated {
		t.Fatalf("recall on graduated: err = %v, want ErrItemGraduated", err)
	}
}

func TestRecall_IncorrectResetsAndLapses(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	uid := uuid.New()
	if err := svc.ScheduleForCompletion(context.Background(), uid, "topic", uuid.New().String(), "dsa"); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	var id uuid.UUID
	for k := range repo.items {
		id = k
	}
	// Advance twice, then miss.
	if _, err := svc.Recall(context.Background(), uid, id, RecallCorrect); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Recall(context.Background(), uid, id, RecallCorrect); err != nil {
		t.Fatal(err)
	}
	it, err := svc.Recall(context.Background(), uid, id, RecallIncorrect)
	if err != nil {
		t.Fatalf("incorrect recall: %v", err)
	}
	if it.Stage != 0 || it.IntervalDays != 1 {
		t.Fatalf("reset state = stage %d interval %d, want 0/1", it.Stage, it.IntervalDays)
	}
	if it.LapseCount != 1 {
		t.Fatalf("lapse_count = %d, want 1", it.LapseCount)
	}
	if it.ReviewCount != 3 {
		t.Fatalf("review_count = %d, want 3", it.ReviewCount)
	}
	wantDue := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if !it.DueAt.Equal(wantDue) {
		t.Fatalf("due = %v, want %v", it.DueAt, wantDue)
	}
}

func TestRecall_Validation(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	if _, err := svc.Recall(context.Background(), uuid.New(), uuid.New(), RecallResult("maybe")); err != ErrInvalidRecall {
		t.Fatalf("invalid recall: err = %v, want ErrInvalidRecall", err)
	}
	if _, err := svc.Recall(context.Background(), uuid.New(), uuid.New(), RecallCorrect); err != ErrItemNotFound {
		t.Fatalf("missing item: err = %v, want ErrItemNotFound", err)
	}
}

// TestRecall_OwnershipEnforced ensures one user cannot recall another's item.
func TestRecall_OwnershipEnforced(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	owner := uuid.New()
	if err := svc.ScheduleForCompletion(context.Background(), owner, "topic", uuid.New().String(), "dsa"); err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	for k := range repo.items {
		id = k
	}
	if _, err := svc.Recall(context.Background(), uuid.New(), id, RecallCorrect); err != ErrItemNotFound {
		t.Fatalf("cross-user recall: err = %v, want ErrItemNotFound", err)
	}
}

func TestDue_OnlyActiveDueReturned(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo)
	uid := uuid.New()

	// Due today.
	repo.items[uuid.New()] = &Item{ID: uuid.New(), UserID: uid, ItemType: ItemTopic, ItemID: uuid.New(), IsActive: true, DueAt: fixedNow().AddDate(0, 0, -1)}
	// Due in the future (not yet due).
	repo.items[uuid.New()] = &Item{ID: uuid.New(), UserID: uid, ItemType: ItemTopic, ItemID: uuid.New(), IsActive: true, DueAt: fixedNow().AddDate(0, 0, 5)}
	// Graduated (inactive).
	repo.items[uuid.New()] = &Item{ID: uuid.New(), UserID: uid, ItemType: ItemTopic, ItemID: uuid.New(), IsActive: false, DueAt: fixedNow().AddDate(0, 0, -1)}

	items, total, err := svc.Due(context.Background(), uid, DueParams{})
	if err != nil {
		t.Fatalf("due: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("due count = %d (len %d), want 1", total, len(items))
	}
}
