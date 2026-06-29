package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// fakeRepository is an in-memory Repository for unit tests.
type fakeRepository struct {
	users         map[uuid.UUID]*User
	usersByEmail  map[string]uuid.UUID
	refresh       map[uuid.UUID]*RefreshToken // keyed by id
	refreshByHash map[string]uuid.UUID
	resets        map[uuid.UUID]*PasswordResetToken
	resetsByHash  map[string]uuid.UUID
	oauth         map[string]*OAuthAccount // key: provider|subject

	createUserErr error
}

func newFakeRepo() *fakeRepository {
	return &fakeRepository{
		users:         map[uuid.UUID]*User{},
		usersByEmail:  map[string]uuid.UUID{},
		refresh:       map[uuid.UUID]*RefreshToken{},
		refreshByHash: map[string]uuid.UUID{},
		resets:        map[uuid.UUID]*PasswordResetToken{},
		resetsByHash:  map[string]uuid.UUID{},
		oauth:         map[string]*OAuthAccount{},
	}
}

func (f *fakeRepository) CreateUser(_ context.Context, u *User) error {
	if f.createUserErr != nil {
		return f.createUserErr
	}
	if _, ok := f.usersByEmail[u.Email]; ok {
		return errors.New("duplicate key value violates unique constraint")
	}
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	cp := *u
	f.users[u.ID] = &cp
	f.usersByEmail[u.Email] = u.ID
	return nil
}

func (f *fakeRepository) GetUserByEmail(_ context.Context, email string) (*User, error) {
	id, ok := f.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *f.users[id]
	return &cp, nil
}

func (f *fakeRepository) GetUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepository) UpdateLastLogin(_ context.Context, id uuid.UUID, at time.Time) error {
	if u, ok := f.users[id]; ok {
		u.LastLoginAt = &at
	}
	return nil
}

func (f *fakeRepository) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if u, ok := f.users[id]; ok {
		u.PasswordHash = &hash
	}
	return nil
}

func (f *fakeRepository) CreateRefreshToken(_ context.Context, t *RefreshToken) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	f.refresh[t.ID] = &cp
	f.refreshByHash[t.TokenHash] = t.ID
	return nil
}

func (f *fakeRepository) GetRefreshTokenByHash(_ context.Context, hash string) (*RefreshToken, error) {
	id, ok := f.refreshByHash[hash]
	if !ok {
		return nil, ErrRefreshInvalid
	}
	cp := *f.refresh[id]
	return &cp, nil
}

func (f *fakeRepository) RevokeRefreshToken(_ context.Context, id uuid.UUID, replacedBy *uuid.UUID, at time.Time) error {
	if t, ok := f.refresh[id]; ok && t.RevokedAt == nil {
		t.RevokedAt = &at
		t.ReplacedBy = replacedBy
	}
	return nil
}

func (f *fakeRepository) RevokeRefreshTokenFamily(_ context.Context, familyID uuid.UUID, at time.Time) error {
	for _, t := range f.refresh {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			rt := at
			t.RevokedAt = &rt
		}
	}
	return nil
}

func (f *fakeRepository) RevokeAllUserRefreshTokens(_ context.Context, userID uuid.UUID, at time.Time) error {
	for _, t := range f.refresh {
		if t.UserID == userID && t.RevokedAt == nil {
			rt := at
			t.RevokedAt = &rt
		}
	}
	return nil
}

func (f *fakeRepository) CreatePasswordResetToken(_ context.Context, t *PasswordResetToken) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	f.resets[t.ID] = &cp
	f.resetsByHash[t.TokenHash] = t.ID
	return nil
}

func (f *fakeRepository) GetResetTokenByHash(_ context.Context, hash string) (*PasswordResetToken, error) {
	id, ok := f.resetsByHash[hash]
	if !ok {
		return nil, ErrResetInvalid
	}
	cp := *f.resets[id]
	return &cp, nil
}

