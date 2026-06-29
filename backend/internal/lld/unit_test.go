package lld

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizePageBounds(t *testing.T) {
	if p := normalizePage(0, 0); p.Page != 1 || p.PageSize != defaultPageSize {
		t.Errorf("defaults not applied: %+v", p)
	}
	if p := normalizePage(3, 1000); p.PageSize != maxPageSize {
		t.Errorf("page size not clamped: %+v", p)
	}
	if p := normalizePage(2, 5); p.Offset() != 5 {
		t.Errorf("expected offset 5, got %d", p.Offset())
	}
}

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{0, 20, 0},
		{7, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{40, 20, 2},
	}
	for _, c := range cases {
		if got := totalPages(c.total, c.pageSize); got != c.want {
			t.Errorf("totalPages(%d,%d)=%d want %d", c.total, c.pageSize, got, c.want)
		}
	}
}

func TestParseSortAllowlist(t *testing.T) {
	got := parseSort("-order_index,title,evil_column", problemSortable)
	if len(got) != 2 {
		t.Fatalf("expected 2 valid sort fields, got %d (%+v)", len(got), got)
	}
	if got[0].Column != "order_index" || !got[0].Desc {
		t.Errorf("expected -order_index first, got %+v", got[0])
	}
	if got[1].Column != "title" || got[1].Desc {
		t.Errorf("expected title asc second, got %+v", got[1])
	}
}

func TestDifficultyParam(t *testing.T) {
	if d := difficultyParam("hard"); d == nil || *d != DifficultyHard {
		t.Errorf("expected hard, got %v", d)
	}
	if d := difficultyParam("impossible"); d != nil {
		t.Errorf("expected nil for invalid difficulty, got %v", *d)
	}
	if d := difficultyParam(""); d != nil {
		t.Errorf("expected nil for empty difficulty, got %v", *d)
	}
}

func TestJSONArrayRoundTrip(t *testing.T) {
	a := JSONArray{"Strategy", "Factory"}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var got JSONArray
	if err := got.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 2 || got[0] != "Strategy" || got[1] != "Factory" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	// nil slice persists as an empty JSON array, and a NULL scans to empty.
	var empty JSONArray
	v, _ = empty.Value()
	if v != "[]" {
		t.Errorf("expected nil JSONArray to serialize as '[]', got %v", v)
	}
	var fromNull JSONArray
	if err := fromNull.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if len(fromNull) != 0 {
		t.Errorf("expected empty slice from NULL, got %+v", fromNull)
	}
}

func TestToProblemDetailResponseMapping(t *testing.T) {
	req := "Park and unpark vehicles."
	prob := &Problem{
		ID:                uuid.New(),
		TrackID:           uuid.New(),
		Slug:              "parking-lot",
		Title:             "Design a Parking Lot",
		Difficulty:        DifficultyMedium,
		OrderIndex:        1,
		RequirementsMD:    &req,
		DesignPatterns:    JSONArray{"Strategy", "Factory"},
		FollowUpQuestions: nil, // must serialize as [], not null
	}
	got := toProblemDetailResponse(prob)
	if got.Slug != "parking-lot" || got.Difficulty != "medium" || got.OrderIndex != 1 {
		t.Errorf("summary fields mismatch: %+v", got.problemResponse)
	}
	if got.RequirementsMD == nil || *got.RequirementsMD != req {
		t.Errorf("requirements not mapped: %v", got.RequirementsMD)
	}
	if len(got.DesignPatterns) != 2 {
		t.Errorf("expected 2 design patterns, got %d", len(got.DesignPatterns))
	}
	if got.FollowUpQuestions == nil {
		t.Error("follow_up_questions must be a non-nil empty slice for JSON []")
	}
}
