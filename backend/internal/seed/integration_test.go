package seed

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to the DATABASE_URL test database, skipping the test when
// it is unset (so unit-only runs stay green).
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

// TestSeedIsIdempotent runs the seeder twice and asserts the row counts are
// identical after the second run (no duplicates introduced).
func TestSeedIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s := NewSeeder(db, zap.NewNop())

	first, err := s.Run(ctx)
	if err != nil {
		t.Fatalf("first seed run: %v", err)
	}
	second, err := s.Run(ctx)
	if err != nil {
		t.Fatalf("second seed run: %v", err)
	}

	if first != second {
		t.Errorf("seed not idempotent: first=%+v second=%+v", first, second)
	}
	if second.Tracks != 1 {
		t.Errorf("expected 1 track, got %d", second.Tracks)
	}
	if second.Pillars != 6 {
		t.Errorf("expected 6 pillars, got %d", second.Pillars)
	}
	if second.Problems < 60 {
		t.Errorf("expected >=60 problems, got %d", second.Problems)
	}
	if second.Companies != 10 {
		t.Errorf("expected 10 companies, got %d", second.Companies)
	}
}
