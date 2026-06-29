// Package seed loads the canonical InterviewOS content/curriculum library into
// the database idempotently. Every entity is upserted by its natural key
// (slug / url / composite) so the seeder is safe to re-run: a second run makes
// no net change to row counts. Dedup logic for the merged DSA problem set lives
// here, not in the read-time service.
package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// Counts summarizes how many rows of each kind exist after a seed run.
type Counts struct {
	Tracks    int64
	Pillars   int64
	Patterns  int64
	Topics    int64
	Subtopics int64
	Resources int64
	Problems  int64
	Companies int64
}

// Seeder loads content into the database.
type Seeder struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewSeeder constructs a Seeder.
func NewSeeder(db *gorm.DB, log *zap.Logger) *Seeder {
	return &Seeder{db: db, log: log}
}

// Run executes the full seed within a single transaction. It is idempotent.
func (s *Seeder) Run(ctx context.Context) (Counts, error) {
	var counts Counts
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		track, err := s.seedTrack(tx)
		if err != nil {
			return fmt.Errorf("seed track: %w", err)
		}
		pillars, err := s.seedPillars(tx, track.ID)
		if err != nil {
			return fmt.Errorf("seed pillars: %w", err)
		}
		patterns, err := s.seedPatterns(tx, track.ID)
		if err != nil {
			return fmt.Errorf("seed patterns: %w", err)
		}
		dsaTopics, err := s.seedDSATopics(tx, track.ID, pillars[content.PillarDSA])
		if err != nil {
			return fmt.Errorf("seed dsa topics: %w", err)
		}
		sdTopics, err := s.seedSystemDesignTopics(tx, track.ID, pillars[content.PillarSystem])
		if err != nil {
			return fmt.Errorf("seed system design topics: %w", err)
		}
		beTopics, err := s.seedBackendEngineeringTopics(tx, track.ID, pillars[content.PillarBackendEng])
		if err != nil {
			return fmt.Errorf("seed backend engineering topics: %w", err)
		}
		resourcesBySlug, err := s.seedResources(tx)
		if err != nil {
			return fmt.Errorf("seed resources: %w", err)
		}
		if err := s.seedTopicResources(tx, dsaTopics, sdTopics, beTopics, resourcesBySlug); err != nil {
			return fmt.Errorf("seed topic resources: %w", err)
		}
		if err := s.seedProblems(tx, track.ID, patterns, dsaTopics); err != nil {
			return fmt.Errorf("seed problems: %w", err)
		}
		companies, err := s.seedCompanies(tx)
		if err != nil {
			return fmt.Errorf("seed companies: %w", err)
		}
		if err := s.seedCompanyWeights(tx, companies, pillars); err != nil {
			return fmt.Errorf("seed company weights: %w", err)
		}
		if err := s.seedProblemCompanyFrequency(tx, companies); err != nil {
			return fmt.Errorf("seed problem company frequency: %w", err)
		}
		var cerr error
		counts, cerr = s.count(tx)
		return cerr
	})
	return counts, err
}

func (s *Seeder) count(tx *gorm.DB) (Counts, error) {
	var c Counts
	for _, q := range []struct {
		model any
		dst   *int64
	}{
		{&content.Track{}, &c.Tracks},
		{&content.Pillar{}, &c.Pillars},
		{&content.Pattern{}, &c.Patterns},
		{&content.Topic{}, &c.Topics},
		{&content.Subtopic{}, &c.Subtopics},
		{&content.Resource{}, &c.Resources},
		{&content.Problem{}, &c.Problems},
		{&content.Company{}, &c.Companies},
	} {
		if err := tx.Model(q.model).Count(q.dst).Error; err != nil {
			return c, err
		}
	}
	return c, nil
}

// ---- Track ----

func (s *Seeder) seedTrack(tx *gorm.DB) (*content.Track, error) {
	t := content.Track{
		Slug:        "backend-sde3",
		Name:        "Backend SDE3",
		Description: ptr("Senior backend engineering interview preparation track covering DSA, System Design, LLD, Backend Engineering, Behavioral, and Resume."),
		Seniority:   ptr("SDE3"),
		IsActive:    true,
		SortOrder:   0,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "description", "seniority", "is_active", "sort_order", "updated_at"}),
	}).Create(&t).Error; err != nil {
		return nil, err
	}
	// Re-read to get the canonical id (OnConflict returns the inserted row id on
	// insert; on update the model id is the conflicting row's id via RETURNING).
	var out content.Track
	if err := tx.Where("slug = ?", t.Slug).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- Pillars ----

func (s *Seeder) seedPillars(tx *gorm.DB, trackID uuid.UUID) (map[content.PillarType]uuid.UUID, error) {
	defs := []struct {
		Type   content.PillarType
		Name   string
		Desc   string
		Weight float64
		Order  int
	}{
		{content.PillarDSA, "Data Structures & Algorithms", "Pattern-based problem solving across the canonical interview patterns.", 1.5, 0},
		{content.PillarSystem, "System Design", "High-level distributed systems design.", 1.5, 1},
		{content.PillarLLD, "Low-Level Design", "Object-oriented design and design patterns.", 1.0, 2},
		{content.PillarBackendEng, "Backend Engineering", "Databases, concurrency, APIs, and backend depth.", 1.0, 3},
		{content.PillarBehavioral, "Behavioral", "STAR stories and leadership-principle interviews.", 0.75, 4},
		{content.PillarResume, "Resume", "Resume and project narrative preparation.", 0.5, 5},
	}
	out := make(map[content.PillarType]uuid.UUID, len(defs))
	for _, d := range defs {
		p := content.Pillar{
			TrackID:     trackID,
			Type:        d.Type,
			Name:        d.Name,
			Description: ptr(d.Desc),
			Weight:      d.Weight,
			SortOrder:   d.Order,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "track_id"}, {Name: "type"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "description", "weight", "sort_order", "updated_at"}),
		}).Create(&p).Error; err != nil {
			return nil, err
		}
		var got content.Pillar
		if err := tx.Where("track_id = ? AND type = ?", trackID, d.Type).First(&got).Error; err != nil {
			return nil, err
		}
		out[d.Type] = got.ID
	}
	return out, nil
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }
