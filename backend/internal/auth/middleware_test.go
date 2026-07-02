package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func testTokenManager(t *testing.T) *TokenManager {
	t.Helper()
	tm, err := NewTokenManager(TokenManagerConfig{
		Secret:    "test-secret",
		AccessTTL: time.Hour,
		Issuer:    "interviewos",
	})
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	return tm
}

// newAdminRouter mounts a single admin-gated endpoint for exercising the
// RequireAdmin middleware end-to-end.
func newAdminRouter(tm *TokenManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", RequireAdmin(tm), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRequireAdmin_Unauthenticated(t *testing.T) {
	tm := testTokenManager(t)
	r := newAdminRouter(tm)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestRequireAdmin_NonAdminForbidden(t *testing.T) {
	tm := testTokenManager(t)
	r := newAdminRouter(tm)

	tok, err := tm.IssueAccessToken(uuid.New(), RoleUser)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", w.Code)
	}
}

func TestRequireAdmin_AdminAllowed(t *testing.T) {
	tm := testTokenManager(t)
	r := newAdminRouter(tm)

	tok, err := tm.IssueAccessToken(uuid.New(), RoleAdmin)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d (body=%s)", w.Code, w.Body.String())
	}
}
