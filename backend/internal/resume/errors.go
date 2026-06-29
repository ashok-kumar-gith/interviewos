package resume

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrProfileNotFound indicates the user has no resume profile yet (404).
	ErrProfileNotFound = errors.New("resume: profile not found")
	// ErrProjectNotFound indicates the project does not exist (404).
	ErrProjectNotFound = errors.New("resume: project not found")
	// ErrForbidden indicates the caller does not own the target resource (403/404).
	ErrForbidden = errors.New("resume: not permitted")
	// ErrValidation indicates an input validation failure (422). It wraps the
	// field-level details surfaced by the handler.
	ErrValidation = errors.New("resume: validation failed")
)
