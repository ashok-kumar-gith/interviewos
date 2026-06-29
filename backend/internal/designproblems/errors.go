package designproblems

import "errors"

// Domain errors returned by the design-problems service. Handlers map these to
// HTTP status codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested design problem does not exist.
	ErrNotFound = errors.New("designproblems: not found")
	// ErrInvalidProgress indicates a progress payload failed validation (422).
	ErrInvalidProgress = errors.New("designproblems: invalid progress")
)
