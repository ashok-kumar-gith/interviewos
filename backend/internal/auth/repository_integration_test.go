package auth

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
// Requires migration 000001_auth to have been applied to the target database.
func TestRepository_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Run everything in a transaction and roll back at the end.
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// CreateUser + GetUserByEmail/ID.
	email := "itest+" + uuid.NewString() + "@example.com"
	pw := "hashed-pw"
	u := &User{Email: email, PasswordHash: &pw, Role: RoleUser, Status: StatusActive}
	require.NoError(t, repo.CreateUser(ctx, u))
	require.NotEqual(t, uuid.Nil, u.ID)

	byEmail, err := repo.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, u.ID, byEmail.ID)

	byID, err := repo.GetUserByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, email, byID.Email)

	// UpdateLastLogin / UpdatePassword.
	require.NoError(t, repo.UpdateLastLogin(ctx, u.ID, now))
	require.NoError(t, repo.UpdatePassword(ctx, u.ID, "new-hash"))
	reloaded, err := repo.GetUserByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.LastLoginAt)
	require.Equal(t, "new-hash", *reloaded.PasswordHash)

	// Refresh token lifecycle: create, get, rotate (revoke + replaced_by), reuse.
	family := uuid.New()
	rt := &RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: HashToken("integration-refresh-" + uuid.NewString()),
		FamilyID:  family,
		ExpiresAt: now.Add(time.Hour),
	}
	require.NoError(t, repo.CreateRefreshToken(ctx, rt))

	got, err := repo.GetRefreshTokenByHash(ctx, rt.TokenHash)
	require.NoError(t, err)
	require.True(t, got.IsActive(now))

	// The replacement token must exist before it can be referenced by
	// replaced_by (FK refresh_tokens_replaced_by_fkey) — mirror how the service
	// inserts the rotated token before revoking the old one.
	replacementTok := &RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: HashToken("integration-refresh-replacement-" + uuid.NewString()),
		FamilyID:  family,
		ExpiresAt: now.Add(time.Hour),
	}
	require.NoError(t, repo.CreateRefreshToken(ctx, replacementTok))
	replacement := replacementTok.ID
	require.NoError(t, repo.RevokeRefreshToken(ctx, rt.ID, &replacement, now))
	got, err = repo.GetRefreshTokenByHash(ctx, rt.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, got.RevokedAt)
	require.NotNil(t, got.ReplacedBy)

	// RevokeAllUserRefreshTokens on a fresh active token.
	rt2 := &RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: HashToken("integration-refresh2-" + uuid.NewString()),
		FamilyID:  family,
		ExpiresAt: now.Add(time.Hour),
	}
	require.NoError(t, repo.CreateRefreshToken(ctx, rt2))
	require.NoError(t, repo.RevokeAllUserRefreshTokens(ctx, u.ID, now))
	got2, err := repo.GetRefreshTokenByHash(ctx, rt2.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, got2.RevokedAt)

	// Password reset token lifecycle.
	prt := &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: HashToken("integration-reset-" + uuid.NewString()),
		ExpiresAt: now.Add(time.Hour),
	}
	require.NoError(t, repo.CreatePasswordResetToken(ctx, prt))
	gotReset, err := repo.GetResetTokenByHash(ctx, prt.TokenHash)
	require.NoError(t, err)
	require.Nil(t, gotReset.UsedAt)
	require.NoError(t, repo.MarkResetTokenUsed(ctx, prt.ID, now))
	gotReset, err = repo.GetResetTokenByHash(ctx, prt.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, gotReset.UsedAt)

	// OAuth upsert + find.
	subject := "subject-" + uuid.NewString()
	acct := &OAuthAccount{UserID: u.ID, Provider: ProviderGoogle, ProviderUserID: subject}
	require.NoError(t, repo.UpsertOAuthAccount(ctx, acct))
	found, err := repo.FindOAuthAccount(ctx, ProviderGoogle, subject)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, u.ID, found.UserID)

	missing, err := repo.FindOAuthAccount(ctx, ProviderGitHub, "no-such-subject")
	require.NoError(t, err)
	require.Nil(t, missing)
}
