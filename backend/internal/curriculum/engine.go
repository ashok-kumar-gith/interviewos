package curriculum

// The Curriculum Engine is a deterministic generator that turns an intake
// profile + the content library + company weights into a dated N-week roadmap
// (03-ARCHITECTURE.md §6.1).
//
// The engine is intentionally free of any database, HTTP, or framework
// dependency: it consumes a plain Input value (assembled by internal/roadmap
// from the content tables) and returns a Plan value (persisted by
// internal/roadmap). Every function is pure and unit-testable, and the same
// Input always yields the same Plan — stable sorts and fixed tie-breakers, no
// randomness (the determinism guarantee from §6.1).

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// PillarType mirrors the pillar_type enum. Re-declared here (rather than imported
// from internal/content) so the engine stays dependency-free and independently
// testable; the roadmap layer maps content values onto these.
type PillarType string

const (
	PillarDSA        PillarType = "dsa"
	PillarSystem     PillarType = "system_design"
	PillarLLD        PillarType = "lld"
	PillarBackendEng PillarType = "backend_engineering"
	PillarBehavioral PillarType = "behavioral"
	PillarResume     PillarType = "resume"
)

// Difficulty / Priority mirror the shared content enums.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// TaskKind mirrors the task_kind enum.
type TaskKind string

const (
	KindStudy  TaskKind = "study"
	KindSolve  TaskKind = "solve"
	KindRead   TaskKind = "read"
	KindWatch  TaskKind = "watch"
	KindRevise TaskKind = "revise"
	KindMock   TaskKind = "mock"
)

// ItemType mirrors the plan_item_type enum (subset used by the engine).
type ItemType string

const (
	ItemTopic    ItemType = "topic"
	ItemProblem  ItemType = "problem"
	ItemResource ItemType = "resource"
)

// --- Input model (assembled by the roadmap layer from content + profile) ---

// Profile is the slice of the intake profile the engine needs.
type Profile struct {
	HoursPerWeek    int                // weekly study-hour budget H
	StartDate       time.Time          // S (roadmap day 0)
	TargetWeeks     int                // N (defaults applied by caller)
	PillarStrengths map[PillarType]int // self-assessed 1..5; missing => neutral
	ActiveDays      int                // study days per week (default 6)
}

// Topic is a learning topic with the ordering signals the engine uses.
type Topic struct {
	ID             uuid.UUID
	Pillar         PillarType
	Slug           string
	Name           string
	Difficulty     Difficulty
	Priority       Priority
	EstimatedHours float64
	Prerequisites  []uuid.UUID // topic ids that must precede this topic
	SortOrder      int         // deterministic tie-breaker from the seed
}

// Problem is a DSA-style practice problem attached to a topic.
type Problem struct {
	ID               uuid.UUID
	TopicID          uuid.UUID
	Slug             string
	Title            string
	Difficulty       Difficulty
	EstimatedMinutes int
	FrequencyScore   float64 // cross-company aggregate
	CompanyFrequency float64 // frequency at the target company (0 if none)
	SortOrder        int
}

// Resource is a primary learning resource linked to a topic.
type Resource struct {
	ID               uuid.UUID
	TopicID          uuid.UUID
	Title            string
	Kind             TaskKind // KindRead or KindWatch
	EstimatedMinutes int
	IsPrimary        bool
	SortOrder        int
}

// PillarMeta carries the per-pillar base weight from the track.
type PillarMeta struct {
	Type   PillarType
	Weight float64 // track default weight (pillars.weight)
}

// Input is the complete, DB-free engine input.
type Input struct {
	Profile    Profile
	Pillars    []PillarMeta
	Topics     []Topic
	Problems   []Problem
	Resources  []Resource
	CompanyMul map[PillarType]float64 // per-pillar company multiplier (>1 emphasize)
}

// --- Output model (persisted by the roadmap layer) ---

// Task is a single emitted plan task (maps to plan_tasks).
type Task struct {
	Kind             TaskKind
	ItemType         ItemType
	ItemID           uuid.UUID
	Pillar           PillarType
	Title            string
	EstimatedMinutes int
	Priority         Priority
	Difficulty       Difficulty // empty for non-graded items (read/watch)
	SortOrder        int
}

// Day is a dated day holding bin-packed tasks (maps to plan_days).
type Day struct {
	Date           time.Time
	IsRestDay      bool
	Tasks          []Task
	PlannedMinutes int
}

