package ai

import (
	"context"

	"gorm.io/gorm"
)

// Repository persists ai_invocations rows. It is an interface so the orchestrator
// is unit-testable against a fake recorder.
type Repository interface {
	// Record inserts one ai_invocations row and back-fills its generated id.
	Record(ctx context.Context, inv *Invocation) error
}

// gormRepository is the GORM-backed Repository.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Record(ctx context.Context, inv *Invocation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}
