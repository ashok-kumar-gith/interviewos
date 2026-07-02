package content

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProblemWrite carries the resolved fields for creating/updating a DSA problem
// together with its join-table associations. Pattern slugs, source names, and
// company frequencies are re-set atomically on every write.
type ProblemWrite struct {
	Problem          Problem
	PatternSlugs     []string
	Sources          []ProblemSourceName
	CompanyFrequency []CompanyFrequencyInput
}

// CompanyFrequencyInput links a problem to a company (by id or slug) with a
// frequency and optional period. Exactly one of CompanyID/CompanySlug is used.
type CompanyFrequencyInput struct {
	CompanyID      *uuid.UUID
	CompanySlug    string
	Frequency      float64
	LastSeenPeriod *string
}

// TopicWrite carries the resolved fields for creating/updating a topic. The
// pillar/track are resolved by the service before reaching the repository.
type TopicWrite struct {
	Topic Topic
}

// WriteRepository abstracts admin write access to the content library. It is a
// separate interface from the read Repository so read-only callers are not
// forced to depend on the write surface. The GORM implementation is
// gormRepository (which satisfies both).
type WriteRepository interface {
	// DSA problems.
	CreateProblem(ctx context.Context, w ProblemWrite) (*ProblemBundle, error)
	UpdateProblem(ctx context.Context, id uuid.UUID, w ProblemWrite) (*ProblemBundle, error)
	DeleteProblem(ctx context.Context, id uuid.UUID) error

	// Topics.
	CreateTopic(ctx context.Context, w TopicWrite) (*TopicBundle, error)
	UpdateTopic(ctx context.Context, id uuid.UUID, w TopicWrite) (*TopicBundle, error)
	DeleteTopic(ctx context.Context, id uuid.UUID) error

	// Lookups used by the service to resolve human-friendly references.
	DefaultTrackID(ctx context.Context) (uuid.UUID, error)
	ResolvePillar(ctx context.Context, pillarID *uuid.UUID, pillarType *PillarType) (*Pillar, error)
	ProblemSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)
	TopicSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)
}

// NewWriteRepository returns a gorm-backed WriteRepository. It shares the same
// underlying gormRepository as NewRepository so read and write live on one type.
func NewWriteRepository(db *gorm.DB) WriteRepository {
	return &gormRepository{db: db}
}

// isUniqueViolation reports whether a database error is a Postgres unique-key
// conflict, so the service can map it to a 409.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "sqlstate 23505")
}

func (r *gormRepository) DefaultTrackID(ctx context.Context) (uuid.UUID, error) {
	var t Track
	err := r.db.WithContext(ctx).Order("sort_order ASC, created_at ASC").First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return t.ID, nil
}

