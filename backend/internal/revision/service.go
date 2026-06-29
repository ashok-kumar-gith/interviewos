package revision

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CompletionScheduler is the narrow port the progress module depends on to
// schedule a revision item when a learning task is completed. Keeping it small
// (and satisfied by *Service) lets progress stay decoupled and nil-safe.
type CompletionScheduler interface {
	// ScheduleForCompletion creates a stage-0 revision item due +1 day for the
	// completed learning item, idempotent on (user, item_type, item_id). It is a
	// no-op (nil error) for non-schedulable item types.
	ScheduleForCompletion(ctx context.Context, userID uuid.UUID, itemType, itemID, pillarType string) error
}

// Service implements the revision use-cases. It depends only on the Repository
// interface and a clock, so it is unit-testable against fakes.
type Service struct {
	repo Repository
	now  func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo Repository
	Now  func() time.Time
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Service{repo: cfg.Repo, now: nowFn}
}

// Compile-time assertion that *Service satisfies the completion port.
var _ CompletionScheduler = (*Service)(nil)

// DueParams paginates the due listing and optionally pins the as-of date.
type DueParams struct {
	OnDate time.Time // zero ⇒ today
	Limit  int
	Offset int
}

// Due returns the user's active revision items due on/before the as-of date.
func (s *Service) Due(ctx context.Context, userID uuid.UUID, p DueParams) ([]Item, int64, error) {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	onDate := p.OnDate
	if onDate.IsZero() {
		onDate = s.now()
	}
	return s.repo.ListDue(ctx, userID, DueFilter{OnDate: onDate, Limit: p.Limit, Offset: p.Offset})
}

// Recall applies a binary recall result to an item owned by the user, advancing
// or resetting the ladder per docs/02-SRS.md §6.1, and persists the new state.
// It returns the updated item. A recall against a graduated item is rejected.
func (s *Service) Recall(ctx context.Context, userID, itemID uuid.UUID, recall RecallResult) (*Item, error) {
	if !recall.Valid() {
		return nil, ErrInvalidRecall
	}
	it, err := s.repo.GetByID(ctx, userID, itemID)
	if err != nil {
		return nil, err
	}
	if !it.IsActive {
		return nil, ErrItemGraduated
	}

	now := s.now().UTC()
	tr := Apply(it.Stage, recall, now)

	it.Stage = tr.Stage
	it.IntervalDays = tr.IntervalDays
	it.DueAt = tr.DueAt
	it.IsActive = tr.IsActive
	lr := tr.LastRecall
	it.LastRecall = &lr
	it.LastReviewedAt = &now
	it.ReviewCount++
	if tr.Lapsed {
		it.LapseCount++
	}

	if err := s.repo.Update(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

// ScheduleForCompletion implements CompletionScheduler. It creates a stage-0
// revision item due +1 day for a completed learning item, deduped on the active
// unique index. Non-schedulable item types are a silent no-op so the progress
// hook can call it unconditionally.
func (s *Service) ScheduleForCompletion(ctx context.Context, userID uuid.UUID, itemType, itemID, pillarType string) error {
	t := ItemType(itemType)
	if !t.Schedulable() {
		return nil
	}
	parsedItem, err := uuid.Parse(itemID)
	if err != nil {
		return ErrItemNotFound
	}
	st := InitialState(s.now())
	it := &Item{
		UserID:       userID,
		ItemType:     t,
		ItemID:       parsedItem,
		PillarType:   pillarType,
		Stage:        st.Stage,
		IntervalDays: st.IntervalDays,
		DueAt:        st.DueAt,
		IsActive:     st.IsActive,
	}
	_, err = s.repo.Create(ctx, it)
	return err
}
