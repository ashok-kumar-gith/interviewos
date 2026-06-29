package auth

import "context"

// OAuthUserInfo is the normalized identity returned by a provider after a
// successful authorization-code exchange.
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	FullName       string
	AvatarURL      string
	Raw            []byte
}

// OAuthProvider abstracts an external identity provider (Google, GitHub). A
// provider exchanges an authorization code (+ state) for normalized user info.
// Implementations that lack configured client credentials return
// ErrOAuthNotConfigured so the handler can surface a clear 501.
type OAuthProvider interface {
	// Name is the auth_provider enum value (e.g. "google").
	Name() Provider
	// Configured reports whether client credentials are present.
	Configured() bool
	// Exchange swaps the authorization code for the user's identity.
	Exchange(ctx context.Context, code, state string) (*OAuthUserInfo, error)
}

// OAuthRegistry holds the available providers keyed by name.
type OAuthRegistry struct {
	providers map[Provider]OAuthProvider
}

// NewOAuthRegistry builds a registry from the given providers.
func NewOAuthRegistry(providers ...OAuthProvider) *OAuthRegistry {
	m := make(map[Provider]OAuthProvider, len(providers))
	for _, p := range providers {
		m[p.Name()] = p
	}
	return &OAuthRegistry{providers: m}
}

// Get returns the provider for name, or (nil, ErrUnsupportedProvider).
func (r *OAuthRegistry) Get(name Provider) (OAuthProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, ErrUnsupportedProvider
	}
	return p, nil
}

// unconfiguredProvider is the default provider used when no client credentials
// are present locally. It is fully unit-testable and always reports a clear,
// structured "not configured" error from Exchange. Real Google/GitHub providers
// implement the same interface and can be registered in their place.
type unconfiguredProvider struct {
	name Provider
}

// NewUnconfiguredProvider returns a placeholder provider for name.
func NewUnconfiguredProvider(name Provider) OAuthProvider {
	return &unconfiguredProvider{name: name}
}

func (p *unconfiguredProvider) Name() Provider { return p.name }

func (p *unconfiguredProvider) Configured() bool { return false }

func (p *unconfiguredProvider) Exchange(_ context.Context, _, _ string) (*OAuthUserInfo, error) {
	return nil, ErrOAuthNotConfigured
}
