package designproblems

import "testing"

func TestNormalizePageBounds(t *testing.T) {
	if p := normalizePage(0, 0); p.Page != 1 || p.PageSize != defaultPageSize {
		t.Errorf("defaults not applied: %+v", p)
	}
	if p := normalizePage(-5, -1); p.Page != 1 || p.PageSize != defaultPageSize {
		t.Errorf("negative inputs not normalized: %+v", p)
	}
	if p := normalizePage(3, 1000); p.PageSize != maxPageSize {
		t.Errorf("page size not clamped: %+v", p)
	}
	if p := normalizePage(2, 5); p.Page != 2 || p.PageSize != 5 {
		t.Errorf("valid inputs altered: %+v", p)
	}
}

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{0, 10, 0},
		{10, 10, 1},
		{11, 10, 2},
		{6, 3, 2},
		{6, 0, 0},
	}
	for _, c := range cases {
		if got := totalPages(c.total, c.pageSize); got != c.want {
			t.Errorf("totalPages(%d,%d)=%d want %d", c.total, c.pageSize, got, c.want)
		}
	}
}

func TestParseSortAllowlist(t *testing.T) {
	got := parseSort("-order_index,title,evil_column", designProblemSortable)
	if len(got) != 2 {
		t.Fatalf("expected 2 valid sort fields, got %d (%+v)", len(got), got)
	}
	if got[0].Column != "order_index" || !got[0].Desc {
		t.Errorf("expected -order_index first, got %+v", got[0])
	}
	if got[1].Column != "title" || got[1].Desc {
		t.Errorf("expected title asc second, got %+v", got[1])
	}
	if parseSort("", designProblemSortable) != nil {
		t.Error("empty sort should yield nil")
	}
}

func TestDifficultyParam(t *testing.T) {
	if d := difficultyParam("hard"); d == nil || *d != DifficultyHard {
		t.Errorf("expected hard, got %v", d)
	}
	if d := difficultyParam("medium"); d == nil || *d != DifficultyMedium {
		t.Errorf("expected medium, got %v", d)
	}
	if d := difficultyParam(""); d != nil {
		t.Errorf("empty should be nil, got %v", *d)
	}
	if d := difficultyParam("impossible"); d != nil {
		t.Errorf("invalid should be nil, got %v", *d)
	}
}

// TestToDetailMapping verifies the model -> detail DTO mapping, including the
// nil follow-up-questions normalization to an empty slice.
func TestToDetailMapping(t *testing.T) {
	req := "reqs"
	dp := DesignProblem{
		Slug:           "url-shortener",
		Title:          "Design a URL Shortener",
		Difficulty:     DifficultyEasy,
		OrderIndex:     1,
		RequirementsMD: &req,
		// FollowUpQuestions intentionally nil.
	}
	out := toDesignProblemDetailResponse(&dp)
	if out.Slug != "url-shortener" || out.Difficulty != "easy" || out.OrderIndex != 1 {
		t.Errorf("base fields mismapped: %+v", out.designProblemResponse)
	}
	if out.RequirementsMD == nil || *out.RequirementsMD != "reqs" {
		t.Errorf("requirements_md mismapped: %v", out.RequirementsMD)
	}
	if out.FollowUpQuestions == nil {
		t.Error("nil follow_up_questions must normalize to empty slice for JSON []")
	}
	if len(out.FollowUpQuestions) != 0 {
		t.Errorf("expected empty follow-ups, got %v", out.FollowUpQuestions)
	}
}
