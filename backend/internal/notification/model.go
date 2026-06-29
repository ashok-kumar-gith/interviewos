package notification

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Type enumerates the notification_type Postgres enum values.
type Type string

const (
	TypeTodayPlan          Type = "today_plan"
	TypeRevisionDue        Type = "revision_due"
	TypeWeeklyReview       Type = "weekly_review"
	TypeMissedGoal         Type = "missed_goal"
	TypeStreakReminder     Type = "streak_reminder"
	TypeReadinessMilestone Type = "readiness_milestone"
	TypeMockScheduled      Type = "mock_scheduled"
	TypeSystem             Type = "system"
)

var validTypes = map[Type]struct{}{
	TypeTodayPlan:          {},
	TypeRevisionDue:        {},
	TypeWeeklyReview:       {},
	TypeMissedGoal:         {},
	TypeStreakReminder:     {},
	TypeReadinessMilestone: {},
	TypeMockScheduled:      {},
	TypeSystem:             {},
}

// Valid reports whether t is a known notification_type value.
func (t Type) Valid() bool {
	_, ok := validTypes[t]
	return ok
}

// Channel enumerates the notification_channel Postgres enum values. Only in_app
// is delivered at GA; email/push are reserved for future channels.
type Channel string

const (
	ChannelInApp Channel = "in_app"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

var validChannels = map[Channel]struct{}{
	ChannelInApp: {},
	ChannelEmail: {},
	ChannelPush:  {},
}

// Valid reports whether c is a known notification_channel value.
func (c Channel) Valid() bool {
	_, ok := validChannels[c]
	return ok
}

// Status enumerates the notification_status Postgres enum values.
type Status string

const (
	StatusUnread    Status = "unread"
	StatusRead      Status = "read"
	StatusDismissed Status = "dismissed"
)

var validStatuses = map[Status]struct{}{
	StatusUnread:    {},
	StatusRead:      {},
	StatusDismissed: {},
}

// Valid reports whether s is a known notification_status value.
func (s Status) Valid() bool {
	_, ok := validStatuses[s]
	return ok
}

// JSONMap is an arbitrary JSON object persisted as non-null JSONB (the
// notification payload / deep-link data). A nil map serializes to an empty
// object so the column never holds SQL NULL (matches the DB default '{}').
type JSONMap map[string]any

// Value implements driver.Valuer.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner for a jsonb column.
func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("notification: unsupported Scan type for JSONMap")
	}
	if len(data) == 0 {
		*m = JSONMap{}
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*m = out
	return nil
}

// Notification is a single in-app notification (table: notifications).
type Notification struct {
	ID        uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Type      Type           `gorm:"column:type;type:notification_type;not null"`
	Channel   Channel        `gorm:"column:channel;type:notification_channel;not null;default:in_app"`
	Status    Status         `gorm:"column:status;type:notification_status;not null;default:unread"`
	Title     string         `gorm:"column:title;not null"`
	Body      *string        `gorm:"column:body"`
	Payload   JSONMap        `gorm:"column:payload;type:jsonb;not null;default:'{}'"`
	// DedupKey, when set, makes the row idempotent under the partial unique index
	// uq_notif_user_dedup (user_id, dedup_key). The generator sets it to a stable
	// per-day key (e.g. "today_plan:2026-06-29"); ad-hoc notifications leave it nil.
	DedupKey  *string        `gorm:"column:dedup_key"`
	ReadAt    *time.Time     `gorm:"column:read_at"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Notification) TableName() string { return "notifications" }
