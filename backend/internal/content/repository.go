package content

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Page describes pagination input (1-based page, bounded size).
type Page struct {
	Page     int
	PageSize int
}

// Offset returns the SQL OFFSET for the page.
func (p Page) Offset() int { return (p.Page - 1) * p.PageSize }

// SortField is a single normalized ordering instruction.
type SortField struct {
	Column string
	Desc   bool
}

// TopicFilter constrains a topic listing.
type TopicFilter struct {
	TrackID    *uuid.UUID
	PillarID   *uuid.UUID
	PillarType *PillarType
	Difficulty *Difficulty
	Priority   *Priority
	Query      string
	Sort       []SortField
}

// ResourceFilter constrains a resource listing.
type ResourceFilter struct {
	Type       *ResourceType
	TopicID    *uuid.UUID
	Difficulty *Difficulty
	Query      string
	Sort       []SortField
}

// ProblemFilter constrains a problem listing. CompanySlug is an alternative to
// CompanyID supporting the human-friendly filter=company:<slug> form.
type ProblemFilter struct {
	Difficulty  *Difficulty
	PatternID   *uuid.UUID
	PatternSlug string
	TopicID     *uuid.UUID
	CompanyID   *uuid.UUID
	CompanySlug string
	Source      *ProblemSourceName
	Query       string
	Sort        []SortField
}

// ProblemBundle is a problem with its related patterns, sources, and company
// frequencies, used by the detail endpoint.
type ProblemBundle struct {
	Problem          Problem
	Patterns         []Pattern
	Sources          []ProblemSource
	CompanyFrequency []CompanyFrequencyRow
}

// CompanyFrequencyRow joins a frequency row with its company name for the API.
type CompanyFrequencyRow struct {
	CompanyID      uuid.UUID
	CompanyName    string
	Frequency      float64
	LastSeenPeriod *string
}

// TopicBundle is a topic with its subtopics and linked resources.
type TopicBundle struct {
	Topic     Topic
	Subtopics []Subtopic
	Resources []Resource
}

// CompanyBundle is a company with its weights.
type CompanyBundle struct {
	Company Company
	Weights []CompanyWeight
}

