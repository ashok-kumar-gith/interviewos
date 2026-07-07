package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// oauthHTTPClient is the shared HTTP client for provider token/userinfo calls.
func oauthHTTPClient() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

// ---- Google (OpenID Connect authorization-code flow) ----

type googleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	client       *http.Client
}

// NewGoogleProvider builds a configured Google OAuth provider. redirectURL must
// exactly match an authorized redirect URI on the Google OAuth client.
func NewGoogleProvider(clientID, clientSecret, redirectURL string) OAuthProvider {
	return &googleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		client:       oauthHTTPClient(),
	}
}

func (p *googleProvider) Name() Provider   { return ProviderGoogle }
func (p *googleProvider) Configured() bool { return p.clientID != "" && p.clientSecret != "" }

// AuthCodeURL builds the Google authorization URL for the given CSRF state.
func (p *googleProvider) AuthCodeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("access_type", "online")
	q.Set("prompt", "select_account")
	return "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
}

func (p *googleProvider) Exchange(ctx context.Context, code, _ string) (*OAuthUserInfo, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("redirect_uri", p.redirectURL)
	form.Set("grant_type", "authorization_code")

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := postForm(ctx, p.client, "https://oauth2.googleapis.com/token", form, nil, &tok); err != nil {
		return nil, fmt.Errorf("google: token exchange: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("google: empty access token")
	}

	var ui struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := getJSON(ctx, p.client, "https://openidconnect.googleapis.com/v1/userinfo",
		map[string]string{"Authorization": "Bearer " + tok.AccessToken}, &ui); err != nil {
		return nil, fmt.Errorf("google: userinfo: %w", err)
	}
	if ui.Sub == "" || ui.Email == "" {
		return nil, fmt.Errorf("google: missing subject or email")
	}
	raw, _ := json.Marshal(ui)
	return &OAuthUserInfo{
		ProviderUserID: ui.Sub,
		Email:          ui.Email,
		FullName:       ui.Name,
		AvatarURL:      ui.Picture,
		Raw:            raw,
	}, nil
}

// ---- GitHub (OAuth2 authorization-code flow) ----

type githubProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	client       *http.Client
}

// NewGitHubProvider builds a configured GitHub OAuth provider.
func NewGitHubProvider(clientID, clientSecret, redirectURL string) OAuthProvider {
	return &githubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		client:       oauthHTTPClient(),
	}
}

func (p *githubProvider) Name() Provider   { return ProviderGitHub }
func (p *githubProvider) Configured() bool { return p.clientID != "" && p.clientSecret != "" }

func (p *githubProvider) AuthCodeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

func (p *githubProvider) Exchange(ctx context.Context, code, _ string) (*OAuthUserInfo, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", p.redirectURL)

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := postForm(ctx, p.client, "https://github.com/login/oauth/access_token", form,
		map[string]string{"Accept": "application/json"}, &tok); err != nil {
		return nil, fmt.Errorf("github: token exchange: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("github: empty access token")
	}

	authH := map[string]string{
		"Authorization": "Bearer " + tok.AccessToken,
		"Accept":        "application/vnd.github+json",
	}
	var gu struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getJSON(ctx, p.client, "https://api.github.com/user", authH, &gu); err != nil {
		return nil, fmt.Errorf("github: user: %w", err)
	}
	if gu.ID == 0 {
		return nil, fmt.Errorf("github: missing user id")
	}

	email := gu.Email
	if email == "" {
		// The primary email is often private; fetch it explicitly.
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, p.client, "https://api.github.com/user/emails", authH, &emails); err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
			if email == "" {
				for _, e := range emails {
					if e.Verified {
						email = e.Email
						break
					}
				}
			}
		}
	}
	if email == "" {
		return nil, fmt.Errorf("github: no verified email available")
	}

	name := gu.Name
	if name == "" {
		name = gu.Login
	}
	raw, _ := json.Marshal(gu)
	return &OAuthUserInfo{
		ProviderUserID: strconv.FormatInt(gu.ID, 10),
		Email:          email,
		FullName:       name,
		AvatarURL:      gu.AvatarURL,
		Raw:            raw,
	}, nil
}

// ---- small HTTP helpers ----

func postForm(ctx context.Context, c *http.Client, endpoint string, form url.Values, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doJSON(c, req, out)
}

func getJSON(ctx context.Context, c *http.Client, endpoint string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doJSON(c, req, out)
}

func doJSON(c *http.Client, req *http.Request, out any) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
