package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// userOwnedTables lists the user-data tables that carry a direct user_id column
// and a deleted_at column. Account deletion soft-deletes every live row a user
// owns across these tables, and data export reads them for the GDPR-style
// bundle. The order is irrelevant (all soft-delete by user_id) but is grouped
// roughly by module for readability.
//
// Tables related to the user only transitively (e.g. roadmap_weeks via
// roadmap_id) are intentionally omitted here: deleting the parent roadmap rows
// already removes them from the user's active view, and they carry no PII.
var userOwnedTables = []string{
	"user_profiles",
	"roadmaps",
	"plan_days",
	"plan_tasks",
	"behavioral_stories",
	"resume_profiles",
	"resume_projects",
	"mock_interviews",
	"mock_findings",
	"notifications",
	"revision_items",
	"user_topic_progress",
	"user_problem_progress",
	"study_sessions",
	"streak_days",
	"readiness_snapshots",
}

// DataExport is the JSON bundle returned by GET /me/export. Profile is the user
// record; Data maps each user-owned table name to its live rows for that user.
type DataExport struct {
	ExportedAt string                      `json:"exported_at"`
	User       *User                       `json:"user"`
	Data       map[string][]map[string]any `json:"data"`
}

// DataRepository abstracts the cross-table reads/writes used by account export
// and deletion. It is deliberately separate from the auth Repository so the
// core auth flows stay focused; the gorm implementation reads/writes the
// user-owned tables generically by name.
type DataRepository interface {
	// ExportUserData returns every live row the user owns, keyed by table name.
	ExportUserData(ctx context.Context, userID uuid.UUID) (map[string][]map[string]any, error)
	// SoftDeleteUserData sets deleted_at on every live row the user owns across
	// the user-owned tables (excluding the users row itself). Returns the total
	// number of rows affected.
	SoftDeleteUserData(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error)
	// SoftDeleteUser sets the users row's deleted_at and status='deleted'.
	SoftDeleteUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

type gormDataRepository struct {
	db *gorm.DB
}

// NewDataRepository returns a gorm-backed DataRepository.
func NewDataRepository(db *gorm.DB) DataRepository {
	return &gormDataRepository{db: db}
}

func (r *gormDataRepository) ExportUserData(ctx context.Context, userID uuid.UUID) (map[string][]map[string]any, error) {
	out := make(map[string][]map[string]any, len(userOwnedTables))
	for _, table := range userOwnedTables {
		var rows []map[string]any
		// Table names come from a fixed, code-owned allowlist (never user input),
		// so interpolating the identifier is safe; the user id is parameterized.
		q := fmt.Sprintf("SELECT * FROM %s WHERE user_id = ? AND deleted_at IS NULL", table)
		if err := r.db.WithContext(ctx).Raw(q, userID).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("auth: exporting %s: %w", table, err)
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out[table] = rows
	}
	return out, nil
}

func (r *gormDataRepository) SoftDeleteUserData(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, table := range userOwnedTables {
			q := fmt.Sprintf("UPDATE %s SET deleted_at = ? WHERE user_id = ? AND deleted_at IS NULL", table)
			res := tx.Exec(q, at, userID)
			if res.Error != nil {
				return fmt.Errorf("auth: soft-deleting %s: %w", table, res.Error)
			}
			total += res.RowsAffected
		}
		return nil
	})
	return total, err
}

func (r *gormDataRepository) SoftDeleteUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE users SET deleted_at = ?, status = 'deleted' WHERE id = ? AND deleted_at IS NULL", at, userID).
		Error
}
