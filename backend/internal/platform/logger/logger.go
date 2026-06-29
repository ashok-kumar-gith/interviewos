// Package logger provides a structured (zap) JSON logger configured from the
// application config. Logs are emitted to stdout as JSON in production and in a
// human-friendly console format in development.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a zap logger for the given environment and level. In production it
// emits JSON to stdout; in development it uses a readable console encoder. The
// level string is one of: debug, info, warn, error.
func New(env, level string) (*zap.Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var cfg zap.Config
	if env == "production" || env == "staging" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	log, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("logger: build: %w", err)
	}
	return log, nil
}

// parseLevel maps a level string to a zapcore.Level.
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("logger: unknown level %q", level)
	}
}
