package coderun

import (
	"context"
	"strings"
	"time"
)

// execTimeout bounds a single code execution end-to-end (runtime resolution +
// execute). The service enforces it via a context deadline so a slow or hung
// upstream never blocks the request indefinitely.
const execTimeout = 10 * time.Second

// Service implements the code-run use-case. It validates input against the
// allowlist and size caps, then delegates to the Executor port. It depends only
// on the interface, so it is unit-testable against a fake executor.
type Service struct {
	exec    Executor
	timeout time.Duration
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Executor Executor
	// Timeout overrides the default per-run timeout when > 0 (used in tests).
	Timeout time.Duration
}

// NewService constructs a Service. A nil Executor defaults to the live Piston
// client.
func NewService(cfg ServiceConfig) *Service {
	exec := cfg.Executor
	if exec == nil {
		exec = NewPistonClient()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = execTimeout
	}
	return &Service{exec: exec, timeout: timeout}
}

// Run validates the request and executes it. Validation failures return a
// typed error (mapped to 422 by the handler). Upstream/transport failures
// return ErrExecutorUnavailable / ErrRuntimeUnavailable. A program that ran but
// exited non-zero is a successful Run (Ran=true) and returns a nil error.
func (s *Service) Run(ctx context.Context, language, source, stdin string) (RunOutput, error) {
	language = strings.TrimSpace(strings.ToLower(language))

	pistonLang, ok := SupportedLanguages[language]
	if !ok {
		return RunOutput{}, ErrUnsupportedLanguage
	}
	if strings.TrimSpace(source) == "" {
		return RunOutput{}, ErrEmptySource
	}
	if len(source) > MaxSourceBytes {
		return RunOutput{}, ErrSourceTooLarge
	}
	if len(stdin) > MaxStdinBytes {
		return RunOutput{}, ErrStdinTooLarge
	}

	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	out, err := s.exec.Execute(runCtx, RunInput{
		Language: pistonLang,
		Source:   source,
		Stdin:    stdin,
	})
	if err != nil {
		return RunOutput{}, err
	}
	// Always report the public language identifier back to the caller, not the
	// internal Piston runtime name (e.g. "cpp" rather than "c++").
	out.Language = language
	return out, nil
}
