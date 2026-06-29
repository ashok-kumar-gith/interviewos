package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/interviewos/backend/internal/behavioral"
	"github.com/interviewos/backend/internal/mock"
	"github.com/interviewos/backend/internal/resume"
)

// TextResult is the result of a free-text feature (planner, coach, daily_plan,
// sd_review). It mirrors the openapi AIResponse fields the handler needs.
type TextResult struct {
	Feature      Feature
	Content      string
	Structured   map[string]any
	Model        *string
	UsedFallback bool
	InvocationID uuid.UUID
}

// StoryResult mirrors the openapi StoryImproveResponse.
type StoryResult struct {
	StoryID       uuid.UUID
	Improved      behavioral.ImprovedSTAR
	Suggestions   []string
	StrengthScore float64
	UsedFallback  bool
	InvocationID  uuid.UUID
}

// ResumeResult mirrors the openapi ResumeScoreResponse (AI resume-review).
type ResumeResult struct {
	ATSScore        float64
	KeywordMatches  []string
	MissingKeywords []string
	Suggestions     []string
	UsedFallback    bool
	InvocationID    uuid.UUID
}

// TopicWeakness mirrors a TopicAnalyticsEntry for the weakness detector.
type TopicWeakness struct {
	TopicID       uuid.UUID
	TopicName     string
	PillarType    string
	Confidence    *int
	CompletionPct float64
}

// WeaknessResult mirrors the openapi WeaknessDetectResponse.
type WeaknessResult struct {
	WeakTopics       []TopicWeakness
	RecommendedTasks []string
	UsedFallback     bool
	InvocationID     uuid.UUID
}

// modelPtr returns a pointer to the configured model, or nil in fallback mode.
func (s *Service) modelPtr(usedFallback bool) *string {
	if usedFallback || s.model == "" {
		return nil
	}
	m := s.model
	return &m
}

// ---- Planner ----

// PlannerInput carries the AIPlannerRequest fields.
type PlannerInput struct {
	FocusPillars []string
	Notes        string
}

// Planner refines the user's study plan. AI path produces prose advice grounded
// in the profile + active roadmap; fallback emits a deterministic plan summary.
func (s *Service) Planner(ctx context.Context, userID uuid.UUID, in PlannerInput) (*TextResult, error) {
	started := s.now()
	profile, _ := s.profiles.Profile(ctx, userID)
	roadmapID, totalWeeks, rErr := s.plans.ActiveRoadmap(ctx, userID)

	system := "You are an expert technical-interview study planner for backend SDE3 candidates. " +
		"Give concrete, prioritized, actionable guidance. Be specific and concise; use short markdown sections."
	var b strings.Builder
	b.WriteString("Refine my interview-prep study plan.\n")
	if profile != nil {
		fmt.Fprintf(&b, "Target role: %s. Weekly study budget: %d hours over %d weeks.\n",
			profile.TargetRole, profile.HoursPerWeek, profile.TargetWeeks)
		if len(profile.PillarStrength) > 0 {
			fmt.Fprintf(&b, "Self-rated pillar strengths (1-5): %s.\n", formatStrengths(profile.PillarStrength))
		}
	}
	if rErr == nil {
		fmt.Fprintf(&b, "Active roadmap spans %d weeks.\n", totalWeeks)
	}
	if len(in.FocusPillars) > 0 {
		fmt.Fprintf(&b, "I want to focus on: %s.\n", strings.Join(in.FocusPillars, ", "))
	}
	if strings.TrimSpace(in.Notes) != "" {
		fmt.Fprintf(&b, "Notes: %s\n", strings.TrimSpace(in.Notes))
	}

	_ = roadmapID // roadmap presence (rErr) is what the prompt/fallback use.

	if c := s.tryComplete(ctx, FeaturePlanner, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeaturePlanner, c, false, started, nil)
		return &TextResult{Feature: FeaturePlanner, Content: c.text, Model: s.modelPtr(false), UsedFallback: false, InvocationID: id}, nil
	}

	content := s.plannerFallback(profile, in, rErr == nil, totalWeeks)
	id := s.record(ctx, userID, FeaturePlanner, completion{}, true, started, nil)
	return &TextResult{Feature: FeaturePlanner, Content: content, Model: nil, UsedFallback: true, InvocationID: id}, nil
}

