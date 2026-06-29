// Command seed is the idempotent content/curriculum seeder. At M0 it is a
// compiling stub wired to config; seed data under backend/seed/ loads later.
package main

import (
	"fmt"
	"os"

	"github.com/interviewos/backend/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: config error:", err)
		os.Exit(1)
	}
	fmt.Printf("seed: target database configured (env=%s)\n", cfg.Env)
	fmt.Println("no seed yet")
}
