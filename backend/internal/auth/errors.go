package auth

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrEmailTaken indicates a registration conflict (409 CONFLICT).
	ErrEmailTaken = errors.New("auth: email already registered")
	// ErrInvalidCredentials indicates a bad email/password (401).
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrUserNotFound indicates the user could not be located.
	ErrUserNotFound = errors.New("auth: user not found")
	// ErrAccountInactive indicates a suspended/deleted account (401).
	ErrAccountInactive = errors.New("auth: account is not active")
	// ErrRefreshInvalid indicates a missing/expired/revoked refresh token (401).
	ErrRefreshInvalid = errors.New("auth: refresh token invalid")
	// ErrResetInvalid indicates a missing/expired/used reset token (400).
	ErrResetInvalid = errors.New("auth: reset token invalid")
	// ErrOAuthNotConfigured indicates OAuth credentials are absent (501).
	ErrOAuthNotConfigured = errors.New("auth: oauth provider not configured")
	// ErrUnsupportedProvider indicates an unknown OAuth provider (400).
	ErrUnsupportedProvider = errors.New("auth: unsupported oauth provider")
)
