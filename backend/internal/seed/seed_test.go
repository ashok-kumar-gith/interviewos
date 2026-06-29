package seed

import (
	"testing"

	"github.com/interviewos/backend/internal/content"
)

// TestCanonicalProblemsAreDeduplicated asserts the in-memory seed set has no
// duplicate slugs and every problem maps to at least one pattern (the schema
// invariant). This runs without a database.
func TestCanonicalProblemsAreDeduplicated(t *testing.T) {
	seen := make(map[string]struct{}, len(canonicalProblems))
	for _, p := range canonicalProblems {
		if _, dup := seen[p.Slug]; dup {
			t.Errorf("duplicate problem slug in seed set: %q", p.Slug)
		}
		seen[p.Slug] = struct{}{}
		if len(p.PatternSlugs) == 0 {
			t.Errorf("problem %q maps to no pattern (schema requires >=1)", p.Slug)
		}
		if len(p.Sources) == 0 {
			t.Errorf("problem %q records no source list", p.Slug)
		}
	}
	if len(seen) < 60 {
		t.Errorf("expected >=60 canonical problems, got %d", len(seen))
	}
}

// TestProblemPatternsReferenceKnownPatterns asserts every pattern slug a problem
// references exists in the canonical pattern set.
func TestProblemPatternsReferenceKnownPatterns(t *testing.T) {
	known := make(map[string]struct{}, len(canonicalPatterns))
	for _, p := range canonicalPatterns {
		known[p.Slug] = struct{}{}
	}
	for _, prob := range canonicalProblems {
		for _, ps := range prob.PatternSlugs {
			if _, ok := known[ps]; !ok {
				t.Errorf("problem %q references unknown pattern %q", prob.Slug, ps)
			}
		}
	}
}

// TestPatternSlugsAreUnique guards against duplicate pattern definitions.
func TestPatternSlugsAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(canonicalPatterns))
	for _, p := range canonicalPatterns {
		if _, dup := seen[p.Slug]; dup {
			t.Errorf("duplicate pattern slug: %q", p.Slug)
		}
		seen[p.Slug] = struct{}{}
	}
}

// TestResourceURLsAreUnique guards the url dedup key for resources.
func TestResourceURLsAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(globalResources))
	for _, r := range globalResources {
		if _, dup := seen[r.URL]; dup {
			t.Errorf("duplicate resource url: %q", r.URL)
		}
		seen[r.URL] = struct{}{}
	}
}

// TestBackendEngineeringTopicsAreWellFormed asserts the Backend Engineering
// pillar has a substantial, unique, fully-populated topic set (no placeholders).
func TestBackendEngineeringTopicsAreWellFormed(t *testing.T) {
	if len(backendEngineeringTopics) < 20 {
		t.Errorf("expected >=20 backend_engineering topics, got %d", len(backendEngineeringTopics))
	}
	seen := make(map[string]struct{}, len(backendEngineeringTopics))
	for _, d := range backendEngineeringTopics {
		if _, dup := seen[d.Slug]; dup {
			t.Errorf("duplicate backend_engineering topic slug: %q", d.Slug)
		}
		seen[d.Slug] = struct{}{}
		if d.Name == "" || d.Summary == "" {
			t.Errorf("topic %q missing name/summary", d.Slug)
		}
		// concept_md must be a real multi-sentence summary, not a placeholder.
		if len(d.Concept) < 80 {
			t.Errorf("topic %q has too-short concept_md (%d chars); expected real content", d.Slug, len(d.Concept))
		}
		switch d.Diff {
		case content.DifficultyEasy, content.DifficultyMedium, content.DifficultyHard:
		default:
			t.Errorf("topic %q has invalid difficulty %q", d.Slug, d.Diff)
		}
	}
}

// TestBackendEngineeringResourceLinksAreKnown asserts every resource slug a BE
// topic links to exists in the global resource set, and topic slugs are known.
func TestBackendEngineeringResourceLinksAreKnown(t *testing.T) {
	resourceSlugs := make(map[string]struct{}, len(globalResources))
	for _, r := range globalResources {
		resourceSlugs[r.Slug] = struct{}{}
	}
	topicSlugs := make(map[string]struct{}, len(backendEngineeringTopics))
	for _, d := range backendEngineeringTopics {
		topicSlugs[d.Slug] = struct{}{}
	}
	for _, l := range beTopicResources {
		if _, ok := topicSlugs[l.topicSlug]; !ok {
			t.Errorf("be topic-resource link references unknown topic %q", l.topicSlug)
		}
		for _, rs := range l.resourceSlugs {
			if _, ok := resourceSlugs[rs]; !ok {
				t.Errorf("be topic %q links unknown resource %q", l.topicSlug, rs)
			}
		}
	}
}
