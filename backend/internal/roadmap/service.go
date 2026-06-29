package roadmap

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/curriculum"
)

// Profile is the projection of the intake profile the roadmap service needs.
// It is returned by the ProfileReader port so the roadmap module never imports
// the intake module's internals (clean module seams, 03-ARCHITECTURE.md §4.4).
type Profile struct {
	ID              uuid.UUID
	TrackID         uuid.UUID
	TargetCompanyID *uuid.UUID
	HoursPerWeek    int
	StartDate       time.Time
	TargetWeeks     int
	PillarStrengths map[string]int
}

// ProfileReader is the read port for a user's intake profile.
type ProfileReader interface {
	// GetProfile returns the user's active intake profile, or an error if none
	// exists. The roadmap service maps a not-found error to ErrProfileRequired.
	GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error)
}

// Service implements the roadmap use-cases. It orchestrates the ProfileReader,
// the ContentReader, the deterministic curriculum engine, and the Repository.
type Service struct {
	repo     Repository
	profiles ProfileReader
	content  ContentReader
	now      func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo     Repository
	Profiles ProfileReader
	Content  ContentReader
	Now      func() time.Time
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Service{repo: cfg.Repo, profiles: cfg.Profiles, content: cfg.Content, now: nowFn}
}

const defaultActiveDays = 6

// GenerateRoadmap reads the user's profile, runs the deterministic engine, and
// persists the resulting roadmap graph. When regenerate is false and an active
// roadmap already exists, it returns ErrActiveRoadmapExists. When regenerate is
// true, the existing active roadmap is archived and a fresh one is generated.
func (s *Service) GenerateRoadmap(ctx context.Context, userID uuid.UUID, regenerate bool) (*Roadmap, error) {
	if !regenerate {
		if existing, err := s.repo.GetActive(ctx, userID); err == nil && existing != nil {
			return nil, ErrActiveRoadmapExists
		} else if err != nil && err != ErrNoActiveRoadmap {
			return nil, err
		}
	}

	prof, err := s.profiles.GetProfile(ctx, userID)
	if err != nil {
		return nil, ErrProfileRequired
	}

	weeks := prof.TargetWeeks
	if weeks <= 0 {
		weeks = curriculum.DefaultWeeks
	}

	engineProfile := curriculum.Profile{
		HoursPerWeek:    prof.HoursPerWeek,
		StartDate:       prof.StartDate,
		TargetWeeks:     weeks,
		ActiveDays:      defaultActiveDays,
		PillarStrengths: toEnginePillarStrengths(prof.PillarStrengths),
	}

	in, err := s.content.LoadEngineInput(ctx, prof.TrackID, prof.TargetCompanyID, engineProfile)
	if err != nil {
		return nil, err
	}

	plan := curriculum.Generate(in)

	rm := s.planToRoadmap(userID, prof, plan)
	if err := s.repo.CreateGraph(ctx, rm, regenerate); err != nil {
		return nil, err
	}
	// Reload with weeks for the response (CreateGraph populated ids in-place; we
	// return the in-memory graph which already carries weeks/days/tasks).
	return rm, nil
}

// GetActive returns the active roadmap with its weeks (summary, no days).
func (s *Service) GetActive(ctx context.Context, userID uuid.UUID) (*Roadmap, error) {
	return s.repo.GetActiveWithWeeks(ctx, userID)
}

// GetWeek returns a single week with its days and tasks.
func (s *Service) GetWeek(ctx context.Context, userID, roadmapID uuid.UUID, weekNumber int) (*RoadmapWeek, error) {
	return s.repo.GetWeek(ctx, userID, roadmapID, weekNumber)
}

// GetPlanDay returns the user's plan-day for a date with its tasks.
func (s *Service) GetPlanDay(ctx context.Context, userID uuid.UUID, date time.Time) (*PlanDay, error) {
	return s.repo.GetPlanDay(ctx, userID, date)
}

// planToRoadmap maps the engine Plan into the persistence graph.
func (s *Service) planToRoadmap(userID uuid.UUID, prof *Profile, plan curriculum.Plan) *Roadmap {
	params, _ := json.Marshal(map[string]any{
		"hours_per_week":   prof.HoursPerWeek,
		"target_weeks":     plan.TotalWeeks,
		"pillar_strengths": prof.PillarStrengths,
		"target_company":   prof.TargetCompanyID,
		"generated_at":     s.now().UTC().Format(time.RFC3339),
	})

	rm := &Roadmap{
		UserID:           userID,
		TrackID:          prof.TrackID,
		ProfileID:        prof.ID,
		TargetCompanyID:  prof.TargetCompanyID,
		StartDate:        plan.StartDate,
		EndDate:          plan.EndDate,
		TotalWeeks:       int16(plan.TotalWeeks),
		HoursPerWeek:     int16(plan.HoursPerWeek),
		Status:           "active",
		IsActive:         true,
		GenerationParams: params,
		GeneratedBy:      "engine",
	}

	for _, w := range plan.Weeks {
		theme := w.Theme
		week := RoadmapWeek{
			WeekNumber:   int16(w.Number),
			StartDate:    w.StartDate,
			EndDate:      w.EndDate,
			Theme:        &theme,
			FocusPillars: toStringSlice(w.FocusPillars),
			PlannedHours: w.PlannedHours,
		}
		for _, d := range w.Days {
			day := PlanDay{
				UserID:         userID,
				Date:           d.Date,
				PlannedMinutes: d.PlannedMinutes,
				IsRestDay:      d.IsRestDay,
			}
			for _, t := range d.Tasks {
				task := PlanTask{
					UserID:           userID,
					Kind:             string(t.Kind),
					ItemType:         string(t.ItemType),
					ItemID:           t.ItemID,
					PillarType:       string(t.Pillar),
					Title:            t.Title,
					Objectives:       JSONStringArray{},
					EstimatedMinutes: t.EstimatedMinutes,
					Priority:         string(t.Priority),
					Status:           "pending",
					SortOrder:        t.SortOrder,
				}
				if t.Difficulty != "" {
					diff := string(t.Difficulty)
					task.Difficulty = &diff
				}
				day.Tasks = append(day.Tasks, task)
			}
			week.Days = append(week.Days, day)
		}
		rm.Weeks = append(rm.Weeks, week)
	}
	return rm
}

func toEnginePillarStrengths(m map[string]int) map[curriculum.PillarType]int {
	out := make(map[curriculum.PillarType]int, len(m))
	for k, v := range m {
		out[curriculum.PillarType(k)] = v
	}
	return out
}

func toStringSlice(ps []curriculum.PillarType) JSONStringArray {
	out := make(JSONStringArray, 0, len(ps))
	for _, p := range ps {
		out = append(out, string(p))
	}
	return out
}