func (s *Service) plannerFallback(profile *Profile, in PlannerInput, hasRoadmap bool, weeks int) string {
	var b strings.Builder
	b.WriteString("# Study plan\n\n")
	if profile != nil {
		fmt.Fprintf(&b, "Targeting **%s** with **%d h/week** over **%d weeks**.\n\n",
			profile.TargetRole, profile.HoursPerWeek, profile.TargetWeeks)
	}
	focus := in.FocusPillars
	if len(focus) == 0 && profile != nil {
		focus = weakestPillars(profile.PillarStrength)
	}
	if len(focus) == 0 {
		focus = []string{"dsa", "system_design", "behavioral"}
	}
	b.WriteString("## Priorities\n")
	for i, p := range focus {
		fmt.Fprintf(&b, "%d. Focus on **%s** — schedule it earliest in the week while you are freshest.\n", i+1, p)
	}
	b.WriteString("\n## Weekly rhythm\n")
	b.WriteString("- Front-load your two weakest pillars on Mon/Tue.\n")
	b.WriteString("- Reserve one session for spaced revision of completed topics.\n")
	b.WriteString("- End the week with one timed mock to surface gaps.\n")
	if hasRoadmap {
		fmt.Fprintf(&b, "\nYour active roadmap (%d weeks) already sequences topics; follow it and use this to re-prioritize when behind.\n", weeks)
	} else {
		b.WriteString("\nNo active roadmap yet — generate one (POST /roadmaps/generate) for a sequenced plan.\n")
	}
	if strings.TrimSpace(in.Notes) != "" {
		fmt.Fprintf(&b, "\n> Note acknowledged: %s\n", strings.TrimSpace(in.Notes))
	}
	return b.String()
}

// ---- Coach ----

// Coach answers an interview-prep question. AI produces a grounded answer;
// fallback gives structured, generally-useful guidance keyed off the question.
func (s *Service) Coach(ctx context.Context, userID uuid.UUID, message string) (*TextResult, error) {
	started := s.now()
	profile, _ := s.profiles.Profile(ctx, userID)

	system := "You are a senior interview coach for backend SDE3 candidates. Answer the candidate's question " +
		"directly and practically with concrete, prioritized advice. Use concise markdown."
	var b strings.Builder
	if profile != nil {
		fmt.Fprintf(&b, "Candidate context: targeting %s, %.0f years experience.\n\n", profile.TargetRole, profile.YearsExp)
	}
	fmt.Fprintf(&b, "Question: %s", strings.TrimSpace(message))

	if c := s.tryComplete(ctx, FeatureCoach, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeatureCoach, c, false, started, nil)
		return &TextResult{Feature: FeatureCoach, Content: c.text, Model: s.modelPtr(false), UsedFallback: false, InvocationID: id}, nil
	}

	content := coachFallback(message)
	id := s.record(ctx, userID, FeatureCoach, completion{}, true, started, nil)
	return &TextResult{Feature: FeatureCoach, Content: content, Model: nil, UsedFallback: true, InvocationID: id}, nil
}

