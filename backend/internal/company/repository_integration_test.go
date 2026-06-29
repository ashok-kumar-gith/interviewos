package company

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

// TestRepository_Integration exercises the Company Mode repository against a
// live PostgreSQL instance. Skipped unless DATABASE_URL is set; runs inside a
// rolled-back transaction.
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

	userID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO users (id, email) VALUES (?, ?)`,
		userID, "itest+"+uuid.NewString()+"@example.com").Error)

	trackID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO tracks (id, slug, name) VALUES (?, ?, ?)`,
		trackID, "itest-"+uuid.NewString()[:8], "Track").Error)

	require.NoError(t, tx.Exec(
		`INSERT INTO user_profiles (user_id, track_id, target_role, start_date)
		 VALUES (?, ?, 'SDE3', CURRENT_DATE)`, userID, trackID).Error)

	companyID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO companies (id, slug, name) VALUES (?, ?, 'Co')`,
		companyID, "itest-"+uuid.NewString()[:8]).Error)

	// GetProfile.
	p, err := repo.GetProfile(ctx, userID)
	require.NoError(t, err)
	require.Nil(t, p.TargetCompanyID, "no target set yet")

	// CompanyExists.
	exists, err := repo.CompanyExists(ctx, companyID)
	require.NoError(t, err)
	require.True(t, exists)
	exists, err = repo.CompanyExists(ctx, uuid.New())
	require.NoError(t, err)
	require.False(t, exists)

	// SetTargetCompany.
	updated, err := repo.SetTargetCompany(ctx, userID, &companyID, time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, updated.TargetCompanyID)
	require.Equal(t, companyID, *updated.TargetCompanyID)
}
