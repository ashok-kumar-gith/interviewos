package analytics

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Service implements the Analytics Engine use-cases: live readiness (SRS §6.2),
// daily snapshot recording, snapshot history, streak, weak/strong topics, and
// time-spent. It orchestrates the Repository and the pure readiness calculator
// so analytics and the dashboard agree on the SRS formula exactly.
type Service struct {
	repo    Repository
	weights ReadinessWeights
	target  float64
	window  int
	now     func() time.Time
}

// ServiceConfig configures a Service. Weights/target/window default to the SRS
// values when zero-valued (FR-ANALYTICS-011 makes them server-configurable).
type ServiceConfig struct {
	Repo    Repository
	Weights *ReadinessWeights
	Target  float64
	Window  int
	Now     func() time.Time
}

// NewService constructs a Service with SRS defaults applied for any unset field.
func NewService(cfg ServiceConfig) *Service {
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	w := DefaultReadinessWeights()
	if cfg.Weights != nil {
		w = *cfg.Weights
	}
	target := cfg.Target
	if target <= 0 {
		target = DefaultReadyTarget
	}
	window := cfg.Window
	if window <= 0 {
		window = DefaultTrendWindowDays
	}
	return &Service{repo: cfg.Repo, weights: w, target: target, window: window, now: nowFn}
}

