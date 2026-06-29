package notification

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrNotFound indicates the notification does not exist or is not owned by
	// the requesting user (404). Ownership failures are surfaced as not-found so
	// a caller cannot probe for the existence of another user's notifications.
	ErrNotFound = errors.New("notification: not found")
	// ErrValidation indicates the request failed domain validation (422).
	ErrValidation = errors.New("notification: validation failed")
	// ErrGeneratorUnavailable indicates the generate endpoint was called on a
	// Service that was constructed without a Generator (503).
	ErrGeneratorUnavailable = errors.New("notification: generator unavailable")
)
