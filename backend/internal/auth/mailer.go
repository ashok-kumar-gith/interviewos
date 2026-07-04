package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

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

// ResendMailer delivers email through the Resend HTTP API (https://resend.com)
// over port 443. Unlike raw SMTP, this works on hosts that block outbound SMTP
// ports (e.g. Render's free tier).
type ResendMailer struct {
	apiKey string
	from   string // e.g. "InterviewOS <onboarding@resend.dev>" or a verified-domain address
	client *http.Client
	log    *zap.Logger
}

// NewResendMailer constructs a ResendMailer. from is the sender; without a
// verified domain Resend only permits "onboarding@resend.dev" (which can email
// the account owner's own address).
func NewResendMailer(apiKey, from string, log *zap.Logger) *ResendMailer {
	if from == "" {
		from = "InterviewOS <onboarding@resend.dev>"
	}
	return &ResendMailer{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 20 * time.Second},
		log:    log,
	}
}

// SendPasswordReset posts the reset email to the Resend API.
func (m *ResendMailer) SendPasswordReset(ctx context.Context, email, _ /*resetToken*/, resetURL string) error {
	payload := map[string]any{
		"from":    m.from,
		"to":      []string{email},
		"subject": "Reset your InterviewOS password",
		"text": strings.Join([]string{
			"Hi,",
			"",
			"We received a request to reset your InterviewOS password.",
			"Open the link below to choose a new password (it expires shortly):",
			"",
			resetURL,
			"",
			"If you didn't request this, you can safely ignore this email.",
			"",
			"— InterviewOS",
		}, "\n"),
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("resend: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		m.log.Error("password reset email failed to send", zap.String("to", email), zap.Error(err))
		return fmt.Errorf("resend: send: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 300 {
		m.log.Error("password reset email rejected by Resend",
			zap.String("to", email), zap.Int("status", resp.StatusCode), zap.ByteString("body", body))
		return fmt.Errorf("resend: status %d: %s", resp.StatusCode, string(body))
	}
	m.log.Info("password reset email sent via Resend", zap.String("to", email))
	return nil
}
