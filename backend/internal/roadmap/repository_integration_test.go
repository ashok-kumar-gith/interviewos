package roadmap

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
// Requires migrations 000001_auth and 000006_roadmap to be applied.
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

	// Seed a user (user-owned cascades).
	userID := uuid.New()
	email := "roadmap-itest+" + uuid.NewString() + "@example.com"
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, email,
	).Error)

	// No active roadmap yet.
	_, err = repo.GetActive(ctx, userID)
	require.ErrorIs(t, err, ErrNoActiveRoadmap)

	start := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	topicID := uuid.New()
	problemID := uuid.New()

	rm := &Roadmap{
		UserID:       userID,
		TrackID:      uuid.New(),
		ProfileID:    uuid.New(),
		StartDate:    start,
		EndDate:      start.AddDate(0, 0, 12*7-1),
		TotalWeeks:   12,
		HoursPerWeek: 10,
		Status:       "active",
		IsActive:     true,
		Weeks: []RoadmapWeek{
			{
				WeekNumber:   1,
				StartDate:    start,
				EndDate:      start.AddDate(0, 0, 6),
				FocusPillars: JSONStringArray{"dsa"},
				PlannedHours: 8,
				Days: []PlanDay{
					{
						Date:           start,
						PlannedMinutes: 90,
						Tasks: []PlanTask{
							{Kind: "study", ItemType: "topic", ItemID: topicID, PillarType: "dsa", Title: "Study: Arrays", EstimatedMinutes: 60, Priority: "high", Objectives: JSONStringArray{}, Status: "pending", SortOrder: 0},
							{Kind: "solve", ItemType: "problem", ItemID: problemID, PillarType: "dsa", Title: "Solve: Two Sum", EstimatedMinutes: 30, Priority: "high", Objectives: JSONStringArray{}, Status: "pending", SortOrder: 1},
						},
					},
				},
			},
		},
	}

	require.NoError(t, repo.CreateGraph(ctx, rm, false))
	require.NotEqual(t, uuid.Nil, rm.ID)
	require.NotEqual(t, uuid.Nil, rm.Weeks[0].ID)
	require.NotEqual(t, uuid.Nil, rm.Weeks[0].Days[0].ID)
	require.NotEqual(t, uuid.Nil, rm.Weeks[0].Days[0].Tasks[0].ID)

	// GetActive returns it.
	active, err := repo.GetActive(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, rm.ID, active.ID)
	require.True(t, active.IsActive)

	// GetActiveWithWeeks returns the week summary.
	withWeeks, err := repo.GetActiveWithWeeks(ctx, userID)
	require.NoError(t, err)
	require.Len(t, withWeeks.Weeks, 1)
	require.Equal(t, int16(1), withWeeks.Weeks[0].WeekNumber)

	// GetWeek returns days + tasks (and enforces ownership).
	week, err := repo.GetWeek(ctx, userID, rm.ID, 1)
	require.NoError(t, err)
	require.Len(t, week.Days, 1)
	require.Len(t, week.Days[0].Tasks, 2)
	require.Equal(t, "study", week.Days[0].Tasks[0].Kind)

	// Ownership: another user cannot read the week.
	_, err = repo.GetWeek(ctx, uuid.New(), rm.ID, 1)
	require.ErrorIs(t, err, ErrNotFound)

	// GetPlanDay by date returns tasks ordered by sort_order.
	day, err := repo.GetPlanDay(ctx, userID, start)
	require.NoError(t, err)
	require.Len(t, day.Tasks, 2)
	require.Equal(t, 0, day.Tasks[0].SortOrder)
	require.Equal(t, 1, day.Tasks[1].SortOrder)

	// Missing plan-day → ErrNotFound.
	_, err = repo.GetPlanDay(ctx, userID, start.AddDate(0, 0, 365))
	require.ErrorIs(t, err, ErrNotFound)

	// Regenerate archives the active and creates a fresh active one.
	rm2 := &Roadmap{
		UserID: userID, TrackID: rm.TrackID, ProfileID: rm.ProfileID,
		StartDate: start, EndDate: rm.EndDate, TotalWeeks: 12, HoursPerWeek: 12,
		Status: "active", IsActive: true,
	}
	require.NoError(t, repo.CreateGraph(ctx, rm2, true))
	active2, err := repo.GetActive(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, rm2.ID, active2.ID)
	require.Equal(t, int16(12), active2.HoursPerWeek)
}

