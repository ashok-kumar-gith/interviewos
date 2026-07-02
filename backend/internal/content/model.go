package content

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONArray is a string slice persisted as a JSONB array. It implements
// sql.Scanner / driver.Valuer so GORM can round-trip the expected_questions and
// prerequisites columns without pulling in a heavier datatypes dependency.
type JSONArray []string

// Value serializes the slice to JSON for storage. A nil slice persists as '[]'.
func (a JSONArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(a))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan deserializes a JSONB value from the database into the slice.
func (a *JSONArray) Scan(src any) error {
	if src == nil {
		*a = JSONArray{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("content: unsupported type for JSONArray scan")
	}
	if len(b) == 0 {
		*a = JSONArray{}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(a))
}

// PillarType enumerates the pillar_type Postgres enum.
type PillarType string

const (
	PillarDSA        PillarType = "dsa"
	PillarSystem     PillarType = "system_design"
	PillarLLD        PillarType = "lld"
	PillarBackendEng PillarType = "backend_engineering"
	PillarBehavioral PillarType = "behavioral"
	PillarResume     PillarType = "resume"
)

// ResourceType enumerates the resource_type Postgres enum.
type ResourceType string

// Difficulty enumerates the difficulty Postgres enum.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Priority enumerates the priority Postgres enum.
type Priority string

// ProblemSourceName enumerates the problem_source_name Postgres enum.
type ProblemSourceName string

// ProblemPlatform enumerates the problem_platform Postgres enum.
type ProblemPlatform string

// Track is a learning track (table: tracks).
type Track struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Slug        string         `gorm:"column:slug;not null"`
	Name        string         `gorm:"column:name;not null"`
	Description *string        `gorm:"column:description"`
	Seniority   *string        `gorm:"column:seniority"`
	IsActive    bool           `gorm:"column:is_active;not null;default:true"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Track) TableName() string { return "tracks" }

// Pillar is a track pillar (table: pillars).
type Pillar struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TrackID     uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	Type        PillarType     `gorm:"column:type;type:pillar_type;not null"`
	Name        string         `gorm:"column:name;not null"`
	Description *string        `gorm:"column:description"`
	Weight      float64        `gorm:"column:weight;type:numeric(5,2);not null;default:1.0"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Pillar) TableName() string { return "pillars" }

// Topic is a learning topic within a pillar (table: topics).
type Topic struct {
	ID                uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	PillarID          uuid.UUID      `gorm:"column:pillar_id;type:uuid;not null"`
	TrackID           uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	Slug              string         `gorm:"column:slug;not null"`
	Name              string         `gorm:"column:name;not null"`
	Summary           *string        `gorm:"column:summary"`
	ConceptMD         *string        `gorm:"column:concept_md"`
	Difficulty        Difficulty     `gorm:"column:difficulty;type:difficulty;not null;default:medium"`
	Priority          Priority       `gorm:"column:priority;type:priority;not null;default:medium"`
	EstimatedHours    float64        `gorm:"column:estimated_hours;type:numeric(5,2);not null;default:2.0"`
	CommonMistakes    *string        `gorm:"column:common_mistakes"`
	ExpectedQuestions JSONArray      `gorm:"column:expected_questions;type:jsonb;not null;default:'[]'"`
	Prerequisites     JSONArray      `gorm:"column:prerequisites;type:jsonb;not null;default:'[]'"`
	SortOrder         int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt         time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Topic) TableName() string { return "topics" }

// Subtopic is a topic subdivision (table: subtopics).
type Subtopic struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TopicID        uuid.UUID      `gorm:"column:topic_id;type:uuid;not null"`
	Slug           string         `gorm:"column:slug;not null"`
	Name           string         `gorm:"column:name;not null"`
	ContentMD      *string        `gorm:"column:content_md"`
	EstimatedHours float64        `gorm:"column:estimated_hours;type:numeric(5,2);not null;default:0.5"`
	SortOrder      int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Subtopic) TableName() string { return "subtopics" }

// Resource is a global, deduplicated learning resource (table: resources).
type Resource struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Type             ResourceType   `gorm:"column:type;type:resource_type;not null"`
	Title            string         `gorm:"column:title;not null"`
	Author           *string        `gorm:"column:author"`
	URL              *string        `gorm:"column:url"`
	Provider         *string        `gorm:"column:provider"`
	Description      *string        `gorm:"column:description"`
	EstimatedMinutes *int           `gorm:"column:estimated_minutes"`
	Difficulty       *Difficulty    `gorm:"column:difficulty;type:difficulty"`
	Priority         Priority       `gorm:"column:priority;type:priority;not null;default:medium"`
	IsFree           bool           `gorm:"column:is_free;not null;default:true"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Resource) TableName() string { return "resources" }

// TopicResource links a topic to a resource (table: topic_resources).
type TopicResource struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TopicID    uuid.UUID `gorm:"column:topic_id;type:uuid;not null"`
	ResourceID uuid.UUID `gorm:"column:resource_id;type:uuid;not null"`
	Relevance  Priority  `gorm:"column:relevance;type:priority;not null;default:medium"`
	IsPrimary  bool      `gorm:"column:is_primary;not null;default:false"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (TopicResource) TableName() string { return "topic_resources" }

