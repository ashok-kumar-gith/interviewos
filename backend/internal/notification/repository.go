package notification

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListFilter narrows and paginates a user's notifications.
type ListFilter struct {
	Status *Status // optional status filter (unread/read/dismissed)
	Limit  int     // page size (>0)
	Offset int     // page offset (>=0)
}

// Repository abstracts persistence for the notification domain so the service
// can be unit-tested against a fake. The gorm implementation is gormRepository.
// All reads/writes are scoped to a user_id and are soft-delete aware (GORM
// filters deleted_at IS NULL via gorm.DeletedAt on the model).
type Repository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, userID, id uuid.UUID) (*Notification, error)
	List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Notification, int64, error)
	// MarkRead transitions a single owned, unread notification to read and stamps
	// read_at. It is idempotent: marking an already-read notification returns the
	// current row without error. Returns ErrNotFound if not owned/absent.
	MarkRead(ctx context.Context, userID, id uuid.UUID) (*Notification, error)
	// MarkAllRead transitions every unread notification for the user to read and
	// returns the number of rows updated.
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *gormRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Notification, error) {
	var n Notification
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *gormRepository) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Notification, int64, error) {
	q := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)
	if f.Status != nil {
		q = q.Where("status = ?", *f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Newest-first (covered by idx_notif_user_created).
	q = q.Order("created_at DESC")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var out []Notification
	if err := q.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *gormRepository) MarkRead(ctx context.Context, userID, id uuid.UUID) (*Notification, error) {
	// Confirm ownership/existence first (404 vs no-op distinction).
	existing, err := r.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	// Idempotent: only flip unread → read; leave read/dismissed untouched.
	if existing.Status != StatusUnread {
		return existing, nil
	}
	res := r.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, StatusUnread).
		Updates(map[string]any{"status": StatusRead, "read_at": gorm.Expr("now()")})
	if res.Error != nil {
		return nil, res.Error
	}
	// Reload to return the persisted read_at/updated_at.
	return r.GetByID(ctx, userID, id)
}

func (r *gormRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	res := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND status = ?", userID, StatusUnread).
		Updates(map[string]any{"status": StatusRead, "read_at": gorm.Expr("now()")})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
