package progress

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormProfileReader is a gorm-backed ProfileReader that reads the user's IANA
// timezone from the migration-owned user_profiles table. It is intentionally
// narrow (timezone only) so the progress module stays decoupled from the intake
// module's internals (mirrors the read-port convention used elsewhere).
type gormProfileReader struct{ db *gorm.DB }

// NewProfileReader returns a gorm-backed ProfileReader over user_profiles.
func NewProfileReader(db *gorm.DB) ProfileReader { return &gormProfileReader{db: db} }

var _ ProfileReader = (*gormProfileReader)(nil)

// Timezone returns the user's profile timezone (IANA name), or "" when the user
// has no profile. An error is returned only for unexpected DB failures; a
// missing profile is not an error (the caller falls back to UTC).
func (r *gormProfileReader) Timezone(ctx context.Context, userID uuid.UUID) (string, error) {
	var tz string
	err := r.db.WithContext(ctx).
		Table("user_profiles").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Limit(1).
		Pluck("timezone", &tz).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	return tz, nil
}
