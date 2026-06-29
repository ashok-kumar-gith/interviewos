package ai

import (
	"time"

	"github.com/google/uuid"
)

// Feature enumerates the ai_feature Postgres enum values — one per /ai/* endpoint.
type Feature string

const (
	FeaturePlanner        Feature = "planner"
	FeatureCoach          Feature = "coach"
	FeatureResumeReview   Feature = "resume_review"
	FeatureStoryImprove   Feature = "story_improve"
	FeatureWeaknessDetect Feature = "weakness_detect"
	FeatureDailyPlan      Feature = "daily_plan"
	FeatureSDReview       Feature = "sd_review"
)

// Status enumerates the ai_invocation_status Postgres enum values.
type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusFallback  Status = "fallback"
)

// Invocation is one logged AI feature call (table: ai_invocations). It records
// whether the deterministic fallback was used, the model and token usage (when a
// real call was made), latency, and any error — for cost tracking and the
// graceful-fallback audit trail (§9).
type Invocation struct {
	ID               uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID `gorm:"column:user_id;type:uuid;not null"`
	Feature          Feature   `gorm:"column:feature;type:ai_feature;not null"`
	Status           Status    `gorm:"column:status;type:ai_invocation_status;not null;default:'pending'"`
	Model            *string   `gorm:"column:model"`
	PromptTokens     *int      `gorm:"column:prompt_tokens"`
	CompletionTokens *int      `gorm:"column:completion_tokens"`
	UsedFallback     bool      `gorm:"column:used_fallback;not null;default:false"`
	LatencyMS        *int      `gorm:"column:latency_ms"`
	Error            *string   `gorm:"column:error"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (Invocation) TableName() string { return "ai_invocations" }
