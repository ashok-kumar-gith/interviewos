package designproblems

import "errors"

// Domain errors returned by the design-problems service. Handlers map these to
// HTTP status codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested design problem does not exist.
	ErrNotFound = errors.New("designproblems: not found")
	// ErrInvalidProgress indicates a progress payload failed validation (422).
	ErrInvalidProgress = errors.New("designproblems: invalid progress")
	// ErrValidation indicates a write payload failed validation (422).
	ErrValidation = errors.New("designproblems: validation failed")
	// ErrConflict indicates a unique-key conflict, e.g. a duplicate slug (409).
	ErrConflict = errors.New("designproblems: conflict")
)
