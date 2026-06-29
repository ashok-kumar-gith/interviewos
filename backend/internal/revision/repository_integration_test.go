package revision

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
// PostgreSQL instance. It is skipped unless DATABASE_URL is set, and runs inside
// a transaction that is always rolled back so it leaves no residue.
//
// Requires migrations 000001_auth and 000012_revision.
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

	userID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, "revision-itest+"+uuid.NewString()+"@example.com").Error)

	itemID := uuid.New()
	st := InitialState(now)
	it := &Item{
		UserID:       userID,
		ItemType:     ItemTopic,
		ItemID:       itemID,
		PillarType:   "dsa",
		Stage:        st.Stage,
		IntervalDays: st.IntervalDays,
		DueAt:        st.DueAt,
		IsActive:     st.IsActive,
	}

	// Create is idempotent on the active unique index.
	created, err := repo.Create(ctx, it)
	require.NoError(t, err)
	require.True(t, created)

	dup := &Item{UserID: userID, ItemType: ItemTopic, ItemID: itemID, PillarType: "dsa",
		Stage: 0, IntervalDays: 1, DueAt: st.DueAt, IsActive: true}
	created, err = repo.Create(ctx, dup)
	require.NoError(t, err)
	require.False(t, created, "duplicate active item must not be created")

	// GetByID resolves ownership.
	got, err := repo.GetByID(ctx, userID, it.ID)
	require.NoError(t, err)
	require.Equal(t, 0, got.Stage)
	require.Equal(t, 1, got.IntervalDays)
	require.True(t, got.IsActive)

	_, err = repo.GetByID(ctx, uuid.New(), it.ID)
	require.ErrorIs(t, err, ErrItemNotFound)

	// Not due yet (due tomorrow): ListDue today returns nothing.
	_, total, err := repo.ListDue(ctx, userID, DueFilter{OnDate: now, Limit: 20})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	// Backdate due_at and confirm it is now due.
	require.NoError(t, tx.Model(&Item{}).Where("id = ?", it.ID).
		Update("due_at", now.AddDate(0, 0, -1)).Error)
	due, total, err := repo.ListDue(ctx, userID, DueFilter{OnDate: now, Limit: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, due, 1)

	// Apply a correct recall transition and persist via Update.
	tr := Apply(got.Stage, RecallCorrect, now)
	got.Stage = tr.Stage
	got.IntervalDays = tr.IntervalDays
	got.DueAt = tr.DueAt
	got.IsActive = tr.IsActive
	lr := tr.LastRecall
	got.LastRecall = &lr
	got.LastReviewedAt = &now
	got.ReviewCount++
	require.NoError(t, repo.Update(ctx, got))

	after, err := repo.GetByID(ctx, userID, it.ID)
	require.NoError(t, err)
	require.Equal(t, 1, after.Stage)
	require.Equal(t, 3, after.IntervalDays)
	require.NotNil(t, after.LastRecall)
	require.Equal(t, RecallCorrect, *after.LastRecall)
	require.Equal(t, 1, after.ReviewCount)

	// Update against a non-owner is a not-found.
	stranger := *after
	stranger.UserID = uuid.New()
	require.ErrorIs(t, repo.Update(ctx, &stranger), ErrItemNotFound)
}
