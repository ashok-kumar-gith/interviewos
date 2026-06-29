package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// TokenPair is the result of issuing access + refresh tokens for a user.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresIn  time.Duration
	RefreshExpiresAt time.Time
	User             *User
}

// RequestContext carries optional request metadata recorded on refresh tokens.
type RequestContext struct {
	UserAgent string
	IPAddress string
}

// Service implements the auth use-cases. It depends only on interfaces so it is
// unit-testable with fakes.
type Service struct {
	repo     Repository
	tokens   *TokenManager
	mailer   Mailer
	oauth    *OAuthRegistry
	log      *zap.Logger
	resetURL string // base URL for reset links, e.g. http://localhost:3000/reset-password
	now      func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo     Repository
	Tokens   *TokenManager
	Mailer   Mailer
	OAuth    *OAuthRegistry
	Logger   *zap.Logger
	ResetURL string
	Now      func() time.Time
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	resetURL := cfg.ResetURL
	if resetURL == "" {
		resetURL = "http://localhost:3000/reset-password"
	}
	return &Service{
		repo:     cfg.Repo,
		tokens:   cfg.Tokens,
		mailer:   cfg.Mailer,
		oauth:    cfg.OAuth,
		log:      cfg.Logger,
		resetURL: resetURL,
		now:      nowFn,
	}
}

// bcryptCost is intentionally the library default (10) for a balance of
// security and latency in local/dev; tune via deployment if needed.
const bcryptCost = bcrypt.DefaultCost

// Register creates a new email/password account and issues a token pair.
// Duplicate (active) email returns ErrEmailTaken.
func (s *Service) Register(ctx context.Context, email, password, fullName string, rc RequestContext) (*TokenPair, error) {
	email = normalizeEmail(email)

	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hashing password: %w", err)
	}
	hashStr := string(hash)

	u := &User{
		Email:        email,
		PasswordHash: &hashStr,
		Role:         RoleUser,
		Status:       StatusActive,
	}
	if fullName != "" {
		u.FullName = &fullName
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		// A race on the unique active-email index surfaces here.
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("auth: creating user: %w", err)
	}

	return s.issueTokenPair(ctx, u, uuid.New(), rc)
}

// Login verifies credentials, records last_login_at, and issues a token pair.
func (s *Service) Login(ctx context.Context, email, password string, rc RequestContext) (*TokenPair, error) {
	email = normalizeEmail(email)

	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Run a dummy compare to keep timing roughly constant.
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinv"), []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	if u.Status != StatusActive {
		return nil, ErrAccountInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	if err := s.repo.UpdateLastLogin(ctx, u.ID, now); err != nil {
		return nil, fmt.Errorf("auth: updating last login: %w", err)
	}
	u.LastLoginAt = &now

	return s.issueTokenPair(ctx, u, uuid.New(), rc)
}

// Refresh validates and ROTATES a refresh token: the presented token is revoked
// and a new token (same family) is issued. If a revoked/replaced token is
// presented (reuse / theft), the entire family is revoked and the call fails.
func (s *Service) Refresh(ctx context.Context, rawToken string, rc RequestContext) (*TokenPair, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, ErrRefreshInvalid
	}

	stored, err := s.repo.GetRefreshTokenByHash(ctx, HashToken(rawToken))
	if err != nil {
		return nil, err // ErrRefreshInvalid for not-found
	}

	now := s.now()

	// Reuse detection: a presented token that is already revoked means it was
	// rotated away (or the family was compromised). Revoke the whole family.
	if stored.RevokedAt != nil {
		_ = s.repo.RevokeRefreshTokenFamily(ctx, stored.FamilyID, now)
		s.log.Warn("refresh token reuse detected; revoking family",
			zap.String("user_id", stored.UserID.String()),
			zap.String("family_id", stored.FamilyID.String()),
		)
		return nil, ErrRefreshInvalid
	}
	if now.After(stored.ExpiresAt) {
		return nil, ErrRefreshInvalid
	}

	u, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return nil, ErrRefreshInvalid
	}
	if u.Status != StatusActive {
		return nil, ErrAccountInactive
	}

	// Issue the replacement token within the same family.
	pair, newID, err := s.issueRefreshAndAccess(ctx, u, stored.FamilyID, rc)
	if err != nil {
		return nil, err
	}

	// Revoke the old token, linking to its replacement.
	if err := s.repo.RevokeRefreshToken(ctx, stored.ID, &newID, now); err != nil {
		return nil, fmt.Errorf("auth: revoking rotated token: %w", err)
	}

	return pair, nil
}

// Logout revokes the presented refresh token's family. Idempotent: an
// unknown/invalid token returns nil so logout never leaks token validity.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil
	}
	stored, err := s.repo.GetRefreshTokenByHash(ctx, HashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrRefreshInvalid) {
			return nil
		}
		return err
	}
	return s.repo.RevokeRefreshTokenFamily(ctx, stored.FamilyID, s.now())
}

// ForgotPassword always succeeds from the caller's perspective (no account
// enumeration). When the email maps to an active account it creates a single-use
// reset token and "sends" it via the mailer.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil // silent
		}
		return err
	}
	if u.Status != StatusActive {
		return nil
	}

	raw, err := GenerateOpaqueToken()
	if err != nil {
		return err
	}
	now := s.now()
	prt := &PasswordResetToken{
		UserID:    u.ID,
		TokenHash: HashToken(raw),
		ExpiresAt: now.Add(s.tokens.ResetTTL()),
	}
	if err := s.repo.CreatePasswordResetToken(ctx, prt); err != nil {
		return fmt.Errorf("auth: creating reset token: %w", err)
	}

	resetLink := fmt.Sprintf("%s?token=%s", s.resetURL, raw)
	if err := s.mailer.SendPasswordReset(ctx, u.Email, raw, resetLink); err != nil {
		// Log but do not surface to the caller (still return 2xx).
		s.log.Error("auth: sending reset email", zap.Error(err))
	}
	return nil
}