func coachFallback(message string) string {
	q := strings.ToLower(message)
	var b strings.Builder
	b.WriteString("# Coach (offline guidance)\n\n")
	fmt.Fprintf(&b, "> You asked: %s\n\n", strings.TrimSpace(message))
	switch {
	case strings.Contains(q, "system design") || strings.Contains(q, "scal"):
		b.WriteString("Use a repeatable system-design framework:\n")
		b.WriteString("1. Clarify functional + non-functional requirements and scale.\n")
		b.WriteString("2. Back-of-envelope capacity estimates.\n")
		b.WriteString("3. API + data model, then a clean high-level diagram.\n")
		b.WriteString("4. Deep-dive the bottleneck (caching, sharding, queueing).\n")
		b.WriteString("5. Call out trade-offs and failure handling explicitly.\n")
	case strings.Contains(q, "behavioral") || strings.Contains(q, "star") || strings.Contains(q, "leadership"):
		b.WriteString("Structure every behavioral answer as STAR: Situation, Task, Action, Result.\n")
		b.WriteString("- Lead with first-person ownership in the Action.\n")
		b.WriteString("- Quantify the Result (%, time, scale, $).\n")
		b.WriteString("- Prepare 6-8 stories covering leadership, conflict, failure, and impact.\n")
	case strings.Contains(q, "dsa") || strings.Contains(q, "leetcode") || strings.Contains(q, "algorithm") || strings.Contains(q, "coding"):
		b.WriteString("For coding rounds, train patterns over problem count:\n")
		b.WriteString("- Drill the core patterns (two pointers, sliding window, BFS/DFS, DP, heaps).\n")
		b.WriteString("- Always state brute force first, then optimize; verbalize complexity.\n")
		b.WriteString("- Do timed sets and review every miss the next day (spaced revision).\n")
	default:
		b.WriteString("General prep guidance:\n")
		b.WriteString("- Prioritize your weakest pillar while time remains; protect daily consistency.\n")
		b.WriteString("- Convert mistakes into revision items so they don't recur.\n")
		b.WriteString("- Run one timed mock per week to calibrate under pressure.\n")
	}
	b.WriteString("\n_(AI coach is offline; this is deterministic guidance.)_\n")
	return b.String()
}

// ---- Daily plan ----

// DailyPlan recommends how to spend a given date. AI refines the day's tasks;
// fallback summarizes the planned tasks deterministically (always useful).
func (s *Service) DailyPlan(ctx context.Context, userID uuid.UUID, date string) (*TextResult, error) {
	started := s.now()
	if strings.TrimSpace(date) == "" {
		date = s.now().UTC().Format("2006-01-02")
	}
	tasks, _ := s.plans.TasksForDate(ctx, userID, date)
	profile, _ := s.profiles.Profile(ctx, userID)

	system := "You are a study-day planner. Given today's planned tasks, produce a focused, time-boxed order " +
		"of attack with brief rationale. Keep it short and actionable in markdown."
	var b strings.Builder
	fmt.Fprintf(&b, "Plan my study day for %s.\n", date)
	if profile != nil {
		fmt.Fprintf(&b, "Weekly budget: %d hours.\n", profile.HoursPerWeek)
	}
	if len(tasks) == 0 {
		b.WriteString("There are no planned tasks for this date.\n")
	} else {
		b.WriteString("Planned tasks:\n")
		for _, t := range tasks {
			fmt.Fprintf(&b, "- [%s] %s (%s, %d min, priority %s, status %s)\n",
				t.PillarType, t.Title, t.Kind, t.EstimatedMinutes, t.Priority, t.Status)
		}
	}

	if c := s.tryComplete(ctx, FeatureDailyPlan, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeatureDailyPlan, c, false, started, nil)
		return &TextResult{Feature: FeatureDailyPlan, Content: c.text, Structured: dailyStructured(date, tasks), Model: s.modelPtr(false), UsedFallback: false, InvocationID: id}, nil
	}

	content := dailyPlanFallback(date, tasks)
	id := s.record(ctx, userID, FeatureDailyPlan, completion{}, true, started, nil)
	return &TextResult{Feature: FeatureDailyPlan, Content: content, Structured: dailyStructured(date, tasks), Model: nil, UsedFallback: true, InvocationID: id}, nil
}

func dailyStructured(date string, tasks []PlanTask) map[string]any {
	total := 0
	pending := 0
	for _, t := range tasks {
		total += t.EstimatedMinutes
		if t.Status == "pending" || t.Status == "in_progress" {
			pending++
		}
	}
	return map[string]any{
		"date":            date,
		"total_tasks":     len(tasks),
		"pending_tasks":   pending,
		"planned_minutes": total,
	}
}

