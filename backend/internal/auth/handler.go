package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/interviewos/backend/internal/platform/server"
)

// refreshCookieName is the HttpOnly cookie carrying the refresh token.
const refreshCookieName = "refresh_token"

// oauthStateCookieName carries the CSRF state across the OAuth redirect round
// trip. It must be SameSite=Lax so it survives the top-level cross-site
// navigation back from the provider (Strict would be dropped).
const oauthStateCookieName = "oauth_state"

// Handler wires the auth HTTP endpoints to the service.
type Handler struct {
	svc      *Service
	tokens   *TokenManager
	validate *validator.Validate
	// secureCookies controls the Secure flag on the refresh cookie (off in dev).
	secureCookies bool
	// rateLimit is an optional per-IP rate-limit middleware applied to the
	// sensitive auth endpoints (register/login/forgot-password/reset-password).
	// When nil, no rate limiting is applied.
	rateLimit gin.HandlerFunc
	// appBaseURL is the public frontend base URL; OAuth callbacks redirect here.
	appBaseURL string
}

// HandlerConfig configures a Handler.
type HandlerConfig struct {
	Service       *Service
	Tokens        *TokenManager
	SecureCookies bool
	// RateLimit, when non-nil, is applied to the credential-sensitive endpoints
	// to throttle brute-force and abuse from a single IP.
	RateLimit gin.HandlerFunc
	// AppBaseURL is the public frontend base URL; OAuth callbacks redirect the
	// browser here after setting the session cookie (e.g. https://app.example.com).
	AppBaseURL string
}

// NewHandler constructs a Handler.
func NewHandler(cfg HandlerConfig) *Handler {
	appBaseURL := strings.TrimRight(cfg.AppBaseURL, "/")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	return &Handler{
		svc:           cfg.Service,
		tokens:        cfg.Tokens,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
		secureCookies: cfg.SecureCookies,
		rateLimit:     cfg.RateLimit,
		appBaseURL:    appBaseURL,
	}
}

// ---- Request DTOs ----

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"omitempty,max=200"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"omitempty"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ---- Response DTOs ----

type userResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	FullName      *string `json:"full_name"`
	AvatarURL     *string `json:"avatar_url"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	LastLoginAt   *string `json:"last_login_at"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type authTokensResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         userResponse `json:"user"`
}

// RegisterRoutes mounts the auth routes onto the given /api/v1 router group and
// returns the RequireAuth middleware for callers wiring protected routes.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	a := v1.Group("/auth")
	{
		// Credential-sensitive endpoints are rate-limited per IP (brute-force /
		// abuse mitigation). The limiter is optional (nil in tests / when
		// disabled), so guard each registration.
		rl := h.rateLimit
		register := []gin.HandlerFunc{h.Register}
		login := []gin.HandlerFunc{h.Login}
		forgot := []gin.HandlerFunc{h.ForgotPassword}
		reset := []gin.HandlerFunc{h.ResetPassword}
		if rl != nil {
			register = append([]gin.HandlerFunc{rl}, register...)
			login = append([]gin.HandlerFunc{rl}, login...)
			forgot = append([]gin.HandlerFunc{rl}, forgot...)
			reset = append([]gin.HandlerFunc{rl}, reset...)
		}
		a.POST("/register", register...)
		a.POST("/login", login...)
		a.POST("/refresh", h.Refresh)
		a.POST("/logout", h.Logout)
		a.POST("/forgot-password", forgot...)
		a.POST("/reset-password", reset...)
		a.GET("/oauth/:provider/start", h.OAuthStart)
		a.GET("/oauth/:provider/callback", h.OAuthCallback)
		a.GET("/me", RequireAuth(h.tokens), h.Me)
	}
	// Convenience alias: GET /api/v1/me (protected) mirrors /auth/me.
	v1.GET("/me", RequireAuth(h.tokens), h.Me)
	// Personal-data export and account deletion (NFR-DATA-003).
	v1.GET("/me/export", RequireAuth(h.tokens), h.ExportData)
	v1.DELETE("/me", RequireAuth(h.tokens), h.DeleteAccount)
}

// Register handles POST /auth/register.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if !h.bindJSON(c, &req) {
		return
	}
	pair, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, req.FullName, reqCtx(c))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, pair)
	c.JSON(http.StatusCreated, h.tokenResponse(pair))
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !h.bindJSON(c, &req) {
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password, reqCtx(c))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, pair)
	c.JSON(http.StatusOK, h.tokenResponse(pair))
}

