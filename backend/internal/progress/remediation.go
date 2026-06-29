package progress

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RemediationPlanner schedules a follow-up study task on a user's plan in
// response to a mock weakness. It satisfies mock.RemediationPlanner. The task is
// placed on the user's next plan day on/after today; if none exists (no active
// roadmap), it returns uuid.Nil so the caller simply records no task.
type RemediationPlanner struct {
	db  *gorm.DB
	now func() time.Time
}

// NewRemediationPlanner returns a gorm-backed RemediationPlanner.
func NewRemediationPlanner(db *gorm.DB) *RemediationPlanner {
	return &RemediationPlanner{db: db, now: time.Now}
}

// RemediationInput mirrors mock.RemediationRequest without importing the mock
// package (avoids a cross-module import cycle; the adapter in main.go maps it).
type RemediationInput struct {
	UserID     uuid.UUID
	PillarType string
	TopicID    *uuid.UUID
	Title      string
	Detail     string
}

// Schedule inserts a "revise"-kind remediation task on the user's next plan day
// and returns its id (uuid.Nil when there is no plan day to attach it to).
func (p *RemediationPlanner) Schedule(ctx context.Context, in RemediationInput) (uuid.UUID, error) {
	today := p.now().UTC().Format("2006-01-02")

	// Find the user's next plan day (today or the soonest future day) on a live
	// roadmap. user_id is denormalized on plan_days for exactly this scoping.
	var day struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	res := p.db.WithContext(ctx).
		Table("plan_days AS pd").
		Joins("JOIN roadmap_weeks rw ON rw.id = pd.roadmap_week_id AND rw.deleted_at IS NULL").
		Joins("JOIN roadmaps rm ON rm.id = rw.roadmap_id AND rm.deleted_at IS NULL AND rm.is_active").
		Where("pd.user_id = ? AND pd.deleted_at IS NULL AND pd.date >= ?", in.UserID, today).
		Order("pd.date ASC").
		Select("pd.id").
		Limit(1).
		Find(&day)
	if res.Error != nil {
		return uuid.Nil, res.Error
	}
	if res.RowsAffected == 0 || day.ID == uuid.Nil {
		return uuid.Nil, nil // no active plan to attach to
	}

	pillar := in.PillarType
	if pillar == "" {
		pillar = "dsa" // a remediation task still needs a non-null pillar_type
	}
	// item_type/item_id are polymorphic: point at the topic when known, else a
	// self-referential placeholder (revise-kind tasks need no content lookup).
	itemType := "topic"
	itemID := uuid.New()
	if in.TopicID != nil {
		itemID = *in.TopicID
	}

	now := p.now().UTC()
	taskID := uuid.New()
	err := p.db.WithContext(ctx).Exec(`
		INSERT INTO plan_tasks
		    (id, plan_day_id, user_id, kind, item_type, item_id, pillar_type,
		     title, description, estimated_minutes, priority, status, sort_order,
		     created_at, updated_at)
		VALUES
		    (?, ?, ?, 'revise', ?, ?, ?, ?, ?, 30, 'high', 'pending', 100, ?, ?)`,
		taskID, day.ID, in.UserID, itemType, itemID, pillar,
		in.Title, nullableDetail(in.Detail), now, now,
	).Error
	if err != nil {
		return uuid.Nil, err
	}
	return taskID, nil
}

func nullableDetail(s string) any {
	if s == "" {
		return nil
	}
	return s
}
