package company

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Profile is the company module's projection of the user_profiles row. Company
// Mode only reads/writes the target_company_id column plus the fields returned
// in the UserProfile response; it shares the table with the intake module but
// keeps its own narrow row type so the two modules stay decoupled.
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
	CreatedAt             time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Profile) TableName() string { return "user_profiles" }