func dailyPlanFallback(date string, tasks []PlanTask) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Recommended plan for %s\n\n", date)
	if len(tasks) == 0 {
		b.WriteString("No tasks are scheduled for this date. Suggestions:\n")
		b.WriteString("- Treat it as a light revision day: clear any due revision items.\n")
		b.WriteString("- Or pull forward your next weakest-pillar topic from the roadmap.\n")
		return b.String()
	}
	// Order: priority desc, then estimated minutes desc, pending first.
	ordered := append([]PlanTask(nil), tasks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		pi, pj := priorityRank(ordered[i].Priority), priorityRank(ordered[j].Priority)
		if pi != pj {
			return pi > pj
		}
		return ordered[i].EstimatedMinutes > ordered[j].EstimatedMinutes
	})
	total := 0
	b.WriteString("Suggested order (highest priority first):\n")
	for i, t := range ordered {
		total += t.EstimatedMinutes
		status := ""
		if t.Status == "completed" {
			status = " ✓"
		}
		fmt.Fprintf(&b, "%d. **%s** — %s, ~%d min (%s priority)%s\n", i+1, t.Title, t.PillarType, t.EstimatedMinutes, t.Priority, status)
	}
	fmt.Fprintf(&b, "\nTotal planned: ~%d minutes (~%.1f h). Take a short break every 50 minutes.\n", total, float64(total)/60.0)
	return b.String()
}