func (r *gormRepository) ResolvePillar(ctx context.Context, pillarID *uuid.UUID, pillarType *PillarType) (*Pillar, error) {
	tx := r.db.WithContext(ctx).Model(&Pillar{})
	switch {
	case pillarID != nil:
		tx = tx.Where("id = ?", *pillarID)
	case pillarType != nil:
		tx = tx.Where("type = ?", *pillarType).Order("sort_order ASC")
	default:
		return nil, ErrValidation
	}
	var p Pillar
	err := tx.First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) ProblemSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	tx := r.db.WithContext(ctx).Model(&Problem{}).Where("slug = ?", slug)
	if excludeID != nil {
		tx = tx.Where("id <> ?", *excludeID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *gormRepository) TopicSlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	tx := r.db.WithContext(ctx).Model(&Topic{}).Where("slug = ?", slug)
	if excludeID != nil {
		tx = tx.Where("id <> ?", *excludeID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// CreateProblem inserts a problem and its join rows in a single transaction.
func (r *gormRepository) CreateProblem(ctx context.Context, w ProblemWrite) (*ProblemBundle, error) {
	var created Problem
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		prob := w.Problem
		prob.ID = uuid.Nil // let the DB assign
		if err := tx.Create(&prob).Error; err != nil {
			return err
		}
		created = prob
		return r.replaceProblemAssociations(tx, prob.ID, w)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return r.GetProblemBundle(ctx, created.ID)
}

// UpdateProblem updates a problem's columns and re-sets its join rows atomically.
func (r *gormRepository) UpdateProblem(ctx context.Context, id uuid.UUID, w ProblemWrite) (*ProblemBundle, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Problem
		if err := tx.Where("id = ?", id).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		// Update the mutable columns explicitly (Select) so zero-values persist.
		upd := w.Problem
		upd.ID = id
		if err := tx.Model(&Problem{}).Where("id = ?", id).Select(
			"track_id", "topic_id", "slug", "title", "difficulty", "platform",
			"external_id", "url", "prompt_summary", "approach_md", "common_mistakes",
			"estimated_minutes", "frequency_score", "is_premium", "updated_at",
		).Updates(map[string]any{
			"track_id":          upd.TrackID,
			"topic_id":          upd.TopicID,
			"slug":              upd.Slug,
			"title":             upd.Title,
			"difficulty":        upd.Difficulty,
			"platform":          upd.Platform,
			"external_id":       upd.ExternalID,
			"url":               upd.URL,
			"prompt_summary":    upd.PromptSummary,
			"approach_md":       upd.ApproachMD,
			"common_mistakes":   upd.CommonMistakes,
			"estimated_minutes": upd.EstimatedMinutes,
			"frequency_score":   upd.FrequencyScore,
			"is_premium":        upd.IsPremium,
			"updated_at":        gorm.Expr("now()"),
		}).Error; err != nil {
			return err
		}
		return r.replaceProblemAssociations(tx, id, w)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return r.GetProblemBundle(ctx, id)
}

// replaceProblemAssociations deletes and re-creates the problem_patterns,
// problem_sources, and problem_company_frequency rows for a problem. Must run
// inside a transaction.
func (r *gormRepository) replaceProblemAssociations(tx *gorm.DB, problemID uuid.UUID, w ProblemWrite) error {
	// Patterns: resolve slugs to ids.
	if err := tx.Where("problem_id = ?", problemID).Delete(&ProblemPattern{}).Error; err != nil {
		return err
	}
	for _, slug := range dedupeStrings(w.PatternSlugs) {
		var pat Pattern
		if err := tx.Where("slug = ?", slug).First(&pat).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUnknownReference
			}
			return err
		}
		if err := tx.Create(&ProblemPattern{ProblemID: problemID, PatternID: pat.ID}).Error; err != nil {
			return err
		}
	}

	// Sources.
	if err := tx.Where("problem_id = ?", problemID).Delete(&ProblemSource{}).Error; err != nil {
		return err
	}
	seenSrc := map[ProblemSourceName]struct{}{}
	for _, s := range w.Sources {
		if _, dup := seenSrc[s]; dup {
			continue
		}
		seenSrc[s] = struct{}{}
		if err := tx.Create(&ProblemSource{ProblemID: problemID, Source: s}).Error; err != nil {
			return err
		}
	}

	// Company frequencies: resolve company id/slug.
	if err := tx.Where("problem_id = ?", problemID).Delete(&ProblemCompanyFrequency{}).Error; err != nil {
		return err
	}
	seenCo := map[uuid.UUID]struct{}{}
	for _, cf := range w.CompanyFrequency {
		companyID := uuid.Nil
		if cf.CompanyID != nil {
			companyID = *cf.CompanyID
		} else if cf.CompanySlug != "" {
			var co Company
			if err := tx.Where("slug = ?", cf.CompanySlug).First(&co).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrUnknownReference
				}
				return err
			}
			companyID = co.ID
		} else {
			return ErrValidation
		}
		if _, dup := seenCo[companyID]; dup {
			continue
		}
		seenCo[companyID] = struct{}{}
		if err := tx.Create(&ProblemCompanyFrequency{
			ProblemID:      problemID,
			CompanyID:      companyID,
			Frequency:      cf.Frequency,
			LastSeenPeriod: cf.LastSeenPeriod,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteProblem soft-deletes a problem (GORM honors the DeletedAt column).
func (r *gormRepository) DeleteProblem(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Problem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateTopic inserts a topic row.
func (r *gormRepository) CreateTopic(ctx context.Context, w TopicWrite) (*TopicBundle, error) {
	t := w.Topic
	t.ID = uuid.Nil
	if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return r.GetTopicBundle(ctx, t.ID)
}

// UpdateTopic updates a topic's mutable columns.
func (r *gormRepository) UpdateTopic(ctx context.Context, id uuid.UUID, w TopicWrite) (*TopicBundle, error) {
	t := w.Topic
	res := r.db.WithContext(ctx).Model(&Topic{}).Where("id = ?", id).Updates(map[string]any{
		"pillar_id":          t.PillarID,
		"track_id":           t.TrackID,
		"slug":               t.Slug,
		"name":               t.Name,
		"summary":            t.Summary,
		"concept_md":         t.ConceptMD,
		"difficulty":         t.Difficulty,
		"priority":           t.Priority,
		"estimated_hours":    t.EstimatedHours,
		"common_mistakes":    t.CommonMistakes,
		"expected_questions": t.ExpectedQuestions,
		"prerequisites":      t.Prerequisites,
		"sort_order":         t.SortOrder,
		"updated_at":         gorm.Expr("now()"),
	})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return nil, ErrConflict
		}
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		// Either not found or no columns changed; distinguish by existence.
		var n int64
		if err := r.db.WithContext(ctx).Model(&Topic{}).Where("id = ?", id).Count(&n).Error; err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrNotFound
		}
	}
	return r.GetTopicBundle(ctx, id)
}

// DeleteTopic soft-deletes a topic.
func (r *gormRepository) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Topic{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// dedupeStrings returns the slice with duplicates and empty values removed,
// preserving first-seen order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