// Week groups days for week N (maps to roadmap_weeks).
type Week struct {
	Number       int
	StartDate    time.Time
	EndDate      time.Time
	Theme        string
	FocusPillars []PillarType
	PlannedHours float64
	Days         []Day
}

// Plan is the full deterministic roadmap (maps to roadmaps + children).
type Plan struct {
	StartDate    time.Time
	EndDate      time.Time
	TotalWeeks   int
	HoursPerWeek int
	Weeks        []Week
}

// Defaults and tuning constants. Budget tolerance is the PRD 7.2 acceptance:
// a week's planned hours never exceed the budget by more than 10%.
const (
	DefaultWeeks        = 12
	DefaultActiveDays   = 6
	budgetTolerance     = 1.10 // weekly planned hours <= H * 1.10
	maxProblemsPerTopic = 4    // cap solve tasks emitted per topic
	maxResourceMinutes  = 60   // a read/watch task is one study session, not a whole book/course
	maxResourcesPerTopic = 2   // cap read/watch tasks emitted per topic
)

// Generate runs the full deterministic algorithm (§6.1) and returns the Plan.
// It never panics on sparse content: pillars with no topics simply contribute
// nothing, and the bin-packer tolerates an empty topic set.
func Generate(in Input) Plan {
	weeks := in.Profile.TargetWeeks
	if weeks <= 0 {
		weeks = DefaultWeeks
	}
	activeDays := in.Profile.ActiveDays
	if activeDays <= 0 || activeDays > 7 {
		activeDays = DefaultActiveDays
	}
	H := in.Profile.HoursPerWeek
	if H <= 0 {
		H = 1
	}

	// 1. Per-pillar hour allocation, skewed by company weight + self-assessment.
	hours := allocatePillarHours(in, H*weeks)

	// 2. Pre-emit the tasks (and their true minute load) for every topic once.
	// Emitted minutes — not the topic's nominal estimate — are the unit used for
	// every downstream budget decision, so weekly totals stay within tolerance.
	indexers := newIndexers(in)
	tp := emitAllTopicTasks(in, indexers)

	// 3. Select & order topics within each pillar's hour budget (by emitted load).
	selectedByPillar := selectTopics(in, hours, indexers, tp)

	// 4. Sequence topics across weeks (round-robin, interleaving pillars).
	weekTopics := sequenceWeeks(in, selectedByPillar, weeks, H, tp)

	// 5. Distribute each week's topics into days using the pre-emitted tasks.
	plan := Plan{
		StartDate:    truncDay(in.Profile.StartDate),
		TotalWeeks:   weeks,
		HoursPerWeek: H,
	}
	for w := 0; w < weeks; w++ {
		week := buildWeek(in, w, weekTopics[w], activeDays, H, tp)
		plan.Weeks = append(plan.Weeks, week)
	}
	if len(plan.Weeks) > 0 {
		plan.EndDate = plan.Weeks[len(plan.Weeks)-1].EndDate
	} else {
		plan.EndDate = plan.StartDate.AddDate(0, 0, weeks*7-1)
	}
	return plan
}

// allocatePillarHours implements step 1 of §6.1:
//
//	score[p] = base * companyMul * (0.6 + 0.4*gap)   where gap = (6-strength)/5
//	hours[p] = normalize(score) * budget
//
// Weaker self-assessed pillars (low strength) get a larger gap and thus more
// hours; the company multiplier emphasizes/de-emphasizes per the target company.
func allocatePillarHours(in Input, budget int) map[PillarType]float64 {
	score := make(map[PillarType]float64, len(in.Pillars))
	var total float64
	for _, p := range in.Pillars {
		base := p.Weight
		if base <= 0 {
			base = 1.0
		}
		cmul := 1.0
		if m, ok := in.CompanyMul[p.Type]; ok && m > 0 {
			cmul = m
		}
		strength := in.Profile.PillarStrengths[p.Type]
		if strength < 1 || strength > 5 {
			strength = 3 // neutral when unrated
		}
		gap := float64(6-strength) / 5.0 // strength 1 => gap 1.0; strength 5 => gap 0.2
		score[p.Type] = base * cmul * (0.6 + 0.4*gap)
		total += score[p.Type]
	}
	hours := make(map[PillarType]float64, len(score))
	if total == 0 {
		return hours
	}
	for p, s := range score {
		hours[p] = (s / total) * float64(budget)
	}
	return hours
}

