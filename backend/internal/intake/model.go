package intake

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Profile is a user's intake profile (table: user_profiles). It captures the
// onboarding answers that drive the Curriculum Engine: target role/company,
// hour budget, start date, and self-assessed pillar strengths.
//
// track_id and target_company_id are stored as plain UUIDs without a hard
// foreign key (the tracks/companies tables live in other migrations); integrity
// is validated at the application layer. See migration 000003_user_profiles.
//
// PillarStrengths and IntakeAnswers are raw JSONB payloads (mirroring the
// auth.OAuthAccount.RawProfile convention) so the module carries no extra
// dependency. They are always valid JSON; the service guarantees non-nil.
type Profile struct {
	ID                    uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID                uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	TrackID               uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	YearsExperience       float64        `gorm:"column:years_experience;type:numeric(4,1);not null;default:0"`
	TargetCompanyID       *uuid.UUID     `gorm:"column:target_company_id;type:uuid"`
	TargetRole            string         `gorm:"column:target_role;type:text;not null"`
	TargetLevel           *string        `gorm:"column:target_level;type:text"`
	HoursPerWeek          int16          `gorm:"column:hours_per_week;type:smallint;not null;default:15"`
	StartDate             time.Time      `gorm:"column:start_date;type:date;not null"`
	TargetWeeks           int16          `gorm:"column:target_weeks;type:smallint;not null;default:12"`
	PillarStrengths       []byte         `gorm:"column:pillar_strengths;type:jsonb;not null;default:'{}'"`
	Timezone              string         `gorm:"column:timezone;type:text;not null;default:UTC"`
	OnboardingCompletedAt *time.Time     `gorm:"column:onboarding_completed_at"`
	IntakeAnswers         []byte         `gorm:"column:intake_answers;type:jsonb;not null;default:'{}'"`
	CreatedAt             time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Profile) TableName() string { return "user_profiles" }

// PillarType enumerates the pillar_type values self-assessed during intake.
// Used to validate pillar_strengths keys.
type PillarType string

const (
	PillarDSA                PillarType = "dsa"
	PillarSystemDesign       PillarType = "system_design"
	PillarLLD                PillarType = "lld"
	PillarBackendEngineering PillarType = "backend_engineering"
	PillarBehavioral         PillarType = "behavioral"
	PillarResume             PillarType = "resume"
)

// validPillarTypes is the set of accepted pillar_strengths keys.
var validPillarTypes = map[PillarType]struct{}{
	PillarDSA:                {},
	PillarSystemDesign:       {},
	PillarLLD:                {},
	PillarBackendEngineering: {},
	PillarBehavioral:         {},
	PillarResume:             {},
}

// IsValidPillarType reports whether s is a known pillar_type.
func IsValidPillarType(s string) bool {
	_, ok := validPillarTypes[PillarType(s)]
	return ok
}
