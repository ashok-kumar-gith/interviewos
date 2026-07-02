package lld

import "errors"

// Domain errors returned by the LLD service. Handlers map these to HTTP status
// codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested LLD problem does not exist.
	ErrNotFound = errors.New("lld: not found")
	// ErrValidation indicates a write payload failed validation (422).
	ErrValidation = errors.New("lld: validation failed")
	// ErrConflict indicates a unique-key conflict, e.g. a duplicate slug (409).
	ErrConflict = errors.New("lld: conflict")
)