// indexers groups problems and resources by topic for fast lookup during
// task emission, keeping selection and emission O(n).
type indexers struct {
	problemsByTopic  map[uuid.UUID][]Problem
	resourcesByTopic map[uuid.UUID][]Resource
}

func newIndexers(in Input) indexers {
	ix := indexers{
		problemsByTopic:  make(map[uuid.UUID][]Problem),
		resourcesByTopic: make(map[uuid.UUID][]Resource),
	}
	for _, p := range in.Problems {
		ix.problemsByTopic[p.TopicID] = append(ix.problemsByTopic[p.TopicID], p)
	}
	for _, r := range in.Resources {
		ix.resourcesByTopic[r.TopicID] = append(ix.resourcesByTopic[r.TopicID], r)
	}
	// Deterministic ordering within each topic: company freq desc, global freq
	// desc, difficulty easy->hard, then seed sort order, then id.
	for tid := range ix.problemsByTopic {
		ps := ix.problemsByTopic[tid]
		sort.SliceStable(ps, func(i, j int) bool { return problemLess(ps[i], ps[j]) })
		ix.problemsByTopic[tid] = ps
	}
	for tid := range ix.resourcesByTopic {
		rs := ix.resourcesByTopic[tid]
		sort.SliceStable(rs, func(i, j int) bool { return resourceLess(rs[i], rs[j]) })
		ix.resourcesByTopic[tid] = rs
	}
	return ix
}

// topicPlans holds the pre-emitted tasks and total minute load for each topic,
// keyed by topic id. The load is the sum of the emitted task minutes — the true
// cost of fully working the topic — which is what the engine budgets against.
type topicPlans struct {
	tasks map[uuid.UUID][]Task
	load  map[uuid.UUID]int // minutes
}

// emitAllTopicTasks pre-emits tasks for every topic once. Emission order is
// deterministic; a stable per-topic sort counter is restarted per topic so the
// SortOrder values are local and re-assigned per day during packing.
func emitAllTopicTasks(in Input, ix indexers) topicPlans {
	tp := topicPlans{
		tasks: make(map[uuid.UUID][]Task, len(in.Topics)),
		load:  make(map[uuid.UUID]int, len(in.Topics)),
	}
	for _, t := range in.Topics {
		counter := 0
		tasks := emitTopicTasks(t, ix, &counter)
		var load int
		for _, tk := range tasks {
			load += tk.EstimatedMinutes
		}
		tp.tasks[t.ID] = tasks
		tp.load[t.ID] = load
	}
	return tp
}

// loadHours returns a topic's emitted load in hours (used for budgeting).
func (tp topicPlans) loadHours(id uuid.UUID) float64 {
	return float64(tp.load[id]) / 60.0
}

func problemLess(a, b Problem) bool {
	if a.CompanyFrequency != b.CompanyFrequency {
		return a.CompanyFrequency > b.CompanyFrequency
	}
	if a.FrequencyScore != b.FrequencyScore {
		return a.FrequencyScore > b.FrequencyScore
	}
	if a.Difficulty != b.Difficulty {
		return difficultyRank(a.Difficulty) < difficultyRank(b.Difficulty)
	}
	if a.SortOrder != b.SortOrder {
		return a.SortOrder < b.SortOrder
	}
	return a.ID.String() < b.ID.String()
}

func resourceLess(a, b Resource) bool {
	if a.IsPrimary != b.IsPrimary {
		return a.IsPrimary // primary first
	}
	if a.SortOrder != b.SortOrder {
		return a.SortOrder < b.SortOrder
	}
	return a.ID.String() < b.ID.String()
}

