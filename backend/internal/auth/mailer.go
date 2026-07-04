package auth

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

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

// SMTPMailer delivers email through a standard SMTP relay over STARTTLS (port
// 587). It works with any provider that accepts SMTP AUTH PLAIN — Gmail
// (smtp.gmail.com), Brevo, SendGrid, Mailgun, SES, etc.
type SMTPMailer struct {
	host string // e.g. smtp.gmail.com
	port int    // e.g. 587
	addr string // host:port
	auth smtp.Auth
	from string
	log  *zap.Logger
}

// NewSMTPMailer constructs an SMTPMailer. username/password are the SMTP AUTH
// credentials; from is the envelope/From address.
func NewSMTPMailer(host string, port int, username, password, from string, log *zap.Logger) *SMTPMailer {
	return &SMTPMailer{
		host: host,
		port: port,
		addr: fmt.Sprintf("%s:%d", host, port),
		auth: smtp.PlainAuth("", username, password, host),
		from: from,
		log:  log,
	}
}

// SendPasswordReset sends the password-reset email. smtp.SendMail upgrades the
// connection to TLS via STARTTLS when the server advertises it (as Gmail/Brevo
// do on 587), so credentials and content are encrypted in transit.
func (m *SMTPMailer) SendPasswordReset(_ context.Context, email, _ /*resetToken*/, resetURL string) error {
	subject := "Reset your InterviewOS password"
	body := strings.Join([]string{
		"Hi,",
		"",
		"We received a request to reset your InterviewOS password.",
		"Click the link below to choose a new password (the link expires shortly):",
		"",
		resetURL,
		"",
		"If you didn't request this, you can safely ignore this email — your password won't change.",
		"",
		"— InterviewOS",
	}, "\r\n")

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", m.from),
		fmt.Sprintf("To: %s", email),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if err := smtp.SendMail(m.addr, m.auth, m.from, []string{email}, []byte(msg)); err != nil {
		m.log.Error("password reset email failed to send",
			zap.String("to", email), zap.Error(err))
		return fmt.Errorf("send password reset email: %w", err)
	}
	m.log.Info("password reset email sent", zap.String("to", email))
	return nil
}
