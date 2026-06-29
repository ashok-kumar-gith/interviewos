package ai

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNotFound is returned by readers when the requested record does not exist
// (e.g. no profile, story, or design problem for the user).
var ErrNotFound = errors.New("ai: record not found")

// ---- Read ports (clean architecture) ----
//
// The orchestrator depends only on these narrow interfaces, never on other
// modules' internals. The GORM implementations below query the existing tables
// directly (consistent with the cross-module read-port convention used by the
// analytics and roadmap modules).

// Profile is the slice of a user's intake profile the AI features need for
// grounding prompts and fallbacks.
type Profile struct {
	TrackID        uuid.UUID
	TargetRole     string
	TargetLevel    string
	YearsExp       float64
	HoursPerWeek   int
	TargetWeeks    int
	StartDate      string
	PillarStrength map[string]int
}

// ProfileReader reads a user's intake profile.
type ProfileReader interface {
	Profile(ctx context.Context, userID uuid.UUID) (*Profile, error)
}

// PlanTask is a single planned task for a date (subset of plan_tasks).
type PlanTask struct {
	Title            string
	Kind             string
	PillarType       string
	Priority         string
	Difficulty       string
	EstimatedMinutes int
	Status           string
}

// PlanReader reads the active roadmap summary and a day's planned tasks.
type PlanReader interface {
	// ActiveRoadmap returns the user's active roadmap id and total weeks, or
	// ErrNotFound when none exists.
	ActiveRoadmap(ctx context.Context, userID uuid.UUID) (id uuid.UUID, totalWeeks int, err error)
	// TasksForDate returns the planned tasks for the user on the given date
	// (YYYY-MM-DD), ordered by sort_order. An empty slice is valid (rest day / no
	// plan-day yet).
	TasksForDate(ctx context.Context, userID uuid.UUID, date string) ([]PlanTask, error)
}

// Story is a STAR story (subset of behavioral_stories).
type Story struct {
	ID        uuid.UUID
	Title     string
	Theme     string
	Situation string
	Task      string
	Action    string
	Result    string
	Metrics   string
}

// StoryReader reads a user's behavioral stories.
type StoryReader interface {
	// Story returns the user's story by id, or ErrNotFound.
	Story(ctx context.Context, userID, storyID uuid.UUID) (*Story, error)
}

// ResumeData is the resume profile + project bullets (subset of resume tables).
type ResumeData struct {
	Headline       string
	Summary        string
	Skills         []string
	TargetKeywords []string
	Bullets        []string
	Projects       int
}

// ResumeReader reads a user's resume profile and projects.
type ResumeReader interface {
	// Resume returns the user's resume data, or ErrNotFound when no profile.
	Resume(ctx context.Context, userID uuid.UUID) (*ResumeData, error)
}

// MockFinding is a single mock-interview finding (subset of mock_findings).
type MockFinding struct {
	PillarType string
	Severity   string
	Category   string
	Detail     string
}

// MockReader reads a user's mock-interview findings.
type MockReader interface {
	// Findings returns all of the user's mock findings (across mocks).
	Findings(ctx context.Context, userID uuid.UUID) ([]MockFinding, error)
}

// WeakTopic is a per-topic analytics row (weak end of the spectrum).
type WeakTopic struct {
	TopicID       uuid.UUID
	TopicName     string
	PillarType    string
	Confidence    *int
	CompletionPct float64
}

// AnalyticsReader reads weak-topic analytics for the weakness detector.
type AnalyticsReader interface {
	// WeakTopics returns the user's topics ranked weakest-first (lowest composite
	// of coverage and confidence), capped at limit.
	WeakTopics(ctx context.Context, userID uuid.UUID, limit int) ([]WeakTopic, error)
}

// DesignProblem is the metadata for an HLD design problem (subset).
type DesignProblem struct {
	ID         uuid.UUID
	Title      string
	Difficulty string
	Slug       string
}

// DesignReader reads design-problem metadata for the SD reviewer.
type DesignReader interface {
	// DesignProblem returns the design problem by id, or ErrNotFound.
	DesignProblem(ctx context.Context, id uuid.UUID) (*DesignProblem, error)
}

// ---- GORM-backed implementations ----

// gormReaders bundles all read ports onto a single *gorm.DB. It is split into the
// individual interfaces above so the orchestrator's dependencies stay narrow and
// each can be faked independently in tests.
type gormReaders struct {
	db *gorm.DB
}

// NewReaders constructs the GORM-backed read ports. The returned value satisfies
// every reader interface in this package.
func NewReaders(db *gorm.DB) *gormReaders { return &gormReaders{db: db} }

