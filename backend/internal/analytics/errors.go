package analytics

import "errors"

// Domain errors returned by the analytics/company services. Handlers map these
// to HTTP status codes and the standard error envelope (see handler.go).
var (
	// ErrProfileNotFound indicates the user has no profile yet (intake not done).
	ErrProfileNotFound = errors.New("analytics: profile not found")
	// ErrCompanyNotFound indicates the target company does not exist (422/404).
	ErrCompanyNotFound = errors.New("analytics: company not found")
	// ErrInvalidDateRange indicates a from/to query range that cannot be parsed
	// or is inverted (422).
	ErrInvalidDateRange = errors.New("analytics: invalid date range")
)
