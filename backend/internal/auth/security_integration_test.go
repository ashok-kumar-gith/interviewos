package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openTestDB opens the integration DB or skips. Each test runs inside a
// transaction that is rolled back, leaving no residue.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

// TestAuditLogger_Integration verifies the audit writer persists a row with the
// expected columns against a live audit_logs table (migration 000016).
func TestAuditLogger_Integration(t *testing.T) {
	tx := openTestDB(t)
	ctx := context.Background()

	repo := NewRepository(tx)
	u := &User{Email: "audit-itest+" + uuid.NewString() + "@example.com", Role: RoleUser, Status: StatusActive}
	require.NoError(t, repo.CreateUser(ctx, u))

	logger := NewAuditLogger(tx, zap.NewNop())
	logger.Record(ctx, AuditEvent{
		UserID:    &u.ID,
		Action:    ActionLoginSuccess,
		IPAddress: "203.0.113.9",
		UserAgent: "integration-agent",
		Metadata:  map[string]any{"email": u.Email},
	})

	var count int64
	require.NoError(t, tx.Table("audit_logs").
		Where("user_id = ? AND action = ?", u.ID, ActionLoginSuccess).
		Count(&count).Error)
	require.Equal(t, int64(1), count, "audit row must be persisted")
}

// TestServiceAudit_Integration verifies that register + login write audit rows
// through the full service against a live DB.
func TestServiceAudit_Integration(t *testing.T) {
	tx := openTestDB(t)
	ctx := context.Background()

	tm := newTestTokenManager(t, nil)
	svc := NewService(ServiceConfig{
		Repo:   NewRepository(tx),
		Data:   NewDataRepository(tx),
		Tokens: tm,
		Mailer: &captureMailer{},
		OAuth:  NewOAuthRegistry(NewUnconfiguredProvider(ProviderGoogle), NewUnconfiguredProvider(ProviderGitHub)),
		Audit:  NewAuditLogger(tx, zap.NewNop()),
		Logger: zap.NewNop(),
	})

	email := "svc-audit+" + uuid.NewString() + "@example.com"
	reg, err := svc.Register(ctx, email, "Str0ngPass!7", "", RequestContext{IPAddress: "198.51.100.5"})
	require.NoError(t, err)

	_, err = svc.Login(ctx, email, "Str0ngPass!7", RequestContext{IPAddress: "198.51.100.5"})
	require.NoError(t, err)

	var actions []string
	require.NoError(t, tx.Table("audit_logs").
		Where("user_id = ?", reg.User.ID).
		Order("created_at").
		Pluck("action", &actions).Error)
	require.Contains(t, actions, ActionRegister)
	require.Contains(t, actions, ActionLoginSuccess)
}

// TestDataExportAndDelete_Integration verifies the DataRepository exports
// user-owned rows and soft-deletes them (and the user) against the live schema.
func TestDataExportAndDelete_Integration(t *testing.T) {
	tx := openTestDB(t)
	ctx := context.Background()

	repo := NewRepository(tx)
	data := NewDataRepository(tx)

	u := &User{Email: "data-itest+" + uuid.NewString() + "@example.com", Role: RoleUser, Status: StatusActive}
	require.NoError(t, repo.CreateUser(ctx, u))

	// Seed a user-owned row (notifications carries user_id + deleted_at).
	require.NoError(t, tx.Exec(
		`INSERT INTO notifications (id, user_id, type, title, body, created_at, updated_at)
		 VALUES (gen_random_uuid(), ?, 'system', 'hi', 'body', now(), now())`, u.ID).Error)

	// Export includes the seeded notification.
	export, err := data.ExportUserData(ctx, u.ID)
	require.NoError(t, err)
	require.Contains(t, export, "notifications")
	require.Len(t, export["notifications"], 1)

	// Soft-delete the user's data, then the user.
	now := time.Now().UTC()
	rows, err := data.SoftDeleteUserData(ctx, u.ID, now)
	require.NoError(t, err)
	require.GreaterOrEqual(t, rows, int64(1))
	require.NoError(t, data.SoftDeleteUser(ctx, u.ID, now))

	// Live export is now empty and the user row is soft-deleted.
	export2, err := data.ExportUserData(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, export2["notifications"], 0)

	var deletedAt *time.Time
	require.NoError(t, tx.Table("users").Where("id = ?", u.ID).Pluck("deleted_at", &deletedAt).Error)
	require.NotNil(t, deletedAt, "users row must be soft-deleted")
}

// TestDeleteAccountRevokesTokens_Integration verifies DeleteAccount revokes all
// refresh tokens through the full service against a live DB.
func TestDeleteAccountRevokesTokens_Integration(t *testing.T) {
	tx := openTestDB(t)
	ctx := context.Background()

	tm := newTestTokenManager(t, nil)
	svc := NewService(ServiceConfig{
		Repo:   NewRepository(tx),
		Data:   NewDataRepository(tx),
		Tokens: tm,
		Mailer: &captureMailer{},
		OAuth:  NewOAuthRegistry(NewUnconfiguredProvider(ProviderGoogle), NewUnconfiguredProvider(ProviderGitHub)),
		Audit:  NewAuditLogger(tx, zap.NewNop()),
		Logger: zap.NewNop(),
	})

	email := "del-itest+" + uuid.NewString() + "@example.com"
	reg, err := svc.Register(ctx, email, "Str0ngPass!7", "", RequestContext{})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteAccount(ctx, reg.User.ID, RequestContext{}))

	// The refresh token issued at registration is now revoked.
	_, err = svc.Refresh(ctx, reg.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)

	// An account-deleted audit row exists.
	var count int64
	require.NoError(t, tx.Table("audit_logs").
		Where("user_id = ? AND action = ?", reg.User.ID, ActionAccountDeleted).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}
