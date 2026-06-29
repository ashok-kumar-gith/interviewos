package behavioral

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRepository_Integration exercises the gorm Repository against a live
// PostgreSQL instance. It is skipped unless DATABASE_URL is set, and it runs
// inside a transaction that is always rolled back so it leaves no residue.
//
// Requires migrations 000001_auth and 000004_behavioral to have been applied to
// the target database (behavioral_stories FK→users).
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

	// Seed two users directly (FK target) to test ownership scoping.
	owner := seedUser(t, tx)
	other := seedUser(t, tx)

	// Create.
	situation := "The on-call rotation was overwhelmed."
	story := &Story{
		UserID:    owner,
		Title:     "Tamed the incident backlog",
		Theme:     ThemeProductionIncident,
		Situation: &situation,
		Tags:      Tags{"oncall", "reliability"},
	}
	require.NoError(t, repo.Create(ctx, story))
	require.NotEqual(t, uuid.Nil, story.ID)

	// GetByID — owner sees it.
	got, err := repo.GetByID(ctx, owner, story.ID)
	require.NoError(t, err)
	require.Equal(t, story.Title, got.Title)
	require.Equal(t, Tags{"oncall", "reliability"}, got.Tags)
	require.NotNil(t, got.Situation)

	// GetByID — other user is denied (ownership scoping).
	_, err = repo.GetByID(ctx, other, story.ID)
	require.ErrorIs(t, err, ErrStoryNotFound)

	// Update — change fields + AI feedback, scoped to owner.
	got.Title = "Eliminated the incident backlog"
	got.AIImproved = true
	score := 88.5
	got.StrengthScore = &score
	got.AIFeedback = JSONMap{"suggestions": []any{"add metrics"}, "used_fallback": true}
	require.NoError(t, repo.Update(ctx, got))

	reloaded, err := repo.GetByID(ctx, owner, story.ID)
	require.NoError(t, err)
	require.Equal(t, "Eliminated the incident backlog", reloaded.Title)
	require.True(t, reloaded.AIImproved)
	require.NotNil(t, reloaded.StrengthScore)
	require.InDelta(t, 88.5, *reloaded.StrengthScore, 0.001)
	require.NotNil(t, reloaded.AIFeedback)
	require.Equal(t, true, reloaded.AIFeedback["used_fallback"])

	// Update by a non-owner must not match any row.
	bad := *reloaded
	bad.UserID = other
	require.ErrorIs(t, repo.Update(ctx, &bad), ErrStoryNotFound)

	// List with theme filter + count.
	second := &Story{UserID: owner, Title: "Led the rewrite", Theme: ThemeLeadership}
	require.NoError(t, repo.Create(ctx, second))

	items, total, err := repo.List(ctx, owner, ListFilter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	theme := ThemeLeadership
	items, total, err = repo.List(ctx, owner, ListFilter{Theme: &theme, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, ThemeLeadership, items[0].Theme)

	// List query search by title.
	items, total, err = repo.List(ctx, owner, ListFilter{Query: "rewrite", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "Led the rewrite", items[0].Title)

	// Other user's list is empty (scoping).
	_, total, err = repo.List(ctx, other, ListFilter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	// Delete (soft) — non-owner denied, owner succeeds, then gone.
	require.ErrorIs(t, repo.Delete(ctx, other, story.ID), ErrStoryNotFound)
	require.NoError(t, repo.Delete(ctx, owner, story.ID))
	_, err = repo.GetByID(ctx, owner, story.ID)
	require.ErrorIs(t, err, ErrStoryNotFound)

	// Deleting again is a no-op not-found.
	require.ErrorIs(t, repo.Delete(ctx, owner, story.ID), ErrStoryNotFound)
}

// seedUser inserts a bare user row and returns its id.
func seedUser(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "behav+" + id.String() + "@example.com"
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		id, email,
	).Error)
	return id
}
