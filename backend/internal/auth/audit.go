package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Audit action constants for security-relevant events (FR-AUDIT-003). These are
// stable strings persisted to audit_logs.action.
const (
	ActionLoginSuccess       = "auth.login.success"
	ActionLoginFailure       = "auth.login.failure"
	ActionRegister           = "auth.register"
	ActionLogout             = "auth.logout"
	ActionPasswordResetReq   = "auth.password_reset.requested"
	ActionPasswordReset      = "auth.password_reset.completed"
	ActionTokenReuseDetected = "auth.refresh.reuse_detected"
	ActionAccountDeleted     = "auth.account.deleted"
	ActionDataExported       = "auth.account.exported"
)

// AuditLog is the audit_logs row model (append-only).
type AuditLog struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    *uuid.UUID `gorm:"column:user_id;type:uuid"`
	Action    string     `gorm:"column:action;not null"`
	IPAddress *string    `gorm:"column:ip_address;type:inet"`
	UserAgent *string    `gorm:"column:user_agent"`
	Metadata  []byte     `gorm:"column:metadata;type:jsonb"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (AuditLog) TableName() string { return "audit_logs" }

// AuditEvent describes a single security event to record.
type AuditEvent struct {
	UserID    *uuid.UUID
	Action    string
	IPAddress string
	UserAgent string
	Metadata  map[string]any
}

// AuditLogger records security-relevant events. Implementations MUST be
// best-effort: a write failure is logged but never surfaced to the caller so an
// audit hiccup can never block authentication or an account action.
type AuditLogger interface {
	Record(ctx context.Context, ev AuditEvent)
}

// gormAuditLogger persists audit events to the audit_logs table.
type gormAuditLogger struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewAuditLogger returns a gorm-backed AuditLogger. A nil db or logger yields a
// no-op-safe logger (writes are skipped / logging is dropped).
func NewAuditLogger(db *gorm.DB, log *zap.Logger) AuditLogger {
	if log == nil {
		log = zap.NewNop()
	}
	return &gormAuditLogger{db: db, log: log}
}

func (a *gormAuditLogger) Record(ctx context.Context, ev AuditEvent) {
	if a.db == nil {
		return
	}
	row := AuditLog{
		UserID: ev.UserID,
		Action: ev.Action,
	}
	if ev.IPAddress != "" {
		ip := ev.IPAddress
		row.IPAddress = &ip
	}
	if ev.UserAgent != "" {
		ua := ev.UserAgent
		row.UserAgent = &ua
	}
	if len(ev.Metadata) > 0 {
		if b, err := json.Marshal(ev.Metadata); err == nil {
			row.Metadata = b
		}
	}
	// Best-effort: detach from the request context's cancellation so a client
	// disconnect cannot drop the audit write, but cap it so it never hangs.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := a.db.WithContext(writeCtx).Create(&row).Error; err != nil {
		a.log.Warn("auth: audit write failed (continuing)",
			zap.String("action", ev.Action), zap.Error(err))
	}
}

// nopAuditLogger discards all events. Used when auditing is not wired (tests /
// degraded mode) so the service never needs nil checks.
type nopAuditLogger struct{}

// NewNopAuditLogger returns an AuditLogger that discards every event.
func NewNopAuditLogger() AuditLogger { return nopAuditLogger{} }

func (nopAuditLogger) Record(context.Context, AuditEvent) {}