func (f *fakeRepository) MarkResetTokenUsed(_ context.Context, id uuid.UUID, at time.Time) error {
	if t, ok := f.resets[id]; ok {
		t.UsedAt = &at
	}
	return nil
}

func (f *fakeRepository) UpsertOAuthAccount(_ context.Context, a *OAuthAccount) error {
	key := string(a.Provider) + "|" + a.ProviderUserID
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	cp := *a
	f.oauth[key] = &cp
	return nil
}

func (f *fakeRepository) FindOAuthAccount(_ context.Context, provider Provider, subject string) (*OAuthAccount, error) {
	if a, ok := f.oauth[string(provider)+"|"+subject]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, nil
}

// captureMailer records the last reset token "sent".
type captureMailer struct {
	lastEmail string
	lastToken string
	lastURL   string
}

func (m *captureMailer) SendPasswordReset(_ context.Context, email, token, url string) error {
	m.lastEmail, m.lastToken, m.lastURL = email, token, url
	return nil
}

func newTestService(t *testing.T, repo Repository, mailer Mailer, now func() time.Time) *Service {
	t.Helper()
	tm := newTestTokenManager(t, now)
	return NewService(ServiceConfig{
		Repo:   repo,
		Tokens: tm,
		Mailer: mailer,
		OAuth:  NewOAuthRegistry(NewUnconfiguredProvider(ProviderGoogle), NewUnconfiguredProvider(ProviderGitHub)),
		Logger: zap.NewNop(),
		Now:    now,
	})
}

func TestRegister_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)

	pair, err := svc.Register(context.Background(), "User@Example.com", "password123", "Test User", RequestContext{})
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
	require.Equal(t, "user@example.com", pair.User.Email, "email normalized")

	// Refresh token persisted as a hash, never plaintext.
	_, ok := repo.refreshByHash[HashToken(pair.RefreshToken)]
	require.True(t, ok)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)

	_, err := svc.Register(context.Background(), "dup@example.com", "password123", "", RequestContext{})
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "dup@example.com", "password123", "", RequestContext{})
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.Register(context.Background(), "a@example.com", "correct-horse", "", RequestContext{})
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), "a@example.com", "wrong-password", RequestContext{})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.Login(context.Background(), "nobody@example.com", "whatever1", RequestContext{})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_SuccessSetsLastLogin(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.Register(context.Background(), "a@example.com", "password123", "", RequestContext{})
	require.NoError(t, err)

	pair, err := svc.Login(context.Background(), "a@example.com", "password123", RequestContext{})
	require.NoError(t, err)
	require.NotNil(t, pair.User.LastLoginAt)
}

