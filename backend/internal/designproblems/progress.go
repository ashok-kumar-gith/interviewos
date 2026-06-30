package designproblems

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Progress is a user's progress on a single HLD design problem
// (table: user_design_problem_progress). It mirrors user_problem_progress so
// design problems track status/confidence/time the same way DSA problems do.
type Progress struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	DesignProblemID  uuid.UUID      `gorm:"column:design_problem_id;type:uuid;not null"`
	Status           string         `gorm:"column:status;type:progress_status;not null;default:not_started"`
	Confidence       *int16         `gorm:"column:confidence"`
	Attempts         int            `gorm:"column:attempts;not null;default:0"`
	TimeSpentMinutes int            `gorm:"column:time_spent_minutes;not null;default:0"`
	Notes            *string        `gorm:"column:notes"`
	LastAttemptAt    *time.Time     `gorm:"column:last_attempt_at"`
	FirstCompletedAt *time.Time     `gorm:"column:first_completed_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table for GORM.
func (Progress) TableName() string { return "user_design_problem_progress" }

// ProgressInput is the validated payload for recording progress on a design
// problem. Status must be a valid progress_status; confidence (when set) 1–5.
type ProgressInput struct {
	Status           string
	Confidence       *int16
	TimeSpentMinutes int
	Notes            *string
}

var validProgressStatus = map[string]struct{}{
	"not_started":  {},
	"in_progress":  {},
	"completed":    {},
	"needs_review": {},
}

// ProgressRepository abstracts persistence for design-problem progress.
type ProgressRepository interface {
	// Get returns the user's progress for a design problem, or (nil, nil) when
	// none exists yet.
	Get(ctx context.Context, userID, designProblemID uuid.UUID) (*Progress, error)
	// Upsert records progress idempotently per (user, design problem), bumping
	// attempts/time and stamping completion the first time status hits completed.
	Upsert(ctx context.Context, userID, designProblemID uuid.UUID, in ProgressInput, now time.Time) (*Progress, error)
	// Delete soft-deletes the user's progress on a design problem. Deleting when
	// no progress exists is a no-op (no error).
	Delete(ctx context.Context, userID, designProblemID uuid.UUID) error
}

// NewProgressRepository returns a gorm-backed ProgressRepository.
func NewProgressRepository(db *gorm.DB) ProgressRepository { return &gormProgressRepository{db: db} }

type gormProgressRepository struct{ db *gorm.DB }

func (r *gormProgressRepository) Get(ctx context.Context, userID, dpID uuid.UUID) (*Progress, error) {
	var p Progress
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND design_problem_id = ?", userID, dpID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormProgressRepository) Upsert(ctx context.Context, userID, dpID uuid.UUID, in ProgressInput, now time.Time) (*Progress, error) {
	completedAtExpr := gorm.Expr("user_design_problem_progress.first_completed_at")
	if in.Status == "completed" {
		completedAtExpr = gorm.Expr("COALESCE(user_design_problem_progress.first_completed_at, ?)", now)
	}

	row := Progress{
		UserID:           userID,
		DesignProblemID:  dpID,
		Status:           in.Status,
		Confidence:       in.Confidence,
		Attempts:         1,
		TimeSpentMinutes: in.TimeSpentMinutes,
		Notes:            in.Notes,
		LastAttemptAt:    &now,
	}
	if in.Status == "completed" {
		row.FirstCompletedAt = &now
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "user_id"}, {Name: "design_problem_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":             in.Status,
			"confidence":         in.Confidence,
			"attempts":           gorm.Expr("user_design_problem_progress.attempts + 1"),
			"time_spent_minutes": gorm.Expr("user_design_problem_progress.time_spent_minutes + ?", in.TimeSpentMinutes),
			"notes":              in.Notes,
			"last_attempt_at":    now,
			"first_completed_at": completedAtExpr,
			"updated_at":         now,
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, userID, dpID)
}

func (r *gormProgressRepository) Delete(ctx context.Context, userID, dpID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND design_problem_id = ?", userID, dpID).
		Delete(&Progress{}).Error
}
