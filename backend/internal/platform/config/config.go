// Package config provides 12-factor environment configuration loading and
// validation for the InterviewOS backend. All config is sourced from the
// environment (with optional .env support for local development) and validated
// at startup so the process fails fast on missing/invalid values.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the API process. Values are loaded
// from environment variables (12-factor) with sane defaults for local dev.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// DatabaseURL is the PostgreSQL DSN/URL (e.g. postgres://user:pass@host:5432/db).
	DatabaseURL string
	// RedisURL is the Redis connection URL (e.g. redis://localhost:6379/0).
	RedisURL string
	// JWTSecret is the signing secret for access/refresh tokens.
	JWTSecret string
	// Env is the deployment environment: development | staging | production.
	Env string
	// LogLevel is the zap log level: debug | info | warn | error.
	LogLevel string
	// CORSOrigins is the allowlist of permitted browser origins.
	CORSOrigins []string
	// AccessTokenTTL is the lifetime of issued JWT access tokens.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is the lifetime of issued refresh tokens.
	RefreshTokenTTL time.Duration
	// ResetTokenTTL is the lifetime of password-reset tokens.
	ResetTokenTTL time.Duration
	// AnthropicAPIKey is the Claude API key. Optional: when empty the AI module
	// runs in deterministic-fallback mode (no external calls are made).
	AnthropicAPIKey string
	// AIModel is the Claude model id used for AI features (default
	// claude-sonnet-4-6).
	AIModel string
	// AIEnabled gates the AI augmentation. When false (or when AnthropicAPIKey is
	// empty) every /ai/* feature serves the deterministic fallback.
	AIEnabled bool
	// MetricsEnabled gates the Prometheus metrics middleware and /metrics
	// endpoint. Default true.
	MetricsEnabled bool
	// RateLimitPerMin is the default per-IP request budget per minute applied to
	// general endpoints. Default 60.
	RateLimitPerMin int
	// AuthRateLimitPerMin is the stricter per-IP request budget per minute applied
	// to sensitive auth endpoints (login/register/forgot-password). Default 10.
	AuthRateLimitPerMin int
}

// Environment helpers.

// IsProduction reports whether the process is running in production.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// IsDevelopment reports whether the process is running in development.
func (c *Config) IsDevelopment() bool { return c.Env == "development" }

// Load reads configuration from the environment (and an optional .env file),
// applies defaults, and validates required fields. It returns an error if a
// required value is missing or invalid so callers can fail fast at startup.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults (12-factor: env overrides these).
	v.SetDefault("PORT", "8080")
	v.SetDefault("DATABASE_URL", "postgres://interviewos:interviewos@localhost:5432/interviewos?sslmode=disable")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("JWT_SECRET", "")
	v.SetDefault("ENV", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("CORS_ORIGINS", "http://localhost:3000")
	v.SetDefault("ACCESS_TOKEN_TTL", "15m")
	v.SetDefault("REFRESH_TOKEN_TTL", "720h")
	v.SetDefault("RESET_TOKEN_TTL", "1h")
	v.SetDefault("ANTHROPIC_API_KEY", "")
	v.SetDefault("AI_MODEL", "claude-sonnet-4-6")
	v.SetDefault("AI_ENABLED", true)
	v.SetDefault("METRICS_ENABLED", true)
	v.SetDefault("RATE_LIMIT_PER_MIN", 60)
	v.SetDefault("AUTH_RATE_LIMIT_PER_MIN", 10)

	// Optional .env support for local dev. Missing file is not an error.
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: reading .env: %w", err)
		}
		// .env not found is acceptable; environment variables still apply.
	}

	// Environment variables take precedence over the .env file.
	v.AutomaticEnv()

	cfg := &Config{
		Port:        v.GetString("PORT"),
		DatabaseURL: v.GetString("DATABASE_URL"),
		RedisURL:    v.GetString("REDIS_URL"),
		JWTSecret:   v.GetString("JWT_SECRET"),
		Env:         strings.ToLower(v.GetString("ENV")),
		LogLevel:    strings.ToLower(v.GetString("LOG_LEVEL")),
		CORSOrigins: splitAndTrim(v.GetString("CORS_ORIGINS")),

		AnthropicAPIKey: strings.TrimSpace(v.GetString("ANTHROPIC_API_KEY")),
		AIModel:         strings.TrimSpace(v.GetString("AI_MODEL")),
		AIEnabled:       v.GetBool("AI_ENABLED"),

		MetricsEnabled:      v.GetBool("METRICS_ENABLED"),
		RateLimitPerMin:     v.GetInt("RATE_LIMIT_PER_MIN"),
		AuthRateLimitPerMin: v.GetInt("AUTH_RATE_LIMIT_PER_MIN"),
	}
	if cfg.AIModel == "" {
		cfg.AIModel = "claude-sonnet-4-6"
	}
	// Guard against non-positive overrides; fall back to safe defaults.
	if cfg.RateLimitPerMin <= 0 {
		cfg.RateLimitPerMin = 60
	}
	if cfg.AuthRateLimitPerMin <= 0 {
		cfg.AuthRateLimitPerMin = 10
	}

	var err error
	if cfg.AccessTokenTTL, err = parseDuration(v.GetString("ACCESS_TOKEN_TTL"), "ACCESS_TOKEN_TTL"); err != nil {
		return nil, err
	}
	if cfg.RefreshTokenTTL, err = parseDuration(v.GetString("REFRESH_TOKEN_TTL"), "REFRESH_TOKEN_TTL"); err != nil {
		return nil, err
	}
	if cfg.ResetTokenTTL, err = parseDuration(v.GetString("RESET_TOKEN_TTL"), "RESET_TOKEN_TTL"); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseDuration parses a Go duration string, returning a config-scoped error.
func parseDuration(s, name string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a valid duration (e.g. 15m, 720h): %w", name, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("config: %s must be positive", name)
	}
	return d, nil
}

// validate enforces required fields and value constraints.
func (c *Config) validate() error {
	if c.Port == "" {
		return fmt.Errorf("config: PORT is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("config: REDIS_URL is required")
	}
	// In non-development environments a real secret is mandatory.
	if !c.IsDevelopment() && c.JWTSecret == "" {
		return fmt.Errorf("config: JWT_SECRET is required outside development")
	}
	switch c.Env {
	case "development", "staging", "production":
	default:
		return fmt.Errorf("config: ENV must be one of development|staging|production, got %q", c.Env)
	}
	return nil
}

// splitAndTrim splits a comma-separated string into a slice of non-empty,
// trimmed values.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
