package server

import (
	"github.com/gin-gonic/gin"
)

// Standard machine error codes (05-API-CONTRACTS.md §1.5). These are stable and
// reused across feature modules.
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthenticated     = "UNAUTHENTICATED"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeRefreshTokenInvalid = "REFRESH_TOKEN_INVALID"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeRateLimited         = "RATE_LIMITED"
	CodeInternal            = "INTERNAL"
)

// FieldError is a single field-level validation detail.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorBody is the inner object of the error envelope.
type ErrorBody struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"request_id,omitempty"`
	Details   []FieldError `json:"details,omitempty"`
}

// ErrorEnvelope is the canonical non-2xx response shape (05-API-CONTRACTS §1.4).
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// AbortError writes the standard error envelope with the request id and aborts
// the handler chain. details may be nil.
func AbortError(c *gin.Context, status int, code, message string, details []FieldError) {
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: RequestIDFromContext(c),
			Details:   details,
		},
	})
}
