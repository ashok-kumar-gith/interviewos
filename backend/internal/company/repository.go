package company

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository abstracts persistence for Company Mode so the service is
// unit-testable against a fake. The GORM implementation is gormRepository.
type Repository interface {
	// GetProfile returns the user's active profile, or ErrProfileNotFound.
	GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error)
	// CompanyExists reports whether a (non-deleted) company with this id exists.
	CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error)
	// SetTargetCompany sets the user profile's target_company_id and returns the
	// updated profile. A nil companyID clears the target.
	SetTargetCompany(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID, now time.Time) (*Profile, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Table("companies").
		Where("id = ? AND deleted_at IS NULL", companyID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRepository) SetTargetCompany(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID, now time.Time) (*Profile, error) {
	var updated *Profile
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Profile
		ferr := tx.WithContext(ctx).
			Where("user_id = ? AND deleted_at IS NULL", userID).
			Order("created_at DESC").First(&p).Error
		if errors.Is(ferr, gorm.ErrRecordNotFound) {
			return ErrProfileNotFound
		}
		if ferr != nil {
			return ferr
		}
		if err := tx.Model(&Profile{}).Where("id = ?", p.ID).
			Updates(map[string]any{"target_company_id": companyID, "updated_at": now}).Error; err != nil {
			return err
		}
		var fresh Profile
		if err := tx.WithContext(ctx).Where("id = ?", p.ID).First(&fresh).Error; err != nil {
			return err
		}
		updated = &fresh
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
