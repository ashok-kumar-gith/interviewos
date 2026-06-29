// Command migrate is the database migration runner. It applies the versioned
// SQL migrations under backend/migrations/ using golang-migrate (file source +
// postgres driver), tracking applied versions in the schema_migrations table.
//
// Usage:
//
//	migrate up        # apply all pending migrations (default)
//	migrate down      # roll back the most recent migration (one step)
//	migrate down-all  # roll back every migration
//	migrate version   # print the current migration version
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/interviewos/backend/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = strings.ToLower(os.Args[1])
	}

	sourceURL, err := migrationsSourceURL()
	if err != nil {
		return err
	}

	// golang-migrate's postgres driver expects a postgres:// (or pgx5://) DSN.
	m, err := migrate.New(sourceURL, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			fmt.Fprintln(os.Stderr, "migrate: closing source:", srcErr)
		}
		if dbErr != nil {
			fmt.Fprintln(os.Stderr, "migrate: closing db:", dbErr)
		}
	}()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		// Roll back a single step (safe default for an interactive down).
		err = m.Steps(-1)
	case "down-all":
		err = m.Down()
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			if errors.Is(verr, migrate.ErrNilVersion) {
				fmt.Println("migrate: no migrations applied yet")
				return nil
			}
			return fmt.Errorf("reading version: %w", verr)
		}
		fmt.Printf("migrate: version=%d dirty=%t\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("unknown command %q (want up|down|down-all|version)", cmd)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("migrate: no change; database is up to date")
			return nil
		}
		return fmt.Errorf("%s: %w", cmd, err)
	}

	fmt.Printf("migrate: %s complete\n", cmd)
	return nil
}

// migrationsSourceURL resolves the migrations directory relative to this source
// file so the runner works regardless of the process working directory, then
// falls back to ./migrations.
func migrationsSourceURL() (string, error) {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "migrations"),
			filepath.Join(wd, "..", "..", "migrations"),
		)
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			abs, err := filepath.Abs(dir)
			if err != nil {
				return "", err
			}
			return "file://" + abs, nil
		}
	}
	return "", fmt.Errorf("could not locate migrations directory (looked in %s)", strings.Join(candidates, ", "))
}
