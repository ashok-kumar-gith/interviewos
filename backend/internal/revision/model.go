package revision

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecallResult enumerates the recall_result Postgres enum values (binary recall
// per ADR D1 / FR-REV-003).
type RecallResult string

const (
	// RecallCorrect advances the item along the interval ladder (or graduates it).
	RecallCorrect RecallResult = "correct"
	// RecallIncorrect resets the item to stage 0 and increments lapse_count.
	RecallIncorrect RecallResult = "incorrect"
)

// Valid reports whether r is a known recall_result value.
func (r RecallResult) Valid() bool {
	return r == RecallCorrect || r == RecallIncorrect
}

// ItemType enumerates the plan_item_type values eligible for a revision item.
// The schema's allowed subset is topic|problem|design_problem|lld_problem; the
// learning roadmap also produces subtopic study tasks, so subtopic is accepted
// as a schedulable learning type.
type ItemType string

const (
	ItemTopic         ItemType = "topic"
	ItemSubtopic      ItemType = "subtopic"
	ItemProblem       ItemType = "problem"
	ItemDesignProblem ItemType = "design_problem"
	ItemLLDProblem    ItemType = "lld_problem"
)

// schedulableTypes is the set of item types that may be scheduled for revision.
var schedulableTypes = map[ItemType]struct{}{
	ItemTopic:         {},
	ItemSubtopic:      {},
	ItemProblem:       {},
	ItemDesignProblem: {},
	ItemLLDProblem:    {},
}

// Schedulable reports whether an item of this type may have a revision item.
func (t ItemType) Schedulable() bool {
	_, ok := schedulableTypes[t]
	return ok
}

// Item is a revision_items row: the spaced-repetition state for one learned
// content item, owned by a user.
type Item struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	ItemType       ItemType       `gorm:"column:item_type;type:plan_item_type;not null"`
	ItemID         uuid.UUID      `gorm:"column:item_id;type:uuid;not null"`
	PillarType     string         `gorm:"column:pillar_type;type:pillar_type;not null"`
	IntervalDays   int            `gorm:"column:interval_days;not null;default:1"`
	Stage          int            `gorm:"column:stage;not null;default:0"`
	Ease           float64        `gorm:"column:ease;type:numeric(4,2);not null;default:2.50"`
	DueAt          time.Time      `gorm:"column:due_at;type:date;not null"`
	LastReviewedAt *time.Time     `gorm:"column:last_reviewed_at"`
	LastRecall     *RecallResult  `gorm:"column:last_recall;type:recall_result"`
	ReviewCount    int            `gorm:"column:review_count;not null;default:0"`
	LapseCount     int            `gorm:"column:lapse_count;not null;default:0"`
	IsActive       bool           `gorm:"column:is_active;not null;default:true"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Title is a non-persisted, best-effort resolved title of the underlying
	// content item (topic name, problem title, etc.). Populated by the repository
	// on reads for the API response.
	Title string `gorm:"-"`
}

// TableName pins the table name for GORM.
func (Item) TableName() string { return "revision_items" }
