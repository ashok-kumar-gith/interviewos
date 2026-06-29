package mock

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
// Requires migrations 000001_auth, 000002_content, and 000009_mock to have been
// applied to the target database (mock_interviews FK→users/topics/companies,
// mock_findings FK→mock_interviews/users).
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

	owner := seedUser(t, tx)
	other := seedUser(t, tx)

	// Create a mock interview.
	score := 72.5
	dur := 45
	m := &Interview{
		UserID:          owner,
		Type:            TypeCoding,
		Outcome:         OutcomeLeanHire,
		OverallScore:    &score,
		DurationMinutes: &dur,
	}
	require.NoError(t, repo.Create(ctx, m))
	require.NotEqual(t, uuid.Nil, m.ID)

	// GetByID — owner sees it; other is denied.
	got, err := repo.GetByID(ctx, owner, m.ID)
	require.NoError(t, err)
	require.Equal(t, TypeCoding, got.Type)
	require.Equal(t, OutcomeLeanHire, got.Outcome)
	require.NotNil(t, got.OverallScore)
	require.InDelta(t, 72.5, *got.OverallScore, 0.001)

	_, err = repo.GetByID(ctx, other, m.ID)
	require.ErrorIs(t, err, ErrMockNotFound)

	// Update — scoped to owner.
	got.Type = TypeSystemDesign
	got.Outcome = OutcomeHire
	require.NoError(t, repo.Update(ctx, got))

	reloaded, err := repo.GetByID(ctx, owner, m.ID)
	require.NoError(t, err)
	require.Equal(t, TypeSystemDesign, reloaded.Type)
	require.Equal(t, OutcomeHire, reloaded.Outcome)

	// Non-owner update must not match.
	bad := *reloaded
	bad.UserID = other
	require.ErrorIs(t, repo.Update(ctx, &bad), ErrMockNotFound)

	// Add findings (with and without a pillar).
	dsa := PillarDSA
	f1 := &Finding{
		MockInterviewID: m.ID,
		UserID:          owner,
		PillarType:      &dsa,
		Severity:        SeverityMajor,
		Category:        "correctness",
		Detail:          "missed an edge case",
	}
	require.NoError(t, repo.AddFinding(ctx, f1))
	require.NotEqual(t, uuid.Nil, f1.ID)

	f2 := &Finding{
		MockInterviewID: m.ID,
		UserID:          owner,
		Severity:        SeverityMinor,
		Category:        "communication",
		Detail:          "rambled on the approach",
	}
	require.NoError(t, repo.AddFinding(ctx, f2))

	// GetByIDWithFindings preloads findings ordered by created_at ASC.
	detail, err := repo.GetByIDWithFindings(ctx, owner, m.ID)
	require.NoError(t, err)
	require.Len(t, detail.Findings, 2)
	require.Equal(t, "correctness", detail.Findings[0].Category)
	require.NotNil(t, detail.Findings[0].PillarType)
	require.Equal(t, PillarDSA, *detail.Findings[0].PillarType)

	// ListFindings is user-scoped.
	findings, err := repo.ListFindings(ctx, owner)
	require.NoError(t, err)
	require.Len(t, findings, 2)
	other2, err := repo.ListFindings(ctx, other)
	require.NoError(t, err)
	require.Len(t, other2, 0)

	// List with type filter + count.
	second := &Interview{UserID: owner, Type: TypeBehavioral, Outcome: OutcomeNotRated}
	require.NoError(t, repo.Create(ctx, second))

	items, total, err := repo.List(ctx, owner, ListFilter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	mt := TypeBehavioral
	items, total, err = repo.List(ctx, owner, ListFilter{Type: &mt, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, TypeBehavioral, items[0].Type)

	// Other user's list is empty (scoping).
	_, total, err = repo.List(ctx, other, ListFilter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	// Delete (soft) — non-owner denied, owner succeeds, then gone.
	require.ErrorIs(t, repo.Delete(ctx, other, m.ID), ErrMockNotFound)
	require.NoError(t, repo.Delete(ctx, owner, m.ID))
	_, err = repo.GetByID(ctx, owner, m.ID)
	require.ErrorIs(t, err, ErrMockNotFound)
	require.ErrorIs(t, repo.Delete(ctx, owner, m.ID), ErrMockNotFound)
}

// seedUser inserts a bare user row and returns its id.
func seedUser(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "mock+" + id.String() + "@example.com"
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		id, email,
	).Error)
	return id
}
