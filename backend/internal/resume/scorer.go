package resume

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ScoreInput is the flattened, scorer-ready view of a resume profile and its
// projects. It is built by the service from the persisted models so the scorer
// has no dependency on the storage layer.
type ScoreInput struct {
	Headline       string
	Summary        string
	Skills         []string
	TargetKeywords []string
	// Bullets are the impact/description lines drawn from projects; each project
	// contributes its description, impact, and metrics lines.
	Bullets []string
	// Projects is the count of projects (used for the "sections present" rule).
	Projects int
}

// ScoreBreakdown is a single rule's contribution to the overall score.
type ScoreBreakdown struct {
	Rule   string  `json:"rule"`
	Score  float64 `json:"score"`  // points awarded for this rule
	Max    float64 `json:"max"`    // maximum points for this rule
	Detail string  `json:"detail"` // human-readable explanation
}

// ScoreResult is the full deterministic scoring output.
type ScoreResult struct {
	ATSScore        float64          `json:"ats_score"` // 0..100
	KeywordMatches  []string         `json:"keyword_matches"`
	MissingKeywords []string         `json:"missing_keywords"`
	Suggestions     []string         `json:"suggestions"`
	Breakdown       []ScoreBreakdown `json:"breakdown"`
	UsedFallback    bool             `json:"used_fallback"`
}

// Scorer computes a resume score. The deterministic rule-based implementation is
// the default; the interface lets a real AI reviewer be substituted later
// (the AI reviewer can fall back to the deterministic scorer when unavailable).
type Scorer interface {
	Score(ctx context.Context, in ScoreInput) (ScoreResult, error)
}

// actionVerbs is a curated set of strong resume action verbs (lowercased).
var actionVerbs = map[string]struct{}{
	"led": {}, "built": {}, "designed": {}, "architected": {}, "developed": {},
	"implemented": {}, "launched": {}, "shipped": {}, "drove": {}, "owned": {},
	"created": {}, "improved": {}, "optimized": {}, "reduced": {}, "increased": {},
	"scaled": {}, "migrated": {}, "automated": {}, "delivered": {}, "spearheaded": {},
	"managed": {}, "mentored": {}, "refactored": {}, "engineered": {}, "established": {},
	"streamlined": {}, "accelerated": {}, "eliminated": {}, "negotiated": {}, "deployed": {},
}

// quantifiedRe matches a quantified-impact signal: a number, optionally with a
// %, x, $, or a magnitude suffix (k/m/b) — e.g. "40%", "3x", "$2M", "1.5k".
var quantifiedRe = regexp.MustCompile(`(?i)(\$?\d[\d,.]*\s*(%|x|k|m|b|ms|s|gb|mb|qps|rps|req/s|users|requests)?)`)

// numberRe matches any digit (used as a lighter "has a number" check).
var numberRe = regexp.MustCompile(`\d`)

// wordRe splits text into lowercase word tokens.
var wordRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+#.\-]*`)

// RuleScorer is the deterministic, no-external-API scorer. Each rule contributes
// a weighted share of the 0..100 total.
type RuleScorer struct{}

// NewRuleScorer returns the default deterministic scorer.
func NewRuleScorer() *RuleScorer { return &RuleScorer{} }

// rule weights (sum = 100).
const (
	maxKeywords   = 35.0 // keyword coverage vs target role keywords
	maxQuantified = 20.0 // quantified impact in bullets
	maxActionVerb = 15.0 // action-verb usage at bullet start
	maxSections   = 15.0 // presence of key sections
	maxBulletLen  = 15.0 // bullet length within an ideal band
)

// Score runs all rules and aggregates a 0..100 score with a breakdown.
func (s *RuleScorer) Score(_ context.Context, in ScoreInput) (ScoreResult, error) {
	res := ScoreResult{UsedFallback: false}

	kw := s.scoreKeywords(in, &res)
	qt := s.scoreQuantified(in, &res)
	av := s.scoreActionVerbs(in, &res)
	sec := s.scoreSections(in, &res)
	bl := s.scoreBulletLength(in, &res)

	total := kw + qt + av + sec + bl
	res.ATSScore = round2(total)
	return res, nil
}

