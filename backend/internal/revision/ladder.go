package revision

import "time"

// Intervals is the fixed interval ladder (days) at GA per docs/02-SRS.md §6.1
// and ADR D1 (PRD OQ1). stage indexes into this slice.
var Intervals = []int{1, 3, 7, 15, 30}

// lastStage is the final ladder index (stage at the 30-day interval).
func lastStage() int { return len(Intervals) - 1 }

// IntervalForStage returns the interval (days) for a ladder stage, clamping
// out-of-range stages to the nearest valid bound.
func IntervalForStage(stage int) int {
	if stage < 0 {
		stage = 0
	}
	if stage > lastStage() {
		stage = lastStage()
	}
	return Intervals[stage]
}

// dayFloor truncates t to midnight UTC (the local "day" granularity used for
// due_at, which is a DATE column).
func dayFloor(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// NewItemState is the initial spaced-repetition state for a freshly scheduled
// learning item (FR-REV-001): stage 0, interval 1 day, due tomorrow, active,
// no recall yet.
type NewItemState struct {
	Stage        int
	IntervalDays int
	DueAt        time.Time
	IsActive     bool
}

// InitialState computes the creation-time state for an item completed at now.
// due_at = local_day(now) + 1 day.
func InitialState(now time.Time) NewItemState {
	return NewItemState{
		Stage:        0,
		IntervalDays: Intervals[0],
		DueAt:        dayFloor(now).AddDate(0, 0, Intervals[0]),
		IsActive:     true,
	}
}

// Transition is the result of applying a recall to an item: the new ladder state
// plus the counters to bump. It is a pure value computed by Apply; persistence
// is the caller's concern.
type Transition struct {
	Stage        int
	IntervalDays int
	DueAt        time.Time
	IsActive     bool
	LastRecall   RecallResult
	Graduated    bool // true when this recall graduated the item
	Lapsed       bool // true when this recall was an incorrect (lapse) recall
}

// Apply computes the next state for an item given a recall result and the
// current local time, implementing docs/02-SRS.md §6.1 exactly:
//
//	correct  at last stage -> graduate (is_active=false), no further due_at change semantics
//	correct  otherwise     -> stage+1, interval=Intervals[stage+1], due_at=today+interval
//	incorrect              -> stage=0, interval=1, due_at=today+1, lapse
//
// stage is the item's current stage; recall must be valid (validated upstream).
func Apply(stage int, recall RecallResult, now time.Time) Transition {
	today := dayFloor(now)
	if recall == RecallCorrect {
		if stage >= lastStage() {
			return Transition{
				Stage:        stage,
				IntervalDays: IntervalForStage(stage),
				DueAt:        today, // unchanged in effect; item is inactive
				IsActive:     false,
				LastRecall:   RecallCorrect,
				Graduated:    true,
			}
		}
		next := stage + 1
		interval := Intervals[next]
		return Transition{
			Stage:        next,
			IntervalDays: interval,
			DueAt:        today.AddDate(0, 0, interval),
			IsActive:     true,
			LastRecall:   RecallCorrect,
		}
	}
	// incorrect: reset to stage 0, due tomorrow, count a lapse.
	return Transition{
		Stage:        0,
		IntervalDays: Intervals[0],
		DueAt:        today.AddDate(0, 0, Intervals[0]),
		IsActive:     true,
		LastRecall:   RecallIncorrect,
		Lapsed:       true,
	}
}
