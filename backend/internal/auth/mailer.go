package auth

import (
	"context"

	"go.uber.org/zap"
)

// Mailer delivers transactional emails. At GA there is no real email provider
// wired locally, so LogMailer logs the message (including the reset link) to the
// structured logger. A real SMTP/SES implementation can replace it later.
type Mailer interface {
	SendPasswordReset(ctx context.Context, email, resetToken, resetURL string) error
}

// LogMailer is a Mailer that logs the reset link instead of sending email.
type LogMailer struct {
	log *zap.Logger
}

// NewLogMailer constructs a LogMailer.
func NewLogMailer(log *zap.Logger) *LogMailer {
	return &LogMailer{log: log}
}

// SendPasswordReset logs the reset token and link at info level so it can be
// retrieved from the server logs during local development.
func (m *LogMailer) SendPasswordReset(_ context.Context, email, resetToken, resetURL string) error {
	m.log.Info("password reset requested (email delivery stubbed)",
		zap.String("to", email),
		zap.String("reset_token", resetToken),
		zap.String("reset_url", resetURL),
	)
	return nil
}
