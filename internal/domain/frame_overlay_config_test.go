package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_NewOverlayConfig(t *testing.T) {
	t.Run("should turn on all four blocks when enabled with the four block names", func(t *testing.T) {
		// given / when
		config, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{
			domain.OverlayBlockDistance, domain.OverlayBlockElevation, domain.OverlayBlockTime, domain.OverlayBlockProfile,
		})

		// then
		require.NoError(t, err)
		assert.True(t, config.Enabled)
		assert.True(t, config.Distance)
		assert.True(t, config.Elevation)
		assert.True(t, config.Time)
		assert.True(t, config.Profile)
	})

	t.Run("should turn on only the blocks named", func(t *testing.T) {
		// given / when
		config, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockDistance, domain.OverlayBlockTime})

		// then
		require.NoError(t, err)
		assert.True(t, config.Distance)
		assert.False(t, config.Elevation)
		assert.True(t, config.Time)
		assert.False(t, config.Profile)
	})

	t.Run("should leave all four blocks off when not enabled, even with blocks named", func(t *testing.T) {
		// given / when
		config, err := domain.NewOverlayConfig(false, []domain.OverlayBlock{
			domain.OverlayBlockDistance, domain.OverlayBlockElevation, domain.OverlayBlockTime, domain.OverlayBlockProfile,
		})

		// then
		require.NoError(t, err)
		assert.False(t, config.Enabled)
		assert.False(t, config.Distance)
		assert.False(t, config.Elevation)
		assert.False(t, config.Time)
		assert.False(t, config.Profile)
	})

	t.Run("should turn on the speed block when named, even though it is not among the four default names", func(t *testing.T) {
		// given / when
		config, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockSpeed})

		// then
		require.NoError(t, err)
		assert.True(t, config.Speed)
		assert.False(t, config.Distance)
		assert.False(t, config.Elevation)
		assert.False(t, config.Time)
		assert.False(t, config.Profile)
	})

	t.Run("should turn on the gain block when named, independent of elevation", func(t *testing.T) {
		// given / when
		config, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{domain.OverlayBlockGain})

		// then
		require.NoError(t, err)
		assert.True(t, config.Gain)
		assert.False(t, config.Elevation)
		assert.False(t, config.Distance)
		assert.False(t, config.Time)
		assert.False(t, config.Profile)
		assert.False(t, config.Speed)
	})

	t.Run("should refuse an unknown block name, naming it and listing the six accepted names", func(t *testing.T) {
		// given / when
		_, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{"altitude"})

		// then
		require.ErrorIs(t, err, domain.ErrInvalidOverlayBlock)
		assert.ErrorContains(t, err, "altitude")
		assert.ErrorContains(t, err, "distance, elevation, gain, time, profile, speed")
	})
}

func Test_OverlayConfig_Fingerprint(t *testing.T) {
	base := func() domain.OverlayConfig {
		config, err := domain.NewOverlayConfig(true, []domain.OverlayBlock{
			domain.OverlayBlockDistance, domain.OverlayBlockElevation, domain.OverlayBlockTime, domain.OverlayBlockProfile,
		})
		require.NoError(t, err)
		return config
	}

	t.Run("should be the same for the same configuration called twice", func(t *testing.T) {
		// given
		config := base()

		// when / then
		assert.Equal(t, config.Fingerprint(), config.Fingerprint())
	})

	t.Run("should change when Enabled changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Enabled = false

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Distance changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Distance = false

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Elevation changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Elevation = false

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Time changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Time = false

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Profile changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Profile = false

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Speed changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Speed = true

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})

	t.Run("should change when Gain changes", func(t *testing.T) {
		// given
		config := base()
		other := config
		other.Gain = true

		// when / then
		assert.NotEqual(t, config.Fingerprint(), other.Fingerprint())
	})
}