func TestRefresh_RotationAndReuseDetection(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)

	reg, err := svc.Register(context.Background(), "rot@example.com", "password123", "", RequestContext{})
	require.NoError(t, err)
	original := reg.RefreshToken

	// First refresh rotates: new token issued, old revoked.
	rotated, err := svc.Refresh(context.Background(), original, RequestContext{})
	require.NoError(t, err)
	require.NotEqual(t, original, rotated.RefreshToken)

	// Reusing the original (now revoked) token is detected and rejected.
	_, err = svc.Refresh(context.Background(), original, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)

	// Reuse detection revokes the whole family, so the rotated token is dead too.
	_, err = svc.Refresh(context.Background(), rotated.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestRefresh_InvalidToken(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.Refresh(context.Background(), "does-not-exist", RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestRefresh_Expired(t *testing.T) {
	repo := newFakeRepo()
	base := time.Now()
	current := base
	svc := newTestService(t, repo, &captureMailer{}, func() time.Time { return current })

	reg, err := svc.Register(context.Background(), "exp@example.com", "password123", "", RequestContext{})
	require.NoError(t, err)

	current = base.Add(721 * time.Hour) // past 720h refresh TTL
	_, err = svc.Refresh(context.Background(), reg.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestLogout_RevokesFamily(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)

	reg, err := svc.Register(context.Background(), "out@example.com", "password123", "", RequestContext{})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), reg.RefreshToken))

	_, err = svc.Refresh(context.Background(), reg.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestForgotAndResetPasswordFlow(t *testing.T) {
	repo := newFakeRepo()
	mailer := &captureMailer{}
	svc := newTestService(t, repo, mailer, nil)

	_, err := svc.Register(context.Background(), "reset@example.com", "oldpassword", "", RequestContext{})
	require.NoError(t, err)

	// Forgot always succeeds; mailer captures the token.
	require.NoError(t, svc.ForgotPassword(context.Background(), "reset@example.com"))
	require.NotEmpty(t, mailer.lastToken)
	require.Contains(t, mailer.lastURL, mailer.lastToken)

	// Reset with the captured token.
	require.NoError(t, svc.ResetPassword(context.Background(), mailer.lastToken, "newpassword1"))

	// Old password fails; new password works.
	_, err = svc.Login(context.Background(), "reset@example.com", "oldpassword", RequestContext{})
	require.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = svc.Login(context.Background(), "reset@example.com", "newpassword1", RequestContext{})
	require.NoError(t, err)

	// The reset token is single-use.
	require.ErrorIs(t, svc.ResetPassword(context.Background(), mailer.lastToken, "another12"), ErrResetInvalid)
}

func TestForgotPassword_UnknownEmailSilent(t *testing.T) {
	repo := newFakeRepo()
	mailer := &captureMailer{}
	svc := newTestService(t, repo, mailer, nil)
	require.NoError(t, svc.ForgotPassword(context.Background(), "ghost@example.com"))
	require.Empty(t, mailer.lastToken)
}

func TestResetPassword_Expired(t *testing.T) {
	repo := newFakeRepo()
	mailer := &captureMailer{}
	base := time.Now()
	current := base
	svc := newTestService(t, repo, mailer, func() time.Time { return current })

	_, err := svc.Register(context.Background(), "exp2@example.com", "oldpassword", "", RequestContext{})
	require.NoError(t, err)
	require.NoError(t, svc.ForgotPassword(context.Background(), "exp2@example.com"))

	current = base.Add(2 * time.Hour) // past 1h reset TTL
	require.ErrorIs(t, svc.ResetPassword(context.Background(), mailer.lastToken, "newpass12"), ErrResetInvalid)
}

func TestResetPassword_RevokesRefreshTokens(t *testing.T) {
	repo := newFakeRepo()
	mailer := &captureMailer{}
	svc := newTestService(t, repo, mailer, nil)

	reg, err := svc.Register(context.Background(), "revoke@example.com", "oldpassword", "", RequestContext{})
	require.NoError(t, err)
	require.NoError(t, svc.ForgotPassword(context.Background(), "revoke@example.com"))
	require.NoError(t, svc.ResetPassword(context.Background(), mailer.lastToken, "newpass123"))

	// Pre-existing refresh token is revoked by the reset.
	_, err = svc.Refresh(context.Background(), reg.RefreshToken, RequestContext{})
	require.ErrorIs(t, err, ErrRefreshInvalid)
}

func TestOAuthCallback_NotConfigured(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.OAuthCallback(context.Background(), "google", "code", "state", RequestContext{})
	require.ErrorIs(t, err, ErrOAuthNotConfigured)
}

func TestOAuthCallback_UnsupportedProvider(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(t, repo, &captureMailer{}, nil)
	_, err := svc.OAuthCallback(context.Background(), "facebook", "code", "state", RequestContext{})
	require.ErrorIs(t, err, ErrUnsupportedProvider)
}

func TestBcryptHashing(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcryptCost)
	require.NoError(t, err)
	require.NotEqual(t, "password123", string(hash))
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("password123")))
	require.Error(t, bcrypt.CompareHashAndPassword(hash, []byte("wrong")))
}
