package progress

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlanTaskRow is the progress module's projection of a plan_tasks row. The
// progress module reads/writes plan_tasks directly (the roadmap module owns the
// schema; this module mutates lifecycle columns) and keeps its own narrow row
// type so the two modules stay decoupled.
type PlanTaskRow struct {
	ID               uuid.UUID  `gorm:"column:id"`
	PlanDayID        uuid.UUID  `gorm:"column:plan_day_id"`
	UserID           uuid.UUID  `gorm:"column:user_id"`
	Kind             string     `gorm:"column:kind"`
	ItemType         string     `gorm:"column:item_type"`
	ItemID           uuid.UUID  `gorm:"column:item_id"`
	PillarType       string     `gorm:"column:pillar_type"`
	Title            string     `gorm:"column:title"`
	Description      *string    `gorm:"column:description"`
	EstimatedMinutes int        `gorm:"column:estimated_minutes"`
	Priority         string     `gorm:"column:priority"`
	Difficulty       *string    `gorm:"column:difficulty"`
	Status           string     `gorm:"column:status"`
	SortOrder        int        `gorm:"column:sort_order"`
	Confidence       *int16     `gorm:"column:confidence"`
	TimeSpentMinutes *int       `gorm:"column:time_spent_minutes"`
	CompletionNotes  *string    `gorm:"column:completion_notes"`
	RevisionItemID   *uuid.UUID `gorm:"column:revision_item_id"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
}

// TableName pins PlanTaskRow to the plan_tasks table.
func (PlanTaskRow) TableName() string { return "plan_tasks" }

// PlanDayRow is the progress module's projection of a plan_days row.
type PlanDayRow struct {
	ID               uuid.UUID `gorm:"column:id"`
	RoadmapWeekID    uuid.UUID `gorm:"column:roadmap_week_id"`
	UserID           uuid.UUID `gorm:"column:user_id"`
	Date             time.Time `gorm:"column:date"`
	PlannedMinutes   int       `gorm:"column:planned_minutes"`
	CompletedMinutes int       `gorm:"column:completed_minutes"`
	IsRestDay        bool      `gorm:"column:is_rest_day"`
	Summary          *string   `gorm:"column:summary"`

	Tasks []PlanTaskRow `gorm:"-"`
}

// TableName pins PlanDayRow to the plan_days table.
func (PlanDayRow) TableName() string { return "plan_days" }

// Repository abstracts persistence for the progress domain so the service is
// unit-testable against a fake. The GORM implementation is gormRepository.
type Repository interface {
	// GetPlanDay returns the user's plan-day for a date (with tasks ordered by
	// sort_order), or ErrPlanDayNotFound.
	GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDayRow, error)
	// HasActiveRoadmap reports whether the user has a non-deleted active roadmap.
	HasActiveRoadmap(ctx context.Context, userID uuid.UUID) (bool, error)
	// GetTask returns a single task owned by the user, or ErrTaskNotFound.
	GetTask(ctx context.Context, userID, taskID uuid.UUID) (*PlanTaskRow, error)

	// CompleteTask transactionally: flips the task to completed (confidence,
	// time, notes, completed_at), upserts the right progress row by item_type,
	// records a study_session, and upserts the streak_day. It returns the updated
	// task and the streak for the day's user.
	CompleteTask(ctx context.Context, userID, taskID uuid.UUID, in CompleteInput, now time.Time) (*PlanTaskRow, error)
	// SkipTask flips a pending/in-progress task to skipped and returns it.
	SkipTask(ctx context.Context, userID, taskID uuid.UUID, reason string, now time.Time) (*PlanTaskRow, error)
	// StartTask flips a pending task to in_progress (a no-op signal of intent).
	StartTask(ctx context.Context, userID, taskID uuid.UUID, now time.Time) (*PlanTaskRow, error)
	// ReopenTask flips a completed/skipped task back to pending and reverses its
	// completion side effects (progress row + streak counters) within a tx.
	ReopenTask(ctx context.Context, userID, taskID uuid.UUID, now time.Time) (*PlanTaskRow, error)
	// RescheduleTask moves a task to the user's plan-day for toDate. The original
	// task is marked rescheduled; a fresh pending clone is created on the target
	// day (rescheduled_from = original). Returns the new task.
	RescheduleTask(ctx context.Context, userID, taskID uuid.UUID, toDate time.Time, now time.Time) (*PlanTaskRow, error)

	// ComputeStreak returns the user's current and longest streak as of asOf
	// (counting consecutive streak_days ending today/yesterday).
	ComputeStreak(ctx context.Context, userID uuid.UUID, asOf time.Time) (Streak, error)
	// PillarAggregates rolls up planned/completed est-minutes (from plan_tasks)
	// and confidence (from progress rows) per pillar for the user's active roadmap.
	PillarAggregates(ctx context.Context, userID uuid.UUID) ([]PillarAggregate, error)
	// RevisionDueCount counts active revision items due on/before asOf, or 0 if
	// the revision_items table is absent.
	RevisionDueCount(ctx context.Context, userID uuid.UUID, asOf time.Time) (int, error)
}

// CompleteInput carries the validated completion payload to the repository.
type CompleteInput struct {
	Confidence       int16
	TimeSpentMinutes int
	Notes            *string
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// dailyGoalMinutes is the minutes-studied threshold that marks a streak day's
// goal as met. It is a deliberately simple GA heuristic.
const dailyGoalMinutes = 30

func (r *gormRepository) GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDayRow, error) {
	var day PlanDayRow
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND date = ? AND deleted_at IS NULL", userID, date.Format("2006-01-02")).
		First(&day).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlanDayNotFound
	}
	if err != nil {
		return nil, err
	}
	tasks, err := r.tasksForDay(ctx, r.db, day.ID)
	if err != nil {
		return nil, err
	}
	day.Tasks = tasks
	return &day, nil
}

// HasActiveRoadmap reports whether the user has a non-deleted active roadmap.
func (r *gormRepository) HasActiveRoadmap(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("roadmaps").
		Where("user_id = ? AND is_active AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRepository) tasksForDay(ctx context.Context, db *gorm.DB, dayID uuid.UUID) ([]PlanTaskRow, error) {
	var tasks []PlanTaskRow
	if err := db.WithContext(ctx).
		Where("plan_day_id = ? AND deleted_at IS NULL", dayID).
		Order("sort_order ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *gormRepository) GetTask(ctx context.Context, userID, taskID uuid.UUID) (*PlanTaskRow, error) {
	return getTaskTx(ctx, r.db, userID, taskID)
}

func getTaskTx(ctx context.Context, db *gorm.DB, userID, taskID uuid.UUID) (*PlanTaskRow, error) {
	var t PlanTaskRow
	err := db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", taskID, userID).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *gormRepository) CompleteTask(ctx context.Context, userID, taskID uuid.UUID, in CompleteInput, now time.Time) (*PlanTaskRow, error) {
	var updated *PlanTaskRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := getTaskTx(ctx, tx, userID, taskID)
		if err != nil {
			return err
		}
		if task.Status == "completed" || task.Status == "skipped" {
			return ErrTaskAlreadyResolved
		}

		// 1. Flip the task to completed.
		conf := in.Confidence
		updates := map[string]any{
			"status":             "completed",
			"confidence":         conf,
			"time_spent_minutes": in.TimeSpentMinutes,
			"completion_notes":   in.Notes,
			"completed_at":       now,
		}
		if err := tx.Model(&PlanTaskRow{}).
			Where("id = ?", task.ID).Updates(updates).Error; err != nil {
			return err
		}

		// 2. Upsert the right progress row by item_type.
		if err := upsertProgress(ctx, tx, task, conf, in.TimeSpentMinutes, now); err != nil {
			return err
		}

		// 3. Record a study session.
		pillar := task.PillarType
		session := StudySession{
			UserID:          userID,
			PlanTaskID:      &task.ID,
			PillarType:      &pillar,
			StartedAt:       now,
			EndedAt:         &now,
			DurationMinutes: in.TimeSpentMinutes,
			Source:          "manual",
		}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}

		// 4. Roll the plan-day completed_minutes forward.
		if err := tx.Model(&PlanDayRow{}).
			Where("id = ?", task.PlanDayID).
			UpdateColumn("completed_minutes", gorm.Expr("completed_minutes + ?", in.TimeSpentMinutes)).Error; err != nil {
			return err
		}

		// 5. Upsert the streak day (today's local date).
		if err := upsertStreakDay(ctx, tx, userID, now, in.TimeSpentMinutes); err != nil {
			return err
		}

		// Re-read the task for the response.
		updated, err = getTaskTx(ctx, tx, userID, taskID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// upsertProgress writes the topic/problem progress row appropriate to the
// task's item_type. Non topic/problem item types (resource/design/etc.) have no
// dedicated progress table at GA and are tracked solely via the task lifecycle.
func upsertProgress(ctx context.Context, tx *gorm.DB, task *PlanTaskRow, conf int16, minutes int, now time.Time) error {
	switch task.ItemType {
	case "topic":
		row := UserTopicProgress{
			UserID:           task.UserID,
			TopicID:          task.ItemID,
			Status:           "completed",
			Confidence:       &conf,
			TimeSpentMinutes: minutes,
			LastStudiedAt:    &now,
			FirstCompletedAt: &now,
		}
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:     []clause.Column{{Name: "user_id"}, {Name: "topic_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":             "completed",
				"confidence":         conf,
				"time_spent_minutes": gorm.Expr("user_topic_progress.time_spent_minutes + ?", minutes),
				"last_studied_at":    now,
				"first_completed_at": gorm.Expr("COALESCE(user_topic_progress.first_completed_at, ?)", now),
				"updated_at":         now,
			}),
		}).Create(&row).Error
	case "problem":
		row := UserProblemProgress{
			UserID:           task.UserID,
			ProblemID:        task.ItemID,
			Status:           "completed",
			Confidence:       &conf,
			Attempts:         1,
			Solved:           true,
			TimeSpentMinutes: minutes,
			LastAttemptAt:    &now,
			SolvedAt:         &now,
		}
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:     []clause.Column{{Name: "user_id"}, {Name: "problem_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":             "completed",
				"confidence":         conf,
				"attempts":           gorm.Expr("user_problem_progress.attempts + 1"),
				"solved":             true,
				"time_spent_minutes": gorm.Expr("user_problem_progress.time_spent_minutes + ?", minutes),
				"last_attempt_at":    now,
				"solved_at":          gorm.Expr("COALESCE(user_problem_progress.solved_at, ?)", now),
				"updated_at":         now,
			}),
		}).Create(&row).Error
	default:
		return nil
	}
}

// upsertStreakDay creates or increments the user's streak_day for the date,
// incrementing tasks_completed and minutes_studied and recomputing goal_met.
func upsertStreakDay(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now time.Time, minutes int) error {
	row := StreakDay{
		UserID:         userID,
		Date:           time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		TasksCompleted: 1,
		MinutesStudied: minutes,
		GoalMet:        minutes >= dailyGoalMinutes,
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "user_id"}, {Name: "date"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
		DoUpdates: clause.Assignments(map[string]any{
			"tasks_completed": gorm.Expr("streak_days.tasks_completed + 1"),
			"minutes_studied": gorm.Expr("streak_days.minutes_studied + ?", minutes),
			"goal_met":        gorm.Expr("(streak_days.minutes_studied + ?) >= ?", minutes, dailyGoalMinutes),
			"updated_at":      now,
		}),
	}).Create(&row).Error
}

func (r *gormRepository) SkipTask(ctx context.Context, userID, taskID uuid.UUID, reason string, now time.Time) (*PlanTaskRow, error) {
	var updated *PlanTaskRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := getTaskTx(ctx, tx, userID, taskID)
		if err != nil {
			return err
		}
		if task.Status == "completed" || task.Status == "skipped" {
			return ErrTaskAlreadyResolved
		}
		updates := map[string]any{"status": "skipped"}
		if reason != "" {
			updates["completion_notes"] = reason
		}
		if err := tx.Model(&PlanTaskRow{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
			return err
		}
		updated, err = getTaskTx(ctx, tx, userID, taskID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *gormRepository) StartTask(ctx context.Context, userID, taskID uuid.UUID, now time.Time) (*PlanTaskRow, error) {
	var updated *PlanTaskRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := getTaskTx(ctx, tx, userID, taskID)
		if err != nil {
			return err
		}
		// Only a not-yet-resolved, not-already-started task can be started.
		if task.Status == "completed" || task.Status == "skipped" {
			return ErrTaskAlreadyResolved
		}
		if task.Status != "in_progress" {
			if err := tx.Model(&PlanTaskRow{}).
				Where("id = ?", task.ID).
				Update("status", "in_progress").Error; err != nil {
				return err
			}
		}
		updated, err = getTaskTx(ctx, tx, userID, taskID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *gormRepository) ReopenTask(ctx context.Context, userID, taskID uuid.UUID, now time.Time) (*PlanTaskRow, error) {
	var updated *PlanTaskRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := getTaskTx(ctx, tx, userID, taskID)
		if err != nil {
			return err
		}
		// Only resolved tasks can be reopened; pending/in_progress is a no-op.
		if task.Status != "completed" && task.Status != "skipped" {
			updated, err = getTaskTx(ctx, tx, userID, taskID)
			return err
		}

		wasCompleted := task.Status == "completed"
		minutes := 0
		if task.TimeSpentMinutes != nil {
			minutes = *task.TimeSpentMinutes
		}

		// 1. Reset the task to pending and clear completion fields.
		if err := tx.Model(&PlanTaskRow{}).Where("id = ?", task.ID).Updates(map[string]any{
			"status":             "pending",
			"confidence":         nil,
			"time_spent_minutes": nil,
			"completion_notes":   nil,
			"completed_at":       nil,
		}).Error; err != nil {
			return err
		}

		// Reversing the streak/session/plan-day bookkeeping only applies when the
		// task had been completed (skipped tasks recorded none of it).
		if wasCompleted {
			// 2. Soft-delete the study session(s) booked for this task.
			if err := tx.Model(&StudySession{}).
				Where("plan_task_id = ? AND deleted_at IS NULL", task.ID).
				Update("deleted_at", now).Error; err != nil {
				return err
			}

			// 3. Roll the plan-day completed_minutes back (never below zero).
			if minutes > 0 {
				if err := tx.Model(&PlanDayRow{}).
					Where("id = ?", task.PlanDayID).
					UpdateColumn("completed_minutes", gorm.Expr("GREATEST(completed_minutes - ?, 0)", minutes)).Error; err != nil {
					return err
				}
			}

			// 4. Decrement the streak day's counters; if it drops to zero tasks,
			//    clear goal_met. (Streak length recomputes from streak_days on read.)
			day := now.Format("2006-01-02")
			if err := tx.Exec(`
				UPDATE streak_days
				   SET tasks_completed = GREATEST(tasks_completed - 1, 0),
				       minutes_studied = GREATEST(minutes_studied - ?, 0),
				       goal_met = (GREATEST(tasks_completed - 1, 0) > 0 AND goal_met),
				       updated_at = ?
				 WHERE user_id = ? AND date = ? AND deleted_at IS NULL`,
				minutes, now, userID, day).Error; err != nil {
				return err
			}
		}

		updated, err = getTaskTx(ctx, tx, userID, taskID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *gormRepository) RescheduleTask(ctx context.Context, userID, taskID uuid.UUID, toDate time.Time, now time.Time) (*PlanTaskRow, error) {
	var created *PlanTaskRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := getTaskTx(ctx, tx, userID, taskID)
		if err != nil {
			return err
		}
		if task.Status == "completed" {
			return ErrTaskAlreadyResolved
		}

		// Find the target plan-day owned by the user.
		var target PlanDayRow
		derr := tx.WithContext(ctx).
			Where("user_id = ? AND date = ? AND deleted_at IS NULL", userID, toDate.Format("2006-01-02")).
			First(&target).Error
		if errors.Is(derr, gorm.ErrRecordNotFound) {
			return ErrNoTargetPlanDay
		}
		if derr != nil {
			return derr
		}

		// Mark the original task rescheduled.
		if err := tx.Model(&PlanTaskRow{}).Where("id = ?", task.ID).
			Update("status", "rescheduled").Error; err != nil {
			return err
		}

		// Clone a fresh pending task onto the target day. Insert with explicit
		// columns so DB defaults (objectives='[]', priority, etc.) and the
		// rescheduled_from link are honored.
		newID := uuid.New()
		if err := tx.WithContext(ctx).Exec(`
			INSERT INTO plan_tasks
			    (id, plan_day_id, user_id, kind, item_type, item_id, pillar_type,
			     title, description, objectives, estimated_minutes, priority,
			     difficulty, status, sort_order, rescheduled_from)
			SELECT ?, ?, user_id, kind, item_type, item_id, pillar_type,
			       title, description, objectives, estimated_minutes, priority,
			       difficulty, 'pending', sort_order, id
			FROM plan_tasks WHERE id = ?`,
			newID, target.ID, task.ID).Error; err != nil {
			return err
		}

		created, err = getTaskTx(ctx, tx, userID, newID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *gormRepository) ComputeStreak(ctx context.Context, userID uuid.UUID, asOf time.Time) (Streak, error) {
	var dates []time.Time
	if err := r.db.WithContext(ctx).Model(&StreakDay{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("date ASC").Pluck("date", &dates).Error; err != nil {
		return Streak{}, err
	}
	return computeStreakFromDates(dates, asOf), nil
}

// computeStreakFromDates derives current+longest consecutive-day streaks from a
// sorted-ascending slice of active study dates. The current streak counts
// backwards from today (or yesterday, allowing the user not to have studied yet
// today) through consecutive days.
func computeStreakFromDates(dates []time.Time, asOf time.Time) Streak {
	if len(dates) == 0 {
		return Streak{}
	}
	// Normalize to day granularity (UTC) and de-duplicate.
	seen := map[string]bool{}
	norm := make([]time.Time, 0, len(dates))
	for _, d := range dates {
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		k := day.Format("2006-01-02")
		if seen[k] {
			continue
		}
		seen[k] = true
		norm = append(norm, day)
	}

	// Longest run of consecutive days.
	longest, run := 1, 1
	for i := 1; i < len(norm); i++ {
		if norm[i].Sub(norm[i-1]) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
	}

	// Current streak: walk back from today; if today missing, allow yesterday.
	today := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	var anchor time.Time
	switch {
	case seen[today.Format("2006-01-02")]:
		anchor = today
	case seen[today.AddDate(0, 0, -1).Format("2006-01-02")]:
		anchor = today.AddDate(0, 0, -1)
	default:
		return Streak{Current: 0, Longest: longest}
	}
	current := 0
	for d := anchor; seen[d.Format("2006-01-02")]; d = d.AddDate(0, 0, -1) {
		current++
	}
	return Streak{Current: current, Longest: longest}
}

func (r *gormRepository) PillarAggregates(ctx context.Context, userID uuid.UUID) ([]PillarAggregate, error) {
	// Resolve the user's active roadmap track once; everything below is scoped to
	// it. No active roadmap ⇒ no pillars (the dashboard renders the onboarding
	// CTA in that case).
	type trackRow struct {
		TrackID uuid.UUID `gorm:"column:track_id"`
	}
	var tr trackRow
	if err := r.db.WithContext(ctx).Table("roadmaps").
		Select("track_id").
		Where("user_id = ? AND is_active AND deleted_at IS NULL", userID).
		Limit(1).Scan(&tr).Error; err != nil {
		return nil, err
	}
	if tr.TrackID == uuid.Nil {
		return nil, nil
	}

	// Seed the aggregate map from the track's pillars so EVERY pillar the track
	// defines appears on the dashboard — including lld/behavioral/resume which may
	// have no plan_tasks yet (they'd otherwise be silently dropped by a GROUP BY
	// on plan_tasks.pillar_type). Order is preserved for a stable dashboard.
	type pRow struct {
		Type   string  `gorm:"column:type"`
		Weight float64 `gorm:"column:weight"`
	}
	var pillars []pRow
	if err := r.db.WithContext(ctx).Table("pillars").
		Select("type, weight").
		Where("track_id = ? AND deleted_at IS NULL", tr.TrackID).
		Order("weight DESC, type ASC").Scan(&pillars).Error; err != nil {
		return nil, err
	}
	order := make([]string, 0, len(pillars))
	byPillar := map[string]*PillarAggregate{}
	for _, p := range pillars {
		w := p.Weight
		if w == 0 {
			w = 1.0
		}
		byPillar[p.Type] = &PillarAggregate{Pillar: p.Type, Weight: w}
		order = append(order, p.Type)
	}
	// Defensive: if a pillar shows up below (plan_task/progress) that the pillars
	// table did not define, ensure it still surfaces rather than being dropped.
	ensure := func(pillar string) *PillarAggregate {
		agg, ok := byPillar[pillar]
		if !ok {
			agg = &PillarAggregate{Pillar: pillar, Weight: 1.0}
			byPillar[pillar] = agg
			order = append(order, pillar)
		}
		return agg
	}

	// Coverage from the active roadmap's plan_tasks: planned vs. completed
	// est-minutes per pillar (revise tasks excluded — they are not "new coverage").
	type covRow struct {
		PillarType string `gorm:"column:pillar_type"`
		Planned    int    `gorm:"column:planned"`
		Completed  int    `gorm:"column:completed"`
	}
	var cov []covRow
	if err := r.db.WithContext(ctx).Table("plan_tasks pt").
		Select(`pt.pillar_type AS pillar_type,
		        SUM(pt.estimated_minutes) AS planned,
		        SUM(CASE WHEN pt.status = 'completed' THEN pt.estimated_minutes ELSE 0 END) AS completed`).
		Where("pt.user_id = ? AND pt.deleted_at IS NULL AND pt.kind <> 'revise'", userID).
		Group("pt.pillar_type").Scan(&cov).Error; err != nil {
		return nil, err
	}
	for _, c := range cov {
		agg := ensure(c.PillarType)
		agg.PlannedMinutes = c.Planned
		agg.CompletedMinutes = c.Completed
	}

	// Average confidence per pillar from completed plan_tasks (the per-pillar
	// confidence signal lives on the task; topic/problem progress mirrors it).
	type confRow struct {
		PillarType string `gorm:"column:pillar_type"`
		Sum        int    `gorm:"column:sum"`
		Cnt        int    `gorm:"column:cnt"`
	}
	var conf []confRow
	if err := r.db.WithContext(ctx).Table("plan_tasks pt").
		Select("pt.pillar_type AS pillar_type, COALESCE(SUM(pt.confidence),0) AS sum, COUNT(pt.confidence) AS cnt").
		Where("pt.user_id = ? AND pt.deleted_at IS NULL AND pt.confidence IS NOT NULL", userID).
		Group("pt.pillar_type").Scan(&conf).Error; err != nil {
		return nil, err
	}
	for _, c := range conf {
		agg := ensure(c.PillarType)
		agg.ConfidenceSum += c.Sum
		agg.ConfidenceCount += c.Cnt
	}

	// --- "Items completed" signals from the problem-progress tables ---
	// These make solving a problem on the detail page (which writes the progress
	// table, NOT a plan_task) visibly move the pillar's readiness. The items
	// coverage is blended with plan-task minute coverage in the service.
	//
	// LLD (lld pillar): there is no per-user LLD progress table at GA — LLD
	// problems are completed only via plan_tasks (item_type='lld_problem'), which
	// the plan-task coverage above already captures. So there is no separate
	// items signal to wire for LLD; behavioral/resume likewise have none.

	// DSA: solved problems in the user's track vs. total track problems.
	type dsaRow struct {
		Total  int `gorm:"column:total"`
		Solved int `gorm:"column:solved"`
	}
	var dsa dsaRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
		  (SELECT COUNT(*) FROM problems
		     WHERE track_id = ? AND deleted_at IS NULL) AS total,
		  (SELECT COUNT(*) FROM user_problem_progress upp
		     JOIN problems p ON p.id = upp.problem_id
		    WHERE upp.user_id = ? AND upp.solved = true
		      AND upp.deleted_at IS NULL AND p.deleted_at IS NULL
		      AND p.track_id = ?) AS solved`,
		tr.TrackID, userID, tr.TrackID).Scan(&dsa).Error; err != nil {
		return nil, err
	}
	if dsa.Total > 0 {
		agg := ensure("dsa")
		agg.ItemsTotal = dsa.Total
		agg.ItemsCompleted = dsa.Solved
	}
	// Confidence from solved DSA problems feeds the pillar confidence too.
	type confItemRow struct {
		Sum int `gorm:"column:sum"`
		Cnt int `gorm:"column:cnt"`
	}
	var dsaConf confItemRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(upp.confidence),0) AS sum, COUNT(upp.confidence) AS cnt
		  FROM user_problem_progress upp
		  JOIN problems p ON p.id = upp.problem_id
		 WHERE upp.user_id = ? AND upp.solved = true AND upp.confidence IS NOT NULL
		   AND upp.deleted_at IS NULL AND p.deleted_at IS NULL AND p.track_id = ?`,
		userID, tr.TrackID).Scan(&dsaConf).Error; err != nil {
		return nil, err
	}
	if dsaConf.Cnt > 0 {
		agg := ensure("dsa")
		agg.ConfidenceSum += dsaConf.Sum
		agg.ConfidenceCount += dsaConf.Cnt
	}

	// System Design: completed HLD design problems in the track vs. total.
	type sdRow struct {
		Total     int `gorm:"column:total"`
		Completed int `gorm:"column:completed"`
	}
	var sd sdRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
		  (SELECT COUNT(*) FROM design_problems
		     WHERE track_id = ? AND deleted_at IS NULL) AS total,
		  (SELECT COUNT(*) FROM user_design_problem_progress udpp
		     JOIN design_problems dp ON dp.id = udpp.design_problem_id
		    WHERE udpp.user_id = ? AND udpp.status = 'completed'
		      AND udpp.deleted_at IS NULL AND dp.deleted_at IS NULL
		      AND dp.track_id = ?) AS completed`,
		tr.TrackID, userID, tr.TrackID).Scan(&sd).Error; err != nil {
		return nil, err
	}
	if sd.Total > 0 {
		agg := ensure("system_design")
		agg.ItemsTotal = sd.Total
		agg.ItemsCompleted = sd.Completed
	}
	var sdConf confItemRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(udpp.confidence),0) AS sum, COUNT(udpp.confidence) AS cnt
		  FROM user_design_problem_progress udpp
		  JOIN design_problems dp ON dp.id = udpp.design_problem_id
		 WHERE udpp.user_id = ? AND udpp.status = 'completed' AND udpp.confidence IS NOT NULL
		   AND udpp.deleted_at IS NULL AND dp.deleted_at IS NULL AND dp.track_id = ?`,
		userID, tr.TrackID).Scan(&sdConf).Error; err != nil {
		return nil, err
	}
	if sdConf.Cnt > 0 {
		agg := ensure("system_design")
		agg.ConfidenceSum += sdConf.Sum
		agg.ConfidenceCount += sdConf.Cnt
	}

	out := make([]PillarAggregate, 0, len(order))
	for _, pillar := range order {
		out = append(out, *byPillar[pillar])
	}
	return out, nil
}

func (r *gormRepository) RevisionDueCount(ctx context.Context, userID uuid.UUID, asOf time.Time) (int, error) {
	// revision_items may not exist yet (separate later feature). Guard on the
	// table's presence; absent ⇒ 0.
	if !r.db.Migrator().HasTable("revision_items") {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Table("revision_items").
		Where("user_id = ? AND is_active = true AND deleted_at IS NULL AND due_at <= ?",
			userID, asOf.Format("2006-01-02")).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
