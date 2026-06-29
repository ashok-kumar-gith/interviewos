package notification

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanReader exposes the minimal plan facts the generator needs (today's and
// yesterday's task counts for the user's active roadmap). The generator depends
// on this interface, not on internal/progress or internal/roadmap, so the
// notification module stays decoupled (mirrors the read-port convention used by
// internal/analytics and internal/roadmap). The gorm implementation queries
// plan_days/plan_tasks directly; the schema is migration-owned and stable.
type PlanReader interface {
	// PlanDay returns the user's plan-day task summary for date, or (nil, nil)
	// when the user has no active roadmap / no plan-day for that date.
	PlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDaySummary, error)
}

// PlanDaySummary is the aggregate task picture for a single plan-day.
type PlanDaySummary struct {
	TotalTasks     int // tasks scheduled that day (excludes skipped/rescheduled)
	CompletedTasks int // tasks completed that day
	EstimatedMins  int // sum of estimated_minutes for non-skipped/rescheduled tasks
}

// RevisionReader exposes the count of revision items due on/before a date.
type RevisionReader interface {
	DueCount(ctx context.Context, userID uuid.UUID, asOf time.Time) (int, error)
}

// StreakReader exposes whether the user logged activity on a date and their
// current streak length, derived from streak_days.
type StreakReader interface {
	// Streak reports the current consecutive-day streak as of asOf and whether
	// the user has a streak_days row for asOf's date (i.e. logged today).
	Streak(ctx context.Context, userID uuid.UUID, asOf time.Time) (current int, loggedToday bool, err error)
}

// ReadinessReader exposes the two most recent readiness snapshot values so the
// generator can detect a milestone crossing.
type ReadinessReader interface {
	// LatestReadiness returns the latest overall readiness and the prior
	// snapshot's overall readiness (0 if only one/none). hasAny is false when the
	// user has no snapshots at all.
	LatestReadiness(ctx context.Context, userID uuid.UUID) (latest, previous float64, hasAny bool, err error)
}

// --- gorm-backed implementations (query migration-owned tables directly) ---

type gormReaders struct{ db *gorm.DB }

// NewReaders returns gorm-backed PlanReader, RevisionReader, StreakReader and
// ReadinessReader (all satisfied by one struct reading the underlying tables).
func NewReaders(db *gorm.DB) *gormReaders { return &gormReaders{db: db} }

var (
	_ PlanReader      = (*gormReaders)(nil)
	_ RevisionReader  = (*gormReaders)(nil)
	_ StreakReader    = (*gormReaders)(nil)
	_ ReadinessReader = (*gormReaders)(nil)
)

func (r *gormReaders) PlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDaySummary, error) {
	d := date.UTC().Format("2006-01-02")

	// Locate the user's plan-day for that date (user_id is denormalized on
	// plan_days; scope to the active roadmap so an archived plan never fires).
	// Scan the id into a typed struct field so GORM resolves the uuid column
	// (a bare Scan into uuid.UUID trips database/sql's scalar conversion).
	var day struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	res := r.db.WithContext(ctx).
		Table("plan_days AS pd").
		Joins("JOIN roadmap_weeks rw ON rw.id = pd.roadmap_week_id AND rw.deleted_at IS NULL").
		Joins("JOIN roadmaps rm ON rm.id = rw.roadmap_id AND rm.deleted_at IS NULL AND rm.is_active").
		Where("pd.user_id = ? AND pd.date = ? AND pd.deleted_at IS NULL", userID, d).
		Select("pd.id").
		Limit(1).
		Find(&day)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	dayID := day.ID

	type agg struct {
		Total     int
		Completed int
		EstMins   int
	}
	var a agg
	if err := r.db.WithContext(ctx).
		Table("plan_tasks").
		Where("plan_day_id = ? AND user_id = ? AND deleted_at IS NULL", dayID, userID).
		Where("status NOT IN ('skipped','rescheduled')").
		Select("COUNT(*) AS total, "+
			"COUNT(*) FILTER (WHERE status = 'completed') AS completed, "+
			"COALESCE(SUM(estimated_minutes), 0) AS est_mins").
		Scan(&a).Error; err != nil {
		return nil, err
	}
	return &PlanDaySummary{TotalTasks: a.Total, CompletedTasks: a.Completed, EstimatedMins: a.EstMins}, nil
}

func (r *gormReaders) DueCount(ctx context.Context, userID uuid.UUID, asOf time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("revision_items").
		Where("user_id = ? AND is_active AND deleted_at IS NULL AND due_at <= ?",
			userID, asOf.UTC().Format("2006-01-02")).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *gormReaders) Streak(ctx context.Context, userID uuid.UUID, asOf time.Time) (int, bool, error) {
	// Pull recent active days (descending) and walk back from asOf to count the
	// consecutive run, mirroring the streak computation in internal/progress.
	var dates []time.Time
	err := r.db.WithContext(ctx).
		Table("streak_days").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("date DESC").
		Limit(400).
		Pluck("date", &dates).Error
	if err != nil {
		return 0, false, err
	}
	if len(dates) == 0 {
		return 0, false, nil
	}

	today := truncDay(asOf.UTC())
	have := make(map[string]struct{}, len(dates))
	for _, d := range dates {
		have[truncDay(d.UTC()).Format("2006-01-02")] = struct{}{}
	}
	_, loggedToday := have[today.Format("2006-01-02")]

	// Current streak: count back from today (or yesterday if today not yet
	// logged) while consecutive days are present.
	cursor := today
	if !loggedToday {
		cursor = today.AddDate(0, 0, -1)
	}
	current := 0
	for {
		if _, ok := have[cursor.Format("2006-01-02")]; !ok {
			break
		}
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return current, loggedToday, nil
}

func (r *gormReaders) LatestReadiness(ctx context.Context, userID uuid.UUID) (float64, float64, bool, error) {
	var vals []float64
	err := r.db.WithContext(ctx).
		Table("readiness_snapshots").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("snapshot_date DESC").
		Limit(2).
		Pluck("overall_readiness", &vals).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, false, err
	}
	switch len(vals) {
	case 0:
		return 0, 0, false, nil
	case 1:
		return vals[0], 0, true, nil
	default:
		return vals[0], vals[1], true, nil
	}
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