// selectTopics implements step 2: topological order by prerequisites, then a
// stable company-weighted priority sort, then greedily take topics until the
// pillar's hour budget is reached.
func selectTopics(in Input, hours map[PillarType]float64, ix indexers, tp topicPlans) map[PillarType][]Topic {
	byPillar := make(map[PillarType][]Topic)
	for _, t := range in.Topics {
		byPillar[t.Pillar] = append(byPillar[t.Pillar], t)
	}

	out := make(map[PillarType][]Topic, len(byPillar))
	for pillar, topics := range byPillar {
		ordered := topoSort(topics)
		// Stable sort by (priority desc, company-frequency-of-best-problem desc,
		// difficulty asc), preserving topo order as the base tie-break by keeping
		// the topo position. We sort a copy carrying the topo index.
		type ranked struct {
			t       Topic
			topoIdx int
			compFre float64
		}
		rk := make([]ranked, len(ordered))
		for i, t := range ordered {
			rk[i] = ranked{t: t, topoIdx: i, compFre: topicCompanyFrequency(t, ix)}
		}
		sort.SliceStable(rk, func(i, j int) bool {
			a, b := rk[i], rk[j]
			if priorityRank(a.t.Priority) != priorityRank(b.t.Priority) {
				return priorityRank(a.t.Priority) > priorityRank(b.t.Priority)
			}
			if a.compFre != b.compFre {
				return a.compFre > b.compFre
			}
			if a.t.Difficulty != b.t.Difficulty {
				return difficultyRank(a.t.Difficulty) < difficultyRank(b.t.Difficulty)
			}
			return a.topoIdx < b.topoIdx
		})

		budget := hours[pillar]
		var acc float64
		var selected []Topic
		for _, r := range rk {
			est := tp.loadHours(r.t.ID) // true emitted load in hours
			if est <= 0 {
				est = 1.0
			}
			// Always take at least one topic if any budget exists; otherwise stop
			// when adding would exceed the pillar budget.
			if len(selected) > 0 && acc+est > budget {
				continue
			}
			if budget <= 0 {
				break
			}
			selected = append(selected, r.t)
			acc += est
		}
		// Re-establish prerequisite-respecting order for the selected set so that
		// week sequencing emits prerequisites first.
		out[pillar] = topoSort(selected)
	}
	return out
}

// topicCompanyFrequency returns the company frequency of the topic's most-asked
// problem, used as a company-weighting signal during topic ranking.
func topicCompanyFrequency(t Topic, ix indexers) float64 {
	var best float64
	for _, p := range ix.problemsByTopic[t.ID] {
		if p.CompanyFrequency > best {
			best = p.CompanyFrequency
		}
	}
	return best
}

// sequenceWeeks implements step 3: round-robin the selected topics (grouped by
// pillar) across N weeks, respecting per-week capacity H and keeping
// prerequisite order within each pillar. Returns topics assigned to each week.
func sequenceWeeks(in Input, byPillar map[PillarType][]Topic, weeks, H int, tp topicPlans) [][]Topic {
	result := make([][]Topic, weeks)
	weekHours := make([]float64, weeks)

	// Deterministic pillar iteration order: by track sort (the order in
	// in.Pillars), so the round-robin is stable.
	order := make([]PillarType, 0, len(in.Pillars))
	for _, p := range in.Pillars {
		if len(byPillar[p.Type]) > 0 {
			order = append(order, p.Type)
		}
	}

	// Cursors into each pillar's ordered topic list.
	cursor := make(map[PillarType]int, len(order))
	remaining := 0
	for _, p := range order {
		remaining += len(byPillar[p])
	}

	week := 0
	cap := float64(H) * budgetTolerance
	for remaining > 0 {
		progressed := false
		for _, p := range order {
			c := cursor[p]
			list := byPillar[p]
			if c >= len(list) {
				continue
			}
			t := list[c]
			est := tp.loadHours(t.ID) // true emitted load in hours
			if est <= 0 {
				est = 1.0
			}
			// Find the earliest week at or after the round's base week whose running
			// total still fits the tolerance budget once this topic is added. We
			// scan forward from `week` so topics flow chronologically (preserving
			// prerequisite order across weeks within a pillar), but fall back to any
			// earlier under-budget week so capacity is never wasted while a later
			// week overflows.
			placed := false
			for w := week; w < weeks; w++ {
				if weekHours[w]+est <= cap {
					result[w] = append(result[w], t)
					weekHours[w] += est
					placed = true
					break
				}
			}
			if !placed {
				for w := 0; w < week; w++ {
					if weekHours[w]+est <= cap {
						result[w] = append(result[w], t)
						weekHours[w] += est
						placed = true
						break
					}
				}
			}
			if !placed {
				// No week can fit this topic within tolerance. Place it in the
				// globally least-loaded week to keep any breach minimal (this only
				// happens when a single topic's load alone exceeds the weekly cap, or
				// when total selected load genuinely exceeds N*cap).
				best := 0
				for w := 1; w < weeks; w++ {
					if weekHours[w] < weekHours[best] {
						best = w
					}
				}
				result[best] = append(result[best], t)
				weekHours[best] += est
			}
			cursor[p] = c + 1
			remaining--
			progressed = true
		}
		// Advance the base week pointer once every pillar has contributed one
		// topic this round, so later topics land in later weeks (interleaving).
		if week < weeks-1 {
			week++
		}
		if !progressed {
			break
		}
	}
	return result
}