func priorityRank(p string) int {
	switch p {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// ---- SD review ----

// SDReviewInput carries the AISdReviewRequest fields.
type SDReviewInput struct {
	DesignProblemID uuid.UUID
	AnswerMD        string
}

// SDReview critiques a user's system-design answer. AI produces a structured
// review; fallback runs a deterministic rubric over the answer text.
func (s *Service) SDReview(ctx context.Context, userID uuid.UUID, in SDReviewInput) (*TextResult, error) {
	started := s.now()
	problem, pErr := s.designs.DesignProblem(ctx, in.DesignProblemID)

	system := "You are a staff engineer reviewing a candidate's system-design answer. Critique it against a " +
		"standard rubric (requirements, estimation, API/data model, high-level design, deep-dive/bottlenecks, " +
		"trade-offs, failure handling). Be specific; cite what's missing. Concise markdown."
	var b strings.Builder
	if pErr == nil {
		fmt.Fprintf(&b, "Design problem: %s (difficulty %s).\n\n", problem.Title, problem.Difficulty)
	}
	fmt.Fprintf(&b, "Candidate answer:\n%s", strings.TrimSpace(in.AnswerMD))

	if c := s.tryComplete(ctx, FeatureSDReview, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeatureSDReview, c, false, started, nil)
		return &TextResult{Feature: FeatureSDReview, Content: c.text, Model: s.modelPtr(false), UsedFallback: false, InvocationID: id}, nil
	}

	content := sdReviewFallback(problem, in.AnswerMD)
	id := s.record(ctx, userID, FeatureSDReview, completion{}, true, started, nil)
	return &TextResult{Feature: FeatureSDReview, Content: content, Structured: sdStructured(in.AnswerMD), Model: nil, UsedFallback: true, InvocationID: id}, nil
}

// sdRubric is the deterministic system-design rubric: a label plus keywords that
// signal the dimension was addressed.
var sdRubric = []struct {
	label    string
	keywords []string
	tip      string
}{
	{"Requirements & scope", []string{"requirement", "functional", "non-functional", "constraint", "assumption"}, "State functional and non-functional requirements and your assumptions up front."},
	{"Capacity estimation", []string{"qps", "rps", "throughput", "storage", "estimate", "bandwidth", "traffic"}, "Add back-of-envelope estimates (QPS, storage, bandwidth) to justify your design."},
	{"API & data model", []string{"api", "endpoint", "schema", "table", "data model", "index"}, "Define the key APIs and the data model (tables/indexes or schema)."},
	{"High-level design", []string{"load balancer", "service", "component", "architecture", "gateway", "diagram"}, "Describe the high-level components and how requests flow between them."},
	{"Scaling & bottlenecks", []string{"cache", "shard", "partition", "replica", "queue", "cdn", "scal"}, "Identify the bottleneck and address it (caching, sharding, queues, replication)."},
	{"Trade-offs", []string{"trade-off", "tradeoff", "consistency", "availability", "cap", "latency vs"}, "Make trade-offs explicit (e.g. consistency vs availability, latency vs cost)."},
	{"Failure handling", []string{"failure", "fault", "retry", "timeout", "fallback", "redundan", "monitor"}, "Cover failure modes: retries, timeouts, redundancy, and monitoring."},
}

func sdReviewFallback(problem *DesignProblem, answer string) string {
	a := strings.ToLower(answer)
	var b strings.Builder
	b.WriteString("# System-design review (rubric)\n\n")
	if problem != nil {
		fmt.Fprintf(&b, "Problem: **%s** (%s)\n\n", problem.Title, problem.Difficulty)
	}
	covered, missing := 0, []string{}
	b.WriteString("## Coverage\n")
	for _, r := range sdRubric {
		hit := false
		for _, kw := range r.keywords {
			if strings.Contains(a, kw) {
				hit = true
				break
			}
		}
		if hit {
			covered++
			fmt.Fprintf(&b, "- ✓ %s — addressed.\n", r.label)
		} else {
			missing = append(missing, r.tip)
			fmt.Fprintf(&b, "- ✗ %s — not clearly addressed.\n", r.label)
		}
	}
	score := int(float64(covered) / float64(len(sdRubric)) * 100)
	fmt.Fprintf(&b, "\n**Rubric coverage: %d%% (%d/%d dimensions).**\n", score, covered, len(sdRubric))
	if len(missing) > 0 {
		b.WriteString("\n## Recommendations\n")
		for _, m := range missing {
			fmt.Fprintf(&b, "- %s\n", m)
		}
	} else {
		b.WriteString("\nStrong, well-rounded answer — rehearse delivery to keep it within time.\n")
	}
	if strings.TrimSpace(answer) == "" {
		b.WriteString("\n> Your answer was empty; write out your design and resubmit for a real critique.\n")
	}
	return b.String()
}

func sdStructured(answer string) map[string]any {
	a := strings.ToLower(answer)
	covered := 0
	dims := map[string]bool{}
	for _, r := range sdRubric {
		hit := false
		for _, kw := range r.keywords {
			if strings.Contains(a, kw) {
				hit = true
				break
			}
		}
		dims[r.label] = hit
		if hit {
			covered++
		}
	}
	return map[string]any{
		"rubric_coverage_pct": int(float64(covered) / float64(len(sdRubric)) * 100),
		"dimensions":          dims,
	}
}

// ---- Story improve ----

// StoryImproveInput carries the AIStoryImproveRequest (by id or inline STAR).
type StoryImproveInput struct {
	StoryID   *uuid.UUID
	Situation string
	Task      string
	Action    string
	Result    string
}

// StoryImprove improves a STAR story. The deterministic fallback reuses the
// behavioral.Improver engine. The AI path asks for suggestions, but always layers
// the deterministic strength score so the response shape is stable.
func (s *Service) StoryImprove(ctx context.Context, userID uuid.UUID, in StoryImproveInput) (*StoryResult, error) {
	started := s.now()

	improveIn := behavioral.ImproveInput{
		Situation: in.Situation,
		Task:      in.Task,
		Action:    in.Action,
		Result:    in.Result,
	}
	var storyID uuid.UUID
	if in.StoryID != nil {
		st, err := s.stories.Story(ctx, userID, *in.StoryID)
		if err != nil {
			return nil, err // ErrNotFound surfaces as 404
		}
		storyID = st.ID
		improveIn = behavioral.ImproveInput{
			Title:     st.Title,
			Theme:     behavioral.Theme(st.Theme),
			Situation: firstNonEmpty(in.Situation, st.Situation),
			Task:      firstNonEmpty(in.Task, st.Task),
			Action:    firstNonEmpty(in.Action, st.Action),
			Result:    firstNonEmpty(in.Result, st.Result),
			Metrics:   st.Metrics,
		}
	}

	// Deterministic engine is the source of the structured score + improved STAR.
	det, _ := s.improver.Improve(ctx, improveIn)

	system := "You are an expert behavioral-interview coach. Improve this STAR story: tighten framing, surface " +
		"first-person ownership, and quantify impact. Return 3-6 concrete bullet suggestions, one per line."
	prompt := fmt.Sprintf("Title: %s\nTheme: %s\nSituation: %s\nTask: %s\nAction: %s\nResult: %s\nMetrics: %s",
		improveIn.Title, improveIn.Theme, improveIn.Situation, improveIn.Task, improveIn.Action, improveIn.Result, improveIn.Metrics)

	if c := s.tryComplete(ctx, FeatureStoryImprove, system, prompt); c.ok {
		id := s.record(ctx, userID, FeatureStoryImprove, c, false, started, nil)
		res := &StoryResult{StoryID: storyID, Suggestions: splitLines(c.text), UsedFallback: false, InvocationID: id}
		if det != nil {
			res.Improved = det.Improved
			res.StrengthScore = det.StrengthScore
		}
		if len(res.Suggestions) == 0 && det != nil {
			res.Suggestions = det.Suggestions
		}
		return res, nil
	}

	id := s.record(ctx, userID, FeatureStoryImprove, completion{}, true, started, nil)
	res := &StoryResult{StoryID: storyID, UsedFallback: true, InvocationID: id}
	if det != nil {
		res.Improved = det.Improved
		res.Suggestions = det.Suggestions
		res.StrengthScore = det.StrengthScore
	}
	return res, nil
}

// ---- Resume review ----

// ResumeReview reviews/scores the user's resume. Fallback reuses resume.Scorer.
func (s *Service) ResumeReview(ctx context.Context, userID uuid.UUID) (*ResumeResult, error) {
	started := s.now()
	data, err := s.resumes.Resume(ctx, userID)
	if err != nil {
		return nil, err // ErrNotFound -> 404
	}

	scoreIn := resume.ScoreInput{
		Headline:       data.Headline,
		Summary:        data.Summary,
		Skills:         data.Skills,
		TargetKeywords: data.TargetKeywords,
		Bullets:        data.Bullets,
		Projects:       data.Projects,
	}
	det, _ := s.scorer.Score(ctx, scoreIn)

	system := "You are an expert technical resume reviewer (ATS-aware). Give 3-6 specific, high-impact " +
		"improvement suggestions, one per line. Focus on quantified impact, keyword coverage, and clarity."
	var b strings.Builder
	fmt.Fprintf(&b, "Headline: %s\nSummary: %s\nSkills: %s\nTarget keywords: %s\nProject bullets:\n",
		data.Headline, data.Summary, strings.Join(data.Skills, ", "), strings.Join(data.TargetKeywords, ", "))
	for _, bl := range data.Bullets {
		fmt.Fprintf(&b, "- %s\n", bl)
	}

	if c := s.tryComplete(ctx, FeatureResumeReview, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeatureResumeReview, c, false, started, nil)
		res := &ResumeResult{
			ATSScore:        det.ATSScore,
			KeywordMatches:  det.KeywordMatches,
			MissingKeywords: det.MissingKeywords,
			Suggestions:     splitLines(c.text),
			UsedFallback:    false,
			InvocationID:    id,
		}
		if len(res.Suggestions) == 0 {
			res.Suggestions = det.Suggestions
		}
		return res, nil
	}

	id := s.record(ctx, userID, FeatureResumeReview, completion{}, true, started, nil)
	return &ResumeResult{
		ATSScore:        det.ATSScore,
		KeywordMatches:  det.KeywordMatches,
		MissingKeywords: det.MissingKeywords,
		Suggestions:     det.Suggestions,
		UsedFallback:    true,
		InvocationID:    id,
	}, nil
}

