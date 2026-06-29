package analytics

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
// Requires migrations 000001-000013 applied to the target database.
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

	ctx := context.Background()
	repo := NewRepository(tx)

	// Seed: a user, a track + two pillars, three topics in pillar A, and a
	// profile pointing at the track. The base track/pillar from the migration
	// seed are insufficient (no topics), so we create our own.
	userID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email) VALUES (?, ?)`,
		userID, "itest+"+uuid.NewString()+"@example.com").Error)

	trackID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO tracks (id, slug, name) VALUES (?, ?, ?)`,
		trackID, "itest-"+uuid.NewString()[:8], "Integration Track").Error)

	pillarDSAID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO pillars (id, track_id, type, name, weight) VALUES (?, ?, 'dsa', 'DSA', 2.0)`,
		pillarDSAID, trackID).Error)
	pillarLLDID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO pillars (id, track_id, type, name, weight) VALUES (?, ?, 'lld', 'LLD', 1.0)`,
		pillarLLDID, trackID).Error)

	// Three DSA topics; user completes two (confidence 5 and 3), one untouched.
	topicA := uuid.New()
	topicB := uuid.New()
	topicC := uuid.New()
	for i, tid := range []uuid.UUID{topicA, topicB, topicC} {
		require.NoError(t, tx.Exec(
			`INSERT INTO topics (id, pillar_id, track_id, slug, name, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			tid, pillarDSAID, trackID, uuid.NewString()[:8], "Topic", i).Error)
	}

	require.NoError(t, tx.Exec(
		`INSERT INTO user_topic_progress (user_id, topic_id, status, confidence)
		 VALUES (?, ?, 'completed', 5)`, userID, topicA).Error)
	require.NoError(t, tx.Exec(
		`INSERT INTO user_topic_progress (user_id, topic_id, status, confidence)
		 VALUES (?, ?, 'completed', 3)`, userID, topicB).Error)

	require.NoError(t, tx.Exec(
		`INSERT INTO user_profiles (user_id, track_id, target_role, start_date)
		 VALUES (?, ?, 'SDE3', CURRENT_DATE)`, userID, trackID).Error)

	// --- PillarInputs ---
	inputs, err := repo.PillarInputs(ctx, userID)
	require.NoError(t, err)

	byPillar := map[string]PillarInputs{}
	for _, in := range inputs {
		byPillar[in.Pillar] = in
	}
	dsa, ok := byPillar["dsa"]
	require.True(t, ok, "dsa pillar present")
	require.Equal(t, 3, dsa.TotalItems, "3 in-scope dsa topics")
	require.Equal(t, 2, dsa.CompletedItems, "2 completed")
	require.InDelta(t, 4.0, dsa.AvgRating, 1e-9, "avg rating (5+3)/2")
	require.InDelta(t, 2.0, dsa.Weight, 1e-9, "dsa pillar weight")
	require.InDelta(t, 1.0, dsa.RevHealth, 1e-9, "revision health defaults to 1.0")

	// Calculator agreement: coverage 2/3, confidence (4-1)/4=0.75, rev 1.
	// readiness = 100 * (2/3) * (0.6*0.75 + 0.4*1) = 100*0.6667*0.85 ≈ 56.667.
	pr := ComputePillarReadiness(dsa, DefaultReadinessWeights())
	require.InDelta(t, 56.6667, pr.Readiness, 0.01, "per-pillar readiness")

	// --- TopicEntries / weak-strong ---
	entries, err := repo.TopicEntries(ctx, userID)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	weak, strong := bucketTopics(entries)
	require.NotEmpty(t, weak)
	require.NotEmpty(t, strong)
	require.Equal(t, topicC, weak[0].TopicID, "untouched topic is weakest")
	require.Equal(t, topicA, strong[0].TopicID, "confidence-5 completed topic is strongest")

	// --- StudySessions / TimeSpent ---
	now := time.Now().UTC()
	require.NoError(t, tx.Exec(
		`INSERT INTO study_sessions (user_id, pillar_type, started_at, duration_minutes, source)
		 VALUES (?, 'dsa', ?, 45, 'manual')`, userID, now).Error)
	require.NoError(t, tx.Exec(
		`INSERT INTO study_sessions (user_id, pillar_type, started_at, duration_minutes, source)
		 VALUES (?, 'lld', ?, 30, 'manual')`, userID, now).Error)

	tsPillar, err := repo.TimeSpent(ctx, userID, time.Time{}, time.Time{}, "pillar")
	require.NoError(t, err)
	require.Equal(t, 75, tsPillar.TotalMinutes)
	require.Equal(t, "pillar", tsPillar.GroupBy)

	tsDay, err := repo.TimeSpent(ctx, userID, time.Time{}, time.Time{}, "day")
	require.NoError(t, err)
	require.Equal(t, 75, tsDay.TotalMinutes)
	require.Len(t, tsDay.Buckets, 1, "single day bucket")

	// --- StreakDays ---
	for i := 0; i < 3; i++ {
		require.NoError(t, tx.Exec(
			`INSERT INTO streak_days (user_id, date, tasks_completed, minutes_studied, goal_met)
			 VALUES (?, ?, 1, 45, true)`, userID, now.AddDate(0, 0, -i).Format("2006-01-02")).Error)
	}
	days, err := repo.StreakDays(ctx, userID, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, days, 3)
	current, longest := computeStreak(days, now)
	require.Equal(t, 3, current)
	require.Equal(t, 3, longest)

	// --- UpsertSnapshot idempotency + ListSnapshots ---
	snap := Snapshot{
		UserID:           userID,
		SnapshotDate:     time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		OverallReadiness: 42.5,
		PillarReadiness:  map[string]float64{"dsa": 56.67, "lld": 0},
		CompletionPct:    33.33,
		WeakTopics:       []uuid.UUID{topicC},
		StrongTopics:     []uuid.UUID{topicA},
	}
	stored1, err := repo.UpsertSnapshot(ctx, snap)
	require.NoError(t, err)
	require.InDelta(t, 42.5, stored1.OverallReadiness, 1e-9)
	require.Equal(t, []uuid.UUID{topicC}, stored1.WeakTopics)
	require.Equal(t, 56.67, stored1.PillarReadiness["dsa"])

	// Re-upsert the same day with a new value: must update, not duplicate.
	snap.OverallReadiness = 60
	stored2, err := repo.UpsertSnapshot(ctx, snap)
	require.NoError(t, err)
	require.Equal(t, stored1.ID, stored2.ID, "same row id (idempotent per day)")
	require.InDelta(t, 60.0, stored2.OverallReadiness, 1e-9)

	list, total, err := repo.ListSnapshots(ctx, userID, time.Time{}, time.Time{}, 50, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "exactly one snapshot for the day")
	require.Len(t, list, 1)

	// --- Company Mode re-weighting via PillarInputs ---
	companyID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO companies (id, slug, name, is_fully_weighted)
		 VALUES (?, ?, 'Itest Co', true)`, companyID, "itest-"+uuid.NewString()[:8]).Error)
	// Heavy DSA multiplier (×5) for this company.
	require.NoError(t, tx.Exec(
		`INSERT INTO company_weights (company_id, pillar_id, weight_multiplier)
		 VALUES (?, ?, 5.0)`, companyID, pillarDSAID).Error)
	require.NoError(t, tx.Exec(
		`UPDATE user_profiles SET target_company_id = ? WHERE user_id = ?`,
		companyID, userID).Error)

	weighted, err := repo.PillarInputs(ctx, userID)
	require.NoError(t, err)
	for _, in := range weighted {
		if in.Pillar == "dsa" {
			// base 2.0 × multiplier 5.0 = 10.0.
			require.InDelta(t, 10.0, in.Weight, 1e-9, "company-weighted dsa weight")
		}
	}
}