func (g *gormReaders) Profile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var row struct {
		TrackID        uuid.UUID      `gorm:"column:track_id"`
		TargetRole     string         `gorm:"column:target_role"`
		TargetLevel    *string        `gorm:"column:target_level"`
		YearsExp       float64        `gorm:"column:years_experience"`
		HoursPerWeek   int            `gorm:"column:hours_per_week"`
		TargetWeeks    int            `gorm:"column:target_weeks"`
		StartDate      string         `gorm:"column:start_date"`
		PillarStrength datatypesJSONB `gorm:"column:pillar_strengths"`
	}
	err := g.db.WithContext(ctx).Table("user_profiles").
		Select("track_id, target_role, target_level, years_experience, hours_per_week, target_weeks, to_char(start_date,'YYYY-MM-DD') AS start_date, pillar_strengths").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.TrackID == uuid.Nil {
		return nil, ErrNotFound
	}
	p := &Profile{
		TrackID:        row.TrackID,
		TargetRole:     row.TargetRole,
		YearsExp:       row.YearsExp,
		HoursPerWeek:   row.HoursPerWeek,
		TargetWeeks:    row.TargetWeeks,
		StartDate:      row.StartDate,
		PillarStrength: row.PillarStrength.toIntMap(),
	}
	if row.TargetLevel != nil {
		p.TargetLevel = *row.TargetLevel
	}
	return p, nil
}

func (g *gormReaders) ActiveRoadmap(ctx context.Context, userID uuid.UUID) (uuid.UUID, int, error) {
	var row struct {
		ID         uuid.UUID `gorm:"column:id"`
		TotalWeeks int       `gorm:"column:total_weeks"`
	}
	err := g.db.WithContext(ctx).Table("roadmaps").
		Select("id, total_weeks").
		Where("user_id = ? AND is_active AND deleted_at IS NULL", userID).
		Limit(1).Scan(&row).Error
	if err != nil {
		return uuid.Nil, 0, err
	}
	if row.ID == uuid.Nil {
		return uuid.Nil, 0, ErrNotFound
	}
	return row.ID, row.TotalWeeks, nil
}

func (g *gormReaders) TasksForDate(ctx context.Context, userID uuid.UUID, date string) ([]PlanTask, error) {
	var rows []struct {
		Title            string  `gorm:"column:title"`
		Kind             string  `gorm:"column:kind"`
		PillarType       string  `gorm:"column:pillar_type"`
		Priority         string  `gorm:"column:priority"`
		Difficulty       *string `gorm:"column:difficulty"`
		EstimatedMinutes int     `gorm:"column:estimated_minutes"`
		Status           string  `gorm:"column:status"`
	}
	err := g.db.WithContext(ctx).Table("plan_tasks AS t").
		Select("t.title, t.kind, t.pillar_type, t.priority, t.difficulty, t.estimated_minutes, t.status").
		Joins("JOIN plan_days d ON d.id = t.plan_day_id").
		Where("t.user_id = ? AND d.date = ? AND t.deleted_at IS NULL", userID, date).
		Order("t.sort_order ASC, t.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]PlanTask, 0, len(rows))
	for _, r := range rows {
		pt := PlanTask{
			Title:            r.Title,
			Kind:             r.Kind,
			PillarType:       r.PillarType,
			Priority:         r.Priority,
			EstimatedMinutes: r.EstimatedMinutes,
			Status:           r.Status,
		}
		if r.Difficulty != nil {
			pt.Difficulty = *r.Difficulty
		}
		out = append(out, pt)
	}
	return out, nil
}

func (g *gormReaders) Story(ctx context.Context, userID, storyID uuid.UUID) (*Story, error) {
	var row struct {
		ID        uuid.UUID `gorm:"column:id"`
		Title     string    `gorm:"column:title"`
		Theme     string    `gorm:"column:theme"`
		Situation *string   `gorm:"column:situation"`
		Task      *string   `gorm:"column:task"`
		Action    *string   `gorm:"column:action"`
		Result    *string   `gorm:"column:result"`
		Metrics   *string   `gorm:"column:metrics"`
	}
	err := g.db.WithContext(ctx).Table("behavioral_stories").
		Select("id, title, theme, situation, task, action, result, metrics").
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", storyID, userID).
		Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, ErrNotFound
	}
	return &Story{
		ID:        row.ID,
		Title:     row.Title,
		Theme:     row.Theme,
		Situation: deref(row.Situation),
		Task:      deref(row.Task),
		Action:    deref(row.Action),
		Result:    deref(row.Result),
		Metrics:   deref(row.Metrics),
	}, nil
}

