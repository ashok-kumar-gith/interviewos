package lld

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to the DATABASE_URL test database (migrated/seeded),
// skipping when unset so unit-only runs stay green.
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

func TestIntegrationListProblemsSeeded(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	probs, total, err := r.ListProblems(context.Background(), ProblemFilter{}, Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListProblems: %v", err)
	}
	if total < 7 {
		t.Fatalf("expected >=7 seeded LLD problems, got %d", total)
	}
	if len(probs) < 7 {
		t.Fatalf("expected >=7 problems on the page, got %d", len(probs))
	}
}

func TestIntegrationListProblemsPagination(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	page1, total, err := r.ListProblems(ctx, ProblemFilter{}, Page{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("ListProblems page1: %v", err)
	}
	if total < 7 {
		t.Fatalf("expected >=7 problems total, got %d", total)
	}
	if len(page1) != 3 {
		t.Fatalf("expected page size 3, got %d", len(page1))
	}

	page2, _, err := r.ListProblems(ctx, ProblemFilter{}, Page{Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("ListProblems page2: %v", err)
	}
	if len(page2) == 0 {
		t.Fatal("expected a non-empty second page")
	}
	seen := map[string]struct{}{}
	for _, p := range page1 {
		seen[p.Slug] = struct{}{}
	}
	for _, p := range page2 {
		if _, dup := seen[p.Slug]; dup {
			t.Errorf("page 2 overlaps page 1 on slug %q", p.Slug)
		}
	}
}

func TestIntegrationListProblemsFilterByDifficulty(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	hard := DifficultyHard
	probs, total, err := r.ListProblems(context.Background(), ProblemFilter{Difficulty: &hard}, Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListProblems hard: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one hard LLD problem")
	}
	for _, p := range probs {
		if p.Difficulty != DifficultyHard {
			t.Errorf("non-hard problem %q in difficulty filter result", p.Slug)
		}
	}
}

func TestIntegrationListProblemsSearch(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	probs, total, err := r.ListProblems(context.Background(), ProblemFilter{Query: "parking"}, Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListProblems q=parking: %v", err)
	}
	if total == 0 {
		t.Fatal("expected a search hit for 'parking'")
	}
	found := false
	for _, p := range probs {
		if p.Slug == "parking-lot" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'parking-lot' in search results for 'parking'")
	}
}

func TestIntegrationGetProblemBySlug(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	prob, err := r.GetProblemBySlug(ctx, "splitwise")
	if err != nil {
		t.Fatalf("GetProblemBySlug splitwise: %v", err)
	}
	if prob.Title == "" {
		t.Error("expected a non-empty title")
	}
	if len(prob.DesignPatterns) == 0 {
		t.Error("expected splitwise to list >=1 design pattern")
	}
	if len(prob.FollowUpQuestions) == 0 {
		t.Error("expected splitwise to list >=1 follow-up question")
	}
	if prob.RequirementsMD == nil || *prob.RequirementsMD == "" {
		t.Error("expected splitwise to have requirements content")
	}

	// Round-trip by id.
	byID, err := r.GetProblem(ctx, prob.ID)
	if err != nil {
		t.Fatalf("GetProblem by id: %v", err)
	}
	if byID.Slug != "splitwise" {
		t.Errorf("expected slug 'splitwise', got %q", byID.Slug)
	}
}

func TestIntegrationGetProblemNotFound(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	if _, err := r.GetProblemBySlug(context.Background(), "does-not-exist"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
