package roadmap

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository abstracts persistence for the roadmap domain so the service is
// unit-testable against a fake. The GORM implementation is gormRepository.
type Repository interface {
	// GetActive returns the user's active roadmap WITHOUT its week graph
	// (lightweight), or ErrNoActiveRoadmap.
	GetActive(ctx context.Context, userID uuid.UUID) (*Roadmap, error)
	// GetActiveWithWeeks returns the active roadmap with its weeks (no days), or
	// ErrNoActiveRoadmap.
	GetActiveWithWeeks(ctx context.Context, userID uuid.UUID) (*Roadmap, error)
	// GetWeek returns a single week (with days + tasks) for a roadmap owned by
	// the user, or ErrNotFound.
	GetWeek(ctx context.Context, userID, roadmapID uuid.UUID, weekNumber int) (*RoadmapWeek, error)
	// GetPlanDay returns the user's plan-day for a date (with tasks), or ErrNotFound.
	GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDay, error)
	// CreateGraph persists a full roadmap graph (roadmap → weeks → days → tasks)
	// in one transaction. If replaceActive is true, any existing active roadmap
	// for the user is archived (is_active=false) first so the partial unique
	// index is honored.
	CreateGraph(ctx context.Context, rm *Roadmap, replaceActive bool) error
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) GetActive(ctx context.Context, userID uuid.UUID) (*Roadmap, error) {
	var rm Roadmap
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = true", userID).
		First(&rm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoActiveRoadmap
	}
	if err != nil {
		return nil, err
	}
	return &rm, nil
}

func (r *gormRepository) GetActiveWithWeeks(ctx context.Context, userID uuid.UUID) (*Roadmap, error) {
	rm, err := r.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	var weeks []RoadmapWeek
	if err := r.db.WithContext(ctx).
		Where("roadmap_id = ?", rm.ID).
		Order("week_number ASC").Find(&weeks).Error; err != nil {
		return nil, err
	}
	rm.Weeks = weeks
	return rm, nil
}

func (r *gormRepository) GetWeek(ctx context.Context, userID, roadmapID uuid.UUID, weekNumber int) (*RoadmapWeek, error) {
	// Ownership check: the roadmap must belong to the user.
	var owned int64
	if err := r.db.WithContext(ctx).Model(&Roadmap{}).
		Where("id = ? AND user_id = ?", roadmapID, userID).
		Count(&owned).Error; err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, ErrNotFound
	}

	var week RoadmapWeek
	err := r.db.WithContext(ctx).
		Where("roadmap_id = ? AND week_number = ?", roadmapID, weekNumber).
		First(&week).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var days []PlanDay
	if err := r.db.WithContext(ctx).
		Where("roadmap_week_id = ?", week.ID).
		Order("date ASC").Find(&days).Error; err != nil {
		return nil, err
	}
	if err := r.loadTasksForDays(ctx, days); err != nil {
		return nil, err
	}
	week.Days = days
	return &week, nil
}

func (r *gormRepository) GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDay, error) {
	var day PlanDay
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND date = ?", userID, date.Format("2006-01-02")).
		First(&day).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadTasksForDays(ctx, []PlanDay{day}); err != nil {
		return nil, err
	}
	// loadTasksForDays mutates copies in the slice; reload tasks onto day directly.
	var tasks []PlanTask
	if err := r.db.WithContext(ctx).
		Where("plan_day_id = ?", day.ID).
		Order("sort_order ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	day.Tasks = tasks
	return &day, nil
}

// loadTasksForDays attaches tasks to each day in the slice with a single query.
func (r *gormRepository) loadTasksForDays(ctx context.Context, days []PlanDay) error {
	if len(days) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(days))
	for i := range days {
		ids[i] = days[i].ID
	}
	var tasks []PlanTask
	if err := r.db.WithContext(ctx).
		Where("plan_day_id IN ?", ids).
		Order("plan_day_id ASC, sort_order ASC").Find(&tasks).Error; err != nil {
		return err
	}
	byDay := make(map[uuid.UUID][]PlanTask, len(days))
	for _, t := range tasks {
		byDay[t.PlanDayID] = append(byDay[t.PlanDayID], t)
	}
	for i := range days {
		days[i].Tasks = byDay[days[i].ID]
	}
	return nil
}

