package analytics

import (
	"math"
	"sort"
	"time"
)

// ReadinessWeights are the server-configurable weights of the readiness model
// (SRS §6.2; FR-ANALYTICS-011, NFR-MAINT-005). All are pure inputs to the
// calculator so the formula is fully unit-testable and deterministic.
type ReadinessWeights struct {
	// Conf and Rev split the depth/retention blend; Conf + Rev = 1.
	Conf float64
	Rev  float64
	// Mock is the blend weight applied once a mock exists for the pillar
	// (SRS §6.2.2): readiness_p ← (1-Mock)·readiness_p + Mock·(100·mock_score_p).
	Mock float64
}

// DefaultReadinessWeights are the SRS §6.2 defaults: w_conf=0.6, w_rev=0.4,
// w_mock=0.2 (applied only when a mock exists).
func DefaultReadinessWeights() ReadinessWeights {
	return ReadinessWeights{Conf: 0.6, Rev: 0.4, Mock: 0.2}
}

// PillarInputs is the raw per-pillar rollup the calculator turns into readiness.
// It mirrors SRS §6.2.1/6.2.2: coverage breadth, mean confidence over completed
// items, mean revision health over completed items, and the (optional) mock
// score. The repository produces these from the user's progress data.
type PillarInputs struct {
	Pillar string
	// Weight is the pillar's relative weight (company weight when a company is
	// targeted, else the pillar's content weight). Normalized across pillars by
	// ComputeReadiness.
	Weight float64

	// CompletedItems is Σ cov_i (items completed, skipped excluded).
	CompletedItems int
	// TotalItems is N_p (in-scope items, skipped excluded).
	TotalItems int

	// AvgRating is the mean 1..5 confidence over completed items, or 0 when no
	// completed item carries a rating. Normalized as (rating-1)/4 per SRS §6.2.1.
	AvgRating float64
	// RevHealth is the mean revision health over completed items, in [0,1]
	// (SRS §6.2.1). Defaults to 1.0 when revision data is absent (per the M2
	// standalone guard).
	RevHealth float64

	// HasMock indicates a mock exists for this pillar; MockScore is in [0,1].
	HasMock   bool
	MockScore float64
}

// PillarReadiness is the explainable per-pillar readiness breakdown
// (SRS §6.2: exposes coverage_p, confidence_p, revhealth_p, and W_p).
type PillarReadiness struct {
	Pillar         string
	Readiness      float64 // 0..100
	Coverage       float64 // coverage_p, 0..1
	Confidence     float64 // confidence_p, 0..1
	RevisionHealth float64 // revhealth_p, 0..1
	Weight         float64 // normalized W_p, Σ = 1
	AvgRating      float64 // mean 1..5 rating (display)
}

// ReadinessResult is the aggregated readiness computation.
type ReadinessResult struct {
	Overall float64 // 0..100
	Pillars []PillarReadiness
}

// ComputePillarReadiness implements the per-pillar readiness of SRS §6.2.2 for a
// single pillar. It is the normative formula and is the unit under test.
//
//	coverage_p   = completed / total                       (breadth; gates score)
//	confidence_p = (avg_rating - 1) / 4                     (depth)
//	revhealth_p  = mean revision health over completed      (retention)
//	readiness_p  = 100 · coverage_p · (w_conf·confidence_p + w_rev·revhealth_p)
//	mock blend   = (1-w_mock)·readiness_p + w_mock·(100·mock_score_p)  if a mock exists
func ComputePillarReadiness(in PillarInputs, w ReadinessWeights) PillarReadiness {
	coverage := 0.0
	if in.TotalItems > 0 {
		coverage = float64(in.CompletedItems) / float64(in.TotalItems)
	}

	// confidence_p = (avg_rating - 1) / 4, clamped to [0,1]. 0 rating ⇒ 0.
	confidence := 0.0
	if in.AvgRating > 0 {
		confidence = clamp01((in.AvgRating - 1) / 4)
	}

	revHealth := clamp01(in.RevHealth)

	readiness := 100 * coverage * (w.Conf*confidence + w.Rev*revHealth)

	if in.HasMock {
		readiness = (1-w.Mock)*readiness + w.Mock*(100*clamp01(in.MockScore))
	}

	return PillarReadiness{
		Pillar:         in.Pillar,
		Readiness:      readiness,
		Coverage:       coverage,
		Confidence:     confidence,
		RevisionHealth: revHealth,
		Weight:         in.Weight,
		AvgRating:      in.AvgRating,
	}
}