// Refresh handles POST /auth/refresh. The token may come from the body or the
// refresh_token cookie.
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	// Body is optional for refresh; tolerate empty/no body.
	_ = c.ShouldBindJSON(&req)
	token := req.RefreshToken
	if token == "" {
		if cookie, err := c.Cookie(refreshCookieName); err == nil {
			token = cookie
		}
	}
	pair, err := h.svc.Refresh(c.Request.Context(), token, reqCtx(c))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, pair)
	c.JSON(http.StatusOK, h.tokenResponse(pair))
}

// Logout handles POST /auth/logout (204).
func (h *Handler) Logout(c *gin.Context) {
	token := ""
	var req refreshRequest
	_ = c.ShouldBindJSON(&req)
	token = req.RefreshToken
	if token == "" {
		if cookie, err := c.Cookie(refreshCookieName); err == nil {
			token = cookie
		}
	}
	if err := h.svc.Logout(c.Request.Context(), token, reqCtx(c)); err != nil {
		h.writeServiceError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

// ForgotPassword handles POST /auth/forgot-password (202, always).
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if !h.bindJSON(c, &req) {
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email, reqCtx(c)); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

// ResetPassword handles POST /auth/reset-password (204).
func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !h.bindJSON(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.Password, reqCtx(c)); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me handles GET /auth/me (and /me alias).
func (h *Handler) Me(c *gin.Context) {
	uid, ok := UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	u, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

// ExportData handles GET /me/export: returns the authenticated user's personal
// data bundle (profile + all user-owned rows) as JSON.
func (h *Handler) ExportData(c *gin.Context) {
	uid, ok := UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	bundle, err := h.svc.ExportData(c.Request.Context(), uid, reqCtx(c))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="interviewos-export.json"`)
	c.JSON(http.StatusOK, bundle)
}

// DeleteAccount handles DELETE /me: soft-deletes the authenticated user's
// account and user-owned data, revokes all refresh tokens (204).
func (h *Handler) DeleteAccount(c *gin.Context) {
	uid, ok := UserIDFromContext(c)
	if !ok {
		server.AbortError(c, http.StatusUnauthorized, server.CodeUnauthenticated, "authentication required", nil)
		return
	}
	if err := h.svc.DeleteAccount(c.Request.Context(), uid, reqCtx(c)); err != nil {
		h.writeServiceError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

// OAuthStart handles GET /auth/oauth/:provider/start. It redirects the browser
// to the provider's authorization URL when configured; when the provider has no
// credentials (the local default) it returns a clear 501 OAUTH_NOT_CONFIGURED
// envelope instead of a raw 404, so the SPA can show a friendly message.
func (h *Handler) OAuthStart(c *gin.Context) {
	provider := c.Param("provider")
	// Mint a CSRF state token, stash it in a Lax cookie, and pass it to the
	// provider so we can verify it on the callback.
	state, err := GenerateOpaqueToken()
	if err != nil {
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "could not start sign-in", nil)
		return
	}
	authURL, err := h.svc.OAuthStart(provider, state)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, state, 600, "/api/v1/auth", "", h.secureCookies, true)
	c.Redirect(http.StatusFound, authURL)
}

// OAuthCallback handles GET /auth/oauth/:provider/callback. It is reached by a
// browser redirect from the provider, so on success it sets the refresh cookie
// and redirects into the app (rather than returning JSON); on any failure it
// redirects to the login page with an ?oauth_error marker.
func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// If the provider reported an error (e.g. user denied), bounce to login.
	if e := c.Query("error"); e != "" {
		h.redirectOAuthError(c, e)
		return
	}
	if code == "" || state == "" {
		h.redirectOAuthError(c, "missing_code")
		return
	}
	// Verify CSRF state against the cookie set in OAuthStart.
	cookieState, _ := c.Cookie(oauthStateCookieName)
	h.clearOAuthStateCookie(c)
	if cookieState == "" || cookieState != state {
		h.redirectOAuthError(c, "state_mismatch")
		return
	}

	pair, err := h.svc.OAuthCallback(c.Request.Context(), provider, code, state, reqCtx(c))
	if err != nil {
		h.redirectOAuthError(c, "exchange_failed")
		return
	}
	h.setRefreshCookie(c, pair)
	// Land on the dashboard; the SPA silently refreshes from the cookie, and a
	// profile-less new account is guided to intake from there.
	c.Redirect(http.StatusFound, h.appBaseURL+"/dashboard")
}

// redirectOAuthError sends the browser back to the login page with a marker the
// SPA can surface as a friendly message.
func (h *Handler) redirectOAuthError(c *gin.Context, reason string) {
	c.Redirect(http.StatusFound, h.appBaseURL+"/login?oauth_error="+url.QueryEscape(reason))
}

// clearOAuthStateCookie expires the CSRF state cookie.
func (h *Handler) clearOAuthStateCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, "", -1, "/api/v1/auth", "", h.secureCookies, true)
}

// ---- helpers ----

// bindJSON binds and validates the request body, writing the appropriate error
// envelope on failure. Returns false if the handler should stop.
func (h *Handler) bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "invalid or malformed JSON body", nil)
		return false
	}
	if err := h.validate.Struct(dst); err != nil {
		var ve validator.ValidationErrors
		details := []server.FieldError{}
		if errors.As(err, &ve) {
			for _, fe := range ve {
				details = append(details, server.FieldError{
					Field:   fe.Field(),
					Message: validationMessage(fe),
				})
			}
		}
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed", details)
		return false
	}
	return true
}

// writeServiceError maps domain errors to HTTP status + error envelope.
func (h *Handler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		server.AbortError(c, http.StatusConflict, server.CodeConflict, "email already registered", nil)
	case errors.Is(err, ErrInvalidCredentials):
		server.AbortError(c, http.StatusUnauthorized, server.CodeInvalidCredentials, "invalid email or password", nil)
	case errors.Is(err, ErrAccountInactive):
		server.AbortError(c, http.StatusUnauthorized, server.CodeInvalidCredentials, "account is not active", nil)
	case errors.Is(err, ErrRefreshInvalid):
		server.AbortError(c, http.StatusUnauthorized, server.CodeRefreshTokenInvalid, "refresh token is invalid, expired, or revoked", nil)
	case errors.Is(err, ErrResetInvalid):
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "reset token is invalid, expired, or already used", nil)
	case errors.Is(err, ErrPasswordTooCommon):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed",
			[]server.FieldError{{Field: "Password", Message: "password is too common"}})
	case errors.Is(err, ErrPasswordTooShort):
		server.AbortError(c, http.StatusUnprocessableEntity, server.CodeValidationError, "validation failed",
			[]server.FieldError{{Field: "Password", Message: "must be at least 8 characters"}})
	case errors.Is(err, ErrDataUnavailable):
		server.AbortError(c, http.StatusServiceUnavailable, server.CodeInternal, "data service unavailable", nil)
	case errors.Is(err, ErrUserNotFound):
		server.AbortError(c, http.StatusNotFound, server.CodeNotFound, "user not found", nil)
	case errors.Is(err, ErrUnsupportedProvider):
		server.AbortError(c, http.StatusBadRequest, server.CodeBadRequest, "unsupported oauth provider", nil)
	case errors.Is(err, ErrOAuthNotConfigured):
		server.AbortError(c, http.StatusNotImplemented, "OAUTH_NOT_CONFIGURED", "oauth provider is not configured", nil)
	default:
		server.AbortError(c, http.StatusInternalServerError, server.CodeInternal, "internal server error", nil)
	}
}

func (h *Handler) tokenResponse(p *TokenPair) authTokensResponse {
	return authTokensResponse{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(p.AccessExpiresIn.Seconds()),
		User:         toUserResponse(p.User),
	}
}

func (h *Handler) setRefreshCookie(c *gin.Context, p *TokenPair) {
	maxAge := int(time.Until(p.RefreshExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, p.RefreshToken, maxAge, "/api/v1/auth", "", h.secureCookies, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, "/api/v1/auth", "", h.secureCookies, true)
}

func toUserResponse(u *User) userResponse {
	r := userResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified(),
		FullName:      u.FullName,
		AvatarURL:     u.AvatarURL,
		Role:          string(u.Role),
		Status:        string(u.Status),
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		s := u.LastLoginAt.UTC().Format(time.RFC3339)
		r.LastLoginAt = &s
	}
	return r
}

func reqCtx(c *gin.Context) RequestContext {
	return RequestContext{
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	}
}

// validationMessage renders a human-friendly message for a field error.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	default:
		return "is invalid"
	}
}
