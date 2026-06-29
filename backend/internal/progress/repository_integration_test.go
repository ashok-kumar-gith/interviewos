package progress

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRepository_Integration exercises the gorm Repository against a live
// PostgreSQL instance. It is skipped unless DATABASE_URL is set, and it runs
// inside a transaction that is always rolled back so it leaves no residue.
//
// Requires migrations 000001_auth, 000006_roadmap, and 000011_progress.
func TestRepository_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx)
	ctx := context.Background()
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

	// Seed a user + minimal roadmap graph (no hard FKs to content needed).
	userID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, "progress-itest+"+uuid.NewString()+"@example.com").Error)

	roadmapID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO roadmaps (id, user_id, track_id, profile_id, start_date, end_date, total_weeks, hours_per_week, status, is_active)
		VALUES (?, ?, ?, ?, ?, ?, 12, 10, 'active', true)`,
		roadmapID, userID, uuid.New(), uuid.New(), now, now.AddDate(0, 0, 83)).Error)

	weekID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO roadmap_weeks (id, roadmap_id, week_number, start_date, end_date)
		VALUES (?, ?, 1, ?, ?)`,
		weekID, roadmapID, now, now.AddDate(0, 0, 6)).Error)

	dayID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO plan_days (id, roadmap_week_id, user_id, date, planned_minutes)
		VALUES (?, ?, ?, ?, 120)`,
		dayID, weekID, userID, now).Error)

	// A future plan-day to reschedule into.
	day2ID := uuid.New()
	tomorrow := now.AddDate(0, 0, 1)
	require.NoError(t, tx.Exec(`
		INSERT INTO plan_days (id, roadmap_week_id, user_id, date, planned_minutes)
		VALUES (?, ?, ?, ?, 60)`,
		day2ID, weekID, userID, tomorrow).Error)

	topicID := uuid.New()
	problemID := uuid.New()
	topicTaskID := uuid.New()
	problemTaskID := uuid.New()
	skipTaskID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO plan_tasks (id, plan_day_id, user_id, kind, item_type, item_id, pillar_type, title, estimated_minutes, priority, status, sort_order)
		VALUES
		  (?, ?, ?, 'study', 'topic', ?, 'dsa', 'Study: Arrays', 60, 'high', 'pending', 0),
		  (?, ?, ?, 'solve', 'problem', ?, 'dsa', 'Solve: Two Sum', 30, 'high', 'pending', 1),
		  (?, ?, ?, 'read', 'resource', ?, 'dsa', 'Read: notes', 15, 'medium', 'pending', 2)`,
		topicTaskID, dayID, userID, topicID,
		problemTaskID, dayID, userID, problemID,
		skipTaskID, dayID, userID, uuid.New()).Error)

	// --- GetPlanDay ---
	day, err := repo.GetPlanDay(ctx, userID, now)
	require.NoError(t, err)
	require.Len(t, day.Tasks, 3)
	require.Equal(t, "pending", day.Tasks[0].Status)

	// --- CompleteTask (topic) ---
	task, err := repo.CompleteTask(ctx, userID, topicTaskID, CompleteInput{Confidence: 4, TimeSpentMinutes: 45, Notes: ptr("solid")}, now)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.NotNil(t, task.Confidence)
	require.Equal(t, int16(4), *task.Confidence)

	// topic progress upserted.
	var utpStatus string
	var utpConf int16
	var utpMin int
	require.NoError(t, tx.Raw(
		`SELECT status, confidence, time_spent_minutes FROM user_topic_progress WHERE user_id = ? AND topic_id = ?`,
		userID, topicID).Row().Scan(&utpStatus, &utpConf, &utpMin))
	require.Equal(t, "completed", utpStatus)
	require.Equal(t, int16(4), utpConf)
	require.Equal(t, 45, utpMin)

	// study session recorded.
	var sessCount int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM study_sessions WHERE plan_task_id = ?`, topicTaskID).Row().Scan(&sessCount))
	require.Equal(t, int64(1), sessCount)

	// streak day upserted.
	var tasksDone, minutes int
	require.NoError(t, tx.Raw(
		`SELECT tasks_completed, minutes_studied FROM streak_days WHERE user_id = ? AND date = ?`,
		userID, now.Format("2006-01-02")).Row().Scan(&tasksDone, &minutes))
	require.Equal(t, 1, tasksDone)
	require.Equal(t, 45, minutes)

	// plan-day completed_minutes rolled forward.
	var completedMin int
	require.NoError(t, tx.Raw(`SELECT completed_minutes FROM plan_days WHERE id = ?`, dayID).Row().Scan(&completedMin))
	require.Equal(t, 45, completedMin)

	// --- CompleteTask (problem) increments streak ---
	_, err = repo.CompleteTask(ctx, userID, problemTaskID, CompleteInput{Confidence: 3, TimeSpentMinutes: 20}, now)
	require.NoError(t, err)
	var uppSolved bool
	require.NoError(t, tx.Raw(`SELECT solved FROM user_problem_progress WHERE user_id = ? AND problem_id = ?`, userID, problemID).Row().Scan(&uppSolved))
	require.True(t, uppSolved)
	require.NoError(t, tx.Raw(
		`SELECT tasks_completed, minutes_studied FROM streak_days WHERE user_id = ? AND date = ?`,
		userID, now.Format("2006-01-02")).Row().Scan(&tasksDone, &minutes))
	require.Equal(t, 2, tasksDone)
	require.Equal(t, 65, minutes)

	// double-complete is rejected.
	_, err = repo.CompleteTask(ctx, userID, topicTaskID, CompleteInput{Confidence: 5, TimeSpentMinutes: 5}, now)
	require.ErrorIs(t, err, ErrTaskAlreadyResolved)

	// --- SkipTask ---
	skipped, err := repo.SkipTask(ctx, userID, skipTaskID, "ran out of time", now)
	require.NoError(t, err)
	require.Equal(t, "skipped", skipped.Status)

	// --- RescheduleTask ---
	// Add a fresh pending task to reschedule.
	rTaskID := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO plan_tasks (id, plan_day_id, user_id, kind, item_type, item_id, pillar_type, title, estimated_minutes, priority, status, sort_order)
		VALUES (?, ?, ?, 'study', 'topic', ?, 'dsa', 'Study: Hashing', 40, 'medium', 'pending', 3)`,
		rTaskID, dayID, userID, uuid.New()).Error)
	cloned, err := repo.RescheduleTask(ctx, userID, rTaskID, tomorrow, now)
	require.NoError(t, err)
	require.Equal(t, "pending", cloned.Status)
	require.Equal(t, day2ID, cloned.PlanDayID)
	// rescheduled_from links the clone back to the original.
	var rescheduledFrom uuid.UUID
	require.NoError(t, tx.Raw(`SELECT rescheduled_from FROM plan_tasks WHERE id = ?`, cloned.ID).Row().Scan(&rescheduledFrom))
	require.Equal(t, rTaskID, rescheduledFrom)
	// original marked rescheduled.
	var origStatus string
	require.NoError(t, tx.Raw(`SELECT status FROM plan_tasks WHERE id = ?`, rTaskID).Row().Scan(&origStatus))
	require.Equal(t, "rescheduled", origStatus)

	// reschedule to a date with no plan-day → ErrNoTargetPlanDay.
	rTask2 := uuid.New()
	require.NoError(t, tx.Exec(`
		INSERT INTO plan_tasks (id, plan_day_id, user_id, kind, item_type, item_id, pillar_type, title, estimated_minutes, priority, status, sort_order)
		VALUES (?, ?, ?, 'study', 'topic', ?, 'dsa', 'Study: Graphs', 40, 'medium', 'pending', 4)`,
		rTask2, dayID, userID, uuid.New()).Error)
	_, err = repo.RescheduleTask(ctx, userID, rTask2, now.AddDate(0, 0, 30), now)
	require.ErrorIs(t, err, ErrNoTargetPlanDay)

	// --- ComputeStreak ---
	streak, err := repo.ComputeStreak(ctx, userID, now)
	require.NoError(t, err)
	require.Equal(t, 1, streak.Current)
	require.Equal(t, 1, streak.Longest)

	// --- PillarAggregates ---
	aggs, err := repo.PillarAggregates(ctx, userID)
	require.NoError(t, err)
	require.NotEmpty(t, aggs)
	var dsa *PillarAggregate
	for i := range aggs {
		if aggs[i].Pillar == "dsa" {
			dsa = &aggs[i]
		}
	}
	require.NotNil(t, dsa)
	require.Greater(t, dsa.CompletedMinutes, 0)
	require.Greater(t, dsa.PlannedMinutes, dsa.CompletedMinutes)

	// --- RevisionDueCount (table may be absent → 0) ---
	cnt, err := repo.RevisionDueCount(ctx, userID, now)
	require.NoError(t, err)
	require.GreaterOrEqual(t, cnt, 0)

	// GetTask ownership: another user cannot read the task.
	_, err = repo.GetTask(ctx, uuid.New(), topicTaskID)
	require.ErrorIs(t, err, ErrTaskNotFound)
}

func ptr(s string) *string { return &s }