// TestCreateGraph_RegenerationPreservesProgress exercises FR-CUR-010: a
// regenerate must NOT delete user_topic_progress / user_problem_progress /
// revision_items, and the freshly generated tasks whose item the user already
// completed must be carried over to 'completed'.
//
// Requires migrations 000006_roadmap, 000011_progress, 000012_revision applied.
func TestCreateGraph_RegenerationPreservesProgress(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, "cur010+"+uuid.NewString()+"@example.com",
	).Error)

	start := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	topicID := uuid.New()
	problemID := uuid.New()

	// User already mastered this topic and solved this problem (independent of
	// any roadmap — these rows persist across regenerations).
	require.NoError(t, tx.Exec(
		`INSERT INTO user_topic_progress (user_id, topic_id, status, confidence, first_completed_at)
		 VALUES (?, ?, 'completed', 4, now())`, userID, topicID).Error)
	require.NoError(t, tx.Exec(
		`INSERT INTO user_problem_progress (user_id, problem_id, status, confidence, solved, solved_at)
		 VALUES (?, ?, 'completed', 5, true, now())`, userID, problemID).Error)
	// A revision item for the topic (must survive regeneration).
	require.NoError(t, tx.Exec(
		`INSERT INTO revision_items (user_id, item_type, item_id, pillar_type, stage, interval_days, due_at, is_active)
		 VALUES (?, 'topic', ?, 'dsa', 0, 1, ?, true)`, userID, topicID, start.AddDate(0, 0, 1)).Error)

	mkRoadmap := func() *Roadmap {
		return &Roadmap{
			UserID: userID, TrackID: uuid.New(), ProfileID: uuid.New(),
			StartDate: start, EndDate: start.AddDate(0, 0, 6), TotalWeeks: 1, HoursPerWeek: 10,
			Status: "active", IsActive: true,
			Weeks: []RoadmapWeek{{
				WeekNumber: 1, StartDate: start, EndDate: start.AddDate(0, 0, 6),
				FocusPillars: JSONStringArray{"dsa"}, PlannedHours: 8,
				Days: []PlanDay{{
					Date: start, PlannedMinutes: 90,
					Tasks: []PlanTask{
						{Kind: "study", ItemType: "topic", ItemID: topicID, PillarType: "dsa", Title: "Study: Arrays", EstimatedMinutes: 60, Priority: "high", Objectives: JSONStringArray{}, Status: "pending", SortOrder: 0},
						{Kind: "solve", ItemType: "problem", ItemID: problemID, PillarType: "dsa", Title: "Solve: Two Sum", EstimatedMinutes: 30, Priority: "high", Objectives: JSONStringArray{}, Status: "pending", SortOrder: 1},
					},
				}},
			}},
		}
	}

	// First generation (not a regenerate): no carry-over, tasks stay pending.
	rm := mkRoadmap()
	require.NoError(t, repo.CreateGraph(ctx, rm, false))

	count := func(table, where string, args ...any) int64 {
		var c int64
		require.NoError(t, tx.Table(table).Where(where, args...).Count(&c).Error)
		return c
	}

	utpBefore := count("user_topic_progress", "user_id = ? AND deleted_at IS NULL", userID)
	uppBefore := count("user_problem_progress", "user_id = ? AND deleted_at IS NULL", userID)
	revBefore := count("revision_items", "user_id = ? AND deleted_at IS NULL", userID)
	require.EqualValues(t, 1, utpBefore)
	require.EqualValues(t, 1, uppBefore)
	require.EqualValues(t, 1, revBefore)

	// Regenerate: archives the old roadmap, builds fresh, and carries completed
	// status forward onto the matching new tasks.
	rm2 := mkRoadmap()
	require.NoError(t, repo.CreateGraph(ctx, rm2, true))

	// Progress + revision rows must be untouched (NOT deleted).
	require.EqualValues(t, utpBefore, count("user_topic_progress", "user_id = ? AND deleted_at IS NULL", userID), "topic progress must survive regen")
	require.EqualValues(t, uppBefore, count("user_problem_progress", "user_id = ? AND deleted_at IS NULL", userID), "problem progress must survive regen")
	require.EqualValues(t, revBefore, count("revision_items", "user_id = ? AND deleted_at IS NULL", userID), "revision items must survive regen")

	// New roadmap's matching tasks are carried over to completed. Query plan_tasks
	// scoped to rm2's plan_days (GetPlanDay-by-date can't disambiguate the archived
	// roadmap's plan-day on the same date).
	var newTasks []PlanTask
	require.NoError(t, tx.
		Table("plan_tasks AS pt").
		Joins("JOIN plan_days pd ON pd.id = pt.plan_day_id").
		Joins("JOIN roadmap_weeks rw ON rw.id = pd.roadmap_week_id").
		Where("rw.roadmap_id = ?", rm2.ID).
		Order("pt.sort_order ASC").
		Find(&newTasks).Error)
	require.Len(t, newTasks, 2)
	byItem := map[uuid.UUID]PlanTask{}
	for _, tsk := range newTasks {
		byItem[tsk.ItemID] = tsk
	}
	require.Equal(t, "completed", byItem[topicID].Status, "completed topic must carry over")
	require.Equal(t, "completed", byItem[problemID].Status, "solved problem must carry over")
	require.NotNil(t, byItem[topicID].CompletedAt)
	require.NotNil(t, byItem[problemID].CompletedAt)
}
