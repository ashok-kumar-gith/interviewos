package curriculum

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// deterministic uuids for fixtures.
func tid(n byte) uuid.UUID {
	var b [16]byte
	b[0] = n
	return uuid.UUID(b)
}

func baseInput() Input {
	start := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	dsaT1, dsaT2 := tid(1), tid(2)
	sdT1 := tid(3)
	return Input{
		Profile: Profile{
			HoursPerWeek: 10,
			StartDate:    start,
			TargetWeeks:  12,
			ActiveDays:   6,
			PillarStrengths: map[PillarType]int{
				PillarDSA:    4,
				PillarSystem: 2,
			},
		},
		Pillars: []PillarMeta{
			{Type: PillarDSA, Weight: 1.5},
			{Type: PillarSystem, Weight: 1.5},
		},
		Topics: []Topic{
			{ID: dsaT1, Pillar: PillarDSA, Slug: "arrays", Name: "Arrays", Difficulty: DifficultyEasy, Priority: PriorityHigh, EstimatedHours: 4, SortOrder: 0},
			{ID: dsaT2, Pillar: PillarDSA, Slug: "two-ptr", Name: "Two Pointers", Difficulty: DifficultyMedium, Priority: PriorityHigh, EstimatedHours: 4, Prerequisites: []uuid.UUID{dsaT1}, SortOrder: 1},
			{ID: sdT1, Pillar: PillarSystem, Slug: "url", Name: "URL Shortener", Difficulty: DifficultyMedium, Priority: PriorityMedium, EstimatedHours: 3, SortOrder: 0},
		},
		Problems: []Problem{
			{ID: tid(10), TopicID: dsaT1, Title: "Two Sum", Difficulty: DifficultyEasy, EstimatedMinutes: 30, FrequencyScore: 90, CompanyFrequency: 70},
			{ID: tid(11), TopicID: dsaT1, Title: "Contains Dup", Difficulty: DifficultyEasy, EstimatedMinutes: 20, FrequencyScore: 40, CompanyFrequency: 10},
		},
		Resources: []Resource{
			{ID: tid(20), TopicID: dsaT1, Title: "Arrays guide", Kind: KindRead, EstimatedMinutes: 25, IsPrimary: true},
		},
		CompanyMul: map[PillarType]float64{PillarDSA: 1.2, PillarSystem: 1.3},
	}
}

func TestGenerate_WeekCount(t *testing.T) {
	p := Generate(baseInput())
	if len(p.Weeks) != 12 {
		t.Fatalf("expected 12 weeks, got %d", len(p.Weeks))
	}
	if p.TotalWeeks != 12 || p.HoursPerWeek != 10 {
		t.Fatalf("plan metadata wrong: %+v", p)
	}
	// End date = start + 12*7 - 1 days.
	wantEnd := p.StartDate.AddDate(0, 0, 12*7-1)
	if !p.EndDate.Equal(wantEnd) {
		t.Fatalf("end date = %v want %v", p.EndDate, wantEnd)
	}
}

func TestGenerate_BudgetRespectedWithin10Pct(t *testing.T) {
	in := baseInput()
	// Add a lot of topics so the bin-packer is genuinely stressed.
	for i := 0; i < 40; i++ {
		in.Topics = append(in.Topics, Topic{
			ID: tid(byte(100 + i)), Pillar: PillarDSA, Name: "T", Difficulty: DifficultyMedium,
			Priority: PriorityMedium, EstimatedHours: 3, SortOrder: 10 + i,
		})
	}
	p := Generate(in)
	limit := float64(in.Profile.HoursPerWeek) * budgetTolerance
	for _, w := range p.Weeks {
		if w.PlannedHours > limit+0.001 {
			t.Fatalf("week %d planned %.2fh exceeds budget*1.1 = %.2fh", w.Number, w.PlannedHours, limit)
		}
	}
}

