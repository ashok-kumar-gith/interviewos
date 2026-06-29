package revision

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DueFilter paginates the due-items listing.
type DueFilter struct {
	OnDate time.Time // due on/before this local day
	Limit  int       // page size (>0)
	Offset int       // page offset (>=0)
}

// Repository abstracts persistence for the revision domain so the service is
// unit-testable against a fake. The gorm implementation is gormRepository. All
// reads/writes are scoped to a user_id and are soft-delete aware.
type Repository interface {
	// Create inserts a new revision item idempotently on the active unique index
	// (user_id, item_type, item_id). On conflict it leaves the existing active
	// item untouched (FR-REV-007 dedupe). Returns true when a row was created.
	Create(ctx context.Context, it *Item) (created bool, err error)
	// GetByID returns an item owned by the user (title resolved), or
	// ErrItemNotFound.
	GetByID(ctx context.Context, userID, id uuid.UUID) (*Item, error)
	// ListDue returns active items due on/before f.OnDate for the user (titles
	// resolved) plus the total matching count, ordered by due_at then created_at.
	ListDue(ctx context.Context, userID uuid.UUID, f DueFilter) ([]Item, int64, error)
	// Update persists the post-recall mutable state (stage, interval, due_at,
	// is_active, last_recall, last_reviewed_at, review_count, lapse_count) for an
	// item owned by the user. Returns ErrItemNotFound when nothing matched.
	Update(ctx context.Context, it *Item) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, it *Item) (bool, error) {
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			// Matches the partial unique index uq_rev_user_item; do nothing on a
			// live active item so re-completion never duplicates (FR-REV-007).
			Columns:     []clause.Column{{Name: "user_id"}, {Name: "item_type"}, {Name: "item_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			DoNothing:   true,
		}).
		Create(it)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *gormRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Item, error) {
	var it Item
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&it).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	r.resolveTitles(ctx, []*Item{&it})
	return &it, nil
}

func (r *gormRepository) ListDue(ctx context.Context, userID uuid.UUID, f DueFilter) ([]Item, int64, error) {
	onDate := f.OnDate
	if onDate.IsZero() {
		onDate = time.Now()
	}
	q := r.db.WithContext(ctx).Model(&Item{}).
		Where("user_id = ? AND is_active = true AND due_at <= ?", userID, dayFloor(onDate).Format("2006-01-02"))

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("due_at ASC, created_at ASC")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var out []Item
	if err := q.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	ptrs := make([]*Item, len(out))
	for i := range out {
		ptrs[i] = &out[i]
	}
	r.resolveTitles(ctx, ptrs)
	return out, total, nil
}

func (r *gormRepository) Update(ctx context.Context, it *Item) error {
	res := r.db.WithContext(ctx).Model(&Item{}).
		Where("id = ? AND user_id = ?", it.ID, it.UserID).
		Select("stage", "interval_days", "due_at", "is_active", "last_recall",
			"last_reviewed_at", "review_count", "lapse_count").
		Updates(map[string]any{
			"stage":            it.Stage,
			"interval_days":    it.IntervalDays,
			"due_at":           it.DueAt,
			"is_active":        it.IsActive,
			"last_recall":      it.LastRecall,
			"last_reviewed_at": it.LastReviewedAt,
			"review_count":     it.ReviewCount,
			"lapse_count":      it.LapseCount,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return nil
}

// resolveTitles best-effort populates each item's Title from the underlying
// content table by item_type. A missing source row (or absent table) leaves the
// title empty — the field is optional in the API contract.
func (r *gormRepository) resolveTitles(ctx context.Context, items []*Item) {
	// Group ids by the table/column that names them.
	type src struct {
		table  string
		column string
	}
	titleSources := map[ItemType]src{
		ItemTopic:         {table: "topics", column: "name"},
		ItemSubtopic:      {table: "subtopics", column: "name"},
		ItemProblem:       {table: "problems", column: "title"},
		ItemDesignProblem: {table: "design_problems", column: "title"},
		ItemLLDProblem:    {table: "lld_problems", column: "title"},
	}
	byType := map[ItemType][]uuid.UUID{}
	for _, it := range items {
		byType[it.ItemType] = append(byType[it.ItemType], it.ItemID)
	}
	for t, ids := range byType {
		s, ok := titleSources[t]
		if !ok || len(ids) == 0 {
			continue
		}
		if !r.db.Migrator().HasTable(s.table) {
			continue
		}
		type row struct {
			ID    uuid.UUID `gorm:"column:id"`
			Title string    `gorm:"column:title"`
		}
		var rows []row
		if err := r.db.WithContext(ctx).Table(s.table).
			Select("id, "+s.column+" AS title").
			Where("id IN ?", ids).Scan(&rows).Error; err != nil {
			continue
		}
		titles := make(map[uuid.UUID]string, len(rows))
		for _, rw := range rows {
			titles[rw.ID] = rw.Title
		}
		for _, it := range items {
			if it.ItemType == t {
				it.Title = titles[it.ItemID]
			}
		}
	}
}
