package resume

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringArray is a []string persisted as a JSONB column. It implements
// driver.Valuer / sql.Scanner so GORM stores it as a JSON array (e.g. skills,
// target_keywords, metrics, tech_stack). A nil slice marshals to "[]".
type StringArray []string

// Value implements driver.Valuer, encoding the slice as a JSON array.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(a))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner, decoding a JSON array from the database.
func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = StringArray{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("resume: unsupported Scan source for StringArray")
	}
	if len(b) == 0 {
		*a = StringArray{}
		return nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*a = StringArray(out)
	return nil
}

// GormDataType pins the JSONB column type for GORM auto-migration/typing.
func (StringArray) GormDataType() string { return "jsonb" }

// Profile is a user's resume profile (table: resume_profiles). Exactly one
// active profile per user (partial unique index on user_id WHERE deleted_at IS NULL).
type Profile struct {
	ID              uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Headline        *string        `gorm:"column:headline"`
	Summary         *string        `gorm:"column:summary"`
	YearsExperience *float64       `gorm:"column:years_experience;type:numeric(4,1)"`
	Skills          StringArray    `gorm:"column:skills;type:jsonb;not null;default:'[]'"`
	TargetKeywords  StringArray    `gorm:"column:target_keywords;type:jsonb;not null;default:'[]'"`
	ATSScore        *float64       `gorm:"column:ats_score;type:numeric(5,2)"`
	LastScoredAt    *time.Time     `gorm:"column:last_scored_at"`
	AIFeedback      []byte         `gorm:"column:ai_feedback;type:jsonb"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Projects is populated by repository preloads; not a stored column.
	Projects []Project `gorm:"foreignKey:ResumeProfileID;references:ID"`
}

// TableName pins the table name for GORM.
func (Profile) TableName() string { return "resume_profiles" }

// Project is a single resume project entry (table: resume_projects).
type Project struct {
	ID              uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	ResumeProfileID uuid.UUID      `gorm:"column:resume_profile_id;type:uuid;not null"`
	UserID          uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Name            string         `gorm:"column:name;not null"`
	Role            *string        `gorm:"column:role"`
	Description     *string        `gorm:"column:description"`
	Impact          *string        `gorm:"column:impact"`
	Metrics         StringArray    `gorm:"column:metrics;type:jsonb;not null;default:'[]'"`
	TechStack       StringArray    `gorm:"column:tech_stack;type:jsonb;not null;default:'[]'"`
	StartDate       *time.Time     `gorm:"column:start_date;type:date"`
	EndDate         *time.Time     `gorm:"column:end_date;type:date"`
	SortOrder       int            `gorm:"column:sort_order;not null;default:0"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Project) TableName() string { return "resume_projects" }

// ResumeFile is a user's uploaded resume file (table: resume_files). The bytes
// are stored inline in the Content column. Exactly one active file per user
// (partial unique index on user_id WHERE deleted_at IS NULL); replacing a resume
// soft-deletes the previous row.
type ResumeFile struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	FileName    string         `gorm:"column:file_name;not null"`
	ContentType string         `gorm:"column:content_type;not null"`
	SizeBytes   int            `gorm:"column:size_bytes;not null"`
	Content     []byte         `gorm:"column:content;type:bytea;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (ResumeFile) TableName() string { return "resume_files" }
