package analytics

import (
	"math"
	"testing"
	"time"
)

const eps = 1e-9

func approx(t *testing.T, got, want float64, msg string) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

// TestCoverageGate verifies SRS §6.2.2: coverage gates the score — zero coverage
// yields zero readiness regardless of confidence/revision health.
func TestCoverageGate(t *testing.T) {
	w := DefaultReadinessWeights()
	p := ComputePillarReadiness(PillarInputs{
		Pillar:         "dsa",
		CompletedItems: 0,
		TotalItems:     10,
		AvgRating:      5, // max confidence
		RevHealth:      1, // perfect retention
	}, w)
	approx(t, p.Coverage, 0, "coverage")
	approx(t, p.Readiness, 0, "readiness must be 0 at 0 coverage")
}

// TestConfidenceNormalization verifies SRS §6.2.1: confidence_p = (rating-1)/4
// so 1★→0.0, 3★→0.5, 5★→1.0.
func TestConfidenceNormalization(t *testing.T) {
	w := DefaultReadinessWeights()
	cases := []struct {
		rating float64
		want   float64
	}{
		{1, 0.0},
		{2, 0.25},
		{3, 0.5},
		{4, 0.75},
		{5, 1.0},
	}
	for _, c := range cases {
		p := ComputePillarReadiness(PillarInputs{
			Pillar:         "dsa",
			CompletedItems: 1,
			TotalItems:     1,
			AvgRating:      c.rating,
			RevHealth:      0, // isolate the confidence term
		}, w)
		approx(t, p.Confidence, c.want, "confidence for rating")
	}
}

// TestPerPillarFormula verifies the exact SRS §6.2.2 multiplicative form:
// readiness_p = 100 · coverage_p · (0.6·confidence_p + 0.4·revhealth_p).
func TestPerPillarFormula(t *testing.T) {
	w := DefaultReadinessWeights() // 0.6 / 0.4
	// coverage = 4/8 = 0.5; rating 4 ⇒ confidence 0.75; revhealth 0.5.
	p := ComputePillarReadiness(PillarInputs{
		Pillar:         "dsa",
		CompletedItems: 4,
		TotalItems:     8,
		AvgRating:      4,
		RevHealth:      0.5,
	}, w)
	approx(t, p.Coverage, 0.5, "coverage")
	approx(t, p.Confidence, 0.75, "confidence")
	approx(t, p.RevisionHealth, 0.5, "revhealth")
	// 100 * 0.5 * (0.6*0.75 + 0.4*0.5) = 100 * 0.5 * (0.45 + 0.2) = 100*0.5*0.65 = 32.5
	approx(t, p.Readiness, 32.5, "readiness")
}

// TestMockBlend verifies SRS §6.2.2 mock blend: readiness_p ← (1-w_mock)·r +
// w_mock·(100·mock_score), with w_mock=0.2.
func TestMockBlend(t *testing.T) {
	w := DefaultReadinessWeights() // mock 0.2
	base := PillarInputs{
		Pillar:         "dsa",
		CompletedItems: 1,
		TotalItems:     1,
		AvgRating:      5, // confidence 1.0
		RevHealth:      1, // perfect
	}
	// Without mock: 100 * 1 * (0.6*1 + 0.4*1) = 100.
	noMock := ComputePillarReadiness(base, w)
	approx(t, noMock.Readiness, 100, "no-mock readiness")

	// With mock score 0.5: (1-0.2)*100 + 0.2*(100*0.5) = 80 + 10 = 90.
	base.HasMock = true
	base.MockScore = 0.5
	withMock := ComputePillarReadiness(base, w)
	approx(t, withMock.Readiness, 90, "blended readiness")
}

// TestOverallWeighting verifies SRS §6.2.3: Overall = Σ (W_p · readiness_p) with
// normalized weights summing to 1.
func TestOverallWeighting(t *testing.T) {
	w := DefaultReadinessWeights()
	inputs := []PillarInputs{
		// readiness 100 (coverage 1, conf 1, rev 1), weight 3.
		{Pillar: "dsa", Weight: 3, CompletedItems: 1, TotalItems: 1, AvgRating: 5, RevHealth: 1},
		// readiness 0 (coverage 0), weight 1.
		{Pillar: "lld", Weight: 1, CompletedItems: 0, TotalItems: 1, AvgRating: 5, RevHealth: 1},
	}
	res := ComputeReadiness(inputs, w)
	// Normalized weights: 0.75 and 0.25. Overall = 0.75*100 + 0.25*0 = 75.
	approx(t, res.Overall, 75, "overall weighted")
	approx(t, res.Pillars[0].Weight, 0.75, "normalized weight dsa")
	approx(t, res.Pillars[1].Weight, 0.25, "normalized weight lld")
}

