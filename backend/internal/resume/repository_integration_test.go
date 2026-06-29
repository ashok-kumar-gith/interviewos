package resume

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
// Requires migrations 000001_auth and 000005_resume to have been applied.
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

	// A resume_profiles.user_id FK requires a real users row.
	userID := uuid.New()
	require.NoError(t, tx.Exec(
		`INSERT INTO users (id, email, role, status) VALUES (?, ?, 'user', 'active')`,
		userID, "resume-itest+"+uuid.NewString()+"@example.com",
	).Error)

	// Profile not found before creation.
	_, err = repo.GetProfileByUserID(ctx, userID)
	require.ErrorIs(t, err, ErrProfileNotFound)

	// CreateProfile.
	headline := "Senior Backend Engineer"
	yoe := 8.5
	profile := &Profile{
		ID:              uuid.New(),
		UserID:          userID,
		Headline:        &headline,
		YearsExperience: &yoe,
		Skills:          StringArray{"Go", "Postgres"},
		TargetKeywords:  StringArray{"Go", "Kafka"},
	}
	require.NoError(t, repo.CreateProfile(ctx, profile))

	got, err := repo.GetProfileByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, profile.ID, got.ID)
	require.Equal(t, "Senior Backend Engineer", *got.Headline)
	require.Equal(t, StringArray{"Go", "Postgres"}, got.Skills)
	require.Equal(t, StringArray{"Go", "Kafka"}, got.TargetKeywords)
	require.Empty(t, got.Projects)

	// UpdateProfile (only editable columns).
	newHead := "Staff Engineer"
	got.Headline = &newHead
	got.Skills = StringArray{"Go", "Kafka", "Kubernetes"}
	require.NoError(t, repo.UpdateProfile(ctx, got))
	reloaded, err := repo.GetProfileByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "Staff Engineer", *reloaded.Headline)
	require.Len(t, reloaded.Skills, 3)

	// CreateProject + ListProjects ordering by sort_order.
	desc := "Built a Go order service handling 50k rps"
	p2 := &Project{
		ID:              uuid.New(),
		ResumeProfileID: profile.ID,
		UserID:          userID,
		Name:            "Order Service",
		Description:     &desc,
		Metrics:         StringArray{"Reduced latency 40%"},
		TechStack:       StringArray{"Go", "Kafka"},
		SortOrder:       2,
	}
	p1 := &Project{
		ID:              uuid.New(),
		ResumeProfileID: profile.ID,
		UserID:          userID,
		Name:            "Auth Service",
		SortOrder:       1,
	}
	require.NoError(t, repo.CreateProject(ctx, p2))
	require.NoError(t, repo.CreateProject(ctx, p1))

	list, err := repo.ListProjects(ctx, profile.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "Auth Service", list[0].Name) // sort_order 1 first
	require.Equal(t, "Order Service", list[1].Name)
	require.Equal(t, StringArray{"Go", "Kafka"}, list[1].TechStack)

	// GetProject scoped by user_id.
	one, err := repo.GetProject(ctx, userID, p2.ID)
	require.NoError(t, err)
	require.Equal(t, "Order Service", one.Name)

	// Foreign user cannot fetch it.
	_, err = repo.GetProject(ctx, uuid.New(), p2.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)

	// UpdateProject.
	one.Name = "Order Service v2"
	one.SortOrder = 5
	require.NoError(t, repo.UpdateProject(ctx, one))
	one, err = repo.GetProject(ctx, userID, p2.ID)
	require.NoError(t, err)
	require.Equal(t, "Order Service v2", one.Name)
	require.Equal(t, 5, one.SortOrder)

	// UpdateScore persists ats_score + last_scored_at + ai_feedback.
	require.NoError(t, repo.UpdateScore(ctx, profile.ID, 87.5, []byte(`{"ok":true}`)))
	scored, err := repo.GetProfileByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, scored.ATSScore)
	require.InDelta(t, 87.5, *scored.ATSScore, 0.001)
	require.NotNil(t, scored.LastScoredAt)

	// DeleteProject soft-deletes (scoped by user_id) and removes from listing.
	require.NoError(t, repo.DeleteProject(ctx, userID, p1.ID))
	_, err = repo.GetProject(ctx, userID, p1.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)
	list, err = repo.ListProjects(ctx, profile.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}
