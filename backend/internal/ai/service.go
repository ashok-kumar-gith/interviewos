package ai

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/behavioral"
	"github.com/interviewos/backend/internal/mock"
	"github.com/interviewos/backend/internal/resume"
)

// Service is the AI orchestrator (§9). For each feature it builds a focused
// system+user prompt from the user's data (read via the narrow read ports),
// calls Claude when enabled and a key is present, and otherwise — or on any
// error/timeout/empty output — produces a useful deterministic result by reusing
// the existing engines (behavioral improver, resume scorer, mock weakness
// detector). Every call records exactly one ai_invocations row.
//
// AI is an augmentation, never a dependency: the deterministic path is always
// correct and self-contained, so the API works perfectly with no API key set.
type Service struct {
	client   Client // nil ⇒ fallback-only mode
	cache    Cache  // nil ⇒ caching disabled
	repo     Repository
	enabled  bool
	model    string
	maxTok   int
	cacheTTL time.Duration
	now      func() time.Time

	profiles ProfileReader
	plans    PlanReader
	stories  StoryReader
	resumes  ResumeReader
	mocks    MockReader
	topics   AnalyticsReader
	designs  DesignReader

	// deterministic engines reused as fallbacks.
	improver behavioral.Improver
	scorer   resume.Scorer
	detector mock.WeaknessDetector
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	// Client is the LLM client. Nil means deterministic-fallback mode.
	Client Client
	// Cache is the optional Redis response cache. Nil disables caching.
	Cache Cache
	// Repo records ai_invocations rows (required).
	Repo Repository
	// Enabled gates the AI augmentation. When false the service serves the
	// deterministic fallback for every feature even if a Client is present.
	Enabled bool
	// Model is the configured model id (recorded on invocations).
	Model string
	// MaxTokens caps each completion. Defaults to 1024 when zero.
	MaxTokens int
	// CacheTTL is the response cache TTL. Defaults to 10m when zero.
	CacheTTL time.Duration
	// Now is injectable for tests. Defaults to time.Now.
	Now func() time.Time

	Profiles ProfileReader
	Plans    PlanReader
	Stories  StoryReader
	Resumes  ResumeReader
	Mocks    MockReader
	Topics   AnalyticsReader
	Designs  DesignReader

	// Improver/Scorer/Detector default to the deterministic engines when nil.
	Improver behavioral.Improver
	Scorer   resume.Scorer
	Detector mock.WeaknessDetector
}

// NewService constructs a Service, applying deterministic-engine defaults.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = 1024
	}
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	imp := cfg.Improver
	if imp == nil {
		imp = behavioral.NewDeterministicImprover()
	}
	sc := cfg.Scorer
	if sc == nil {
		sc = resume.NewRuleScorer()
	}
	det := cfg.Detector
	if det == nil {
		det = mock.NewDeterministicWeaknessDetector()
	}
	return &Service{
		client:   cfg.Client,
		cache:    cfg.Cache,
		repo:     cfg.Repo,
		enabled:  cfg.Enabled,
		model:    cfg.Model,
		maxTok:   maxTok,
		cacheTTL: ttl,
		now:      nowFn,
		profiles: cfg.Profiles,
		plans:    cfg.Plans,
		stories:  cfg.Stories,
		resumes:  cfg.Resumes,
		mocks:    cfg.Mocks,
		topics:   cfg.Topics,
		designs:  cfg.Designs,
		improver: imp,
		scorer:   sc,
		detector: det,
	}
}

// aiActive reports whether a live Claude call should be attempted.
func (s *Service) aiActive() bool { return s.enabled && s.client != nil }

// completion is the result of an LLM attempt.
type completion struct {
	text  string
	usage Usage
	ok    bool // true ⇒ a usable text response was obtained
}

// generate runs the shared orchestration pipeline for a text-producing feature:
// cache lookup -> Claude call -> parse, recording one invocation. It returns the
// completion (ok=false signals the caller should fall back) without writing the
// invocation row; the caller records it once the final used_fallback is known.
func (s *Service) tryComplete(ctx context.Context, feature Feature, system, prompt string) completion {
	if !s.aiActive() {
		return completion{}
	}
	key := cacheKey(feature, s.model, system, prompt)
	if s.cache != nil {
		if cached, hit := s.cache.Get(ctx, key); hit && strings.TrimSpace(cached) != "" {
			return completion{text: cached, ok: true}
		}
	}
	text, usage, err := s.client.Complete(ctx, system, prompt, s.maxTok)
	if err != nil || strings.TrimSpace(text) == "" {
		return completion{}
	}
	if s.cache != nil {
		s.cache.Set(ctx, key, text, s.cacheTTL)
	}
	return completion{text: text, usage: usage, ok: true}
}

// record persists exactly one ai_invocations row. It is best-effort: a persistence
// error must not fail the user-facing response (the AI value was already produced),
// so the error is swallowed after the row is attempted.
func (s *Service) record(ctx context.Context, userID uuid.UUID, feature Feature, c completion, usedFallback bool, startedAt time.Time, callErr error) uuid.UUID {
	inv := &Invocation{
		UserID:       userID,
		Feature:      feature,
		UsedFallback: usedFallback,
	}
	latency := int(s.now().Sub(startedAt).Milliseconds())
	inv.LatencyMS = &latency

	switch {
	case usedFallback:
		inv.Status = StatusFallback
		if callErr != nil {
			msg := callErr.Error()
			inv.Error = &msg
		}
	default:
		inv.Status = StatusSucceeded
		if s.model != "" {
			m := s.model
			inv.Model = &m
		}
		if c.usage.PromptTokens > 0 {
			pt := c.usage.PromptTokens
			inv.PromptTokens = &pt
		}
		if c.usage.CompletionTokens > 0 {
			ct := c.usage.CompletionTokens
			inv.CompletionTokens = &ct
		}
	}
	_ = s.repo.Record(ctx, inv)
	return inv.ID
}
