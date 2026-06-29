package roadmap

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONStringArray is a string slice persisted as a JSONB array (focus_pillars,
// objectives). It round-trips via sql.Scanner / driver.Valuer so GORM needs no
// extra datatypes dependency (mirrors content.JSONArray).
type JSONStringArray []string

// Value serializes to a JSON array; a nil slice persists as '[]'.
func (a JSONStringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(a))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan deserializes a JSONB array.
func (a *JSONStringArray) Scan(src any) error {
	if src == nil {
		*a = JSONStringArray{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("roadmap: unsupported type for JSONStringArray scan")
	}
	if len(b) == 0 {
		*a = JSONStringArray{}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(a))
}

// Roadmap is a generated study plan (table: roadmaps).
type Roadmap struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	TrackID          uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	ProfileID        uuid.UUID      `gorm:"column:profile_id;type:uuid;not null"`
	TargetCompanyID  *uuid.UUID     `gorm:"column:target_company_id;type:uuid"`
	StartDate        time.Time      `gorm:"column:start_date;type:date;not null"`
	EndDate          time.Time      `gorm:"column:end_date;type:date;not null"`
	TotalWeeks       int16          `gorm:"column:total_weeks;type:smallint;not null;default:12"`
	HoursPerWeek     int16          `gorm:"column:hours_per_week;type:smallint;not null"`
	Status           string         `gorm:"column:status;type:text;not null;default:active"`
	IsActive         bool           `gorm:"column:is_active;not null;default:true"`
	GenerationParams []byte         `gorm:"column:generation_params;type:jsonb;not null;default:'{}'"`
	GeneratedBy      string         `gorm:"column:generated_by;type:text;not null;default:engine"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Weeks []RoadmapWeek `gorm:"-"` // loaded explicitly, not via association
}

// TableName pins the table name for GORM.
func (Roadmap) TableName() string { return "roadmaps" }

// RoadmapWeek is one week of a roadmap (table: roadmap_weeks).
type RoadmapWeek struct {
	ID           uuid.UUID       `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RoadmapID    uuid.UUID       `gorm:"column:roadmap_id;type:uuid;not null"`
	WeekNumber   int16           `gorm:"column:week_number;type:smallint;not null"`
	StartDate    time.Time       `gorm:"column:start_date;type:date;not null"`
	EndDate      time.Time       `gorm:"column:end_date;type:date;not null"`
	Theme        *string         `gorm:"column:theme;type:text"`
	FocusPillars JSONStringArray `gorm:"column:focus_pillars;type:jsonb;not null;default:'[]'"`
	PlannedHours float64         `gorm:"column:planned_hours;type:numeric(6,2);not null;default:0"`
	CreatedAt    time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt    gorm.DeletedAt  `gorm:"column:deleted_at;index"`

	Days []PlanDay `gorm:"-"`
}

// TableName pins the table name for GORM.
func (RoadmapWeek) TableName() string { return "roadmap_weeks" }

// PlanDay is a dated day within a week (table: plan_days).
type PlanDay struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	RoadmapWeekID    uuid.UUID      `gorm:"column:roadmap_week_id;type:uuid;not null"`
	UserID           uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Date             time.Time      `gorm:"column:date;type:date;not null"`
	PlannedMinutes   int            `gorm:"column:planned_minutes;not null;default:0"`
	CompletedMinutes int            `gorm:"column:completed_minutes;not null;default:0"`
	IsRestDay        bool           `gorm:"column:is_rest_day;not null;default:false"`
	Summary          *string        `gorm:"column:summary;type:text"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Tasks []PlanTask `gorm:"-"`
}

// TableName pins the table name for GORM.
func (PlanDay) TableName() string { return "plan_days" }

// PlanTask is a single unified Today-list task (table: plan_tasks).
type PlanTask struct {
	ID               uuid.UUID       `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	PlanDayID        uuid.UUID       `gorm:"column:plan_day_id;type:uuid;not null"`
	UserID           uuid.UUID       `gorm:"column:user_id;type:uuid;not null"`
	Kind             string          `gorm:"column:kind;type:task_kind;not null"`
	ItemType         string          `gorm:"column:item_type;type:plan_item_type;not null"`
	ItemID           uuid.UUID       `gorm:"column:item_id;type:uuid;not null"`
	PillarType       string          `gorm:"column:pillar_type;type:pillar_type;not null"`
	Title            string          `gorm:"column:title;type:text;not null"`
	Description      *string         `gorm:"column:description;type:text"`
	Objectives       JSONStringArray `gorm:"column:objectives;type:jsonb;not null;default:'[]'"`
	EstimatedMinutes int             `gorm:"column:estimated_minutes;not null;default:30"`
	Priority         string          `gorm:"column:priority;type:priority;not null;default:medium"`
	Difficulty       *string         `gorm:"column:difficulty;type:difficulty"`
	Status           string          `gorm:"column:status;type:task_status;not null;default:pending"`
	SortOrder        int             `gorm:"column:sort_order;not null;default:0"`
	Confidence       *int16          `gorm:"column:confidence;type:smallint"`
	TimeSpentMinutes *int            `gorm:"column:time_spent_minutes"`
	CompletionNotes  *string         `gorm:"column:completion_notes;type:text"`
	RevisionItemID   *uuid.UUID      `gorm:"column:revision_item_id;type:uuid"`
	RescheduledFrom  *uuid.UUID      `gorm:"column:rescheduled_from;type:uuid"`
	CompletedAt      *time.Time      `gorm:"column:completed_at"`
	CreatedAt        time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt  `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (PlanTask) TableName() string { return "plan_tasks" }
