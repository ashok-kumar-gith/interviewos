package behavioral

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Theme enumerates the story_theme Postgres enum values.
type Theme string

const (
	ThemeLeadership            Theme = "leadership"
	ThemeOwnership             Theme = "ownership"
	ThemeConflict              Theme = "conflict"
	ThemeFailure               Theme = "failure"
	ThemeMentorship            Theme = "mentorship"
	ThemeStakeholderManagement Theme = "stakeholder_management"
	ThemeProjectRescue         Theme = "project_rescue"
	ThemeProductionIncident    Theme = "production_incident"
	ThemeAmbiguity             Theme = "ambiguity"
	ThemeImpact                Theme = "impact"
)

// validThemes is the set of accepted story_theme values.
var validThemes = map[Theme]struct{}{
	ThemeLeadership:            {},
	ThemeOwnership:             {},
	ThemeConflict:              {},
	ThemeFailure:               {},
	ThemeMentorship:            {},
	ThemeStakeholderManagement: {},
	ThemeProjectRescue:         {},
	ThemeProductionIncident:    {},
	ThemeAmbiguity:             {},
	ThemeImpact:                {},
}

// Valid reports whether t is a known story_theme value.
func (t Theme) Valid() bool {
	_, ok := validThemes[t]
	return ok
}

// Tags is a string slice persisted as a JSONB array. It implements the
// sql.Scanner / driver.Valuer pair so GORM maps the jsonb column directly.
type Tags []string

// Value implements driver.Valuer, serializing to a JSON array (never SQL NULL).
func (t Tags) Value() (driver.Value, error) {
	if t == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(t))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner for a jsonb column.
func (t *Tags) Scan(src any) error {
	if src == nil {
		*t = Tags{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("behavioral: unsupported Scan type for Tags")
	}
	if len(data) == 0 {
		*t = Tags{}
		return nil
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*t = out
	return nil
}

// JSONMap is an arbitrary JSON object persisted as nullable JSONB (used for
// ai_feedback, the last story-improve output).
type JSONMap map[string]any

// Value implements driver.Valuer; a nil map serializes to SQL NULL.
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner for a nullable jsonb column.
func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("behavioral: unsupported Scan type for JSONMap")
	}
	if len(data) == 0 {
		*m = nil
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*m = out
	return nil
}

// Story is a STAR behavioral story (table: behavioral_stories).
type Story struct {
	ID            uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Title         string         `gorm:"column:title;not null"`
	Theme         Theme          `gorm:"column:theme;type:story_theme;not null"`
	Situation     *string        `gorm:"column:situation"`
	Task          *string        `gorm:"column:task"`
	Action        *string        `gorm:"column:action"`
	Result        *string        `gorm:"column:result"`
	Metrics       *string        `gorm:"column:metrics"`
	Tags          Tags           `gorm:"column:tags;type:jsonb;not null;default:'[]'"`
	AIImproved    bool           `gorm:"column:ai_improved;not null;default:false"`
	AIFeedback    JSONMap        `gorm:"column:ai_feedback;type:jsonb"`
	StrengthScore *float64       `gorm:"column:strength_score;type:numeric(5,2)"`
	CreatedAt     time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Story) TableName() string { return "behavioral_stories" }