// scoreKeywords measures coverage of target keywords across the resume's
// searchable text (headline, summary, skills, bullets).
func (s *RuleScorer) scoreKeywords(in ScoreInput, res *ScoreResult) float64 {
	res.Breakdown = append(res.Breakdown, ScoreBreakdown{}) // placeholder index
	idx := len(res.Breakdown) - 1

	targets := normalizeUnique(in.TargetKeywords)
	if len(targets) == 0 {
		res.Breakdown[idx] = ScoreBreakdown{
			Rule: "keyword_coverage", Score: 0, Max: maxKeywords,
			Detail: "no target keywords set; add target role keywords to enable ATS keyword matching",
		}
		res.Suggestions = append(res.Suggestions, "Add target-role keywords so the resume can be matched against an ATS.")
		res.MissingKeywords = []string{}
		res.KeywordMatches = []string{}
		return 0
	}

	haystack := buildTokenSet(in)
	var matches, missing []string
	for _, kw := range targets {
		if keywordPresent(kw, haystack, in) {
			matches = append(matches, kw)
		} else {
			missing = append(missing, kw)
		}
	}
	sort.Strings(matches)
	sort.Strings(missing)
	res.KeywordMatches = matches
	res.MissingKeywords = missing

	coverage := float64(len(matches)) / float64(len(targets))
	score := round2(coverage * maxKeywords)
	res.Breakdown[idx] = ScoreBreakdown{
		Rule: "keyword_coverage", Score: score, Max: maxKeywords,
		Detail: detailf("%d of %d target keywords present (%.0f%% coverage)", len(matches), len(targets), coverage*100),
	}
	if len(missing) > 0 {
		res.Suggestions = append(res.Suggestions,
			"Incorporate missing target keywords where truthful: "+strings.Join(missing, ", ")+".")
	}
	return score
}

// scoreQuantified rewards bullets that contain quantified impact (numbers/%/x/$).
func (s *RuleScorer) scoreQuantified(in ScoreInput, res *ScoreResult) float64 {
	if len(in.Bullets) == 0 {
		res.Breakdown = append(res.Breakdown, ScoreBreakdown{
			Rule: "quantified_impact", Score: 0, Max: maxQuantified,
			Detail: "no project bullets found",
		})
		res.Suggestions = append(res.Suggestions, "Add project bullets describing what you did and the measurable impact.")
		return 0
	}
	quantified := 0
	for _, b := range in.Bullets {
		if numberRe.MatchString(b) && quantifiedRe.MatchString(b) {
			quantified++
		}
	}
	ratio := float64(quantified) / float64(len(in.Bullets))
	score := round2(ratio * maxQuantified)
	res.Breakdown = append(res.Breakdown, ScoreBreakdown{
		Rule: "quantified_impact", Score: score, Max: maxQuantified,
		Detail: detailf("%d of %d bullets contain quantified impact (%.0f%%)", quantified, len(in.Bullets), ratio*100),
	})
	if ratio < 0.5 {
		res.Suggestions = append(res.Suggestions, "Quantify more bullets with concrete metrics (%, latency, throughput, $, scale).")
	}
	return score
}

// scoreActionVerbs rewards bullets that start with a strong action verb.
func (s *RuleScorer) scoreActionVerbs(in ScoreInput, res *ScoreResult) float64 {
	if len(in.Bullets) == 0 {
		res.Breakdown = append(res.Breakdown, ScoreBreakdown{
			Rule: "action_verbs", Score: 0, Max: maxActionVerb,
			Detail: "no project bullets found",
		})
		return 0
	}
	withVerb := 0
	for _, b := range in.Bullets {
		if startsWithActionVerb(b) {
			withVerb++
		}
	}
	ratio := float64(withVerb) / float64(len(in.Bullets))
	score := round2(ratio * maxActionVerb)
	res.Breakdown = append(res.Breakdown, ScoreBreakdown{
		Rule: "action_verbs", Score: score, Max: maxActionVerb,
		Detail: detailf("%d of %d bullets start with a strong action verb (%.0f%%)", withVerb, len(in.Bullets), ratio*100),
	})
	if ratio < 0.6 {
		res.Suggestions = append(res.Suggestions, "Start bullets with strong action verbs (Led, Built, Reduced, Scaled, …).")
	}
	return score
}

