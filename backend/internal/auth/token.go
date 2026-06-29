package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the access-token JWT payload. sub is the user id; jti uniquely
// identifies the token (for future revocation lists).
type Claims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager mints and parses HS256 access tokens and generates opaque,
// cryptographically-random refresh/reset secrets. Only SHA-256 hashes of the
// opaque secrets are ever persisted.
type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	resetTTL   time.Duration
	issuer     string
	now        func() time.Time
}

// TokenManagerConfig configures a TokenManager.
type TokenManagerConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	ResetTTL   time.Duration
	Issuer     string
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// NewTokenManager constructs a TokenManager, returning an error if the secret
// is empty.
func NewTokenManager(cfg TokenManagerConfig) (*TokenManager, error) {
	if cfg.Secret == "" {
		return nil, errors.New("auth: JWT secret must not be empty")
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = "interviewos"
	}
	return &TokenManager{
		secret:     []byte(cfg.Secret),
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		resetTTL:   cfg.ResetTTL,
		issuer:     issuer,
		now:        nowFn,
	}, nil
}

// AccessTTL exposes the configured access-token lifetime (for expires_in).
func (m *TokenManager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *TokenManager) RefreshTTL() time.Duration { return m.refreshTTL }
func (m *TokenManager) ResetTTL() time.Duration   { return m.resetTTL }

// IssueAccessToken mints a signed HS256 access token for the user.
func (m *TokenManager) IssueAccessToken(userID uuid.UUID, role Role) (string, error) {
	now := m.now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken validates the signature/expiry and returns the claims.
func (m *TokenManager) ParseAccessToken(raw string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(m.issuer),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return nil, fmt.Errorf("auth: parse access token: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("auth: invalid access token")
	}
	return claims, nil
}

// UserIDFromClaims extracts the subject as a uuid.
func UserIDFromClaims(c *Claims) (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

// GenerateOpaqueToken returns a URL-safe, 256-bit random secret. The plaintext
// is given to the client; only HashToken(secret) is persisted.
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: generating random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 of an opaque token. Deterministic,
// so the hash can be looked up by unique index.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
