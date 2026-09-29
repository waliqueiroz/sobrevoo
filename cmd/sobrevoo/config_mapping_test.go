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
		defaults := config.PlanDefaults{FrameRate: 24, Distance: config.LevelHigh, Tilt: config.LevelLow, AspectWidth: 9, AspectHeight: 16}

		// when
		parameters := domainPlanParameters(defaults)

		// then
		assert.Nil(t, parameters.Duration)
		assert.Equal(t, 24.0, parameters.FrameRate)
		assert.Equal(t, domain.LevelHigh, parameters.Distance)
		assert.Equal(t, domain.LevelLow, parameters.Tilt)
		assert.Equal(t, domain.AspectRatio{Width: 9, Height: 16}, parameters.Aspect)
	})
}

func Test_domainSliceTuning(t *testing.T) {
	t.Run("should map the configuration's slice tuning to the values the domain tests are built on", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		tuning := domainSliceTuning(cfg.SliceTuning)

		// then
		assert.Equal(t, builddomain.NewSliceTuningBuilder().Build(), tuning)
	})
}

func Test_domainRenderTuning(t *testing.T) {
	t.Run("should map the configuration's render tuning to the values the domain tests are built on", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		tuning := domainRenderTuning(cfg.RenderTuning)

		// then: the number of goroutines is the one of the machine
		assert.Equal(t, builddomain.NewRenderTuningBuilder().WithWorkers(cfg.RenderTuning.Workers).Build(), tuning)
	})
}

func Test_domainRenderResolution(t *testing.T) {
	t.Run("should map the default resolution to a valid one", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		resolution, err := domainRenderResolution(cfg.RenderDefaults)

		// then
		require.NoError(t, err)
		assert.Equal(t, domain.Resolution{Width: 1080, Height: 1920}, resolution)
	})

	t.Run("should refuse a default resolution outside the limits", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{Width: 1921, Height: 1080}

		// when
		_, err := domainRenderResolution(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidResolution)
	})
}

func Test_domainAppearance(t *testing.T) {
	t.Run("should map the default appearance to the tool's orange trail, red marker and dark background", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		appearance, err := domainAppearance(cfg.RenderDefaults)

		// then
		require.NoError(t, err)
		assert.Equal(t, builddomain.NewAppearanceBuilder().Build(), appearance)
	})

	t.Run("should propagate an invalid trail color", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{TrailColor: "orange", MarkerColor: "#E5252A", BackgroundColor: "#20262E", TrailWidthRatio: 0.005, MarkerRadiusRatio: 0.012}

		// when
		_, err := domainAppearance(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should propagate an invalid marker color", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{TrailColor: "#FFB000", MarkerColor: "not-a-color", BackgroundColor: "#20262E", TrailWidthRatio: 0.005, MarkerRadiusRatio: 0.012}

		// when
		_, err := domainAppearance(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should propagate an invalid background color", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{TrailColor: "#FFB000", MarkerColor: "#E5252A", BackgroundColor: "nope", TrailWidthRatio: 0.005, MarkerRadiusRatio: 0.012}

		// when
		_, err := domainAppearance(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidColor)
	})

	t.Run("should propagate an out-of-range trail width ratio", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{TrailColor: "#FFB000", MarkerColor: "#E5252A", BackgroundColor: "#20262E", TrailWidthRatio: 1, MarkerRadiusRatio: 0.012}

		// when
		_, err := domainAppearance(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidTrailWidth)
	})
}

func Test_domainOverlayConfig(t *testing.T) {
	t.Run("should map the default overlay configuration to enabled, with the four blocks", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when
		overlay, err := domainOverlayConfig(cfg.RenderDefaults)

		// then
		require.NoError(t, err)
		assert.Equal(t, builddomain.NewOverlayConfigBuilder().Build(), overlay)
	})

	t.Run("should turn on only the blocks named", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{OverlaysEnabled: true, OverlayBlocks: []string{"distance", "time"}}

		// when
		overlay, err := domainOverlayConfig(defaults)

		// then
		require.NoError(t, err)
		assert.Equal(t, builddomain.NewOverlayConfigBuilder().WithoutElevation().WithoutProfile().Build(), overlay)
	})

	t.Run("should propagate an unknown block name", func(t *testing.T) {
		// given
		defaults := config.RenderDefaults{OverlaysEnabled: true, OverlayBlocks: []string{"altitude"}}

		// when
		_, err := domainOverlayConfig(defaults)

		// then
		assert.ErrorIs(t, err, domain.ErrInvalidOverlayBlock)
	})
}

func Test_domainVideoQuality(t *testing.T) {
	t.Run("should map the low level to the low quality", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, domain.VideoQualityLow, domainVideoQuality(config.LevelLow))
	})

	t.Run("should map the medium level to the medium quality", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, domain.VideoQualityMedium, domainVideoQuality(config.LevelMedium))
	})

	t.Run("should map the high level to the high quality", func(t *testing.T) {
		// given / when / then
		assert.Equal(t, domain.VideoQualityHigh, domainVideoQuality(config.LevelHigh))
	})

	t.Run("should map the default of the configuration to medium", func(t *testing.T) {
		// given
		cfg, err := config.Load()
		require.NoError(t, err)

		// when / then
		assert.Equal(t, domain.VideoQualityMedium, domainVideoQuality(cfg.VideoDefaults.Quality))
	})
}
