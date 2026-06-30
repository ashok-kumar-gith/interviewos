// Package dsaprogress owns per-user progress on DSA problems: solved state,
// when it was solved, and the solution the user wrote (code + language + note).
// It is an auth-scoped companion to the public content catalog (internal/content),
// mirroring the designproblems progress module. It reads/writes the
// user_problem_progress table (000011 + solution columns from 000020).
package dsaprogress

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Domain errors mapped to HTTP status by the handler.
var (
	// ErrProblemNotFound indicates the referenced DSA problem does not exist.
	ErrProblemNotFound = errors.New("dsaprogress: problem not found")
	// ErrValidation indicates an invalid progress/solution payload (422).
	ErrValidation = errors.New("dsaprogress: validation failed")
)

const (
	maxCodeLen  = 64 * 1024
	maxNotesLen = 5000
	maxLangLen  = 40
)

// Progress is a user's progress + stored solution for a DSA problem
// (table: user_problem_progress).
type Progress struct {
	ID                uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID            uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	ProblemID         uuid.UUID      `gorm:"column:problem_id;type:uuid;not null"`
	Status            string         `gorm:"column:status;type:progress_status;not null;default:not_started"`
	Confidence        *int16         `gorm:"column:confidence"`
	Attempts          int            `gorm:"column:attempts;not null;default:0"`
	Solved            bool           `gorm:"column:solved;not null;default:false"`
	TimeSpentMinutes  int            `gorm:"column:time_spent_minutes;not null;default:0"`
	Notes             *string        `gorm:"column:notes"`
	SolvedAt          *time.Time     `gorm:"column:solved_at"`
	LastAttemptAt     *time.Time     `gorm:"column:last_attempt_at"`
	SolutionCode      *string        `gorm:"column:solution_code"`
	SolutionLanguage  *string        `gorm:"column:solution_language"`
	SolutionNotes     *string        `gorm:"column:solution_notes"`
	SolutionUpdatedAt *time.Time     `gorm:"column:solution_updated_at"`
	CreatedAt         time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table for GORM.
func (Progress) TableName() string { return "user_problem_progress" }

// Input is the validated payload to record solve state + an optional solution.
type Input struct {
	Solved           bool
	Confidence       *int16
	TimeSpentMinutes int
	SolutionCode     *string
	SolutionLanguage *string
	SolutionNotes    *string
}