func (r *gormRepository) CreateGraph(ctx context.Context, rm *Roadmap, replaceActive bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if replaceActive {
			// Archive any existing active roadmap to free the partial unique index.
			if err := tx.Model(&Roadmap{}).
				Where("user_id = ? AND is_active = true", rm.UserID).
				Updates(map[string]any{"is_active": false, "status": "archived"}).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(rm).Error; err != nil {
			return err
		}
		for wi := range rm.Weeks {
			w := &rm.Weeks[wi]
			w.RoadmapID = rm.ID
			if err := tx.Create(w).Error; err != nil {
				return err
			}
			for di := range w.Days {
				d := &w.Days[di]
				d.RoadmapWeekID = w.ID
				d.UserID = rm.UserID
				if err := tx.Create(d).Error; err != nil {
					return err
				}
				if len(d.Tasks) == 0 {
					continue
				}
				for ti := range d.Tasks {
					d.Tasks[ti].PlanDayID = d.ID
					d.Tasks[ti].UserID = rm.UserID
				}
				if err := tx.Create(&d.Tasks).Error; err != nil {
					return err
				}
			}
		}

		// FR-CUR-010: regeneration must preserve the user's hard-won progress.
		// The user_topic_progress / user_problem_progress / revision_items rows are
		// never deleted here (we only archive the prior roadmap above), so they
		// survive untouched. Additionally, carry the *completed* status forward
		// onto the freshly generated tasks so a re-roll doesn't reset items the
		// user already mastered. We do this in-DB (set-based) so it scales and
		// stays consistent within the same transaction.
		if replaceActive {
			if err := carryOverCompletedProgress(tx, rm.ID, rm.UserID); err != nil {
				return err
			}
		}
		return nil
	})
}

// carryOverCompletedProgress marks newly generated plan_tasks as completed when
// the user already has a completed progress row for the same content item, so
// regenerating a roadmap (FR-CUR-010) never discards mastered work. It copies
// the prior confidence and completion timestamp where available. Matching is by
// the polymorphic (item_type, item_id): topic/subtopic tasks match
// user_topic_progress.topic_id; problem/lld_problem/design_problem tasks match
// user_problem_progress.problem_id. Idempotent and safe to run once per regen.
func carryOverCompletedProgress(tx *gorm.DB, roadmapID, userID uuid.UUID) error {
	// Topics & subtopics ← user_topic_progress (status = 'completed'). The
	// correlated subquery against user_topic_progress drives both the row filter
	// and the carried-over confidence/timestamp, so plan_tasks need not appear in
	// a FROM clause (Postgres forbids the target table there).
	topicSQL := `
UPDATE plan_tasks pt
SET status = 'completed',
    confidence = COALESCE((
        SELECT utp.confidence FROM user_topic_progress utp
        WHERE utp.user_id = pt.user_id AND utp.topic_id = pt.item_id AND utp.deleted_at IS NULL
        ORDER BY utp.updated_at DESC LIMIT 1), pt.confidence),
    completed_at = COALESCE((
        SELECT COALESCE(utp.first_completed_at, utp.last_studied_at) FROM user_topic_progress utp
        WHERE utp.user_id = pt.user_id AND utp.topic_id = pt.item_id AND utp.deleted_at IS NULL
        ORDER BY utp.updated_at DESC LIMIT 1), now()),
    updated_at = now()
WHERE pt.user_id = ?
  AND pt.deleted_at IS NULL
  AND pt.status = 'pending'
  AND pt.item_type IN ('topic','subtopic')
  AND pt.plan_day_id IN (
        SELECT pd.id FROM plan_days pd
        JOIN roadmap_weeks rw ON rw.id = pd.roadmap_week_id
        WHERE rw.roadmap_id = ?)
  AND EXISTS (
        SELECT 1 FROM user_topic_progress utp
        WHERE utp.user_id = pt.user_id AND utp.topic_id = pt.item_id
          AND utp.deleted_at IS NULL AND utp.status = 'completed')`
	if err := tx.Exec(topicSQL, userID, roadmapID).Error; err != nil {
		return err
	}

	// Problems (DSA/LLD/HLD) ← user_problem_progress (solved OR status completed).
	problemSQL := `
UPDATE plan_tasks pt
SET status = 'completed',
    confidence = COALESCE((
        SELECT upp.confidence FROM user_problem_progress upp
        WHERE upp.user_id = pt.user_id AND upp.problem_id = pt.item_id AND upp.deleted_at IS NULL
        ORDER BY upp.updated_at DESC LIMIT 1), pt.confidence),
    completed_at = COALESCE((
        SELECT COALESCE(upp.solved_at, upp.last_attempt_at) FROM user_problem_progress upp
        WHERE upp.user_id = pt.user_id AND upp.problem_id = pt.item_id AND upp.deleted_at IS NULL
        ORDER BY upp.updated_at DESC LIMIT 1), now()),
    updated_at = now()
WHERE pt.user_id = ?
  AND pt.deleted_at IS NULL
  AND pt.status = 'pending'
  AND pt.item_type IN ('problem','lld_problem','design_problem')
  AND pt.plan_day_id IN (
        SELECT pd.id FROM plan_days pd
        JOIN roadmap_weeks rw ON rw.id = pd.roadmap_week_id
        WHERE rw.roadmap_id = ?)
  AND EXISTS (
        SELECT 1 FROM user_problem_progress upp
        WHERE upp.user_id = pt.user_id AND upp.problem_id = pt.item_id
          AND upp.deleted_at IS NULL AND (upp.solved = true OR upp.status = 'completed'))`
	if err := tx.Exec(problemSQL, userID, roadmapID).Error; err != nil {
		return err
	}
	return nil
}
