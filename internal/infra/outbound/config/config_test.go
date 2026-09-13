package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/config"
)

func Test_Load(t *testing.T) {
	t.Run("should return the internal thresholds used by the treatment pipeline", func(t *testing.T) {
		// given: no external configuration source exists yet (research.md item 9)

		// when
		cfg := config.Load()

		// then
		assert.Equal(t, 2, cfg.MinPoints)
		assert.Equal(t, 130.0, cfg.MaxPlausibleSpeedKmh)
		assert.Equal(t, domain.LevelMedium, cfg.DefaultLevel)
	})
}
