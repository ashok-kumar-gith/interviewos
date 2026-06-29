package content

import "errors"

// Domain errors returned by the content service. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested content entity does not exist.
	ErrNotFound = errors.New("content: not found")
)
