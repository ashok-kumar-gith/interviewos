package notification

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// maxLen caps free-text fields to keep payloads sane and the DB tidy.
const (
	maxTitleLen = 200
	maxBodyLen  = 2000
)

// CreateInput is the validated payload for enqueuing a notification. It is used
// by other modules (revision, streak, planner, …) via the Notifier interface to
// raise in-app notifications for a user.
type CreateInput struct {
	UserID  uuid.UUID
	Type    Type
	Channel Channel // optional; defaults to in_app when empty
	Title   string
	Body    string
	Payload map[string]any
}

// FieldError is a single field-level validation detail surfaced to the handler.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError carries field-level details and wraps ErrValidation so callers
// can branch with errors.Is.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return ErrValidation.Error() }
func (e *ValidationError) Unwrap() error { return ErrValidation }

// Notifier is the clean, write-only entrypoint other modules depend on to
// enqueue notifications. Keeping it narrow (and separate from the read/query
// API) means modules like revision or streak can take a Notifier without
// pulling in the full Service surface, and a fake is trivial to inject in tests.
type Notifier interface {
	Create(ctx context.Context, in CreateInput) (*Notification, error)
}

// Service implements the notification use-cases. It depends only on the
// Repository interface so it is unit-testable with a fake. It satisfies
// Notifier (Create) for other modules.
//
// The optional generator powers POST /notifications/generate (digest-style
// notifications). It is nil-safe: a Service built without one rejects Generate
// with ErrGeneratorUnavailable rather than panicking.
type Service struct {
	repo Repository
	gen  *Generator
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo Repository
	// Generator powers POST /notifications/generate. Optional; when nil the
	// generate endpoint returns a 503-style ErrGeneratorUnavailable.
	Generator *Generator
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	return &Service{repo: cfg.Repo, gen: cfg.Generator}
}

// compile-time assertion that Service implements Notifier.
var _ Notifier = (*Service)(nil)

// Create validates and persists a new notification for in.UserID. This is the
// create path used by other modules (via the Notifier interface) to enqueue
// notifications; it is not exposed as a user-facing HTTP endpoint.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Notification, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}
	channel := in.Channel
	if channel == "" {
		channel = ChannelInApp
	}
	n := &Notification{
		UserID:  in.UserID,
		Type:    in.Type,
		Channel: channel,
		Status:  StatusUnread,
		Title:   strings.TrimSpace(in.Title),
		Body:    strOrNil(in.Body),
		Payload: JSONMap(in.Payload),
	}
	if n.Payload == nil {
		n.Payload = JSONMap{}
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// List returns a page of the user's notifications plus the total count, applying
// pagination clamps and an optional status filter.
func (s *Service) List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]Notification, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Status != nil && !f.Status.Valid() {
		return nil, 0, &ValidationError{Fields: []FieldError{{Field: "status", Message: "must be a valid notification status"}}}
	}
	return s.repo.List(ctx, userID, f)
}

// MarkRead marks a single owned notification read (ownership enforced) and
// returns the updated row. Idempotent: an already-read notification is returned
// unchanged.
func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) (*Notification, error) {
	return s.repo.MarkRead(ctx, userID, id)
}

// MarkAllRead marks all of the user's unread notifications read and returns the
// number updated.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID)
}

// Generate runs the notification generator for userID (idempotent per day) and
// returns the current set of generated notifications. Returns
// ErrGeneratorUnavailable when no generator was wired.
func (s *Service) Generate(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	if s.gen == nil {
		return nil, ErrGeneratorUnavailable
	}
	return s.gen.Generate(ctx, userID)
}

// --- helpers ---

func validateCreate(in CreateInput) error {
	var fields []FieldError
	if in.UserID == uuid.Nil {
		fields = append(fields, FieldError{Field: "user_id", Message: "is required"})
	}
	if in.Type == "" {
		fields = append(fields, FieldError{Field: "type", Message: "is required"})
	} else if !in.Type.Valid() {
		fields = append(fields, FieldError{Field: "type", Message: "must be a valid notification type"})
	}
	if in.Channel != "" && !in.Channel.Valid() {
		fields = append(fields, FieldError{Field: "channel", Message: "must be a valid notification channel"})
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		fields = append(fields, FieldError{Field: "title", Message: "is required"})
	} else if len(title) > maxTitleLen {
		fields = append(fields, FieldError{Field: "title", Message: "must be at most 200 characters"})
	}
	if len(in.Body) > maxBodyLen {
		fields = append(fields, FieldError{Field: "body", Message: "must be at most 2000 characters"})
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func strOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
