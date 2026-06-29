package mock

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrMockNotFound indicates the mock interview does not exist or is not owned
	// by the requesting user (404). Ownership failures are surfaced as not-found
	// so a caller cannot probe for the existence of another user's mocks.
	ErrMockNotFound = errors.New("mock: mock interview not found")
	// ErrValidation indicates the request failed domain validation (422).
	ErrValidation = errors.New("mock: validation failed")
)
