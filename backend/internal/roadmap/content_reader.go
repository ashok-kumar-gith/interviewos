package roadmap

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/interviewos/backend/internal/curriculum"
)

// ContentReader is the read port the engine input is assembled from. The gorm
// implementation queries the content tables directly; the service depends only
// on this interface so it can be faked in tests.
type ContentReader interface {
	// LoadEngineInput assembles the full curriculum.Input for a user's profile,
	// reading pillars/topics/problems/resources for the track and company weights
	// for the optional target company.
	LoadEngineInput(ctx context.Context, profileTrackID uuid.UUID, targetCompanyID *uuid.UUID, prof curriculum.Profile) (curriculum.Input, error)
}

// gormContentReader implements ContentReader against the live content tables.
type gormContentReader struct {
	db *gorm.DB
}

// NewContentReader returns a gorm-backed ContentReader.
func NewContentReader(db *gorm.DB) ContentReader {
	return &gormContentReader{db: db}
}

// pillarRow is a minimal projection of pillars for the engine.
type pillarRow struct {
	ID     uuid.UUID
	Type   string
	Weight float64
}

type topicRow struct {
	ID             uuid.UUID
	PillarType     string
	Slug           string
	Name           string
	Difficulty     string
	Priority       string
	EstimatedHours float64
	Prerequisites  []byte
	SortOrder      int
}

type problemRow struct {
	ID               uuid.UUID
	TopicID          *uuid.UUID
	Title            string
	Difficulty       string
	EstimatedMinutes int
	FrequencyScore   float64
	CompanyFrequency float64
}

type resourceRow struct {
	ID               uuid.UUID
	TopicID          uuid.UUID
	Title            string
	Type             string
	EstimatedMinutes *int
	IsPrimary        bool
	SortOrder        int
}

func (r *gormContentReader) LoadEngineInput(ctx context.Context, trackID uuid.UUID, companyID *uuid.UUID, prof curriculum.Profile) (curriculum.Input, error) {
	in := curriculum.Input{Profile: prof, CompanyMul: map[curriculum.PillarType]float64{}}

	// Pillars for the track.
	var pillars []pillarRow
	if err := r.db.WithContext(ctx).Table("pillars").
		Select("id, type, weight").
		Where("track_id = ? AND deleted_at IS NULL", trackID).
		Order("sort_order ASC").Scan(&pillars).Error; err != nil {
		return in, err
	}
	pillarIDByType := map[string]uuid.UUID{}
	for _, p := range pillars {
		in.Pillars = append(in.Pillars, curriculum.PillarMeta{
			Type: curriculum.PillarType(p.Type), Weight: p.Weight,
		})
		pillarIDByType[p.Type] = p.ID
	}

	// Topics for the track, with their pillar type.
	var topics []topicRow
	if err := r.db.WithContext(ctx).Table("topics t").
		Select("t.id, p.type AS pillar_type, t.slug, t.name, t.difficulty, t.priority, t.estimated_hours, t.prerequisites, t.sort_order").
		Joins("JOIN pillars p ON p.id = t.pillar_id").
		Where("t.track_id = ? AND t.deleted_at IS NULL", trackID).
		Order("t.sort_order ASC").Scan(&topics).Error; err != nil {
		return in, err
	}
	for _, t := range topics {
		var prereqStrs []string
		if len(t.Prerequisites) > 0 {
			_ = json.Unmarshal(t.Prerequisites, &prereqStrs)
		}
		var prereqs []uuid.UUID
		for _, s := range prereqStrs {
			if id, err := uuid.Parse(s); err == nil {
				prereqs = append(prereqs, id)
			}
		}
		in.Topics = append(in.Topics, curriculum.Topic{
			ID:             t.ID,
			Pillar:         curriculum.PillarType(t.PillarType),
			Slug:           t.Slug,
			Name:           t.Name,
			Difficulty:     curriculum.Difficulty(t.Difficulty),
			Priority:       curriculum.Priority(t.Priority),
			EstimatedHours: t.EstimatedHours,
			Prerequisites:  prereqs,
			SortOrder:      t.SortOrder,
		})
	}

	// Problems for the track, joined with target-company frequency (0 if no company).
	q := r.db.WithContext(ctx).Table("problems pr").
		Select("pr.id, pr.topic_id, pr.title, pr.difficulty, pr.estimated_minutes, pr.frequency_score, COALESCE(pcf.frequency, 0) AS company_frequency").
		Where("pr.track_id = ? AND pr.deleted_at IS NULL AND pr.topic_id IS NOT NULL", trackID).
		Order("pr.frequency_score DESC")
	if companyID != nil {
		q = q.Joins("LEFT JOIN problem_company_frequency pcf ON pcf.problem_id = pr.id AND pcf.company_id = ?", *companyID)
	} else {
		q = q.Joins("LEFT JOIN problem_company_frequency pcf ON pcf.problem_id = pr.id AND false")
	}
	var problems []problemRow
	if err := q.Scan(&problems).Error; err != nil {
		return in, err
	}
	for _, p := range problems {
		if p.TopicID == nil {
			continue
		}
		in.Problems = append(in.Problems, curriculum.Problem{
			ID:               p.ID,
			TopicID:          *p.TopicID,
			Title:            p.Title,
			Difficulty:       curriculum.Difficulty(p.Difficulty),
			EstimatedMinutes: p.EstimatedMinutes,
			FrequencyScore:   p.FrequencyScore,
			CompanyFrequency: p.CompanyFrequency,
		})
	}

	// Primary resources linked to the track's topics.
	var resources []resourceRow
	if err := r.db.WithContext(ctx).Table("resources r").
		Select("r.id, tr.topic_id, r.title, r.type, r.estimated_minutes, tr.is_primary, tr.sort_order").
		Joins("JOIN topic_resources tr ON tr.resource_id = r.id").
		Joins("JOIN topics t ON t.id = tr.topic_id").
		Where("t.track_id = ? AND r.deleted_at IS NULL", trackID).
		Order("tr.sort_order ASC").Scan(&resources).Error; err != nil {
		return in, err
	}
	for _, rs := range resources {
		est := 30
		if rs.EstimatedMinutes != nil {
			est = *rs.EstimatedMinutes
		}
		kind := curriculum.KindRead
		if rs.Type == "video" || rs.Type == "course" {
			kind = curriculum.KindWatch
		}
		in.Resources = append(in.Resources, curriculum.Resource{
			ID:               rs.ID,
			TopicID:          rs.TopicID,
			Title:            rs.Title,
			Kind:             kind,
			EstimatedMinutes: est,
			IsPrimary:        rs.IsPrimary,
			SortOrder:        rs.SortOrder,
		})
	}

	// Company pillar multipliers for the target company.
	if companyID != nil {
		type cwRow struct {
			PillarID         *uuid.UUID
			WeightMultiplier float64
		}
		var rows []cwRow
		if err := r.db.WithContext(ctx).Table("company_weights").
			Select("pillar_id, weight_multiplier").
			Where("company_id = ? AND pillar_id IS NOT NULL", *companyID).
			Scan(&rows).Error; err != nil {
			return in, err
		}
		typeByPillarID := map[uuid.UUID]string{}
		for typ, id := range pillarIDByType {
			typeByPillarID[id] = typ
		}
		for _, row := range rows {
			if row.PillarID == nil {
				continue
			}
			if typ, ok := typeByPillarID[*row.PillarID]; ok {
				in.CompanyMul[curriculum.PillarType(typ)] = row.WeightMultiplier
			}
		}
	}

	return in, nil
}
