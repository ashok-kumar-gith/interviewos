package intake

import "errors"

// Domain errors returned by the service layer. Handlers map these to HTTP
// status codes and the standard error envelope (see handler.go).
var (
	// ErrProfileNotFound indicates the user has no (active) intake profile (404).
	ErrProfileNotFound = errors.New("intake: profile not found")
	// ErrValidation indicates the upsert payload failed domain validation (422).
	// The accompanying field details are carried by ValidationError.
	ErrValidation = errors.New("intake: validation failed")
)

// FieldViolation is a single field-level validation failure.
type FieldViolation struct {
	Field   string
	Message string
}

// ValidationError aggregates field-level violations. It satisfies error and
// wraps ErrValidation so callers can errors.Is(err, ErrValidation).
type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string { return ErrValidation.Error() }

// Unwrap lets errors.Is(err, ErrValidation) succeed.
func (e *ValidationError) Unwrap() error { return ErrValidation }
