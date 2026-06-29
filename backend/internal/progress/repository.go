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
	// GetTask returns a single task owned by the user, or ErrTaskNotFound.
	GetTask(ctx context.Context, userID, taskID uuid.UUID) (*PlanTaskRow, error)

	// CompleteTask transactionally: flips the task to completed (confidence,
	// time, notes, completed_at), upserts the right progress row by item_type,
	// records a study_session, and upserts the streak_day. It returns the updated
	// task and the streak for the day's user.
	CompleteTask(ctx context.Context, userID, taskID uuid.UUID, in CompleteInput, now time.Time) (*PlanTaskRow, error)
	// SkipTask flips a pending/in-progress task to skipped and returns it.
	SkipTask(ctx context.Context, userID, taskID uuid.UUID, reason string, now time.Time) (*PlanTaskRow, error)
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
		Joins("JOIN roadmaps r ON r.id = (SELECT roadmaps.id FROM roadmaps WHERE roadmaps.user_id = pt.user_id AND roadmaps.is_active AND roadmaps.deleted_at IS NULL LIMIT 1)").
		Where("pt.user_id = ? AND pt.deleted_at IS NULL AND pt.kind <> 'revise'", userID).
		Group("pt.pillar_type").Scan(&cov).Error; err != nil {
		return nil, err
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
	confByPillar := map[string]confRow{}
	for _, c := range conf {
		confByPillar[c.PillarType] = c
	}

	// Pillar weights for the user's active roadmap track.
	type wRow struct {
		Type   string  `gorm:"column:type"`
		Weight float64 `gorm:"column:weight"`
	}
	var weights []wRow
	if err := r.db.WithContext(ctx).Table("pillars p").
		Select("p.type AS type, p.weight AS weight").
		Joins("JOIN roadmaps r ON r.track_id = p.track_id AND r.user_id = ? AND r.is_active AND r.deleted_at IS NULL", userID).
		Where("p.deleted_at IS NULL").Scan(&weights).Error; err != nil {
		return nil, err
	}
	weightByPillar := map[string]float64{}
	for _, w := range weights {
		weightByPillar[w.Type] = w.Weight
	}

	out := make([]PillarAggregate, 0, len(cov))
	for _, c := range cov {
		agg := PillarAggregate{
			Pillar:           c.PillarType,
			Weight:           weightByPillar[c.PillarType],
			PlannedMinutes:   c.Planned,
			CompletedMinutes: c.Completed,
		}
		if agg.Weight == 0 {
			agg.Weight = 1.0
		}
		if cf, ok := confByPillar[c.PillarType]; ok {
			agg.ConfidenceSum = cf.Sum
			agg.ConfidenceCount = cf.Cnt
		}
		out = append(out, agg)
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
