package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
)

func Test_Load(t *testing.T) {
	t.Run("should return the internal thresholds used by the treatment pipeline", func(t *testing.T) {
		// given: no external configuration source exists yet (research.md item 9)

		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, cfg.MinPoints)
		assert.Equal(t, 130.0, cfg.MaxPlausibleSpeedKmh)
		assert.Equal(t, config.LevelMedium, cfg.DefaultLevel)
	})

	t.Run("should resolve the registry path under the user's home directory regardless of the working directory", func(t *testing.T) {
		// given
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		expected := filepath.Join(home, ".sobrevoo", "registry.json")

		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, expected, cfg.RegistryPath)
	})

	t.Run("should provide the initial camera tuning of research.md", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then: a few of the values; the composition root's test checks the whole set against the domain's
		require.NoError(t, err)
		assert.Equal(t, 0.10, cfg.CameraTuning.OpeningFraction)
		assert.Equal(t, 45.0, cfg.CameraTuning.MaxHeadingRateDegPerSecond)
		assert.Equal(t, 20*time.Second, cfg.CameraTuning.AutoDurationMin)
		assert.Equal(t, 120*time.Second, cfg.CameraTuning.AutoDurationMax)
		assert.Equal(t, 2_000_000.0, cfg.CameraTuning.MaxTrackSpanMeters)
		assert.Equal(t, config.LevelValues{Low: 300, Medium: 600, High: 1200}, cfg.CameraTuning.BaseDistanceMeters)
		assert.Equal(t, config.LevelValues{Low: 2, Medium: 4, High: 8}, cfg.CameraTuning.LookAheadSeconds)
		assert.Equal(t, config.LevelValues{Low: 25, Medium: 45, High: 65}, cfg.CameraTuning.TiltDegrees)
	})

	t.Run("should default the plan to 30 fps, medium distance and medium tilt", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, config.PlanDefaults{FrameRate: 30, Distance: config.LevelMedium, Tilt: config.LevelMedium}, cfg.PlanDefaults)
	})
}