// TestOverallUniformFallback verifies uniform weighting when no weights are set.
func TestOverallUniformFallback(t *testing.T) {
	w := DefaultReadinessWeights()
	inputs := []PillarInputs{
		{Pillar: "a", CompletedItems: 1, TotalItems: 1, AvgRating: 5, RevHealth: 1}, // 100
		{Pillar: "b", CompletedItems: 0, TotalItems: 1, AvgRating: 5, RevHealth: 1}, // 0
	}
	res := ComputeReadiness(inputs, w)
	// Both default to weight 1 ⇒ normalized 0.5 each ⇒ overall 50.
	approx(t, res.Overall, 50, "uniform overall")
}

// TestEstimatedReadyDate_AlreadyReady verifies SRS §6.2.4: at/above target ⇒ today.
func TestEstimatedReadyDate_AlreadyReady(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	got := EstimatedReadyDate(today, 85, 80, nil, 14)
	if got == nil {
		t.Fatal("expected today, got nil")
	}
	if !got.Equal(today) {
		t.Fatalf("expected %v, got %v", today, *got)
	}
}

// TestEstimatedReadyDate_RateNonPositive verifies SRS §6.2.4: rate<=0 ⇒ undefined.
func TestEstimatedReadyDate_RateNonPositive(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	// Flat history: rate 0.
	flat := []SnapshotPoint{
		{Date: today.AddDate(0, 0, -10), Overall: 40},
		{Date: today, Overall: 40},
	}
	if got := EstimatedReadyDate(today, 40, 80, flat, 14); got != nil {
		t.Fatalf("flat trend: expected nil, got %v", *got)
	}
	// Declining history: rate < 0.
	decl := []SnapshotPoint{
		{Date: today.AddDate(0, 0, -10), Overall: 50},
		{Date: today, Overall: 40},
	}
	if got := EstimatedReadyDate(today, 40, 80, decl, 14); got != nil {
		t.Fatalf("declining trend: expected nil, got %v", *got)
	}
	// Insufficient data (<2 in-window points): undefined.
	if got := EstimatedReadyDate(today, 40, 80, []SnapshotPoint{{Date: today, Overall: 40}}, 14); got != nil {
		t.Fatalf("single point: expected nil, got %v", *got)
	}
}

// TestEstimatedReadyDate_Projection verifies SRS §6.2.4 forward projection:
// days_remaining = ceil((target - R_today) / rate).
func TestEstimatedReadyDate_Projection(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	// Over 10 days readiness went 40→60 ⇒ rate = 2/day. Current 60, target 80 ⇒
	// (80-60)/2 = 10 days remaining ⇒ today+10.
	history := []SnapshotPoint{
		{Date: today.AddDate(0, 0, -10), Overall: 40},
		{Date: today, Overall: 60},
	}
	got := EstimatedReadyDate(today, 60, 80, history, 14)
	if got == nil {
		t.Fatal("expected a date, got nil")
	}
	want := today.AddDate(0, 0, 10)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, *got)
	}
}

// TestEstimatedReadyDate_WindowExcludesStale verifies the trailing window
// excludes snapshots older than windowDays.
func TestEstimatedReadyDate_WindowExcludesStale(t *testing.T) {
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	// A stale point 30 days ago is outside a 14-day window; only the in-window
	// pair (−10 → today: 50→70, rate 2) counts. (80-70)/2 = 5 days.
	history := []SnapshotPoint{
		{Date: today.AddDate(0, 0, -30), Overall: 10},
		{Date: today.AddDate(0, 0, -10), Overall: 50},
		{Date: today, Overall: 70},
	}
	got := EstimatedReadyDate(today, 70, 80, history, 14)
	if got == nil {
		t.Fatal("expected a date, got nil")
	}
	want := today.AddDate(0, 0, 5)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, *got)
	}
}

// TestRevisionHealthDefault verifies revhealth defaulting to 1.0 (the M2
// standalone guard) yields the expected depth/retention blend.
func TestRevisionHealthDefault(t *testing.T) {
	w := DefaultReadinessWeights()
	// coverage 1, confidence 0 (unrated→0 here), revhealth 1 ⇒ 100*1*(0.6*0+0.4*1)=40.
	p := ComputePillarReadiness(PillarInputs{
		Pillar:         "dsa",
		CompletedItems: 1,
		TotalItems:     1,
		AvgRating:      0,
		RevHealth:      1,
	}, w)
	approx(t, p.Readiness, 40, "readiness with default revhealth")
}

func TestRound2(t *testing.T) {
	if got := round2(32.555); math.Abs(got-32.56) > eps {
		t.Fatalf("round2(32.555)=%v", got)
	}
}
