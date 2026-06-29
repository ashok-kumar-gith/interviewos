package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// companyDef defines a target company. Slug is the natural key.
type companyDef struct {
	Slug          string
	Name          string
	Style         string
	FullyWeighted bool
}

var seedCompanies = []companyDef{
	{"amazon", "Amazon", "Leadership-Principles heavy: every round probes LP behavioral signals alongside coding and design. Bar-raiser round is decisive.", true},
	{"google", "Google", "Algorithmic depth and clean coding; Googleyness & Leadership round; data-structure rigor.", true},
	{"uber", "Uber", "Practical system design (real-time, geo, matching) plus strong DSA; pragmatic backend depth.", true},
	{"microsoft", "Microsoft", "Balanced DSA and design with collaboration signals; pragmatic problem solving.", false},
	{"flipkart", "Flipkart", "Scale-oriented system design and strong DSA; e-commerce domain problems.", false},
	{"atlassian", "Atlassian", "Values-based behavioral, practical coding, and craft-focused design discussions.", false},
	{"rubrik", "Rubrik", "Systems-heavy backend and DSA with depth on storage/distributed systems.", false},
	{"phonepe", "PhonePe", "High-scale payments system design and strong DSA fundamentals.", false},
	{"razorpay", "Razorpay", "Payments/fintech backend depth, API design, and solid DSA.", false},
	{"swiggy", "Swiggy", "Real-time logistics system design and pragmatic DSA.", false},
}

// seedCompanies upserts companies (dedup by slug) and returns slug→id.
func (s *Seeder) seedCompanies(tx *gorm.DB) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID, len(seedCompanies))
	for i, d := range seedCompanies {
		c := content.Company{
			Slug:             d.Slug,
			Name:             d.Name,
			InterviewStyleMD: ptr(d.Style),
			IsFullyWeighted:  d.FullyWeighted,
			SortOrder:        i,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "interview_style_md", "is_fully_weighted", "sort_order", "updated_at"}),
		}).Create(&c).Error; err != nil {
			return nil, err
		}
		var got content.Company
		if err := tx.Where("slug = ?", d.Slug).First(&got).Error; err != nil {
			return nil, err
		}
		out[d.Slug] = got.ID
	}
	return out, nil
}

// pillarWeightProfile is a company's per-pillar emphasis multipliers.
type pillarWeightProfile map[content.PillarType]float64

// companyPillarWeights defines per-company pillar emphasis. Fully weighted
// companies get a tuned profile; others get reasonable defaults.
var companyPillarWeights = map[string]pillarWeightProfile{
	"amazon": {content.PillarDSA: 1.2, content.PillarSystem: 1.3, content.PillarLLD: 1.1, content.PillarBackendEng: 1.0, content.PillarBehavioral: 1.6, content.PillarResume: 0.8},
	"google": {content.PillarDSA: 1.6, content.PillarSystem: 1.3, content.PillarLLD: 0.9, content.PillarBackendEng: 1.0, content.PillarBehavioral: 1.0, content.PillarResume: 0.7},
	"uber":   {content.PillarDSA: 1.3, content.PillarSystem: 1.5, content.PillarLLD: 1.1, content.PillarBackendEng: 1.2, content.PillarBehavioral: 1.0, content.PillarResume: 0.7},
}

// defaultPillarWeights applies to non-fully-weighted companies.
var defaultPillarWeights = pillarWeightProfile{
	content.PillarDSA: 1.2, content.PillarSystem: 1.2, content.PillarLLD: 1.0,
	content.PillarBackendEng: 1.0, content.PillarBehavioral: 1.0, content.PillarResume: 0.8,
}

// seedCompanyWeights upserts per-company pillar weight multipliers.
func (s *Seeder) seedCompanyWeights(tx *gorm.DB, companies map[string]uuid.UUID, pillars map[content.PillarType]uuid.UUID) error {
	for slug, companyID := range companies {
		profile, ok := companyPillarWeights[slug]
		if !ok {
			profile = defaultPillarWeights
		}
		for pillarType, mult := range profile {
			pid, ok := pillars[pillarType]
			if !ok {
				continue
			}
			pidCopy := pid
			cw := content.CompanyWeight{
				CompanyID:        companyID,
				PillarID:         &pidCopy,
				WeightMultiplier: mult,
			}
			// Upsert on the (company_id, pillar_id) partial unique index
			// (topic_id IS NULL).
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "company_id"}, {Name: "pillar_id"}},
				TargetWhere: clause.Where{Exprs: []clause.Expression{
					clause.Expr{SQL: "topic_id IS NULL"},
				}},
				DoUpdates: clause.AssignmentColumns([]string{"weight_multiplier", "note", "updated_at"}),
			}).Create(&cw).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// companyProblemFrequency maps a company slug to (problem slug → frequency).
// Provided for Amazon/Google/Uber as representative company-frequency rows.
var companyProblemFrequency = map[string]map[string]float64{
	"amazon": {
		"two-sum": 92, "number-of-islands": 88, "merge-intervals": 70,
		"group-anagrams": 64, "course-schedule": 60, "valid-parentheses": 58, "3sum": 66,
		"word-search": 55, "trapping-rain-water": 62,
	},
	"google": {
		"two-sum": 70, "longest-substring-without-repeating-characters": 80, "merge-intervals": 72,
		"binary-tree-maximum-path-sum": 60, "word-ladder": 55, "median-of-two-sorted-arrays": 64,
		"course-schedule": 58, "3sum": 62,
	},
	"uber": {
		"merge-intervals": 76, "number-of-islands": 70, "two-sum": 60, "lowest-common-ancestor-of-a-binary-search-tree": 50,
		"meeting-rooms-ii": 72, "course-schedule": 64,
	},
}

// seedProblemCompanyFrequency upserts representative company-frequency rows for
// problems that exist (unknown problem slugs are skipped).
func (s *Seeder) seedProblemCompanyFrequency(tx *gorm.DB, companies map[string]uuid.UUID) error {
	const period = "2025-H2"
	for slug, freqs := range companyProblemFrequency {
		companyID, ok := companies[slug]
		if !ok {
			continue
		}
		for problemSlug, freq := range freqs {
			var prob content.Problem
			if err := tx.Where("slug = ?", problemSlug).First(&prob).Error; err != nil {
				// Problem not in the seeded set; skip silently.
				continue
			}
			pcf := content.ProblemCompanyFrequency{
				ProblemID:      prob.ID,
				CompanyID:      companyID,
				Frequency:      freq,
				LastSeenPeriod: ptr(period),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "problem_id"}, {Name: "company_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"frequency", "last_seen_period", "updated_at"}),
			}).Create(&pcf).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