// buildWeek implements step 5: gather each topic's pre-emitted tasks, then
// bin-pack them into the week's active days by estimated minutes.
func buildWeek(in Input, weekIdx int, topics []Topic, activeDays, H int, tp topicPlans) Week {
	weekStart := truncDay(in.Profile.StartDate).AddDate(0, 0, weekIdx*7)
	week := Week{
		Number:    weekIdx + 1,
		StartDate: weekStart,
		EndDate:   weekStart.AddDate(0, 0, 6),
	}

	// Build the 7 calendar days; the first activeDays are study days, the rest
	// are rest days (deterministic: rest days at the tail of the week).
	days := make([]Day, 7)
	for i := 0; i < 7; i++ {
		d := Day{Date: weekStart.AddDate(0, 0, i)}
		if i >= activeDays {
			d.IsRestDay = true
		}
		days[i] = d
	}

	// Gather the pre-emitted tasks for every topic in this week.
	var tasks []Task
	focus := map[PillarType]bool{}
	for _, t := range topics {
		focus[t.Pillar] = true
		tasks = append(tasks, tp.tasks[t.ID]...)
	}

	// Bin-pack tasks into the active days, balancing by accumulated minutes.
	dayBudgetMin := (H * 60) / max(activeDays, 1)
	packTasks(days, tasks, activeDays, dayBudgetMin)

	// Finalize per-day and per-week planned totals.
	var weekMinutes int
	for i := range days {
		var m int
		for _, tk := range days[i].Tasks {
			m += tk.EstimatedMinutes
		}
		days[i].PlannedMinutes = m
		weekMinutes += m
	}
	week.Days = days
	week.PlannedHours = roundHours(float64(weekMinutes) / 60.0)
	week.FocusPillars = orderedFocus(in, focus)
	week.Theme = themeFor(week.FocusPillars)
	return week
}

// emitTopicTasks produces the ordered tasks for a single topic: one study task,
// up to maxProblemsPerTopic solve tasks (company-frequency ordered), and the
// primary resources as read/watch tasks.
func emitTopicTasks(t Topic, ix indexers, sortCounter *int) []Task {
	var out []Task
	next := func() int { *sortCounter++; return *sortCounter }

	studyMin := int(t.EstimatedHours * 60)
	if studyMin <= 0 {
		studyMin = 60
	}
	// Split theory study from practice: the study task covers ~60% of the topic's
	// estimated hours, leaving room for solve/read tasks within the same budget.
	theoryMin := studyMin * 6 / 10
	if theoryMin < 30 {
		theoryMin = minInt(studyMin, 30)
	}
	out = append(out, Task{
		Kind: KindStudy, ItemType: ItemTopic, ItemID: t.ID, Pillar: t.Pillar,
		Title: "Study: " + t.Name, EstimatedMinutes: theoryMin,
		Priority: t.Priority, Difficulty: t.Difficulty, SortOrder: next(),
	})

	probs := ix.problemsByTopic[t.ID]
	limit := minInt(len(probs), maxProblemsPerTopic)
	for i := 0; i < limit; i++ {
		p := probs[i]
		est := p.EstimatedMinutes
		if est <= 0 {
			est = 30
		}
		out = append(out, Task{
			Kind: KindSolve, ItemType: ItemProblem, ItemID: p.ID, Pillar: t.Pillar,
			Title: "Solve: " + p.Title, EstimatedMinutes: est,
			Priority: solvePriority(p), Difficulty: p.Difficulty, SortOrder: next(),
		})
	}

	emitted := 0
	for _, r := range ix.resourcesByTopic[t.ID] {
		if !r.IsPrimary {
			continue
		}
		if emitted >= maxResourcesPerTopic {
			break
		}
		// A resource task is one study session, not the whole book/course: cap its
		// minutes so a 20-hour book does not dominate the weekly budget. The
		// resource is still linked for the user to continue beyond the session.
		est := r.EstimatedMinutes
		if est <= 0 {
			est = 30
		}
		if est > maxResourceMinutes {
			est = maxResourceMinutes
		}
		kind := r.Kind
		if kind != KindWatch {
			kind = KindRead
		}
		out = append(out, Task{
			Kind: kind, ItemType: ItemResource, ItemID: r.ID, Pillar: t.Pillar,
			Title: titleForResource(kind, r.Title), EstimatedMinutes: est,
			Priority: PriorityMedium, SortOrder: next(),
		})
		emitted++
	}
	return out
}

