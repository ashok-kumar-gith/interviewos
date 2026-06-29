package behavioral

import (
	"context"
	"math"
	"regexp"
	"strings"
)

// ImproveInput is the STAR content fed to an Improver. It is decoupled from the
// persisted Story so the improver can also run on inline (unsaved) STAR text.
type ImproveInput struct {
	Title     string
	Theme     Theme
	Situation string
	Task      string
	Action    string
	Result    string
	Metrics   string
}

// ImprovedSTAR carries optional tightened STAR rewrites. Empty fields mean the
// improver had no concrete rewrite to offer for that section.
type ImprovedSTAR struct {
	Situation string `json:"situation,omitempty"`
	Task      string `json:"task,omitempty"`
	Action    string `json:"action,omitempty"`
	Result    string `json:"result,omitempty"`
	Metrics   string `json:"metrics,omitempty"`
}

// ImproveResult is the structured suggestion payload (mirrors the openapi
// StoryImproveResponse, sans story_id which the handler attaches).
type ImproveResult struct {
	Improved      ImprovedSTAR `json:"improved"`
	Suggestions   []string     `json:"suggestions"`
	StrengthScore float64      `json:"strength_score"`
	UsedFallback  bool         `json:"used_fallback"`
}

// Improver produces structured suggestions for strengthening a STAR story. The
// deterministic stub satisfies it today; a Claude-API-backed implementation can
// be dropped in later without touching the service or handler.
type Improver interface {
	Improve(ctx context.Context, in ImproveInput) (*ImproveResult, error)
}

// weakActionVerbs are low-signal verbs that read as passive/vague in the Action
// section. We flag them and nudge toward stronger ownership language.
var weakActionVerbs = map[string]struct{}{
	"helped":      {},
	"assisted":    {},
	"worked":      {},
	"participated": {},
	"involved":    {},
	"contributed": {},
	"supported":   {},
	"tried":       {},
	"handled":     {},
	"did":         {},
}

// metricPattern matches quantified impact: numbers, percentages, multipliers,
// currency, or latency/throughput units.
var metricPattern = regexp.MustCompile(`(?i)(\d+(\.\d+)?\s?(%|x|k|m|b|ms|s|sec|min|hours?|days?|weeks?|months?|rps|qps|gb|mb|tb))|(\$\s?\d)|(\b\d{2,}\b)|(\d+(\.\d+)?\s?(percent|times|million|billion|thousand))`)

var wordSplit = regexp.MustCompile(`[^a-zA-Z]+`)

// DeterministicImprover is a fully offline, pure-function Improver. Given the
// same input it always returns the same suggestions and score, which makes it
// trivially testable and a safe fallback when no AI provider is configured.
type DeterministicImprover struct{}

// NewDeterministicImprover returns the deterministic stub Improver.
func NewDeterministicImprover() *DeterministicImprover { return &DeterministicImprover{} }

// Improve analyzes the STAR sections and returns deterministic suggestions plus
// a 0–100 strength score. It never calls an external service and never errors.
func (DeterministicImprover) Improve(_ context.Context, in ImproveInput) (*ImproveResult, error) {
	res := &ImproveResult{UsedFallback: true}
	var suggestions []string

	situation := strings.TrimSpace(in.Situation)
	task := strings.TrimSpace(in.Task)
	action := strings.TrimSpace(in.Action)
	result := strings.TrimSpace(in.Result)
	metrics := strings.TrimSpace(in.Metrics)

	// --- Completeness: each STAR section should be present. ---
	score := 0.0
	if situation != "" {
		score += 15
	} else {
		suggestions = append(suggestions, "Add a Situation: set the scene with the context, your role, and the stakes.")
	}
	if task != "" {
		score += 15
	} else {
		suggestions = append(suggestions, "Add a Task: state the specific problem or goal you were responsible for.")
	}
	if action != "" {
		score += 20
	} else {
		suggestions = append(suggestions, "Add an Action: describe what *you* personally did, step by step.")
	}
	if result != "" {
		score += 20
	} else {
		suggestions = append(suggestions, "Add a Result: describe the outcome and what changed because of your action.")
	}

	// --- Quantification: results/metrics should carry numbers. ---
	hasMetrics := metricPattern.MatchString(metrics) || metricPattern.MatchString(result)
	if hasMetrics {
		score += 15
	} else {
		suggestions = append(suggestions,
			"Quantify the impact: add concrete metrics (e.g. \"reduced p99 latency by 40%\", \"saved 12 engineer-hours/week\").")
	}

	// --- Weak action verbs: nudge toward first-person ownership. ---
	if action != "" {
		if weak := findWeakVerbs(action); len(weak) > 0 {
			suggestions = append(suggestions,
				"Strengthen weak action verbs ("+strings.Join(weak, ", ")+"): use decisive, first-person verbs like \"designed\", \"led\", \"built\", \"automated\".")
		} else {
			score += 5
		}
		// Reward explicit first-person ownership in the action.
		if mentionsFirstPerson(action) {
			score += 5
		} else {
			suggestions = append(suggestions,
				"Make ownership explicit in the Action: lead with \"I\" so it is clear what you (not the team) did.")
		}
	}

	// --- Concision: very long sections read as rambling. ---
	if longestSectionWords(situation, task, action, result) > 150 {
		suggestions = append(suggestions,
			"Tighten the framing: trim to ~30s of speaking per section; cut background that does not change the outcome.")
	} else {
		score += 5
	}

	// --- Title presence is a small signal of a well-formed story. ---
	if strings.TrimSpace(in.Title) == "" {
		suggestions = append(suggestions, "Give the story a short, memorable title so you can recall it under pressure.")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Strong STAR story: clear structure, quantified impact, and decisive ownership. Rehearse it to ~90 seconds.")
	}

	res.StrengthScore = clampScore(score)
	res.Suggestions = suggestions
	res.Improved = buildImproved(in, hasMetrics)
	return res, nil
}

// findWeakVerbs returns the distinct weak verbs present in text, in first-seen
// order, for stable (deterministic) output.
func findWeakVerbs(text string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, w := range wordSplit.Split(strings.ToLower(text), -1) {
		if w == "" {
			continue
		}
		if _, ok := weakActionVerbs[w]; ok {
			if _, dup := seen[w]; !dup {
				seen[w] = struct{}{}
				out = append(out, w)
			}
		}
	}
	return out
}

func mentionsFirstPerson(text string) bool {
	for _, w := range wordSplit.Split(strings.ToLower(text), -1) {
		if w == "i" || w == "my" || w == "me" {
			return true
		}
	}
	return false
}

func longestSectionWords(sections ...string) int {
	max := 0
	for _, s := range sections {
		n := len(strings.Fields(s))
		if n > max {
			max = n
		}
	}
	return max
}

// buildImproved offers light, deterministic rewrite hints. It does not fabricate
// content; it only proposes a metrics scaffold when quantification is missing.
func buildImproved(in ImproveInput, hasMetrics bool) ImprovedSTAR {
	out := ImprovedSTAR{}
	if !hasMetrics && strings.TrimSpace(in.Result) != "" {
		out.Result = strings.TrimSpace(in.Result) +
			" — quantify this: by how much, in what units, over what timeframe?"
		out.Metrics = "e.g. percentage change, absolute counts, time/cost saved, scale handled"
	}
	return out
}

func clampScore(s float64) float64 {
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	// Round to 2 decimals to match the numeric(5,2) column.
	return math.Round(s*100) / 100
}
