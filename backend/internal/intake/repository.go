package intake

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository abstracts persistence for the intake domain so the service can be
// unit-tested against a fake. The gorm implementation is gormRepository.
type Repository interface {
	// GetByUserID returns the active (non-deleted) profile for a user, or
	// ErrProfileNotFound if none exists.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	// Upsert inserts or updates the user's profile keyed on the active-user
	// unique index. On return p is populated with the persisted row.
	Upsert(ctx context.Context, p *Profile) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	// gorm.DeletedAt on the model auto-filters deleted_at IS NULL.
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// upsertColumns are the user-supplied fields written on an update. id, user_id,
// created_at and deleted_at are never touched here; updated_at is maintained by
// the set_updated_at() trigger.
var upsertColumns = []string{
	"track_id", "years_experience", "target_company_id", "target_role",
	"target_level", "hours_per_week", "start_date", "target_weeks",
	"pillar_strengths", "timezone", "onboarding_completed_at", "intake_answers",
}

func (r *gormRepository) Upsert(ctx context.Context, p *Profile) error {
	// Find-then-create/update inside a transaction. This honors the partial
	// unique index (one active profile per user) without relying on ON CONFLICT
	// arbiter-predicate inference, which GORM cannot express for a partial index.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Profile
		err := tx.Where("user_id = ?", p.UserID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if cerr := tx.Create(p).Error; cerr != nil {
				return cerr
			}
		case err != nil:
			return err
		default:
			p.ID = existing.ID
			if uerr := tx.Model(&Profile{}).
				Where("id = ?", existing.ID).
				Select(upsertColumns).
				Updates(p).Error; uerr != nil {
				return uerr
			}
		}
		// Reload the canonical persisted row (db-generated id, trigger-updated
		// timestamps).
		return tx.Where("user_id = ?", p.UserID).First(p).Error
	})
}
