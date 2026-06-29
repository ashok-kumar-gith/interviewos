package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// gormProfileReader implements ProfileReader by reading user_profiles directly.
// It deliberately queries the table rather than importing intake's repository so
// the roadmap module stays decoupled from another module's internals
// (03-ARCHITECTURE.md §4.4); the profile shape is stable and migration-owned.
type gormProfileReader struct {
	db *gorm.DB
}

// NewProfileReader returns a gorm-backed ProfileReader.
func NewProfileReader(db *gorm.DB) ProfileReader {
	return &gormProfileReader{db: db}
}

type profileRow struct {
	ID              uuid.UUID
	TrackID         uuid.UUID
	TargetCompanyID *uuid.UUID
	HoursPerWeek    int
	StartDate       time.Time
	TargetWeeks     int
	PillarStrengths []byte
}

func (r *gormProfileReader) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var row profileRow
	err := r.db.WithContext(ctx).Table("user_profiles").
		Select("id, track_id, target_company_id, hours_per_week, start_date, target_weeks, pillar_strengths").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileRequired
	}
	if err != nil {
		return nil, err
	}

	strengths := map[string]int{}
	if len(row.PillarStrengths) > 0 {
		_ = json.Unmarshal(row.PillarStrengths, &strengths)
	}

	return &Profile{
		ID:              row.ID,
		TrackID:         row.TrackID,
		TargetCompanyID: row.TargetCompanyID,
		HoursPerWeek:    row.HoursPerWeek,
		StartDate:       row.StartDate,
		TargetWeeks:     row.TargetWeeks,
		PillarStrengths: strengths,
	}, nil
}
