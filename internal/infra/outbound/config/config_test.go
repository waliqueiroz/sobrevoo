package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
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
		assert.Equal(t, domain.LevelMedium, cfg.DefaultLevel)
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
}