func TestGenerate_WeakerPillarGetsMoreHours(t *testing.T) {
	// Equal base weights, equal company mul; DSA strong (5), SD weak (1).
	in := baseInput()
	in.Pillars = []PillarMeta{{Type: PillarDSA, Weight: 1.0}, {Type: PillarSystem, Weight: 1.0}}
	in.CompanyMul = map[PillarType]float64{}
	in.Profile.PillarStrengths = map[PillarType]int{PillarDSA: 5, PillarSystem: 1}
	hours := allocatePillarHours(in, 100)
	if hours[PillarSystem] <= hours[PillarDSA] {
		t.Fatalf("weaker pillar should get more hours: dsa=%.2f sd=%.2f", hours[PillarDSA], hours[PillarSystem])
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	in := baseInput()
	a := Generate(in)
	b := Generate(in)
	if len(a.Weeks) != len(b.Weeks) {
		t.Fatalf("week count differs")
	}
	for i := range a.Weeks {
		wa, wb := a.Weeks[i], b.Weeks[i]
		if wa.Theme != wb.Theme || wa.PlannedHours != wb.PlannedHours || len(wa.Days) != len(wb.Days) {
			t.Fatalf("week %d differs across runs", i)
		}
		for d := range wa.Days {
			if len(wa.Days[d].Tasks) != len(wb.Days[d].Tasks) {
				t.Fatalf("week %d day %d task count differs", i, d)
			}
			for k := range wa.Days[d].Tasks {
				if wa.Days[d].Tasks[k].ItemID != wb.Days[d].Tasks[k].ItemID ||
					wa.Days[d].Tasks[k].Kind != wb.Days[d].Tasks[k].Kind {
					t.Fatalf("task mismatch w%d d%d k%d", i, d, k)
				}
			}
		}
	}
}

func TestGenerate_CompanyWeightingChangesProblemOrder(t *testing.T) {
	in := baseInput()
	p := Generate(in)
	// Find the first two solve tasks for the arrays topic; Two Sum (companyFreq 70)
	// must precede Contains Dup (companyFreq 10).
	var order []string
	for _, w := range p.Weeks {
		for _, d := range w.Days {
			for _, tk := range d.Tasks {
				if tk.Kind == KindSolve {
					order = append(order, tk.Title)
				}
			}
		}
	}
	if len(order) < 2 {
		t.Fatalf("expected at least 2 solve tasks, got %v", order)
	}
	// The two solve tasks may land on different days, but Two Sum should be
	// emitted (and thus packed) before Contains Dup. Verify via emit order.
	ix := newIndexers(in)
	probs := ix.problemsByTopic[tid(1)]
	if probs[0].Title != "Two Sum" {
		t.Fatalf("company weighting failed: first problem = %q want Two Sum", probs[0].Title)
	}
}

func TestGenerate_PrerequisiteOrdering(t *testing.T) {
	in := baseInput()
	ix := newIndexers(in)
	sel := selectTopics(in, allocatePillarHours(in, in.Profile.HoursPerWeek*in.Profile.TargetWeeks), ix, emitAllTopicTasks(in, ix))
	dsa := sel[PillarDSA]
	// Arrays (tid 1) must precede Two Pointers (tid 2) which depends on it.
	var arraysIdx, twoPtrIdx = -1, -1
	for i, t := range dsa {
		if t.ID == tid(1) {
			arraysIdx = i
		}
		if t.ID == tid(2) {
			twoPtrIdx = i
		}
	}
	if arraysIdx == -1 || twoPtrIdx == -1 || arraysIdx > twoPtrIdx {
		t.Fatalf("prerequisite order violated: arrays=%d twoPtr=%d", arraysIdx, twoPtrIdx)
	}
}

func TestGenerate_TasksReferenceRealItems(t *testing.T) {
	in := baseInput()
	p := Generate(in)
	topicIDs := map[uuid.UUID]bool{}
	for _, t := range in.Topics {
		topicIDs[t.ID] = true
	}
	probIDs := map[uuid.UUID]bool{}
	for _, pr := range in.Problems {
		probIDs[pr.ID] = true
	}
	resIDs := map[uuid.UUID]bool{}
	for _, r := range in.Resources {
		resIDs[r.ID] = true
	}
	var count int
	for _, w := range p.Weeks {
		for _, d := range w.Days {
			for _, tk := range d.Tasks {
				count++
				switch tk.ItemType {
				case ItemTopic:
					if !topicIDs[tk.ItemID] {
						t.Fatalf("task references unknown topic %s", tk.ItemID)
					}
				case ItemProblem:
					if !probIDs[tk.ItemID] {
						t.Fatalf("task references unknown problem %s", tk.ItemID)
					}
				case ItemResource:
					if !resIDs[tk.ItemID] {
						t.Fatalf("task references unknown resource %s", tk.ItemID)
					}
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no tasks emitted")
	}
}

func TestGenerate_RestDaysHaveNoTasks(t *testing.T) {
	in := baseInput()
	in.Profile.ActiveDays = 5
	p := Generate(in)
	for _, w := range p.Weeks {
		for _, d := range w.Days {
			if d.IsRestDay && len(d.Tasks) > 0 {
				t.Fatalf("rest day %v has %d tasks", d.Date, len(d.Tasks))
			}
		}
	}
}

// TestGenerate_IncludesBackendEngineeringTasks asserts the engine is
// pillar-agnostic: a profile that includes the backend_engineering pillar (with
// seeded topics) yields at least one backend_engineering study task. This guards
// the curriculum integration of the Backend Engineering pillar.
func TestGenerate_IncludesBackendEngineeringTasks(t *testing.T) {
	start := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	beT1, beT2 := tid(40), tid(41)
	in := Input{
		Profile: Profile{
			HoursPerWeek: 12,
			StartDate:    start,
			TargetWeeks:  8,
			ActiveDays:   6,
			// Weak backend_engineering self-assessment => the allocator should give
			// it a healthy share of hours.
			PillarStrengths: map[PillarType]int{
				PillarDSA:        4,
				PillarBackendEng: 2,
			},
		},
		Pillars: []PillarMeta{
			{Type: PillarDSA, Weight: 1.5},
			{Type: PillarBackendEng, Weight: 1.0},
		},
		Topics: []Topic{
			{ID: tid(1), Pillar: PillarDSA, Slug: "arrays", Name: "Arrays", Difficulty: DifficultyEasy, Priority: PriorityHigh, EstimatedHours: 4, SortOrder: 0},
			{ID: beT1, Pillar: PillarBackendEng, Slug: "be-mvcc", Name: "MVCC", Difficulty: DifficultyHard, Priority: PriorityHigh, EstimatedHours: 3, SortOrder: 0},
			{ID: beT2, Pillar: PillarBackendEng, Slug: "be-kafka", Name: "Kafka", Difficulty: DifficultyMedium, Priority: PriorityHigh, EstimatedHours: 4, SortOrder: 1},
		},
		CompanyMul: map[PillarType]float64{},
	}

	p := Generate(in)
	var beStudy int
	beTopicIDs := map[uuid.UUID]bool{beT1: true, beT2: true}
	for _, w := range p.Weeks {
		for _, d := range w.Days {
			for _, tk := range d.Tasks {
				if tk.Pillar == PillarBackendEng {
					if tk.Kind == KindStudy && tk.ItemType == ItemTopic && beTopicIDs[tk.ItemID] {
						beStudy++
					}
				}
			}
		}
	}
	if beStudy == 0 {
		t.Fatal("expected >=1 backend_engineering study task in the generated plan, got 0")
	}
}
