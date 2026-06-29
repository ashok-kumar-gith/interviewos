package lld

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONArray is a string slice persisted as a JSONB array. It implements
// sql.Scanner / driver.Valuer so GORM can round-trip the design_patterns and
// follow_up_questions columns without a heavier datatypes dependency.
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
		return errors.New("lld: unsupported type for JSONArray scan")
	}
	if len(b) == 0 {
		*a = JSONArray{}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(a))
}

// Difficulty enumerates the difficulty Postgres enum (shared across modules).
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Problem is a canonical Low-Level Design problem (table: lld_problems).
type Problem struct {
	ID                uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	TrackID           uuid.UUID      `gorm:"column:track_id;type:uuid;not null"`
	PillarID          *uuid.UUID     `gorm:"column:pillar_id;type:uuid"`
	Slug              string         `gorm:"column:slug;not null"`
	Title             string         `gorm:"column:title;not null"`
	Difficulty        Difficulty     `gorm:"column:difficulty;type:difficulty;not null"`
	OrderIndex        int            `gorm:"column:order_index;not null;default:0"`
	RequirementsMD    *string        `gorm:"column:requirements_md"`
	EntitiesMD        *string        `gorm:"column:entities_md"`
	ClassDiagramMD    *string        `gorm:"column:class_diagram_md"`
	DesignPatterns    JSONArray      `gorm:"column:design_patterns;type:jsonb;not null;default:'[]'"`
	SolidNotesMD      *string        `gorm:"column:solid_notes_md"`
	APIOrInterfaceMD  *string        `gorm:"column:api_or_interface_md"`
	TradeoffsMD       *string        `gorm:"column:tradeoffs_md"`
	FollowUpQuestions JSONArray      `gorm:"column:follow_up_questions;type:jsonb;not null;default:'[]'"`
	CreatedAt         time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (Problem) TableName() string { return "lld_problems" }
