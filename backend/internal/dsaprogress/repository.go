package dsaprogress

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository abstracts persistence for DSA problem progress + solutions.
type Repository interface {
	// ProblemExists reports whether the DSA problem id is real (for 404s).
	ProblemExists(ctx context.Context, problemID uuid.UUID) (bool, error)
	// Get returns the user's progress for a problem, or (nil, nil) when none.
	Get(ctx context.Context, userID, problemID uuid.UUID) (*Progress, error)
	// Upsert records solve state + solution idempotently per (user, problem).
	Upsert(ctx context.Context, userID, problemID uuid.UUID, in Input, now time.Time) (*Progress, error)
	// List returns the user's recorded problem progress rows (solved first,
	// newest solve first) for the "what I've solved" view.
	List(ctx context.Context, userID uuid.UUID) ([]Progress, error)
	// Delete soft-deletes the user's progress on a problem.
	Delete(ctx context.Context, userID, problemID uuid.UUID) error
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

type gormRepository struct{ db *gorm.DB }

func (r *gormRepository) ProblemExists(ctx context.Context, problemID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("problems").
		Where("id = ? AND deleted_at IS NULL", problemID).
		Count(&count).Error
	return count > 0, err
}

func (r *gormRepository) Get(ctx context.Context, userID, problemID uuid.UUID) (*Progress, error) {
	var p Progress
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND problem_id = ?", userID, problemID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) List(ctx context.Context, userID uuid.UUID) ([]Progress, error) {
	var out []Progress
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("solved DESC, solved_at DESC NULLS LAST, updated_at DESC").
		Find(&out).Error
	return out, err
}

func (r *gormRepository) Delete(ctx context.Context, userID, problemID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND problem_id = ?", userID, problemID).
		Delete(&Progress{}).Error
}

func (r *gormRepository) Upsert(ctx context.Context, userID, problemID uuid.UUID, in Input, now time.Time) (*Progress, error) {
	status := "in_progress"
	if in.Solved {
		status = "completed"
	}

	row := Progress{
		UserID:           userID,
		ProblemID:        problemID,
		Status:           status,
		Confidence:       in.Confidence,
		Attempts:         1,
		Solved:           in.Solved,
		TimeSpentMinutes: in.TimeSpentMinutes,
		LastAttemptAt:    &now,
		SolutionCode:     in.SolutionCode,
		SolutionLanguage: in.SolutionLanguage,
		SolutionNotes:    in.SolutionNotes,
	}
	if in.Solved {
		row.SolvedAt = &now
	}
	if in.SolutionCode != nil || in.SolutionLanguage != nil || in.SolutionNotes != nil {
		row.SolutionUpdatedAt = &now
	}

	// Stamp solved_at the first time it flips solved; preserve solution_updated_at
	// only when a solution field is actually being written this call.
	solvedAtExpr := clause.Expr{SQL: "user_problem_progress.solved_at"}
	if in.Solved {
		solvedAtExpr = clause.Expr{SQL: "COALESCE(user_problem_progress.solved_at, ?)", Vars: []any{now}}
	}
	solUpdatedExpr := clause.Expr{SQL: "user_problem_progress.solution_updated_at"}
	if row.SolutionUpdatedAt != nil {
		solUpdatedExpr = clause.Expr{SQL: "?", Vars: []any{now}}
	}

	assignments := map[string]any{
		"status":            status,
		"confidence":        in.Confidence,
		"attempts":          gorm.Expr("user_problem_progress.attempts + 1"),
		"solved":            in.Solved,
		"time_spent_minutes": gorm.Expr("user_problem_progress.time_spent_minutes + ?", in.TimeSpentMinutes),
		"last_attempt_at":   now,
		"solved_at":         solvedAtExpr,
		"updated_at":        now,
	}
	// Only overwrite solution fields when provided, so saving solve-state alone
	// doesn't wipe a previously stored solution.
	if in.SolutionCode != nil {
		assignments["solution_code"] = in.SolutionCode
	}
	if in.SolutionLanguage != nil {
		assignments["solution_language"] = in.SolutionLanguage
	}
	if in.SolutionNotes != nil {
		assignments["solution_notes"] = in.SolutionNotes
	}
	if row.SolutionUpdatedAt != nil {
		assignments["solution_updated_at"] = solUpdatedExpr
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "user_id"}, {Name: "problem_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
		DoUpdates:   clause.Assignments(assignments),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, userID, problemID)
}
