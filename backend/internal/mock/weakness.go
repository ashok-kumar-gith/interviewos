package mock

import (
	"context"
	"math"
	"sort"
	"strings"
)

// WeaknessItem is a single ranked weakness area aggregated from findings.
type WeaknessItem struct {
	// Area is the grouping key: the finding category (normalized). Categories
	// like "communication", "correctness", "scalability" come from findings.
	Area string `json:"area"`
	// Pillar is the most common pillar_type associated with this area, if any.
	Pillar *Pillar `json:"pillar,omitempty"`
	// Count is the number of findings in this area.
	Count int `json:"count"`
	// Score is the aggregated severity-weighted score (higher = weaker).
	Score int `json:"score"`
	// MaxSeverity is the most severe finding seen in this area.
	MaxSeverity Severity `json:"max_severity"`
	// SeverityCounts breaks down findings by severity for this area.
	SeverityCounts map[Severity]int `json:"severity_counts"`
}

// WeaknessSummary is the ranked weakness report for a user.
type WeaknessSummary struct {
	// Items are weakness areas ranked most-to-least severe.
	Items []WeaknessItem `json:"items"`
	// TotalFindings is the number of findings considered.
	TotalFindings int `json:"total_findings"`
	// GeneratedBy names the detector implementation ("deterministic" today).
	GeneratedBy string `json:"generated_by"`
}

// WeaknessDetector aggregates a user's mock findings into a ranked weakness
// summary. The deterministic implementation satisfies it today; a
// Claude-API-backed implementation can be dropped in later without touching the
// service or handler.
type WeaknessDetector interface {
	Detect(ctx context.Context, findings []Finding) (*WeaknessSummary, error)
}

// DeterministicWeaknessDetector is a fully offline, pure-function detector.
// Given the same findings it always returns the same ranked summary, which
// makes it trivially testable and a safe default when no AI provider is set.
//
// Ranking: areas are grouped by normalized category. Each area accumulates a
// severity-weighted score (sum of Severity.Weight()). Areas are sorted by score
// desc, then count desc, then area name asc for stable, deterministic output.
type DeterministicWeaknessDetector struct{}

// NewDeterministicWeaknessDetector returns the deterministic detector.
func NewDeterministicWeaknessDetector() *DeterministicWeaknessDetector {
	return &DeterministicWeaknessDetector{}
}

type areaAgg struct {
	area        string
	count       int
	score       int
	maxSeverity Severity
	sevCounts   map[Severity]int
	pillarVotes map[Pillar]int
}

// Detect aggregates findings by category/severity and ranks the result. It
// never calls an external service and never errors.
func (DeterministicWeaknessDetector) Detect(_ context.Context, findings []Finding) (*WeaknessSummary, error) {
	aggs := map[string]*areaAgg{}

	for _, f := range findings {
		area := normalizeArea(f.Category)
		if area == "" {
			area = "uncategorized"
		}
		a := aggs[area]
		if a == nil {
			a = &areaAgg{
				area:        area,
				sevCounts:   map[Severity]int{},
				pillarVotes: map[Pillar]int{},
			}
			aggs[area] = a
		}
		a.count++
		a.score += f.Severity.Weight()
		a.sevCounts[f.Severity]++
		if f.Severity.Weight() > a.maxSeverity.Weight() {
			a.maxSeverity = f.Severity
		}
		if f.PillarType != nil {
			a.pillarVotes[*f.PillarType]++
		}
	}

	items := make([]WeaknessItem, 0, len(aggs))
	for _, a := range aggs {
		item := WeaknessItem{
			Area:           a.area,
			Count:          a.count,
			Score:          a.score,
			MaxSeverity:    a.maxSeverity,
			SeverityCounts: a.sevCounts,
			Pillar:         dominantPillar(a.pillarVotes),
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Area < items[j].Area
	})

	return &WeaknessSummary{
		Items:         items,
		TotalFindings: len(findings),
		GeneratedBy:   "deterministic",
	}, nil
}

// normalizeArea lower-cases and trims a category so equivalent categories group
// together.
func normalizeArea(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// dominantPillar returns the most-voted pillar, breaking ties by pillar name for
// determinism. Returns nil when there are no votes.
func dominantPillar(votes map[Pillar]int) *Pillar {
	if len(votes) == 0 {
		return nil
	}
	best := Pillar("")
	bestN := math.MinInt
	for p, n := range votes {
		if n > bestN || (n == bestN && string(p) < string(best)) {
			best = p
			bestN = n
		}
	}
	return &best
}
