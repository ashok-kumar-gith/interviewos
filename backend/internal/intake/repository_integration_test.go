package intake

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
// Requires migrations 000001_auth and 000003_user_profiles to be applied.
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

	// Seed a user (user_profiles.user_id FK -> users.id).
	userID := uuid.New()
	email := "intake-itest+" + uuid.NewString() + "@example.com"
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, email,
	).Error)

	// No profile yet.
	_, err = repo.GetByUserID(ctx, userID)
	require.ErrorIs(t, err, ErrProfileNotFound)

	// Insert via Upsert.
	companyID := uuid.New()
	p := &Profile{
		UserID:          userID,
		TrackID:         uuid.New(),
		YearsExperience: 7.5,
		TargetCompanyID: &companyID,
		TargetRole:      "SDE3 / Senior Backend",
		HoursPerWeek:    20,
		StartDate:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		TargetWeeks:     12,
		PillarStrengths: []byte(`{"dsa":3,"system_design":2}`),
		Timezone:        "UTC",
		IntakeAnswers:   []byte(`{}`),
	}
	require.NoError(t, repo.Upsert(ctx, p))
	require.NotEqual(t, uuid.Nil, p.ID, "id should be populated after insert")
	insertedID := p.ID

	// GetByUserID returns the persisted row.
	got, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, insertedID, got.ID)
	require.Equal(t, "SDE3 / Senior Backend", got.TargetRole)
	require.Equal(t, int16(20), got.HoursPerWeek)
	require.NotNil(t, got.TargetCompanyID)
	require.Equal(t, companyID, *got.TargetCompanyID)
	require.JSONEq(t, `{"dsa":3,"system_design":2}`, string(got.PillarStrengths))

	// Update via Upsert keeps the same id (one active profile per user).
	upd := &Profile{
		UserID:          userID,
		TrackID:         got.TrackID,
		YearsExperience: 8,
		TargetRole:      "Staff Backend",
		HoursPerWeek:    35,
		StartDate:       got.StartDate,
		TargetWeeks:     16,
		PillarStrengths: []byte(`{"dsa":5}`),
		Timezone:        "Asia/Kolkata",
		IntakeAnswers:   []byte(`{"source":"wizard"}`),
	}
	require.NoError(t, repo.Upsert(ctx, upd))
	require.Equal(t, insertedID, upd.ID, "upsert must update the existing row, not insert a new one")

	reloaded, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, insertedID, reloaded.ID)
	require.Equal(t, "Staff Backend", reloaded.TargetRole)
	require.Equal(t, int16(35), reloaded.HoursPerWeek)
	require.Equal(t, int16(16), reloaded.TargetWeeks)
	require.Equal(t, "Asia/Kolkata", reloaded.Timezone)
	require.Nil(t, reloaded.TargetCompanyID, "company cleared on update")
	require.JSONEq(t, `{"dsa":5}`, string(reloaded.PillarStrengths))
}
