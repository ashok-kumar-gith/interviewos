package mock

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Type enumerates the mock_type Postgres enum values.
type Type string

const (
	TypeCoding             Type = "coding"
	TypeSystemDesign       Type = "system_design"
	TypeLLD                Type = "lld"
	TypeBehavioral         Type = "behavioral"
	TypeBackendEngineering Type = "backend_engineering"
)

var validTypes = map[Type]struct{}{
	TypeCoding:             {},
	TypeSystemDesign:       {},
	TypeLLD:                {},
	TypeBehavioral:         {},
	TypeBackendEngineering: {},
}

// Valid reports whether t is a known mock_type value.
func (t Type) Valid() bool {
	_, ok := validTypes[t]
	return ok
}

// Outcome enumerates the mock_outcome Postgres enum values.
type Outcome string

const (
	OutcomeStrongHire   Outcome = "strong_hire"
	OutcomeHire         Outcome = "hire"
	OutcomeLeanHire     Outcome = "lean_hire"
	OutcomeNoHire       Outcome = "no_hire"
	OutcomeStrongNoHire Outcome = "strong_no_hire"
	OutcomeNotRated     Outcome = "not_rated"
)

var validOutcomes = map[Outcome]struct{}{
	OutcomeStrongHire:   {},
	OutcomeHire:         {},
	OutcomeLeanHire:     {},
	OutcomeNoHire:       {},
	OutcomeStrongNoHire: {},
	OutcomeNotRated:     {},
}

// Valid reports whether o is a known mock_outcome value.
func (o Outcome) Valid() bool {
	_, ok := validOutcomes[o]
	return ok
}

// Severity enumerates the finding_severity Postgres enum values, ordered from
// least to most severe. Weight returns a numeric weight used by the weakness
// aggregator to rank areas.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityMinor   Severity = "minor"
	SeverityMajor   Severity = "major"
	SeverityBlocker Severity = "blocker"
)

var validSeverities = map[Severity]struct{}{
	SeverityInfo:    {},
	SeverityMinor:   {},
	SeverityMajor:   {},
	SeverityBlocker: {},
}

// Valid reports whether s is a known finding_severity value.
func (s Severity) Valid() bool {
	_, ok := validSeverities[s]
	return ok
}

// Weight returns the relative severity weight used when ranking weaknesses.
// Higher means more severe. Unknown severities weigh 0.
func (s Severity) Weight() int {
	switch s {
	case SeverityInfo:
		return 1
	case SeverityMinor:
		return 2
	case SeverityMajor:
		return 4
	case SeverityBlocker:
		return 8
	default:
		return 0
	}
}

// Pillar enumerates the pillar_type Postgres enum values (shared with content).
type Pillar string

const (
	PillarDSA                Pillar = "dsa"
	PillarSystemDesign       Pillar = "system_design"
	PillarLLD                Pillar = "lld"
	PillarBackendEngineering Pillar = "backend_engineering"
	PillarBehavioral         Pillar = "behavioral"
	PillarResume             Pillar = "resume"
)

var validPillars = map[Pillar]struct{}{
	PillarDSA:                {},
	PillarSystemDesign:       {},
	PillarLLD:                {},
	PillarBackendEngineering: {},
	PillarBehavioral:         {},
	PillarResume:             {},
}

// Valid reports whether p is a known pillar_type value.
func (p Pillar) Valid() bool {
	_, ok := validPillars[p]
	return ok
}

// Interview is a single mock interview record (table: mock_interviews).
type Interview struct {
	ID              uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Type            Type           `gorm:"column:type;type:mock_type;not null"`
	TopicID         *uuid.UUID     `gorm:"column:topic_id;type:uuid"`
	DesignProblemID *uuid.UUID     `gorm:"column:design_problem_id;type:uuid"`
	CompanyID       *uuid.UUID     `gorm:"column:company_id;type:uuid"`
	ScheduledAt     *time.Time     `gorm:"column:scheduled_at"`
	ConductedAt     *time.Time     `gorm:"column:conducted_at"`
	DurationMinutes *int           `gorm:"column:duration_minutes"`
	Outcome         Outcome        `gorm:"column:outcome;type:mock_outcome;not null;default:'not_rated'"`
	OverallScore    *float64       `gorm:"column:overall_score;type:numeric(5,2)"`
	Interviewer     *string        `gorm:"column:interviewer"`
	TranscriptMD    *string        `gorm:"column:transcript_md"`
	Summary         *string        `gorm:"column:summary"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Findings is populated by GetByID for the detail view (preloaded).
	Findings []Finding `gorm:"foreignKey:MockInterviewID;references:ID"`
}

// TableName pins the table name for GORM.
func (Interview) TableName() string { return "mock_interviews" }

// Finding is a per-mock weakness (table: mock_findings).
type Finding struct {
	ID                uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	MockInterviewID   uuid.UUID      `gorm:"column:mock_interview_id;type:uuid;not null"`
	UserID            uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	PillarType        *Pillar        `gorm:"column:pillar_type;type:pillar_type"`
	TopicID           *uuid.UUID     `gorm:"column:topic_id;type:uuid"`
	Severity          Severity       `gorm:"column:severity;type:finding_severity;not null;default:'minor'"`
	Category          string         `gorm:"column:category;not null"`
	Detail            string         `gorm:"column:detail;not null"`
	RemediationTaskID *uuid.UUID     `gorm:"column:remediation_task_id;type:uuid"`
	CreatedAt         time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Finding) TableName() string { return "mock_findings" }
