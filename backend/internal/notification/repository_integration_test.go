package notification

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
// Requires migrations 000001_auth and 000010_notifications to have been applied
// to the target database (notifications FK→users).
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

	// Create — payload round-trips, defaults applied.
	body := "You have 3 spaced-repetition items due."
	n := &Notification{
		UserID:  owner,
		Type:    TypeRevisionDue,
		Title:   "3 items due for revision",
		Body:    &body,
		Payload: JSONMap{"due_count": float64(3), "kind": "revision"},
	}
	require.NoError(t, repo.Create(ctx, n))
	require.NotEqual(t, uuid.Nil, n.ID)

	// GetByID — owner sees it; defaults persisted (channel=in_app, status=unread).
	got, err := repo.GetByID(ctx, owner, n.ID)
	require.NoError(t, err)
	require.Equal(t, ChannelInApp, got.Channel)
	require.Equal(t, StatusUnread, got.Status)
	require.NotNil(t, got.Body)
	require.Equal(t, float64(3), got.Payload["due_count"])
	require.Nil(t, got.ReadAt)

	// GetByID — other user is denied (ownership scoping).
	_, err = repo.GetByID(ctx, other, n.ID)
	require.ErrorIs(t, err, ErrNotFound)

	// A second, already-created notification for the owner (kept unread).
	n2 := &Notification{UserID: owner, Type: TypeStreakReminder, Title: "Keep your streak alive"}
	require.NoError(t, repo.Create(ctx, n2))

	// A notification for the other user (scoping check on list / mark-all).
	otherNotif := &Notification{UserID: other, Type: TypeSystem, Title: "Other user"}
	require.NoError(t, repo.Create(ctx, otherNotif))

	// List — owner sees 2, scoped, newest-first.
	items, total, err := repo.List(ctx, owner, ListFilter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	// List with status filter (all unread initially).
	unread := StatusUnread
	items, total, err = repo.List(ctx, owner, ListFilter{Status: &unread, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	// Pagination: limit 1 returns 1 row but total still 2.
	items, total, err = repo.List(ctx, owner, ListFilter{Limit: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 1)

	// MarkRead — non-owner denied.
	_, err = repo.MarkRead(ctx, other, n.ID)
	require.ErrorIs(t, err, ErrNotFound)

	// MarkRead — owner flips unread → read and stamps read_at.
	read, err := repo.MarkRead(ctx, owner, n.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRead, read.Status)
	require.NotNil(t, read.ReadAt)

	// MarkRead is idempotent: marking an already-read row keeps it read.
	firstReadAt := *read.ReadAt
	read2, err := repo.MarkRead(ctx, owner, n.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRead, read2.Status)
	require.NotNil(t, read2.ReadAt)
	require.WithinDuration(t, firstReadAt, *read2.ReadAt, 0) // unchanged

	// Status filter now reflects the read row.
	readStatus := StatusRead
	_, total, err = repo.List(ctx, owner, ListFilter{Status: &readStatus, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	// MarkAllRead — clears the owner's remaining unread (n2), returns count=1,
	// and does NOT touch the other user's notification.
	count, err := repo.MarkAllRead(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	_, total, err = repo.List(ctx, owner, ListFilter{Status: &unread, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	// Other user still has their unread notification.
	otherGot, err := repo.GetByID(ctx, other, otherNotif.ID)
	require.NoError(t, err)
	require.Equal(t, StatusUnread, otherGot.Status)

	// MarkAllRead again is a no-op (zero rows).
	count, err = repo.MarkAllRead(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

// seedUser inserts a bare user row and returns its id.
func seedUser(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "notif+" + id.String() + "@example.com"
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		id, email,
	).Error)
	return id
}