// ---- Weakness detect ----

// WeaknessDetect ranks the user's weaknesses from analytics + mock findings.
// Fallback reuses mock.WeaknessDetector + the weak-topic analytics reader.
func (s *Service) WeaknessDetect(ctx context.Context, userID uuid.UUID) (*WeaknessResult, error) {
	started := s.now()
	weak, _ := s.topics.WeakTopics(ctx, userID, 8)
	findings, _ := s.mocks.Findings(ctx, userID)

	// Deterministic weakness ranking from mock findings.
	mf := make([]mock.Finding, 0, len(findings))
	for _, f := range findings {
		var p *mock.Pillar
		if f.PillarType != "" {
			pp := mock.Pillar(f.PillarType)
			p = &pp
		}
		mf = append(mf, mock.Finding{PillarType: p, Severity: mock.Severity(f.Severity), Category: f.Category, Detail: f.Detail})
	}
	summary, _ := s.detector.Detect(ctx, mf)

	weakTopics := make([]TopicWeakness, 0, len(weak))
	for _, w := range weak {
		weakTopics = append(weakTopics, TopicWeakness{
			TopicID:       w.TopicID,
			TopicName:     w.TopicName,
			PillarType:    w.PillarType,
			Confidence:    w.Confidence,
			CompletionPct: w.CompletionPct,
		})
	}

	system := "You are a study advisor. From the candidate's weak topics and mock-interview weaknesses, list " +
		"3-6 concrete recommended next tasks, one per line, ordered by impact."
	var b strings.Builder
	b.WriteString("Weak topics:\n")
	for _, w := range weak {
		fmt.Fprintf(&b, "- %s (%s, completion %.0f%%)\n", w.TopicName, w.PillarType, w.CompletionPct)
	}
	if summary != nil && len(summary.Items) > 0 {
		b.WriteString("Mock weaknesses (by area):\n")
		for _, it := range summary.Items {
			fmt.Fprintf(&b, "- %s (%d findings, max severity %s)\n", it.Area, it.Count, it.MaxSeverity)
		}
	}

	if c := s.tryComplete(ctx, FeatureWeaknessDetect, system, b.String()); c.ok {
		id := s.record(ctx, userID, FeatureWeaknessDetect, c, false, started, nil)
		tasks := splitLines(c.text)
		if len(tasks) == 0 {
			tasks = weaknessFallbackTasks(weak, summary)
		}
		return &WeaknessResult{WeakTopics: weakTopics, RecommendedTasks: tasks, UsedFallback: false, InvocationID: id}, nil
	}

	id := s.record(ctx, userID, FeatureWeaknessDetect, completion{}, true, started, nil)
	return &WeaknessResult{
		WeakTopics:       weakTopics,
		RecommendedTasks: weaknessFallbackTasks(weak, summary),
		UsedFallback:     true,
		InvocationID:     id,
	}, nil
}

