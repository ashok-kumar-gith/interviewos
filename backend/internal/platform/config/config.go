// Package config provides 12-factor environment configuration loading and
// validation for the InterviewOS backend. All config is sourced from the
// environment (with optional .env support for local development) and validated
// at startup so the process fails fast on missing/invalid values.
package config

import (
	"fmt"
	"strings"

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
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
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
