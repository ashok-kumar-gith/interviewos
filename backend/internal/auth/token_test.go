package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestTokenManager(t *testing.T, now func() time.Time) *TokenManager {
	t.Helper()
	tm, err := NewTokenManager(TokenManagerConfig{
		Secret:     "test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
		ResetTTL:   time.Hour,
		Issuer:     "interviewos-test",
		Now:        now,
	})
	require.NoError(t, err)
	return tm
}

func TestNewTokenManager_EmptySecret(t *testing.T) {
	_, err := NewTokenManager(TokenManagerConfig{Secret: ""})
	require.Error(t, err)
}

func TestIssueAndParseAccessToken(t *testing.T) {
	tm := newTestTokenManager(t, nil)
	uid := uuid.New()

	tok, err := tm.IssueAccessToken(uid, RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := tm.ParseAccessToken(tok)
	require.NoError(t, err)
	require.Equal(t, uid.String(), claims.Subject)
	require.Equal(t, RoleAdmin, claims.Role)

	parsed, err := UserIDFromClaims(claims)
	require.NoError(t, err)
	require.Equal(t, uid, parsed)
}

func TestParseAccessToken_Expired(t *testing.T) {
	base := time.Now()
	current := base
	tm := newTestTokenManager(t, func() time.Time { return current })

	tok, err := tm.IssueAccessToken(uuid.New(), RoleUser)
	require.NoError(t, err)

	// Advance past the access TTL.
	current = base.Add(16 * time.Minute)
	_, err = tm.ParseAccessToken(tok)
	require.Error(t, err)
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	tm1 := newTestTokenManager(t, nil)
	tok, err := tm1.IssueAccessToken(uuid.New(), RoleUser)
	require.NoError(t, err)

	tm2, err := NewTokenManager(TokenManagerConfig{
		Secret: "different-secret", AccessTTL: time.Minute, RefreshTTL: time.Hour, ResetTTL: time.Hour, Issuer: "interviewos-test",
	})
	require.NoError(t, err)
	_, err = tm2.ParseAccessToken(tok)
	require.Error(t, err)
}

func TestParseAccessToken_Garbage(t *testing.T) {
	tm := newTestTokenManager(t, nil)
	_, err := tm.ParseAccessToken("not.a.jwt")
	require.Error(t, err)
}

func TestGenerateOpaqueTokenAndHash(t *testing.T) {
	a, err := GenerateOpaqueToken()
	require.NoError(t, err)
	b, err := GenerateOpaqueToken()
	require.NoError(t, err)
	require.NotEqual(t, a, b, "tokens must be random/unique")

	// Hash is deterministic and 64 hex chars (SHA-256).
	require.Equal(t, HashToken(a), HashToken(a))
	require.NotEqual(t, HashToken(a), HashToken(b))
	require.Len(t, HashToken(a), 64)
	require.NotEqual(t, a, HashToken(a), "hash must differ from plaintext")
}