func weaknessFallbackTasks(weak []WeakTopic, summary *mock.WeaknessSummary) []string {
	var tasks []string
	for _, w := range weak {
		if w.CompletionPct >= 100 {
			continue
		}
		tasks = append(tasks, fmt.Sprintf("Study/practice %s (%s) — currently %.0f%% complete.", w.TopicName, w.PillarType, w.CompletionPct))
		if len(tasks) >= 5 {
			break
		}
	}
	if summary != nil {
		for _, it := range summary.Items {
			if len(tasks) >= 6 {
				break
			}
			tasks = append(tasks, fmt.Sprintf("Address recurring mock weakness: %s (%d findings, max %s).", it.Area, it.Count, it.MaxSeverity))
		}
	}
	if len(tasks) == 0 {
		tasks = []string{"No weaknesses detected yet — keep logging mocks and completing topics to surface gaps."}
	}
	return tasks
}

// ---- shared helpers ----

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// splitLines turns model text into clean bullet suggestions: split on newlines,
// strip common list markers, and drop empties.
func splitLines(text string) []string {
	var out []string
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		ln = strings.TrimLeft(ln, "-*•0123456789.) \t")
		ln = strings.TrimSpace(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out
}

func formatStrengths(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}

// weakestPillars returns up to 3 pillars with the lowest self-rated strength.
func weakestPillars(m map[string]int) []string {
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.SliceStable(arr, func(i, j int) bool {
		if arr[i].v != arr[j].v {
			return arr[i].v < arr[j].v
		}
		return arr[i].k < arr[j].k
	})
	var out []string
	for i := 0; i < len(arr) && i < 3; i++ {
		out = append(out, arr[i].k)
	}
	return out
}