// ResetPassword consumes a valid reset token, sets the new password, and revokes
// all of the user's refresh tokens (force re-login everywhere).
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return ErrResetInvalid
	}
	prt, err := s.repo.GetResetTokenByHash(ctx, HashToken(rawToken))
	if err != nil {
		return err // ErrResetInvalid for not-found
	}
	now := s.now()
	if prt.UsedAt != nil || now.After(prt.ExpiresAt) {
		return ErrResetInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("auth: hashing password: %w", err)
	}
	if err := s.repo.UpdatePassword(ctx, prt.UserID, string(hash)); err != nil {
		return fmt.Errorf("auth: updating password: %w", err)
	}
	if err := s.repo.MarkResetTokenUsed(ctx, prt.ID, now); err != nil {
		return fmt.Errorf("auth: marking reset token used: %w", err)
	}
	if err := s.repo.RevokeAllUserRefreshTokens(ctx, prt.UserID, now); err != nil {
		return fmt.Errorf("auth: revoking refresh tokens: %w", err)
	}
	return nil
}

// Me returns the current user by id.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// OAuthCallback exchanges an authorization code and links/creates the account.
// With no configured credentials it returns ErrOAuthNotConfigured (→ 501).
func (s *Service) OAuthCallback(ctx context.Context, providerName, code, state string, rc RequestContext) (*TokenPair, error) {
	provider, err := s.oauth.Get(Provider(providerName))
	if err != nil {
		return nil, err
	}
	if !provider.Configured() {
		return nil, ErrOAuthNotConfigured
	}
	info, err := provider.Exchange(ctx, code, state)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.FindOAuthAccount(ctx, provider.Name(), info.ProviderUserID)
	if err != nil {
		return nil, err
	}

	var u *User
	if existing != nil {
		u, err = s.repo.GetUserByID(ctx, existing.UserID)
		if err != nil {
			return nil, err
		}
	} else {
		// Link to an existing user by email, or create a new one.
		email := normalizeEmail(info.Email)
		u, err = s.repo.GetUserByEmail(ctx, email)
		if errors.Is(err, ErrUserNotFound) {
			u = &User{Email: email, Role: RoleUser, Status: StatusActive}
			if info.FullName != "" {
				u.FullName = &info.FullName
			}
			if info.AvatarURL != "" {
				u.AvatarURL = &info.AvatarURL
			}
			now := s.now()
			u.EmailVerifiedAt = &now // OAuth providers verify email
			if cerr := s.repo.CreateUser(ctx, u); cerr != nil {
				return nil, fmt.Errorf("auth: creating oauth user: %w", cerr)
			}
		} else if err != nil {
			return nil, err
		}
		acct := &OAuthAccount{
			UserID:         u.ID,
			Provider:       provider.Name(),
			ProviderUserID: info.ProviderUserID,
			RawProfile:     info.Raw,
		}
		if info.Email != "" {
			acct.Email = &info.Email
		}
		if err := s.repo.UpsertOAuthAccount(ctx, acct); err != nil {
			return nil, fmt.Errorf("auth: linking oauth account: %w", err)
		}
	}

	if u.Status != StatusActive {
		return nil, ErrAccountInactive
	}
	now := s.now()
	_ = s.repo.UpdateLastLogin(ctx, u.ID, now)
	u.LastLoginAt = &now
	return s.issueTokenPair(ctx, u, uuid.New(), rc)
}

// issueTokenPair mints access + refresh tokens, persisting the refresh hash in a
// fresh family.
func (s *Service) issueTokenPair(ctx context.Context, u *User, familyID uuid.UUID, rc RequestContext) (*TokenPair, error) {
	pair, _, err := s.issueRefreshAndAccess(ctx, u, familyID, rc)
	return pair, err
}

// issueRefreshAndAccess does the shared work of minting tokens and persisting
// the refresh record, returning the new refresh-token row id for rotation links.
func (s *Service) issueRefreshAndAccess(ctx context.Context, u *User, familyID uuid.UUID, rc RequestContext) (*TokenPair, uuid.UUID, error) {
	access, err := s.tokens.IssueAccessToken(u.ID, u.Role)
	if err != nil {
		return nil, uuid.Nil, err
	}
	rawRefresh, err := GenerateOpaqueToken()
	if err != nil {
		return nil, uuid.Nil, err
	}
	now := s.now()
	expiresAt := now.Add(s.tokens.RefreshTTL())
	rt := &RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: HashToken(rawRefresh),
		FamilyID:  familyID,
		ExpiresAt: expiresAt,
	}
	if rc.UserAgent != "" {
		rt.UserAgent = &rc.UserAgent
	}
	if rc.IPAddress != "" {
		rt.IPAddress = &rc.IPAddress
	}
	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, uuid.Nil, fmt.Errorf("auth: persisting refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      access,
		RefreshToken:     rawRefresh,
		AccessExpiresIn:  s.tokens.AccessTTL(),
		RefreshExpiresAt: expiresAt,
		User:             u,
	}, rt.ID, nil
}

// normalizeEmail lowercases and trims an email for consistent lookups (the
// column is citext, but we normalize defensively).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "SQLSTATE 23505") ||
		strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}
