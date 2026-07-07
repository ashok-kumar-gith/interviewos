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
	repo       Repository
	data       DataRepository
	tokens     *TokenManager
	mailer     Mailer
	oauth      *OAuthRegistry
	audit      AuditLogger
	log        *zap.Logger
	resetURL   string // base URL for reset links, e.g. http://localhost:3000/reset-password
	bcryptCost int
	now        func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo Repository
	// Data is the cross-table reader/writer for account export and deletion.
	// When nil, export/delete return ErrDataUnavailable.
	Data   DataRepository
	Tokens *TokenManager
	Mailer Mailer
	OAuth  *OAuthRegistry
	// Audit records security-relevant events (best-effort). When nil a no-op
	// logger is used so the service never needs nil checks.
	Audit    AuditLogger
	Logger   *zap.Logger
	ResetURL string
	// BcryptCost is the work factor for new password hashes. Values below the
	// NFR-SEC floor (12) are clamped up; zero/unset defaults to the floor.
	BcryptCost int
	Now        func() time.Time
}

// minBcryptCost is the NFR-SEC floor for the password-hashing work factor.
const minBcryptCost = 12

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
	audit := cfg.Audit
	if audit == nil {
		audit = NewNopAuditLogger()
	}
	cost := cfg.BcryptCost
	if cost < minBcryptCost {
		cost = minBcryptCost
	}
	log := cfg.Logger
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:       cfg.Repo,
		data:       cfg.Data,
		tokens:     cfg.Tokens,
		mailer:     cfg.Mailer,
		oauth:      cfg.OAuth,
		audit:      audit,
		log:        log,
		resetURL:   resetURL,
		bcryptCost: cost,
		now:        nowFn,
	}
}

