package progress

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserTopicProgress is a user's mastery state for a content topic
// (table: user_topic_progress).
type UserTopicProgress struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	TopicID          uuid.UUID      `gorm:"column:topic_id;type:uuid;not null"`
	Status           string         `gorm:"column:status;type:progress_status;not null;default:not_started"`
	Confidence       *int16         `gorm:"column:confidence;type:smallint"`
	TimeSpentMinutes int            `gorm:"column:time_spent_minutes;not null;default:0"`
	TimesRevised     int            `gorm:"column:times_revised;not null;default:0"`
	LastStudiedAt    *time.Time     `gorm:"column:last_studied_at"`
	FirstCompletedAt *time.Time     `gorm:"column:first_completed_at"`
	Notes            *string        `gorm:"column:notes;type:text"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (UserTopicProgress) TableName() string { return "user_topic_progress" }

// UserProblemProgress is a user's solve state for a DSA problem
// (table: user_problem_progress).
type UserProblemProgress struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	ProblemID        uuid.UUID      `gorm:"column:problem_id;type:uuid;not null"`
	Status           string         `gorm:"column:status;type:progress_status;not null;default:not_started"`
	Confidence       *int16         `gorm:"column:confidence;type:smallint"`
	Attempts         int            `gorm:"column:attempts;not null;default:0"`
	Solved           bool           `gorm:"column:solved;not null;default:false"`
	TimeSpentMinutes int            `gorm:"column:time_spent_minutes;not null;default:0"`
	LastAttemptAt    *time.Time     `gorm:"column:last_attempt_at"`
	SolvedAt         *time.Time     `gorm:"column:solved_at"`
	Notes            *string        `gorm:"column:notes;type:text"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (UserProblemProgress) TableName() string { return "user_problem_progress" }

// StudySession is a single time-tracking record (table: study_sessions).
type StudySession struct {
	ID              uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	PlanTaskID      *uuid.UUID     `gorm:"column:plan_task_id;type:uuid"`
	PillarType      *string        `gorm:"column:pillar_type;type:pillar_type"`
	StartedAt       time.Time      `gorm:"column:started_at;not null"`
	EndedAt         *time.Time     `gorm:"column:ended_at"`
	DurationMinutes int            `gorm:"column:duration_minutes;not null;default:0"`
	Source          string         `gorm:"column:source;type:text;not null;default:timer"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (StudySession) TableName() string { return "study_sessions" }

// StreakDay is one active study day per user (table: streak_days).
type StreakDay struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Date           time.Time      `gorm:"column:date;type:date;not null"`
	TasksCompleted int            `gorm:"column:tasks_completed;not null;default:0"`
	MinutesStudied int            `gorm:"column:minutes_studied;not null;default:0"`
	GoalMet        bool           `gorm:"column:goal_met;not null;default:false"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (StreakDay) TableName() string { return "streak_days" }

// Streak summarizes a user's study streak.
type Streak struct {
	Current int
	Longest int
}

// PillarReadiness is the per-pillar readiness breakdown for the dashboard.
type PillarReadiness struct {
	Pillar         string
	Readiness      float64
	Coverage       float64
	AvgConfidence  float64
	RevisionHealth float64
}

// PillarAggregate is the raw per-pillar coverage/confidence rollup the
// dashboard service turns into readiness. It is produced by the repository from
// the user's plan_tasks (completed vs. planned est-minutes) AND from the
// problem-progress tables (solved DSA problems, completed HLD design problems),
// so progress made on the problem detail pages — not just via plan tasks —
// moves dashboard readiness for the relevant pillar.
type PillarAggregate struct {
	Pillar           string
	Weight           float64
	PlannedMinutes   int
	CompletedMinutes int
	ConfidenceSum    int
	ConfidenceCount  int
	// ItemsCompleted/ItemsTotal carry the per-pillar "items completed" signal
	// from the problem-progress tables (DSA: user_problem_progress.solved;
	// System Design: user_design_problem_progress.status='completed'). When
	// ItemsTotal > 0 this contributes a coverage term blended with the plan-task
	// minute coverage. Pillars without a dedicated progress table (LLD,
	// behavioral, resume) leave these zero and rely on plan-task coverage alone.
	ItemsCompleted int
	ItemsTotal     int
}

// TodaySummary is the dashboard's "today" block.
type TodaySummary struct {
	Date           time.Time
	TotalTasks     int
	CompletedTasks int
	EstimatedHours float64
	RemainingHours float64
}

// Dashboard is the assembled dashboard aggregate (mirrors DashboardResponse).
type Dashboard struct {
	OverallReadiness       float64
	EstimatedReadinessDate *time.Time
	PillarReadiness        []PillarReadiness
	Streak                 Streak
	Today                  TodaySummary
	RevisionDueCount       int
	GeneratedAt            time.Time
}
