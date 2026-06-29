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

	// The Backend Engineering pillar must be seeded with a substantial topic set
	// (>=20), and re-running must not change the count (idempotency at the row
	// level for this pillar specifically).
	beCount := func() int64 {
		var n int64
		if err := db.WithContext(ctx).Table("topics t").
			Joins("JOIN pillars p ON p.id = t.pillar_id").
			Where("p.type = ? AND t.deleted_at IS NULL", "backend_engineering").
			Count(&n).Error; err != nil {
			t.Fatalf("counting backend_engineering topics: %v", err)
		}
		return n
	}
	if got := beCount(); got < 20 {
		t.Errorf("expected >=20 backend_engineering topics after seed, got %d", got)
	}
	before := beCount()
	if _, err := s.Run(ctx); err != nil {
		t.Fatalf("third seed run: %v", err)
	}
	if after := beCount(); after != before {
		t.Errorf("backend_engineering topic count not idempotent: before=%d after=%d", before, after)
	}
}