// Register creates a new email/password account and issues a token pair.
// Duplicate (active) email returns ErrEmailTaken.
func (s *Service) Register(ctx context.Context, email, password, fullName string, rc RequestContext) (*TokenPair, error) {
	email = normalizeEmail(email)

	if err := validatePassword(password); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
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

	s.audit.Record(ctx, AuditEvent{
		UserID:    &u.ID,
		Action:    ActionRegister,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
		Metadata:  map[string]any{"email": email},
	})

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
			s.recordLoginFailure(ctx, nil, email, "unknown_account", rc)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.PasswordHash == nil {
		s.recordLoginFailure(ctx, &u.ID, email, "no_password", rc)
		return nil, ErrInvalidCredentials
	}
	if u.Status != StatusActive {
		s.recordLoginFailure(ctx, &u.ID, email, "inactive", rc)
		return nil, ErrAccountInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		s.recordLoginFailure(ctx, &u.ID, email, "invalid_credentials", rc)
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	if err := s.repo.UpdateLastLogin(ctx, u.ID, now); err != nil {
		return nil, fmt.Errorf("auth: updating last login: %w", err)
	}
	u.LastLoginAt = &now

	s.audit.Record(ctx, AuditEvent{
		UserID:    &u.ID,
		Action:    ActionLoginSuccess,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
		Metadata:  map[string]any{"email": email},
	})

	return s.issueTokenPair(ctx, u, uuid.New(), rc)
}

// recordLoginFailure writes a best-effort login-failure audit row. userID may be
// nil when the email maps to no account (no enumeration leak — the audit row is
// internal, not returned to the caller).
func (s *Service) recordLoginFailure(ctx context.Context, userID *uuid.UUID, email, reason string, rc RequestContext) {
	s.audit.Record(ctx, AuditEvent{
		UserID:    userID,
		Action:    ActionLoginFailure,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
		Metadata:  map[string]any{"email": email, "reason": reason},
	})
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
		uid := stored.UserID
		s.audit.Record(ctx, AuditEvent{
			UserID:    &uid,
			Action:    ActionTokenReuseDetected,
			IPAddress: rc.IPAddress,
			UserAgent: rc.UserAgent,
			Metadata:  map[string]any{"family_id": stored.FamilyID.String()},
		})
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
func (s *Service) Logout(ctx context.Context, rawToken string, rc RequestContext) error {
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
	if err := s.repo.RevokeRefreshTokenFamily(ctx, stored.FamilyID, s.now()); err != nil {
		return err
	}
	uid := stored.UserID
	s.audit.Record(ctx, AuditEvent{
		UserID:    &uid,
		Action:    ActionLogout,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
	})
	return nil
}

// ForgotPassword always succeeds from the caller's perspective (no account
// enumeration). When the email maps to an active account it creates a single-use
// reset token and "sends" it via the mailer.
func (s *Service) ForgotPassword(ctx context.Context, email string, rc RequestContext) error {
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
	// The mailer may be an AsyncMailer in production (returns immediately, delivers
	// out of band) so a slow/unreachable provider never blocks the request. In
	// tests a synchronous fake mailer is injected, so the call is deterministic.
	if err := s.mailer.SendPasswordReset(ctx, u.Email, raw, resetLink); err != nil {
		// Log but do not surface to the caller (still return 2xx).
		s.log.Error("auth: sending reset email", zap.Error(err))
	}

	s.audit.Record(ctx, AuditEvent{
		UserID:    &u.ID,
		Action:    ActionPasswordResetReq,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
		Metadata:  map[string]any{"email": email},
	})
	return nil
}

// ResetPassword consumes a valid reset token, sets the new password, and revokes
// all of the user's refresh tokens (force re-login everywhere).
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string, rc RequestContext) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return ErrResetInvalid
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	prt, err := s.repo.GetResetTokenByHash(ctx, HashToken(rawToken))
	if err != nil {
		return err // ErrResetInvalid for not-found
	}
	now := s.now()
	if prt.UsedAt != nil || now.After(prt.ExpiresAt) {
		return ErrResetInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
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

	uid := prt.UserID
	s.audit.Record(ctx, AuditEvent{
		UserID:    &uid,
		Action:    ActionPasswordReset,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
	})
	return nil
}

// Me returns the current user by id.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// ExportData assembles the user's personal-data bundle (NFR-DATA-003): the user
// record plus every live row they own across the user-owned tables. The export
// itself is recorded as an audit event (best-effort).
func (s *Service) ExportData(ctx context.Context, userID uuid.UUID, rc RequestContext) (*DataExport, error) {
	if s.data == nil {
		return nil, ErrDataUnavailable
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Do not leak the password hash in the export bundle.
	u.PasswordHash = nil

	data, err := s.data.ExportUserData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: exporting user data: %w", err)
	}

	s.audit.Record(ctx, AuditEvent{
		UserID:    &userID,
		Action:    ActionDataExported,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
	})

	return &DataExport{
		ExportedAt: s.now().UTC().Format(time.RFC3339),
		User:       u,
		Data:       data,
	}, nil
}

// DeleteAccount soft-deletes the user's account and all of their user-owned rows
// (set deleted_at), revokes every refresh token (logout everywhere), and writes
// an audit row (NFR-DATA-003). The operation is idempotent: deleting an already
// soft-deleted user simply affects no further rows.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, rc RequestContext) error {
	if s.data == nil {
		return ErrDataUnavailable
	}
	// Confirm the account exists (and isn't already gone) before mutating.
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return err
	}

	now := s.now()
	rows, err := s.data.SoftDeleteUserData(ctx, userID, now)
	if err != nil {
		return err
	}
	if err := s.data.SoftDeleteUser(ctx, userID, now); err != nil {
		return fmt.Errorf("auth: soft-deleting user: %w", err)
	}
	if err := s.repo.RevokeAllUserRefreshTokens(ctx, userID, now); err != nil {
		return fmt.Errorf("auth: revoking refresh tokens: %w", err)
	}

	s.audit.Record(ctx, AuditEvent{
		UserID:    &userID,
		Action:    ActionAccountDeleted,
		IPAddress: rc.IPAddress,
		UserAgent: rc.UserAgent,
		Metadata:  map[string]any{"rows_soft_deleted": rows},
	})
	return nil
}

// OAuthStart resolves the provider and returns the URL the browser should be
// redirected to in order to begin the authorization-code flow. With no
// configured credentials it returns ErrOAuthNotConfigured (→ 501) so the
// frontend can show a clear "not available" message instead of a raw 404.
func (s *Service) OAuthStart(providerName, state string) (string, error) {
	provider, err := s.oauth.Get(Provider(providerName))
	if err != nil {
		return "", err
	}
	if !provider.Configured() {
		return "", ErrOAuthNotConfigured
	}
	// Configured Google/GitHub providers expose AuthCodeURL(state); the state is a
	// CSRF token the handler also stores in a cookie and verifies on callback.
	if au, ok := provider.(interface{ AuthCodeURL(string) string }); ok {
		return au.AuthCodeURL(state), nil
	}
	return "", ErrOAuthNotConfigured
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