// ComputeReadiness implements SRS §6.2.2–6.2.3: per-pillar readiness plus the
// pillar-weighted overall. Pillar weights are normalized so Σ W_p = 1
// (SRS §6.2.3); a non-positive total weight falls back to a uniform split.
func ComputeReadiness(inputs []PillarInputs, w ReadinessWeights) ReadinessResult {
	pillars := make([]PillarReadiness, 0, len(inputs))
	var totalWeight float64
	for _, in := range inputs {
		if in.Weight <= 0 {
			in.Weight = 1.0
		}
		totalWeight += in.Weight
		pillars = append(pillars, ComputePillarReadiness(in, w))
	}

	var overall float64
	if totalWeight > 0 {
		for i := range pillars {
			norm := pillars[i].Weight / totalWeight
			pillars[i].Weight = norm
			overall += norm * pillars[i].Readiness
		}
	} else if n := len(pillars); n > 0 {
		uniform := 1.0 / float64(n)
		for i := range pillars {
			pillars[i].Weight = uniform
			overall += uniform * pillars[i].Readiness
		}
	}

	return ReadinessResult{Overall: overall, Pillars: pillars}
}

// SnapshotPoint is one historical readiness observation used to project the
// estimated-ready date (the trailing-window slope source, SRS §6.2.4).
type SnapshotPoint struct {
	Date    time.Time
	Overall float64
}

const (
	// DefaultReadyTarget is the target overall readiness threshold (SRS §6.2.4).
	DefaultReadyTarget = 80.0
	// DefaultTrendWindowDays is the trailing window over which the daily
	// readiness-gain rate is measured (SRS §6.2.4).
	DefaultTrendWindowDays = 14
)

// EstimatedReadyDate implements SRS §6.2.4. Given current overall readiness and
// the snapshot history, it projects the date the user is expected to reach the
// target threshold:
//
//	R_today >= target        ⇒ today ("interview-ready")
//	rate <= 0                ⇒ nil   (insufficient momentum / undefined)
//	else                     ⇒ today + ceil((target - R_today) / rate)
//
// rate is the mean daily readiness gain over the trailing windowDays of history
// (the slope between the earliest and latest in-window snapshots, per day).
func EstimatedReadyDate(today time.Time, current, target float64, history []SnapshotPoint, windowDays int) *time.Time {
	today = truncateDay(today)
	if current >= target {
		t := today
		return &t
	}

	rate := dailyGainRate(today, history, windowDays)
	if rate <= 0 {
		return nil
	}

	daysRemaining := int(math.Ceil((target - current) / rate))
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	d := today.AddDate(0, 0, daysRemaining)
	return &d
}

// dailyGainRate computes the mean daily readiness gain over the trailing
// windowDays, as the slope between the first and last in-window snapshots
// divided by the number of days between them. Fewer than two in-window points
// ⇒ 0 (undefined momentum).
func dailyGainRate(today time.Time, history []SnapshotPoint, windowDays int) float64 {
	if windowDays <= 0 || len(history) < 2 {
		return 0
	}
	cutoff := today.AddDate(0, 0, -windowDays)

	pts := make([]SnapshotPoint, 0, len(history))
	for _, p := range history {
		d := truncateDay(p.Date)
		if d.Before(cutoff) || d.After(today) {
			continue
		}
		pts = append(pts, SnapshotPoint{Date: d, Overall: p.Overall})
	}
	if len(pts) < 2 {
		return 0
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Date.Before(pts[j].Date) })

	first, last := pts[0], pts[len(pts)-1]
	days := last.Date.Sub(first.Date).Hours() / 24
	if days <= 0 {
		return 0
	}
	return (last.Overall - first.Overall) / days
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func truncateDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// round2 rounds to two decimals (display precision shared across the module).
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
