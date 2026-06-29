package lld

import "errors"

// Domain errors returned by the LLD service. Handlers map these to HTTP status
// codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested LLD problem does not exist.
	ErrNotFound = errors.New("lld: not found")
)
