// Command migrate is the database migration runner. At M0 it is a compiling
// stub wired to config; golang-migrate execution over migrations/ lands later.
package main

import (
	"fmt"
	"os"

	"github.com/interviewos/backend/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate: config error:", err)
		os.Exit(1)
	}
	// Reference the resolved DSN so config wiring is exercised at runtime.
	fmt.Printf("migrate: target database configured (env=%s)\n", cfg.Env)
	fmt.Println("no migrations yet")
}
