package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoad_BcryptCostClampedToFloor verifies the NFR-SEC bcrypt floor: a value
// below 12 (here 4) is clamped up, and an unset value defaults to >= 12.
func TestLoad_BcryptCostClampedToFloor(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("BCRYPT_COST", "4")

	cfg, err := Load()
	require.NoError(t, err)
	require.GreaterOrEqual(t, cfg.BcryptCost, MinBcryptCost)
	require.Equal(t, MinBcryptCost, cfg.BcryptCost)
}

func TestLoad_BcryptCostHonoredWhenAboveFloor(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("BCRYPT_COST", "14")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 14, cfg.BcryptCost)
}

func TestLoad_RateLimitDefaults(t *testing.T) {
	t.Setenv("ENV", "development")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 60, cfg.RateLimitPerMin)
	require.Equal(t, 120, cfg.UserRateLimitPerMin)
	require.Equal(t, 10, cfg.AuthRateLimitPerMin)
	require.GreaterOrEqual(t, cfg.BcryptCost, MinBcryptCost)
}
