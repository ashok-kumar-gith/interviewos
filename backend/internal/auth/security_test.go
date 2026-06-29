package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// captureAudit records audit events for assertions.
type captureAudit struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (a *captureAudit) Record(_ context.Context, ev AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, ev)
}

func (a *captureAudit) actions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.events))
	for i, e := range a.events {
		out[i] = e.Action
	}
	return out
}

func (a *captureAudit) has(action string) bool {
	for _, x := range a.actions() {
		if x == action {
			return true
		}
	}
	return false
}

// fakeDataRepo is an in-memory DataRepository for export/delete unit tests.
type fakeDataRepo struct {
	export        map[string][]map[string]any
	softDeleted   int64
	userDeleted   bool
	deleteDataErr error
}

func (f *fakeDataRepo) ExportUserData(_ context.Context, _ uuid.UUID) (map[string][]map[string]any, error) {
	return f.export, nil
}

func (f *fakeDataRepo) SoftDeleteUserData(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	if f.deleteDataErr != nil {
		return 0, f.deleteDataErr
	}
	return f.softDeleted, nil
}

func (f *fakeDataRepo) SoftDeleteUser(_ context.Context, _ uuid.UUID, _ time.Time) error {
	f.userDeleted = true
	return nil
}

func newSecurityService(t *testing.T, repo Repository, data DataRepository, audit AuditLogger, cost int) *Service {
	t.Helper()
	tm := newTestTokenManager(t, nil)
	return NewService(ServiceConfig{
		Repo:       repo,
		Data:       data,
		Tokens:     tm,
		Mailer:     &captureMailer{},
		OAuth:      NewOAuthRegistry(NewUnconfiguredProvider(ProviderGoogle), NewUnconfiguredProvider(ProviderGitHub)),
		Audit:      audit,
		BcryptCost: cost,
	})
}

func TestPasswordPolicy_BcryptCostAtLeast12(t *testing.T) {
	repo := newFakeRepo()
	svc := newSecurityService(t, repo, &fakeDataRepo{}, &captureAudit{}, 0) // 0 -> clamped to 12

	_, err := svc.Register(context.Background(), "cost@example.com", "Str0ngPass!7", "", RequestContext{})
	require.NoError(t, err)

	u, err := repo.GetUserByEmail(context.Background(), "cost@example.com")
	require.NoError(t, err)
	require.NotNil(t, u.PasswordHash)

	cost, err := bcrypt.Cost([]byte(*u.PasswordHash))
	require.NoError(t, err)
	require.GreaterOrEqual(t, cost, 12, "new hashes must use bcrypt cost >= 12 (NFR-SEC)")
}

func TestPasswordPolicy_RejectsCommonPasswordOnRegister(t *testing.T) {
	svc := newSecurityService(t, newFakeRepo(), &fakeDataRepo{}, &captureAudit{}, 12)

	_, err := svc.Register(context.Background(), "weak@example.com", "password123", "", RequestContext{})
	require.ErrorIs(t, err, ErrPasswordTooCommon)
}

func TestPasswordPolicy_RejectsCommonPasswordOnReset(t *testing.T) {
	repo := newFakeRepo()
	mailer := &captureMailer{}
	tm := newTestTokenManager(t, nil)
	svc := NewService(ServiceConfig{
		Repo: repo, Data: &fakeDataRepo{}, Tokens: tm, Mailer: mailer,
		OAuth:      NewOAuthRegistry(NewUnconfiguredProvider(ProviderGoogle), NewUnconfiguredProvider(ProviderGitHub)),
		BcryptCost: 12,
	})

	_, err := svc.Register(context.Background(), "rc@example.com", "Str0ngPass!7", "", RequestContext{})
	require.NoError(t, err)
	require.NoError(t, svc.ForgotPassword(context.Background(), "rc@example.com", RequestContext{}))
	require.NotEmpty(t, mailer.lastToken)

	err = svc.ResetPassword(context.Background(), mailer.lastToken, "password", RequestContext{})
	require.ErrorIs(t, err, ErrPasswordTooCommon)
}

func TestPasswordPolicy_RejectsShortPassword(t *testing.T) {
	svc := newSecurityService(t, newFakeRepo(), &fakeDataRepo{}, &captureAudit{}, 12)
	_, err := svc.Register(context.Background(), "short@example.com", "ab12", "", RequestContext{})
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestAudit_RegisterAndLoginRecorded(t *testing.T) {
	repo := newFakeRepo()
	audit := &captureAudit{}
	svc := newSecurityService(t, repo, &fakeDataRepo{}, audit, 12)

	_, err := svc.Register(context.Background(), "audit@example.com", "Str0ngPass!7", "", RequestContext{IPAddress: "1.2.3.4", UserAgent: "go-test"})
	require.NoError(t, err)
	require.True(t, audit.has(ActionRegister), "register must write an audit row")

	_, err = svc.Login(context.Background(), "audit@example.com", "Str0ngPass!7", RequestContext{IPAddress: "1.2.3.4"})
	require.NoError(t, err)
	require.True(t, audit.has(ActionLoginSuccess), "login success must write an audit row")

	_, err = svc.Login(context.Background(), "audit@example.com", "wrongpass!", RequestContext{})
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.True(t, audit.has(ActionLoginFailure), "login failure must write an audit row")
}

func TestExportData_ReturnsBundle(t *testing.T) {
	repo := newFakeRepo()
	data := &fakeDataRepo{export: map[string][]map[string]any{
		"behavioral_stories": {{"id": "s1", "title": "story"}},
		"notifications":      {},
	}}
	audit := &captureAudit{}
	svc := newSecurityService(t, repo, data, audit, 12)

	reg, err := svc.Register(context.Background(), "export@example.com", "Str0ngPass!7", "", RequestContext{})
	require.NoError(t, err)

	bundle, err := svc.ExportData(context.Background(), reg.User.ID, RequestContext{})
	require.NoError(t, err)
	require.NotNil(t, bundle.User)
	require.Equal(t, "export@example.com", bundle.User.Email)
	require.Nil(t, bundle.User.PasswordHash, "export must not leak the password hash")
	require.Contains(t, bundle.Data, "behavioral_stories")
	require.Len(t, bundle.Data["behavioral_stories"], 1)
	require.True(t, audit.has(ActionDataExported))
	require.NotEmpty(t, bundle.ExportedAt)
}

func TestDeleteAccount_SoftDeletesAndRevokesTokens(t *testing.T) {
	repo := newFakeRepo()
	data := &fakeDataRepo{softDeleted: 7}
	audit := &captureAudit{}
	svc := newSecurityService(t, repo, data, audit, 12)

	reg, err := svc.Register(context.Background(), "del@example.com", "Str0ngPass!7", "", RequestContext{})
	require.NoError(t, err)

	err = svc.DeleteAccount(context.Background(), reg.User.ID, RequestContext{})
	require.NoError(t, err)

	require.True(t, data.userDeleted, "users row must be soft-deleted")
	require.True(t, audit.has(ActionAccountDeleted))

	// All refresh tokens revoked: the original refresh token is now invalid.
	_, err = svc.Refresh(context.Background(), reg.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestCommonPasswordDenylist_LoadedAndSized(t *testing.T) {
	require.GreaterOrEqual(t, len(commonPasswords), 200, "denylist must contain >= 200 entries (NFR-SEC)")
	require.True(t, isCommonPassword("PASSWORD"), "denylist match must be case-insensitive")
	require.False(t, isCommonPassword("Str0ngPass!7"))
}