// Repository abstracts read access to the content library so the service can be
// tested against a fake. The GORM implementation is gormRepository.
type Repository interface {
	ListTracks(ctx context.Context, q string, sort []SortField, p Page) ([]Track, int64, error)
	ListPillars(ctx context.Context, trackID *uuid.UUID, sort []SortField, p Page) ([]Pillar, int64, error)
	ListTopics(ctx context.Context, f TopicFilter, p Page) ([]Topic, int64, error)
	GetTopicBundle(ctx context.Context, id uuid.UUID) (*TopicBundle, error)
	ListResources(ctx context.Context, f ResourceFilter, p Page) ([]Resource, int64, error)
	ListPatterns(ctx context.Context, trackID *uuid.UUID, q string, sort []SortField, p Page) ([]Pattern, int64, error)
	ListProblems(ctx context.Context, f ProblemFilter, p Page) ([]Problem, int64, error)
	GetProblemBundle(ctx context.Context, id uuid.UUID) (*ProblemBundle, error)
	ListCompanies(ctx context.Context, q string, sort []SortField, p Page) ([]Company, int64, error)
	GetCompanyBundle(ctx context.Context, id uuid.UUID) (*CompanyBundle, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// applySort appends ORDER BY clauses from a validated sort list, falling back to
// the supplied default column (ascending) when no sort is given.
func applySort(q *gorm.DB, sort []SortField, def string) *gorm.DB {
	if len(sort) == 0 {
		return q.Order(def)
	}
	for _, s := range sort {
		dir := " ASC"
		if s.Desc {
			dir = " DESC"
		}
		q = q.Order(s.Column + dir)
	}
	return q
}

func (r *gormRepository) ListTracks(ctx context.Context, q string, sort []SortField, p Page) ([]Track, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Track{})
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Track
	err := applySort(tx, sort, "sort_order ASC, name ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) ListPillars(ctx context.Context, trackID *uuid.UUID, sort []SortField, p Page) ([]Pillar, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Pillar{})
	if trackID != nil {
		tx = tx.Where("track_id = ?", *trackID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Pillar
	err := applySort(tx, sort, "sort_order ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) ListTopics(ctx context.Context, f TopicFilter, p Page) ([]Topic, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Topic{})
	if f.TrackID != nil {
		tx = tx.Where("topics.track_id = ?", *f.TrackID)
	}
	if f.PillarID != nil {
		tx = tx.Where("topics.pillar_id = ?", *f.PillarID)
	}
	if f.PillarType != nil {
		tx = tx.Where("topics.pillar_id IN (SELECT id FROM pillars WHERE type = ?)", *f.PillarType)
	}
	if f.Difficulty != nil {
		tx = tx.Where("topics.difficulty = ?", *f.Difficulty)
	}
	if f.Priority != nil {
		tx = tx.Where("topics.priority = ?", *f.Priority)
	}
	if f.Query != "" {
		tx = tx.Where("topics.name ILIKE ?", "%"+f.Query+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Topic
	err := applySort(tx, f.Sort, "sort_order ASC, name ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) GetTopicBundle(ctx context.Context, id uuid.UUID) (*TopicBundle, error) {
	var t Topic
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var subs []Subtopic
	if err := r.db.WithContext(ctx).Where("topic_id = ?", id).
		Order("sort_order ASC").Find(&subs).Error; err != nil {
		return nil, err
	}
	var res []Resource
	if err := r.db.WithContext(ctx).
		Joins("JOIN topic_resources tr ON tr.resource_id = resources.id").
		Where("tr.topic_id = ?", id).
		Order("tr.sort_order ASC").Find(&res).Error; err != nil {
		return nil, err
	}
	return &TopicBundle{Topic: t, Subtopics: subs, Resources: res}, nil
}

func (r *gormRepository) ListResources(ctx context.Context, f ResourceFilter, p Page) ([]Resource, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Resource{})
	if f.TopicID != nil {
		tx = tx.Where("resources.id IN (SELECT resource_id FROM topic_resources WHERE topic_id = ?)", *f.TopicID)
	}
	if f.Type != nil {
		tx = tx.Where("resources.type = ?", *f.Type)
	}
	if f.Difficulty != nil {
		tx = tx.Where("resources.difficulty = ?", *f.Difficulty)
	}
	if f.Query != "" {
		tx = tx.Where("resources.title ILIKE ?", "%"+f.Query+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Resource
	err := applySort(tx, f.Sort, "title ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) ListPatterns(ctx context.Context, trackID *uuid.UUID, q string, sort []SortField, p Page) ([]Pattern, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Pattern{})
	if trackID != nil {
		tx = tx.Where("track_id = ?", *trackID)
	}
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Pattern
	err := applySort(tx, sort, "sort_order ASC, name ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) ListProblems(ctx context.Context, f ProblemFilter, p Page) ([]Problem, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Problem{})
	if f.Difficulty != nil {
		tx = tx.Where("problems.difficulty = ?", *f.Difficulty)
	}
	if f.TopicID != nil {
		tx = tx.Where("problems.topic_id = ?", *f.TopicID)
	}
	if f.PatternID != nil {
		tx = tx.Where("problems.id IN (SELECT problem_id FROM problem_patterns WHERE pattern_id = ?)", *f.PatternID)
	} else if f.PatternSlug != "" {
		tx = tx.Where("problems.id IN (SELECT pp.problem_id FROM problem_patterns pp JOIN patterns pt ON pt.id = pp.pattern_id WHERE pt.slug = ?)", f.PatternSlug)
	}
	if f.CompanyID != nil {
		tx = tx.Where("problems.id IN (SELECT problem_id FROM problem_company_frequency WHERE company_id = ?)", *f.CompanyID)
	} else if f.CompanySlug != "" {
		tx = tx.Where("problems.id IN (SELECT pcf.problem_id FROM problem_company_frequency pcf JOIN companies co ON co.id = pcf.company_id WHERE co.slug = ?)", f.CompanySlug)
	}
	if f.Source != nil {
		tx = tx.Where("problems.id IN (SELECT problem_id FROM problem_sources WHERE source = ?)", *f.Source)
	}
	if f.Query != "" {
		tx = tx.Where("problems.title ILIKE ?", "%"+f.Query+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Problem
	err := applySort(tx, f.Sort, "frequency_score DESC, title ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) GetProblemBundle(ctx context.Context, id uuid.UUID) (*ProblemBundle, error) {
	var prob Problem
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&prob).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var pats []Pattern
	if err := r.db.WithContext(ctx).
		Joins("JOIN problem_patterns pp ON pp.pattern_id = patterns.id").
		Where("pp.problem_id = ?", id).
		Order("patterns.sort_order ASC").Find(&pats).Error; err != nil {
		return nil, err
	}
	var srcs []ProblemSource
	if err := r.db.WithContext(ctx).Where("problem_id = ?", id).
		Order("source ASC").Find(&srcs).Error; err != nil {
		return nil, err
	}
	var freqs []CompanyFrequencyRow
	if err := r.db.WithContext(ctx).Model(&ProblemCompanyFrequency{}).
		Select("problem_company_frequency.company_id, companies.name AS company_name, problem_company_frequency.frequency, problem_company_frequency.last_seen_period").
		Joins("JOIN companies ON companies.id = problem_company_frequency.company_id").
		Where("problem_company_frequency.problem_id = ?", id).
		Order("problem_company_frequency.frequency DESC").
		Scan(&freqs).Error; err != nil {
		return nil, err
	}
	return &ProblemBundle{Problem: prob, Patterns: pats, Sources: srcs, CompanyFrequency: freqs}, nil
}

func (r *gormRepository) ListCompanies(ctx context.Context, q string, sort []SortField, p Page) ([]Company, int64, error) {
	tx := r.db.WithContext(ctx).Model(&Company{})
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Company
	err := applySort(tx, sort, "sort_order ASC, name ASC").
		Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

func (r *gormRepository) GetCompanyBundle(ctx context.Context, id uuid.UUID) (*CompanyBundle, error) {
	var c Company
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var weights []CompanyWeight
	if err := r.db.WithContext(ctx).Where("company_id = ?", id).Find(&weights).Error; err != nil {
		return nil, err
	}
	return &CompanyBundle{Company: c, Weights: weights}, nil
}

// sortColumnAllowed reports whether a client-supplied sort column is in the
// allowlist, guarding against SQL injection through the sort parameter.
func sortColumnAllowed(col string, allowed map[string]struct{}) bool {
	_, ok := allowed[strings.ToLower(col)]
	return ok
}
