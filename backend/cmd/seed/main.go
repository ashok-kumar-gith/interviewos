// Command seed is the idempotent content/curriculum seeder. It loads the
// canonical Backend SDE3 content library (track, pillars, DSA patterns/problems,
// System Design topics, resources, and companies) into the database. Every
// entity is upserted by its natural key so the seeder is safe to re-run.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/interviewos/backend/internal/content"
	"github.com/interviewos/backend/internal/platform/config"
	"github.com/interviewos/backend/internal/platform/database"
	"github.com/interviewos/backend/internal/platform/logger"
	"github.com/interviewos/backend/internal/seed"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	ctx := context.Background()
	db, err := database.NewPostgres(ctx, database.DefaultPostgresConfig(cfg.DatabaseURL))
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer func() { _ = database.ClosePostgres(db) }()

	// Guard: the content tables must exist (migration 000002 applied).
	if !db.Migrator().HasTable(&content.Track{}) {
		return fmt.Errorf("content tables not found; run `migrate up` first")
	}

	seeder := seed.NewSeeder(db, log)
	counts, err := seeder.Run(ctx)
	if err != nil {
		return fmt.Errorf("seeding: %w", err)
	}

	fmt.Printf("seed: complete (env=%s)\n", cfg.Env)
	fmt.Printf("  tracks=%d pillars=%d patterns=%d topics=%d subtopics=%d resources=%d problems=%d companies=%d\n",
		counts.Tracks, counts.Pillars, counts.Patterns, counts.Topics, counts.Subtopics,
		counts.Resources, counts.Problems, counts.Companies)
	return nil
}
