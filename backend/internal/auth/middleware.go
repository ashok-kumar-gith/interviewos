package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/platform/server"
)

// Context keys for the authenticated principal.
const (
	ctxUserIDKey = "auth_user_id"
	ctxRoleKey   = "auth_user_role"
)

// RequireAuth returns Gin middleware that validates the bearer access token,
// loads the principal id/role into the context, and rejects missing/invalid
// tokens with a 401 error envelope.
func RequireAuth(tokens *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c)
		if !ok {
			server.AbortError(c, 401, server.CodeUnauthenticated, "missing or malformed authorization header", nil)
			return
		}
		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			server.AbortError(c, 401, server.CodeUnauthenticated, "invalid or expired access token", nil)
			return
		}
		uid, err := UserIDFromClaims(claims)
		if err != nil {
			server.AbortError(c, 401, server.CodeUnauthenticated, "invalid token subject", nil)
			return
		}
		c.Set(ctxUserIDKey, uid)
		c.Set(ctxRoleKey, claims.Role)
		c.Next()
	}
}

// RateLimitUserKey returns a rate-limit key function (for
// server.RateLimitWithKey) that buckets requests by authenticated user id when a
// valid bearer access token is present, and otherwise returns "" so the limiter
// falls back to the client IP. It is intentionally best-effort: an invalid /
// missing token is NOT rejected here (RequireAuth does that per route) — it just
// means the request is rate-limited by IP instead of user. This powers the
// per-authenticated-user budget (NFR-SEC-004) at the /api/v1 group level, before
// per-route RequireAuth runs.
func RateLimitUserKey(tokens *TokenManager) func(c *gin.Context) string {
	return func(c *gin.Context) string {
		raw, ok := bearerToken(c)
		if !ok {
			return ""
		}
		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			return ""
		}
		uid, err := UserIDFromClaims(claims)
		if err != nil {
			return ""
		}
		return "user:" + uid.String()
	}
}

// RequireRole returns middleware enforcing that the principal has one of the
// allowed roles. Must run after RequireAuth.
func RequireRole(roles ...Role) gin.HandlerFunc {
	allowed := make(map[Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := RoleFromContext(c)
		if !ok {
			server.AbortError(c, 401, server.CodeUnauthenticated, "authentication required", nil)
			return
		}
		if _, ok := allowed[role]; !ok {
			server.AbortError(c, 403, server.CodeForbidden, "insufficient permissions", nil)
			return
		}
		c.Next()
	}
}

// RequireAdmin returns middleware enforcing that the authenticated principal has
// the admin role. It must run AFTER RequireAuth (which loads the role from the
// validated access-token claims into the context). A missing principal yields a
// 401; a non-admin principal yields a 403 FORBIDDEN. It is a thin convenience
// wrapper over RequireRole(RoleAdmin) for the content-authoring endpoints.
func RequireAdmin(tokens *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c)
		if !ok {
			server.AbortError(c, 401, server.CodeUnauthenticated, "missing or malformed authorization header", nil)
			return
		}
		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			server.AbortError(c, 401, server.CodeUnauthenticated, "invalid or expired access token", nil)
			return
		}
		uid, err := UserIDFromClaims(claims)
		if err != nil {
			server.AbortError(c, 401, server.CodeUnauthenticated, "invalid token subject", nil)
			return
		}
		if claims.Role != RoleAdmin {
			server.AbortError(c, 403, server.CodeForbidden, "admin role required", nil)
			return
		}
		c.Set(ctxUserIDKey, uid)
		c.Set(ctxRoleKey, claims.Role)
		c.Next()
	}
}

// UserIDFromContext returns the authenticated user id set by RequireAuth.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RoleFromContext returns the authenticated user role set by RequireAuth.
func RoleFromContext(c *gin.Context) (Role, bool) {
	v, ok := c.Get(ctxRoleKey)
	if !ok {
		return "", false
	}
	r, ok := v.(Role)
	return r, ok
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(c *gin.Context) (string, bool) {
	h := c.GetHeader("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	tok := strings.TrimSpace(h[len(prefix):])
	if tok == "" {
		return "", false
	}
	return tok, true
}
