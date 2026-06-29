package company

import "errors"

// Domain errors returned by the company service. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrProfileNotFound indicates the user has no profile yet (intake not done).
	ErrProfileNotFound = errors.New("company: profile not found")
	// ErrCompanyNotFound indicates the requested target company does not exist.
	ErrCompanyNotFound = errors.New("company: company not found")
)
