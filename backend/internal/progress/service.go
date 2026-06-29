package progress

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service implements the progress/Today/dashboard use-cases. It orchestrates the
// Repository and applies the readiness model (03-ARCHITECTURE.md §8.1, ADR D15).
type Service struct {
	repo Repository
	now  func() time.Time
}

// ServiceConfig configures a Service.
type ServiceConfig struct {
	Repo Repository
	Now  func() time.Time
}

// NewService constructs a Service.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Service{repo: cfg.Repo, now: nowFn}
}

// CompleteParams is the validated input to CompleteTask.
type CompleteParams struct {
	Confidence       int
	TimeSpentMinutes int
	Notes            string
}

// GetToday returns the user's plan-day for the current date (server-local UTC),
// enriched with its tasks. Returns ErrPlanDayNotFound when none exists.
func (s *Service) GetToday(ctx context.Context, userID uuid.UUID) (*PlanDayRow, error) {
	return s.repo.GetPlanDay(ctx, userID, s.today())
}

// CompleteTask validates the payload and transactionally completes the task,
// upserting progress + session + streak. It returns the updated task and the
// user's refreshed streak.
func (s *Service) CompleteTask(ctx context.Context, userID, taskID uuid.UUID, p CompleteParams) (*PlanTaskRow, Streak, error) {
	if p.Confidence < 1 || p.Confidence > 5 {
		return nil, Streak{}, ErrInvalidConfidence
	}
	if p.TimeSpentMinutes < 0 {
		return nil, Streak{}, ErrInvalidConfidence
	}
	in := CompleteInput{
		Confidence:       int16(p.Confidence),
		TimeSpentMinutes: p.TimeSpentMinutes,
	}
	if p.Notes != "" {
		notes := p.Notes
		in.Notes = &notes
	}
	now := s.now().UTC()
	task, err := s.repo.CompleteTask(ctx, userID, taskID, in, now)
	if err != nil {
		return nil, Streak{}, err
	}
	streak, err := s.repo.ComputeStreak(ctx, userID, now)
	if err != nil {
		return nil, Streak{}, err
	}
	return task, streak, nil
}

// SkipTask marks a task skipped (with an optional reason).
func (s *Service) SkipTask(ctx context.Context, userID, taskID uuid.UUID, reason string) (*PlanTaskRow, error) {
	return s.repo.SkipTask(ctx, userID, taskID, reason, s.now().UTC())
}

// RescheduleTask moves a task to the plan-day for toDate.
func (s *Service) RescheduleTask(ctx context.Context, userID, taskID uuid.UUID, toDate time.Time) (*PlanTaskRow, error) {
	if toDate.IsZero() {
		return nil, ErrInvalidReschedule
	}
	return s.repo.RescheduleTask(ctx, userID, taskID, toDate, s.now().UTC())
}

// readinessThreshold is the target overall readiness used for the estimated date
// projection (ADR D15 / §8.1).
const readinessThreshold = 80.0

// GetDashboard assembles the DashboardResponse aggregate: per-pillar readiness
// via the multiplicative SRS form, the weighted overall, streak, today counts,
// and revision-due count.
func (s *Service) GetDashboard(ctx context.Context, userID uuid.UUID) (*Dashboard, error) {
	now := s.now().UTC()

	aggs, err := s.repo.PillarAggregates(ctx, userID)
	if err != nil {
		return nil, err
	}
	streak, err := s.repo.ComputeStreak(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	revDue, err := s.repo.RevisionDueCount(ctx, userID, now)
	if err != nil {
		return nil, err
	}

	pillars := make([]PillarReadiness, 0, len(aggs))
	var weightedSum, totalWeight float64
	for _, a := range aggs {
		coverage := 0.0
		if a.PlannedMinutes > 0 {
			coverage = float64(a.CompletedMinutes) / float64(a.PlannedMinutes)
		}
		// confidence_p = (avg_rating - 1) / 4, mapping [1..5] → [0..1].
		var confidence float64
		avgRating := 0.0
		if a.ConfidenceCount > 0 {
			avgRating = float64(a.ConfidenceSum) / float64(a.ConfidenceCount)
			confidence = (avgRating - 1) / 4
		}
		// revhealth defaults to 1.0 until revision data exists (per spec).
		revHealth := 1.0
		readiness := 100 * coverage * (0.6*confidence + 0.4*revHealth)

		pillars = append(pillars, PillarReadiness{
			Pillar:         a.Pillar,
			Readiness:      round2(readiness),
			Coverage:       round2(coverage),
			AvgConfidence:  round2(avgRating),
			RevisionHealth: round2(revHealth),
		})
		weightedSum += a.Weight * readiness
		totalWeight += a.Weight
	}

	overall := 0.0
	if totalWeight > 0 {
		overall = weightedSum / totalWeight
	}

	// Today counts.
	today := TodaySummary{Date: s.today()}
	if day, derr := s.repo.GetPlanDay(ctx, userID, s.today()); derr == nil {
		var estMin, remainMin int
		for _, t := range day.Tasks {
			today.TotalTasks++
			estMin += t.EstimatedMinutes
			if t.Status == "completed" {
				today.CompletedTasks++
			} else if t.Status != "skipped" && t.Status != "rescheduled" {
				remainMin += t.EstimatedMinutes
			}
		}
		today.EstimatedHours = round2(float64(estMin) / 60)
		today.RemainingHours = round2(float64(remainMin) / 60)
	} else if derr != ErrPlanDayNotFound {
		return nil, derr
	}

	return &Dashboard{
		OverallReadiness:       round2(overall),
		EstimatedReadinessDate: estimatedReadinessDate(overall, readinessThreshold),
		PillarReadiness:        pillars,
		Streak:                 streak,
		Today:                  today,
		RevisionDueCount:       revDue,
		GeneratedAt:            now,
	}, nil
}

// estimatedReadinessDate is null until enough trend data exists. At GA there is
// no readiness-snapshot history to derive a daily gain rate from, so the
// estimate is deliberately undefined (returns nil) unless already at/above the
// threshold, in which case the user is ready today. This satisfies the contract
// ("null until enough data") without fabricating a projection.
func estimatedReadinessDate(overall, threshold float64) *time.Time {
	if overall >= threshold {
		now := time.Now().UTC()
		d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return &d
	}
	return nil
}

func (s *Service) today() time.Time {
	n := s.now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
