package analytics

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReadinessSnapshot is a user's daily readiness rollup
// (table: readiness_snapshots; SRS §6.2 / FR-ANALYTICS-008). PillarReadiness,
// WeakTopics, and StrongTopics are raw JSONB payloads (mirroring the
// intake.Profile convention) so the module carries no extra JSON dependency on
// GORM tags; the repository marshals/unmarshals them.
type ReadinessSnapshot struct {
	ID                 uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID             uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	RoadmapID          *uuid.UUID     `gorm:"column:roadmap_id;type:uuid"`
	SnapshotDate       time.Time      `gorm:"column:snapshot_date;type:date;not null"`
	OverallReadiness   float64        `gorm:"column:overall_readiness;type:numeric(5,2);not null;default:0"`
	PillarReadiness    []byte         `gorm:"column:pillar_readiness;type:jsonb;not null;default:'{}'"`
	CompletionPct      float64        `gorm:"column:completion_pct;type:numeric(5,2);not null;default:0"`
	AvgConfidence      *float64       `gorm:"column:avg_confidence;type:numeric(4,2)"`
	RevisionHealth     *float64       `gorm:"column:revision_health;type:numeric(5,2)"`
	EstimatedReadyDate *time.Time     `gorm:"column:estimated_ready_date;type:date"`
	WeakTopics         []byte         `gorm:"column:weak_topics;type:jsonb;not null;default:'[]'"`
	StrongTopics       []byte         `gorm:"column:strong_topics;type:jsonb;not null;default:'[]'"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (ReadinessSnapshot) TableName() string { return "readiness_snapshots" }

// Snapshot is the service-level view of a readiness snapshot with the JSONB
// fields decoded into native Go maps/slices.
type Snapshot struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	RoadmapID          *uuid.UUID
	SnapshotDate       time.Time
	OverallReadiness   float64
	PillarReadiness    map[string]float64
	CompletionPct      float64
	AvgConfidence      *float64
	RevisionHealth     *float64
	EstimatedReadyDate *time.Time
	WeakTopics         []uuid.UUID
	StrongTopics       []uuid.UUID
}

// Readiness is the live (recomputed) readiness view served by
// GET /analytics/readiness. It is the same shape as a snapshot but never
// persisted — it always reflects current data.
type Readiness struct {
	SnapshotDate       time.Time
	OverallReadiness   float64
	PillarReadiness    map[string]float64
	CompletionPct      float64
	AvgConfidence      *float64
	RevisionHealth     *float64
	EstimatedReadyDate *time.Time
	WeakTopics         []uuid.UUID
	StrongTopics       []uuid.UUID
	// Pillars carries the explainable per-pillar breakdown (SRS §6.2).
	Pillars []PillarReadiness
}

// Streak summarizes a user's study streak plus optional per-day activity.
type Streak struct {
	Current int
	Longest int
	Days    []StreakDay
}

// StreakDay is one active study day (mirrors streak_days, read-only here).
type StreakDay struct {
	Date           time.Time
	TasksCompleted int
	MinutesStudied int
	GoalMet        bool
}

// TopicAnalytics buckets per-topic readiness contributors into weak/strong sets.
type TopicAnalytics struct {
	Weak   []TopicEntry
	Strong []TopicEntry
}

// TopicEntry is a single per-topic analytics row (SRS FR-ANALYTICS-004).
type TopicEntry struct {
	TopicID          uuid.UUID
	TopicName        string
	PillarType       string
	Confidence       *int
	CompletionPct    float64
	RevisionAccuracy *float64
	// Score is the composite ranking key (coverage × confidence) used to bucket
	// weak vs. strong; not serialized.
	Score float64
}

// TimeSpent is the time-spent aggregation (SRS FR-ANALYTICS-005).
type TimeSpent struct {
	TotalMinutes int
	GroupBy      string
	Buckets      []TimeBucket
}

// TimeBucket is one key→minutes aggregation entry (key = date or pillar_type).
type TimeBucket struct {
	Key     string
	Minutes int
}
