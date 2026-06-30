package coderun

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP status
// codes and the standard error envelope (see handler.go). Note that execution
// failures (compile errors, runtime panics in the user's program) are NOT
// errors here — they are a successful run that produced stderr and a non-zero
// exit code, and are returned in the response envelope with ran=true.
var (
	// ErrUnsupportedLanguage indicates the requested language is not in the
	// allowlist (422).
	ErrUnsupportedLanguage = errors.New("coderun: unsupported language")
	// ErrSourceTooLarge indicates the submitted source exceeds the size cap (422).
	ErrSourceTooLarge = errors.New("coderun: source exceeds size limit")
	// ErrStdinTooLarge indicates the submitted stdin exceeds the size cap (422).
	ErrStdinTooLarge = errors.New("coderun: stdin exceeds size limit")
	// ErrEmptySource indicates an empty source was submitted (422).
	ErrEmptySource = errors.New("coderun: source must not be empty")
	// ErrRuntimeUnavailable indicates the executor could not resolve a runtime
	// version for the language (502 upstream).
	ErrRuntimeUnavailable = errors.New("coderun: no runtime available for language")
	// ErrExecutorUnavailable indicates the upstream executor was unreachable,
	// timed out, or returned an unexpected response (502/504 upstream). The
	// handler surfaces this as a clear, non-panicking error envelope.
	ErrExecutorUnavailable = errors.New("coderun: code executor unavailable")
)
