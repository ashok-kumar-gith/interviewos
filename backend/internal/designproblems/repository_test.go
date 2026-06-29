package designproblems

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to the DATABASE_URL test database (migrated/seeded),
// skipping when it is unset so unit-only runs stay green.
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

func TestListIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	items, total, err := r.List(ctx, Filter{}, Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 6 {
		t.Fatalf("expected >=6 seeded design problems, got total=%d", total)
	}
	if len(items) < 6 {
		t.Fatalf("expected >=6 items on first page, got %d", len(items))
	}
	// Default ordering is by order_index ascending: URL Shortener first.
	if items[0].Slug != "url-shortener" {
		t.Errorf("expected url-shortener first by order_index, got %q", items[0].Slug)
	}
	// order_index must be non-decreasing under the default sort.
	for i := 1; i < len(items); i++ {
		if items[i].OrderIndex < items[i-1].OrderIndex {
			t.Errorf("order_index not ascending at %d: %d < %d", i, items[i].OrderIndex, items[i-1].OrderIndex)
		}
	}
}

func TestListPaginationIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	page1, total, err := r.List(ctx, Filter{}, Page{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if total < 6 {
		t.Fatalf("expected >=6 total, got %d", total)
	}
	if len(page1) != 3 {
		t.Fatalf("expected page size 3, got %d", len(page1))
	}
	page2, _, err := r.List(ctx, Filter{}, Page{Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) == 0 {
		t.Fatal("expected a non-empty second page")
	}
	seen := map[string]struct{}{}
	for _, d := range page1 {
		seen[d.Slug] = struct{}{}
	}
	for _, d := range page2 {
		if _, dup := seen[d.Slug]; dup {
			t.Errorf("page 2 overlaps page 1 on slug %q", d.Slug)
		}
	}
}

func TestListFilterByDifficultyIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	hard := DifficultyHard
	items, total, err := r.List(context.Background(), Filter{Difficulty: &hard}, Page{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("List hard: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one hard design problem")
	}
	for _, d := range items {
		if d.Difficulty != DifficultyHard {
			t.Errorf("non-hard problem %q in difficulty filter result", d.Slug)
		}
	}
}

func TestListSearchIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	items, total, err := r.List(context.Background(), Filter{Query: "url"}, Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List q=url: %v", err)
	}
	if total == 0 {
		t.Fatal("expected a search hit for 'url'")
	}
	found := false
	for _, d := range items {
		if d.Slug == "url-shortener" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'url-shortener' in search results for 'url'")
	}
}

func TestGetByIDIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	ctx := context.Background()

	dp, err := r.GetBySlug(ctx, "twitter")
	if err != nil {
		t.Fatalf("GetBySlug twitter: %v", err)
	}
	got, err := r.GetByID(ctx, dp.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Slug != "twitter" {
		t.Errorf("expected twitter, got %q", got.Slug)
	}
	// Detail sections must be populated (substantive seed content, not nil).
	if got.RequirementsMD == nil || *got.RequirementsMD == "" {
		t.Error("expected requirements_md to be populated")
	}
	if len(got.FollowUpQuestions) == 0 {
		t.Error("expected follow_up_questions to be populated")
	}
}

func TestGetByIDNotFoundIntegration(t *testing.T) {
	db := openTestDB(t)
	r := NewRepository(db)
	_, err := r.GetBySlug(context.Background(), "does-not-exist")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
