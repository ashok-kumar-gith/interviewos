package seed

import "testing"

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
