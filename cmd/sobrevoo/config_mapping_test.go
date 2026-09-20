package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
)

func Test_domainLevel(t *testing.T) {
	t.Run("should map each configuration level to the domain level", func(t *testing.T) {
		// when / then
		assert.Equal(t, domain.LevelLow, domainLevel(config.LevelLow))
		assert.Equal(t, domain.LevelMedium, domainLevel(config.LevelMedium))
		assert.Equal(t, domain.LevelHigh, domainLevel(config.LevelHigh))
	})

	t.Run("should fall back to medium for an unknown level", func(t *testing.T) {
		// when / then
		assert.Equal(t, domain.LevelMedium, domainLevel(config.Level("nonsense")))
	})
}

func Test_domainCameraTuning(t *testing.T) {
	t.Run("should map the configuration's camera tuning to the values the domain tests are built on", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		tuning := domainCameraTuning(cfg.CameraTuning)

		// then
		assert.Equal(t, builddomain.NewCameraTuningBuilder().Build(), tuning)
	})

	t.Run("should place each per-level value at the domain level's position", func(t *testing.T) {
		// given
		values := config.LevelValues{Low: 1, Medium: 2, High: 3}

		// when
		table := domainLevelValues(values)

		// then
		assert.Equal(t, 1.0, table[domain.LevelLow])
		assert.Equal(t, 2.0, table[domain.LevelMedium])
		assert.Equal(t, 3.0, table[domain.LevelHigh])
	})
}

func Test_domainPlanParameters(t *testing.T) {
	t.Run("should map the plan defaults, leaving the duration automatic", func(t *testing.T) {
		// given
		defaults := config.PlanDefaults{FrameRate: 24, Distance: config.LevelHigh, Tilt: config.LevelLow}

		// when
		parameters := domainPlanParameters(defaults)

		// then
		assert.Nil(t, parameters.Duration)
		assert.Equal(t, 24.0, parameters.FrameRate)
		assert.Equal(t, domain.LevelHigh, parameters.Distance)
		assert.Equal(t, domain.LevelLow, parameters.Tilt)
	})
}