// scoreSections rewards presence of the key resume sections.
func (s *RuleScorer) scoreSections(in ScoreInput, res *ScoreResult) float64 {
	present := 0
	const totalSections = 4
	var missing []string
	if strings.TrimSpace(in.Summary) != "" {
		present++
	} else {
		missing = append(missing, "summary")
	}
	if len(in.Skills) > 0 {
		present++
	} else {
		missing = append(missing, "skills")
	}
	if in.Projects > 0 {
		present++
	} else {
		missing = append(missing, "projects")
	}
	if strings.TrimSpace(in.Headline) != "" {
		present++
	} else {
		missing = append(missing, "headline")
	}
	score := round2(float64(present) / float64(totalSections) * maxSections)
	res.Breakdown = append(res.Breakdown, ScoreBreakdown{
		Rule: "sections_present", Score: score, Max: maxSections,
		Detail: detailf("%d of %d key sections present (headline, summary, skills, projects)", present, totalSections),
	})
	if len(missing) > 0 {
		res.Suggestions = append(res.Suggestions, "Add the missing resume sections: "+strings.Join(missing, ", ")+".")
	}
	return score
}

// scoreBulletLength rewards bullets that fall within an ideal word band (8..30
// words). Too-short bullets lack substance; too-long bullets hurt scannability.
func (s *RuleScorer) scoreBulletLength(in ScoreInput, res *ScoreResult) float64 {
	if len(in.Bullets) == 0 {
		res.Breakdown = append(res.Breakdown, ScoreBreakdown{
			Rule: "bullet_length", Score: 0, Max: maxBulletLen,
			Detail: "no project bullets found",
		})
		return 0
	}
	const minWords, maxWords = 8, 30
	ideal := 0
	for _, b := range in.Bullets {
		n := len(strings.Fields(b))
		if n >= minWords && n <= maxWords {
			ideal++
		}
	}
	ratio := float64(ideal) / float64(len(in.Bullets))
	score := round2(ratio * maxBulletLen)
	res.Breakdown = append(res.Breakdown, ScoreBreakdown{
		Rule: "bullet_length", Score: score, Max: maxBulletLen,
		Detail: detailf("%d of %d bullets are within the ideal %d-%d word range", ideal, len(in.Bullets), minWords, maxWords),
	})
	if ratio < 0.6 {
		res.Suggestions = append(res.Suggestions, detailf("Keep bullets between %d and %d words for scannability.", minWords, maxWords))
	}
	return score
}

// --- helpers ---

// buildTokenSet collects the lowercase word tokens across the resume's
// searchable text for single-word keyword matching.
func buildTokenSet(in ScoreInput) map[string]struct{} {
	set := map[string]struct{}{}
	add := func(text string) {
		for _, w := range wordRe.FindAllString(strings.ToLower(text), -1) {
			set[w] = struct{}{}
		}
	}
	add(in.Headline)
	add(in.Summary)
	for _, s := range in.Skills {
		add(s)
	}
	for _, b := range in.Bullets {
		add(b)
	}
	return set
}

// keywordPresent reports whether a (possibly multi-word) keyword is present.
// Multi-word keywords are matched as a substring against the joined corpus;
// single-word keywords match the token set.
func keywordPresent(kw string, tokens map[string]struct{}, in ScoreInput) bool {
	kw = strings.ToLower(strings.TrimSpace(kw))
	if kw == "" {
		return false
	}
	if strings.ContainsAny(kw, " /") || strings.Contains(kw, "-") {
		corpus := strings.ToLower(strings.Join(append([]string{in.Headline, in.Summary},
			append(append([]string{}, in.Skills...), in.Bullets...)...), " "))
		return strings.Contains(corpus, kw)
	}
	_, ok := tokens[kw]
	return ok
}

// startsWithActionVerb reports whether the first word of a bullet is a known
// strong action verb.
func startsWithActionVerb(bullet string) bool {
	fields := wordRe.FindAllString(strings.ToLower(strings.TrimSpace(bullet)), 1)
	if len(fields) == 0 {
		return false
	}
	_, ok := actionVerbs[fields[0]]
	return ok
}

// normalizeUnique lowercases, trims, and de-duplicates while preserving order.
func normalizeUnique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// round2 rounds to two decimal places.
func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}

// detailf is a small Sprintf alias kept local so the rule code stays terse.
func detailf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}
