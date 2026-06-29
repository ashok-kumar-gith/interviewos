package revision

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 9, 30, 0, 0, time.UTC)
}

func TestInitialState(t *testing.T) {
	now := day(2026, 6, 29)
	st := InitialState(now)
	if st.Stage != 0 {
		t.Fatalf("stage = %d, want 0", st.Stage)
	}
	if st.IntervalDays != 1 {
		t.Fatalf("interval = %d, want 1", st.IntervalDays)
	}
	if !st.IsActive {
		t.Fatal("expected active")
	}
	want := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if !st.DueAt.Equal(want) {
		t.Fatalf("due = %v, want %v (today+1)", st.DueAt, want)
	}
}

// TestLadderAdvance walks correct recalls up the ladder 1->3->7->15->30 and
// then graduates at the final stage.
func TestLadderAdvance(t *testing.T) {
	now := day(2026, 6, 29)
	today := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		fromStage    int
		wantStage    int
		wantInterval int
		wantOffset   int
		wantActive   bool
		wantGrad     bool
	}{
		{fromStage: 0, wantStage: 1, wantInterval: 3, wantOffset: 3, wantActive: true},
		{fromStage: 1, wantStage: 2, wantInterval: 7, wantOffset: 7, wantActive: true},
		{fromStage: 2, wantStage: 3, wantInterval: 15, wantOffset: 15, wantActive: true},
		{fromStage: 3, wantStage: 4, wantInterval: 30, wantOffset: 30, wantActive: true},
		{fromStage: 4, wantStage: 4, wantInterval: 30, wantOffset: 0, wantActive: false, wantGrad: true},
	}
	for _, c := range cases {
		tr := Apply(c.fromStage, RecallCorrect, now)
		if tr.Stage != c.wantStage {
			t.Fatalf("stage %d: got stage %d, want %d", c.fromStage, tr.Stage, c.wantStage)
		}
		if tr.IntervalDays != c.wantInterval {
			t.Fatalf("stage %d: got interval %d, want %d", c.fromStage, tr.IntervalDays, c.wantInterval)
		}
		if tr.IsActive != c.wantActive {
			t.Fatalf("stage %d: got active %v, want %v", c.fromStage, tr.IsActive, c.wantActive)
		}
		if tr.Graduated != c.wantGrad {
			t.Fatalf("stage %d: got graduated %v, want %v", c.fromStage, tr.Graduated, c.wantGrad)
		}
		if tr.LastRecall != RecallCorrect {
			t.Fatalf("stage %d: last recall = %v, want correct", c.fromStage, tr.LastRecall)
		}
		if tr.Lapsed {
			t.Fatalf("stage %d: correct recall should not lapse", c.fromStage)
		}
		if c.wantActive {
			want := today.AddDate(0, 0, c.wantOffset)
			if !tr.DueAt.Equal(want) {
				t.Fatalf("stage %d: due = %v, want %v", c.fromStage, tr.DueAt, want)
			}
		}
	}
}

// TestLadderResetOnIncorrect verifies an incorrect recall at any stage resets to
// stage 0 / interval 1 / due tomorrow and flags a lapse.
func TestLadderResetOnIncorrect(t *testing.T) {
	now := day(2026, 6, 29)
	wantDue := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	for stage := 0; stage <= 4; stage++ {
		tr := Apply(stage, RecallIncorrect, now)
		if tr.Stage != 0 {
			t.Fatalf("stage %d: reset stage = %d, want 0", stage, tr.Stage)
		}
		if tr.IntervalDays != 1 {
			t.Fatalf("stage %d: reset interval = %d, want 1", stage, tr.IntervalDays)
		}
		if !tr.IsActive {
			t.Fatalf("stage %d: incorrect recall must keep item active", stage)
		}
		if tr.Graduated {
			t.Fatalf("stage %d: incorrect must not graduate", stage)
		}
		if !tr.Lapsed {
			t.Fatalf("stage %d: incorrect must flag a lapse", stage)
		}
		if tr.LastRecall != RecallIncorrect {
			t.Fatalf("stage %d: last recall = %v, want incorrect", stage, tr.LastRecall)
		}
		if !tr.DueAt.Equal(wantDue) {
			t.Fatalf("stage %d: due = %v, want %v", stage, tr.DueAt, wantDue)
		}
	}
}

func TestIntervalForStage(t *testing.T) {
	for stage, want := range Intervals {
		if got := IntervalForStage(stage); got != want {
			t.Fatalf("stage %d: interval = %d, want %d", stage, got, want)
		}
	}
	if got := IntervalForStage(-1); got != 1 {
		t.Fatalf("clamp low: got %d, want 1", got)
	}
	if got := IntervalForStage(99); got != 30 {
		t.Fatalf("clamp high: got %d, want 30", got)
	}
}
