package content

import "errors"

// Domain errors returned by the content service. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the requested content entity does not exist.
	ErrNotFound = errors.New("content: not found")
	// ErrValidation indicates a write payload failed validation (422).
	ErrValidation = errors.New("content: validation failed")
	// ErrConflict indicates a unique-key conflict, e.g. a duplicate slug (409).
	ErrConflict = errors.New("content: conflict")
	// ErrUnknownReference indicates a referenced pattern/company/pillar slug or id
	// does not exist (422).
	ErrUnknownReference = errors.New("content: unknown reference")
)