// packTasks bin-packs tasks across the active days using a balanced
// least-loaded-day heuristic, keeping task emission order stable as a tie-break.
// It assigns sequential sort_order per day so the Today list renders in a sane
// order. Rest days never receive tasks.
func packTasks(days []Day, tasks []Task, activeDays, dayBudgetMin int) {
	if activeDays <= 0 {
		activeDays = 1
	}
	loads := make([]int, activeDays)
	for _, tk := range tasks {
		// Pick the least-loaded active day; prefer the earliest day on a tie so
		// packing is deterministic and front-loads the week slightly.
		best := 0
		for d := 1; d < activeDays; d++ {
			if loads[d] < loads[best] {
				best = d
			}
		}
		// If the chosen day is already at/over the daily budget but an emptier day
		// exists, the least-loaded pick already handles it; budget is advisory and
		// the weekly tolerance is enforced upstream in sequencing.
		_ = dayBudgetMin
		tk.SortOrder = len(days[best].Tasks)
		days[best].Tasks = append(days[best].Tasks, tk)
		loads[best] += tk.EstimatedMinutes
	}
}

// --- small deterministic helpers ---

func difficultyRank(d Difficulty) int {
	switch d {
	case DifficultyEasy:
		return 0
	case DifficultyMedium:
		return 1
	case DifficultyHard:
		return 2
	default:
		return 1
	}
}

func priorityRank(p Priority) int {
	switch p {
	case PriorityCritical:
		return 3
	case PriorityHigh:
		return 2
	case PriorityMedium:
		return 1
	case PriorityLow:
		return 0
	default:
		return 1
	}
}

// solvePriority promotes high-company-frequency problems to a higher priority so
// they bubble up the Today list.
func solvePriority(p Problem) Priority {
	switch {
	case p.CompanyFrequency >= 60 || p.FrequencyScore >= 80:
		return PriorityHigh
	case p.CompanyFrequency >= 30 || p.FrequencyScore >= 50:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

func titleForResource(kind TaskKind, title string) string {
	if kind == KindWatch {
		return "Watch: " + title
	}
	return "Read: " + title
}

// topoSort returns the topics in an order that respects prerequisites (a topic
// appears after every prerequisite that is present in the set). It is stable:
// ties break by SortOrder then id. Prerequisites pointing outside the set are
// ignored. Cycles (which the seed forbids) degrade gracefully to sort order.
func topoSort(topics []Topic) []Topic {
	if len(topics) <= 1 {
		return append([]Topic(nil), topics...)
	}
	idx := make(map[uuid.UUID]Topic, len(topics))
	for _, t := range topics {
		idx[t.ID] = t
	}
	// Stable base order.
	base := append([]Topic(nil), topics...)
	sort.SliceStable(base, func(i, j int) bool {
		if base[i].SortOrder != base[j].SortOrder {
			return base[i].SortOrder < base[j].SortOrder
		}
		return base[i].ID.String() < base[j].ID.String()
	})

	visited := make(map[uuid.UUID]int) // 0=unseen 1=in-progress 2=done
	var out []Topic
	var visit func(t Topic)
	visit = func(t Topic) {
		switch visited[t.ID] {
		case 2:
			return
		case 1:
			return // cycle guard
		}
		visited[t.ID] = 1
		for _, pre := range t.Prerequisites {
			if p, ok := idx[pre]; ok {
				visit(p)
			}
		}
		visited[t.ID] = 2
		out = append(out, t)
	}
	for _, t := range base {
		visit(t)
	}
	return out
}

func orderedFocus(in Input, focus map[PillarType]bool) []PillarType {
	var out []PillarType
	for _, p := range in.Pillars {
		if focus[p.Type] {
			out = append(out, p.Type)
		}
	}
	return out
}

func themeFor(focus []PillarType) string {
	if len(focus) == 0 {
		return "Rest & consolidation"
	}
	names := map[PillarType]string{
		PillarDSA:        "DSA",
		PillarSystem:     "System Design",
		PillarLLD:        "LLD",
		PillarBackendEng: "Backend Engineering",
		PillarBehavioral: "Behavioral",
		PillarResume:     "Resume",
	}
	switch len(focus) {
	case 1:
		return names[focus[0]] + " focus"
	default:
		return names[focus[0]] + " + " + names[focus[1]]
	}
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func roundHours(h float64) float64 { return float64(int(h*100+0.5)) / 100 }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