func (g *gormReaders) Resume(ctx context.Context, userID uuid.UUID) (*ResumeData, error) {
	var prof struct {
		ID             uuid.UUID      `gorm:"column:id"`
		Headline       *string        `gorm:"column:headline"`
		Summary        *string        `gorm:"column:summary"`
		Skills         datatypesJSONB `gorm:"column:skills"`
		TargetKeywords datatypesJSONB `gorm:"column:target_keywords"`
	}
	err := g.db.WithContext(ctx).Table("resume_profiles").
		Select("id, headline, summary, skills, target_keywords").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Limit(1).Scan(&prof).Error
	if err != nil {
		return nil, err
	}
	if prof.ID == uuid.Nil {
		return nil, ErrNotFound
	}

	var projects []struct {
		Description *string        `gorm:"column:description"`
		Impact      *string        `gorm:"column:impact"`
		Metrics     datatypesJSONB `gorm:"column:metrics"`
	}
	err = g.db.WithContext(ctx).Table("resume_projects").
		Select("description, impact, metrics").
		Where("resume_profile_id = ? AND deleted_at IS NULL", prof.ID).
		Order("sort_order ASC, created_at ASC").
		Scan(&projects).Error
	if err != nil {
		return nil, err
	}

	var bullets []string
	for _, p := range projects {
		if v := deref(p.Description); v != "" {
			bullets = append(bullets, v)
		}
		if v := deref(p.Impact); v != "" {
			bullets = append(bullets, v)
		}
		bullets = append(bullets, p.Metrics.toStrings()...)
	}

	return &ResumeData{
		Headline:       deref(prof.Headline),
		Summary:        deref(prof.Summary),
		Skills:         prof.Skills.toStrings(),
		TargetKeywords: prof.TargetKeywords.toStrings(),
		Bullets:        bullets,
		Projects:       len(projects),
	}, nil
}

func (g *gormReaders) Findings(ctx context.Context, userID uuid.UUID) ([]MockFinding, error) {
	var rows []struct {
		PillarType *string `gorm:"column:pillar_type"`
		Severity   string  `gorm:"column:severity"`
		Category   string  `gorm:"column:category"`
		Detail     string  `gorm:"column:detail"`
	}
	err := g.db.WithContext(ctx).Table("mock_findings").
		Select("pillar_type, severity, category, detail").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]MockFinding, 0, len(rows))
	for _, r := range rows {
		out = append(out, MockFinding{
			PillarType: deref(r.PillarType),
			Severity:   r.Severity,
			Category:   r.Category,
			Detail:     r.Detail,
		})
	}
	return out, nil
}

func (g *gormReaders) WeakTopics(ctx context.Context, userID uuid.UUID, limit int) ([]WeakTopic, error) {
	if limit <= 0 {
		limit = 10
	}
	// Join user_topic_progress to topics for the user's track; rank weakest first
	// by (completion, confidence). Topics never started are the weakest of all.
	var rows []struct {
		TopicID       uuid.UUID `gorm:"column:topic_id"`
		TopicName     string    `gorm:"column:topic_name"`
		PillarType    string    `gorm:"column:pillar_type"`
		Confidence    *int      `gorm:"column:confidence"`
		CompletionPct float64   `gorm:"column:completion_pct"`
	}
	err := g.db.WithContext(ctx).Raw(`
		SELECT tp.id AS topic_id,
		       tp.name AS topic_name,
		       pl.type AS pillar_type,
		       utp.confidence AS confidence,
		       CASE utp.status WHEN 'completed' THEN 100.0 WHEN 'in_progress' THEN 50.0 ELSE 0.0 END AS completion_pct
		FROM user_topic_progress utp
		JOIN topics tp ON tp.id = utp.topic_id
		JOIN pillars pl ON pl.id = tp.pillar_id
		WHERE utp.user_id = ?
		ORDER BY completion_pct ASC, COALESCE(utp.confidence, 0) ASC, tp.name ASC
		LIMIT ?`, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]WeakTopic, 0, len(rows))
	for _, r := range rows {
		out = append(out, WeakTopic{
			TopicID:       r.TopicID,
			TopicName:     r.TopicName,
			PillarType:    r.PillarType,
			Confidence:    r.Confidence,
			CompletionPct: r.CompletionPct,
		})
	}
	return out, nil
}

func (g *gormReaders) DesignProblem(ctx context.Context, id uuid.UUID) (*DesignProblem, error) {
	var row struct {
		ID         uuid.UUID `gorm:"column:id"`
		Title      string    `gorm:"column:title"`
		Difficulty string    `gorm:"column:difficulty"`
		Slug       string    `gorm:"column:slug"`
	}
	err := g.db.WithContext(ctx).Table("design_problems").
		Select("id, title, difficulty, slug").
		Where("id = ? AND deleted_at IS NULL", id).
		Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, ErrNotFound
	}
	return &DesignProblem{ID: row.ID, Title: row.Title, Difficulty: row.Difficulty, Slug: row.Slug}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