// Pattern is a DSA solving pattern (table: patterns).
type Pattern struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TrackID     uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	Slug        string         `gorm:"column:slug;not null"`
	Name        string         `gorm:"column:name;not null"`
	Description *string        `gorm:"column:description"`
	WhenToUse   *string        `gorm:"column:when_to_use"`
	SortOrder   int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Pattern) TableName() string { return "patterns" }

// Problem is a canonical, deduplicated DSA problem (table: problems).
type Problem struct {
	ID               uuid.UUID       `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TrackID          uuid.UUID       `gorm:"column:track_id;type:uuid;not null"`
	TopicID          *uuid.UUID      `gorm:"column:topic_id;type:uuid"`
	Slug             string          `gorm:"column:slug;not null"`
	Title            string          `gorm:"column:title;not null"`
	Difficulty       Difficulty      `gorm:"column:difficulty;type:difficulty;not null"`
	Platform         ProblemPlatform `gorm:"column:platform;type:problem_platform;not null;default:leetcode"`
	ExternalID       *string         `gorm:"column:external_id"`
	URL              *string         `gorm:"column:url"`
	PromptSummary    *string         `gorm:"column:prompt_summary"`
	ApproachMD       *string         `gorm:"column:approach_md"`
	CommonMistakes   *string         `gorm:"column:common_mistakes"`
	EstimatedMinutes int             `gorm:"column:estimated_minutes;not null;default:30"`
	FrequencyScore   float64         `gorm:"column:frequency_score;type:numeric(5,2);not null;default:0"`
	IsPremium        bool            `gorm:"column:is_premium;not null;default:false"`
	CreatedAt        time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt  `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Problem) TableName() string { return "problems" }

// ProblemPattern links a problem to a pattern (table: problem_patterns).
type ProblemPattern struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ProblemID uuid.UUID `gorm:"column:problem_id;type:uuid;not null"`
	PatternID uuid.UUID `gorm:"column:pattern_id;type:uuid;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (ProblemPattern) TableName() string { return "problem_patterns" }

// ProblemSource records a curated list a problem belongs to (table: problem_sources).
type ProblemSource struct {
	ID         uuid.UUID         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ProblemID  uuid.UUID         `gorm:"column:problem_id;type:uuid;not null"`
	Source     ProblemSourceName `gorm:"column:source;type:problem_source_name;not null"`
	SourceRank *int              `gorm:"column:source_rank"`
	SourceURL  *string           `gorm:"column:source_url"`
	CreatedAt  time.Time         `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt  time.Time         `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (ProblemSource) TableName() string { return "problem_sources" }

// Company is an interview target company (table: companies).
type Company struct {
	ID               uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Slug             string         `gorm:"column:slug;not null"`
	Name             string         `gorm:"column:name;not null"`
	LogoURL          *string        `gorm:"column:logo_url"`
	Description      *string        `gorm:"column:description"`
	InterviewStyleMD *string        `gorm:"column:interview_style_md"`
	IsFullyWeighted  bool           `gorm:"column:is_fully_weighted;not null;default:false"`
	SortOrder        int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Company) TableName() string { return "companies" }

// CompanyWeight is a per-company pillar/topic multiplier (table: company_weights).
type CompanyWeight struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	CompanyID        uuid.UUID  `gorm:"column:company_id;type:uuid;not null"`
	PillarID         *uuid.UUID `gorm:"column:pillar_id;type:uuid"`
	TopicID          *uuid.UUID `gorm:"column:topic_id;type:uuid"`
	WeightMultiplier float64    `gorm:"column:weight_multiplier;type:numeric(5,2);not null;default:1.0"`
	Note             *string    `gorm:"column:note"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (CompanyWeight) TableName() string { return "company_weights" }

// ProblemCompanyFrequency records how often a problem is asked at a company
// (table: problem_company_frequency).
type ProblemCompanyFrequency struct {
	ID             uuid.UUID `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ProblemID      uuid.UUID `gorm:"column:problem_id;type:uuid;not null"`
	CompanyID      uuid.UUID `gorm:"column:company_id;type:uuid;not null"`
	Frequency      float64   `gorm:"column:frequency;type:numeric(5,2);not null;default:0"`
	LastSeenPeriod *string   `gorm:"column:last_seen_period"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (ProblemCompanyFrequency) TableName() string { return "problem_company_frequency" }