// Readiness computes the user's current overall + per-pillar readiness and the
// estimated interview-readiness date (SRS §6.2). It never persists — it always
// reflects current data. The estimated date is projected from snapshot history.
func (s *Service) Readiness(ctx context.Context, userID uuid.UUID) (*Readiness, error) {
	inputs, err := s.repo.PillarInputs(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := ComputeReadiness(inputs, s.weights)

	// History for the estimated-ready-date projection (trailing window).
	history, err := s.snapshotHistory(ctx, userID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	est := EstimatedReadyDate(today, result.Overall, s.target, history, s.window)

	out := &Readiness{
		SnapshotDate:       today,
		OverallReadiness:   round2(result.Overall),
		PillarReadiness:    map[string]float64{},
		EstimatedReadyDate: est,
		Pillars:            result.Pillars,
	}

	// Roll up explainable aggregates (completion %, avg confidence, revision
	// health) across pillars for the snapshot/response.
	var covSum, confSum, revSum float64
	var confCount, revCount int
	for i := range result.Pillars {
		p := &result.Pillars[i]
		p.Readiness = round2(p.Readiness)
		p.Coverage = round2(p.Coverage)
		p.Confidence = round2(p.Confidence)
		p.RevisionHealth = round2(p.RevisionHealth)
		p.Weight = round2(p.Weight)
		out.PillarReadiness[p.Pillar] = p.Readiness
		covSum += p.Coverage
		if p.AvgRating > 0 {
			confSum += p.AvgRating
			confCount++
		}
		revSum += p.RevisionHealth
		revCount++
	}
	if n := len(result.Pillars); n > 0 {
		out.CompletionPct = round2(covSum / float64(n) * 100)
	}
	if confCount > 0 {
		v := round2(confSum / float64(confCount))
		out.AvgConfidence = &v
	}
	if revCount > 0 {
		v := round2(revSum / float64(revCount) * 100)
		out.RevisionHealth = &v
	}
	out.Pillars = result.Pillars

	// Weak/strong topic ids for the snapshot payload.
	weak, strong, err := s.weakStrongTopicIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	out.WeakTopics = weak
	out.StrongTopics = strong
	return out, nil
}

// RecordSnapshot computes today's readiness and upserts the daily snapshot
// idempotently (FR-ANALYTICS-008, NFR-REL-006). Returns the stored snapshot.
func (s *Service) RecordSnapshot(ctx context.Context, userID uuid.UUID) (*Snapshot, error) {
	r, err := s.Readiness(ctx, userID)
	if err != nil {
		return nil, err
	}
	roadmapID, err := s.repo.ActiveRoadmapID(ctx, userID)
	if err != nil {
		return nil, err
	}
	snap := Snapshot{
		UserID:             userID,
		RoadmapID:          roadmapID,
		SnapshotDate:       s.today(),
		OverallReadiness:   r.OverallReadiness,
		PillarReadiness:    r.PillarReadiness,
		CompletionPct:      r.CompletionPct,
		AvgConfidence:      r.AvgConfidence,
		RevisionHealth:     r.RevisionHealth,
		EstimatedReadyDate: r.EstimatedReadyDate,
		WeakTopics:         r.WeakTopics,
		StrongTopics:       r.StrongTopics,
	}
	stored, err := s.repo.UpsertSnapshot(ctx, snap)
	if err != nil {
		return nil, err
	}
	return &stored, nil
}

// Snapshots returns the user's persisted readiness snapshots over [from,to]
// (zero = unbounded) with paging, plus the total count.
func (s *Service) Snapshots(ctx context.Context, userID uuid.UUID, from, to time.Time, limit, offset int) ([]Snapshot, int64, error) {
	return s.repo.ListSnapshots(ctx, userID, from, to, limit, offset)
}

// Streak returns the user's current/longest streak and per-day activity over
// [from,to] (SRS §6.2.5). The current streak counts consecutive local days
// ending today (or yesterday if today has no completion yet).
func (s *Service) Streak(ctx context.Context, userID uuid.UUID, from, to time.Time) (*Streak, error) {
	days, err := s.repo.StreakDays(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	// Longest/current streak must be computed over the user's full history, not
	// just the windowed view; fetch unbounded dates for the streak counters.
	all := days
	if !from.IsZero() || !to.IsZero() {
		all, err = s.repo.StreakDays(ctx, userID, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
	}
	current, longest := computeStreak(all, s.today())
	return &Streak{Current: current, Longest: longest, Days: days}, nil
}

// Topics returns weak/strong topic analytics (SRS FR-ANALYTICS-004). Topics are
// ranked by composite score (coverage × confidence); the lowest-scoring
// completed-or-in-scope topics are "weak", the highest "strong".
func (s *Service) Topics(ctx context.Context, userID uuid.UUID) (*TopicAnalytics, error) {
	entries, err := s.repo.TopicEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	weak, strong := bucketTopics(entries)
	return &TopicAnalytics{Weak: weak, Strong: strong}, nil
}

// TimeSpent returns the time-spent aggregation over [from,to] grouped by day or
// pillar (SRS FR-ANALYTICS-005).
func (s *Service) TimeSpent(ctx context.Context, userID uuid.UUID, from, to time.Time, groupBy string) (*TimeSpent, error) {
	ts, err := s.repo.TimeSpent(ctx, userID, from, to, groupBy)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

// weakStrongTopicIDs returns just the ids for the snapshot payload.
func (s *Service) weakStrongTopicIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	entries, err := s.repo.TopicEntries(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	weak, strong := bucketTopics(entries)
	weakIDs := make([]uuid.UUID, 0, len(weak))
	for _, e := range weak {
		weakIDs = append(weakIDs, e.TopicID)
	}
	strongIDs := make([]uuid.UUID, 0, len(strong))
	for _, e := range strong {
		strongIDs = append(strongIDs, e.TopicID)
	}
	return weakIDs, strongIDs, nil
}

// snapshotHistory loads all persisted snapshots for the estimated-date slope.
func (s *Service) snapshotHistory(ctx context.Context, userID uuid.UUID) ([]SnapshotPoint, error) {
	snaps, _, err := s.repo.ListSnapshots(ctx, userID, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	pts := make([]SnapshotPoint, 0, len(snaps))
	for _, sn := range snaps {
		pts = append(pts, SnapshotPoint{Date: sn.SnapshotDate, Overall: sn.OverallReadiness})
	}
	return pts, nil
}

func (s *Service) today() time.Time {
	n := s.now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// topTopicBucket caps weak/strong lists so the response stays focused.
const topTopicBucket = 5

// bucketTopics ranks topics by composite score and splits into weak (lowest) and
// strong (highest), each capped at topTopicBucket entries. Topics with no
// coverage are eligible to be weak; fully-mastered topics are eligible to be
// strong. Only in-scope topics (returned by the repository) are considered.
func bucketTopics(entries []TopicEntry) (weak, strong []TopicEntry) {
	if len(entries) == 0 {
		return nil, nil
	}
	sorted := make([]TopicEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Score != sorted[j].Score {
			return sorted[i].Score < sorted[j].Score
		}
		return sorted[i].TopicName < sorted[j].TopicName
	})

	weakN := min(topTopicBucket, len(sorted))
	weak = append(weak, sorted[:weakN]...)

	strongN := min(topTopicBucket, len(sorted))
	for i := len(sorted) - 1; i >= len(sorted)-strongN; i-- {
		strong = append(strong, sorted[i])
	}
	return weak, strong
}

// computeStreak derives current+longest consecutive-day streaks from streak
// days. Mirrors progress.computeStreakFromDates so the streak counters agree.
func computeStreak(days []StreakDay, asOf time.Time) (current, longest int) {
	if len(days) == 0 {
		return 0, 0
	}
	seen := map[string]bool{}
	norm := make([]time.Time, 0, len(days))
	for _, d := range days {
		day := time.Date(d.Date.Year(), d.Date.Month(), d.Date.Day(), 0, 0, 0, 0, time.UTC)
		k := day.Format("2006-01-02")
		if seen[k] {
			continue
		}
		seen[k] = true
		norm = append(norm, day)
	}
	sort.Slice(norm, func(i, j int) bool { return norm[i].Before(norm[j]) })

	longest, run := 1, 1
	for i := 1; i < len(norm); i++ {
		if norm[i].Sub(norm[i-1]) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
	}

	today := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)
	var anchor time.Time
	switch {
	case seen[today.Format("2006-01-02")]:
		anchor = today
	case seen[today.AddDate(0, 0, -1).Format("2006-01-02")]:
		anchor = today.AddDate(0, 0, -1)
	default:
		return 0, longest
	}
	for d := anchor; seen[d.Format("2006-01-02")]; d = d.AddDate(0, 0, -1) {
		current++
	}
	return current, longest
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
