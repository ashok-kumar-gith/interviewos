package behavioral

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrStoryNotFound indicates the story does not exist or is not owned by the
	// requesting user (404). Ownership failures are surfaced as not-found so a
	// caller cannot probe for the existence of another user's stories.
	ErrStoryNotFound = errors.New("behavioral: story not found")
	// ErrValidation indicates the request failed domain validation (422).
	ErrValidation = errors.New("behavioral: validation failed")
)
