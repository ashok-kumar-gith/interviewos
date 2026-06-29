package content

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to the DATABASE_URL test database (seeded), skipping when
// it is unset so unit-only runs stay green.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	return db
}

func TestListTracks(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	tracks, total, err := r.ListTracks(context.Background(), "", nil, Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListTracks: %v", err)
	}
	if total < 1 || len(tracks) < 1 {
		t.Fatalf("expected at least one track, got total=%d len=%d", total, len(tracks))
	}
}

func TestListProblemsPagination(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	page1, total, err := r.ListProblems(ctx, ProblemFilter{}, Page{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProblems page1: %v", err)
	}
	if total < 60 {
		t.Fatalf("expected >=60 problems total, got %d", total)
	}
	if len(page1) != 10 {
		t.Fatalf("expected page size 10, got %d", len(page1))
	}

	page2, _, err := r.ListProblems(ctx, ProblemFilter{}, Page{Page: 2, PageSize: 10})
	if err != nil {
		t.Fatalf("ListProblems page2: %v", err)
	}
	if len(page2) == 0 {
		t.Fatal("expected a non-empty second page")
	}
	// Pages must not overlap.
	first := map[string]struct{}{}
	for _, p := range page1 {
		first[p.Slug] = struct{}{}
	}
	for _, p := range page2 {
		if _, dup := first[p.Slug]; dup {
			t.Errorf("page 2 overlaps page 1 on slug %q", p.Slug)
		}
	}
}

func TestListProblemsFilterByDifficulty(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	easy := DifficultyEasy
	probs, total, err := r.ListProblems(context.Background(), ProblemFilter{Difficulty: &easy}, Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListProblems easy: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one easy problem")
	}
	for _, p := range probs {
		if p.Difficulty != DifficultyEasy {
			t.Errorf("non-easy problem %q in difficulty filter result", p.Slug)
		}
	}
}

func TestListProblemsFilterByCompanySlug(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	_, total, err := r.ListProblems(context.Background(), ProblemFilter{CompanySlug: "amazon"}, Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListProblems company=amazon: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one amazon-frequency problem")
	}
}

func TestListProblemsSearch(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	probs, total, err := r.ListProblems(context.Background(), ProblemFilter{Query: "two sum"}, Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListProblems q=two sum: %v", err)
	}
	if total == 0 {
		t.Fatal("expected a search hit for 'two sum'")
	}
	found := false
	for _, p := range probs {
		if p.Slug == "two-sum" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'two-sum' in search results for 'two sum'")
	}
}

func TestGetProblemBundle(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()
	probs, _, err := r.ListProblems(ctx, ProblemFilter{Query: "two sum"}, Page{Page: 1, PageSize: 1})
	if err != nil || len(probs) == 0 {
		t.Fatalf("seed lookup failed: %v", err)
	}
	b, err := r.GetProblemBundle(ctx, probs[0].ID)
	if err != nil {
		t.Fatalf("GetProblemBundle: %v", err)
	}
	if len(b.Patterns) == 0 {
		t.Error("expected the problem to map to >=1 pattern")
	}
	if len(b.Sources) == 0 {
		t.Error("expected the problem to record >=1 source")
	}
}

func TestNormalizePageBounds(t *testing.T) {
	if p := normalizePage(0, 0); p.Page != 1 || p.PageSize != defaultPageSize {
		t.Errorf("defaults not applied: %+v", p)
	}
	if p := normalizePage(3, 1000); p.PageSize != maxPageSize {
		t.Errorf("page size not clamped: %+v", p)
	}
}

func TestParseSortAllowlist(t *testing.T) {
	got := parseSort("-frequency_score,title,evil_column", problemSortable)
	if len(got) != 2 {
		t.Fatalf("expected 2 valid sort fields, got %d (%+v)", len(got), got)
	}
	if got[0].Column != "frequency_score" || !got[0].Desc {
		t.Errorf("expected -frequency_score first, got %+v", got[0])
	}
	if got[1].Column != "title" || got[1].Desc {
		t.Errorf("expected title asc second, got %+v", got[1])
	}
}

func TestParseFilter(t *testing.T) {
	f := parseFilter("difficulty:hard,company:amazon,bad")
	if f["difficulty"] != "hard" || f["company"] != "amazon" {
		t.Errorf("unexpected filter parse: %+v", f)
	}
	if _, ok := f["bad"]; ok {
		t.Error("malformed filter segment should be dropped")
	}
}
