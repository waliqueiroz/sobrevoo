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

	t.Run("should provide the initial slice tuning of research.md", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, 1.0, cfg.SliceTuning.MarginFactor)
		assert.Equal(t, 1920.0, cfg.SliceTuning.ReferenceHeightPixels)
		assert.Equal(t, 2.0, cfg.SliceTuning.TexelScreenRatio)
		assert.Equal(t, int64(65_536), cfg.SliceTuning.EstimatedTileBytes)
		assert.Equal(t, int64(268_435_456), cfg.SliceTuning.MaxSizeBytes)
	})

	t.Run("should default the plan to 30 fps, medium distance, medium tilt and a vertical 9:16 video", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, config.PlanDefaults{FrameRate: 30, Distance: config.LevelMedium, Tilt: config.LevelMedium, AspectWidth: 9, AspectHeight: 16}, cfg.PlanDefaults)
	})

	t.Run("should provide the initial render tuning of research.md", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, 2.0, cfg.RenderTuning.MinCameraClearanceMeters)
		assert.Equal(t, 1.0, cfg.RenderTuning.MinTiltForTargetDegrees)
		assert.Equal(t, 0.3, cfg.RenderTuning.TrailLiftMeters)
		assert.Equal(t, 1.0, cfg.RenderTuning.DepthBiasMeters)
		assert.Equal(t, 0.002, cfg.RenderTuning.DepthBiasRatio)
		assert.Equal(t, int64(268_435_456), cfg.RenderTuning.TileCacheBytes)
	})

	t.Run("should use one field of view for the camera plan and for drawing", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, 45.0, cfg.RenderTuning.VerticalFOVDegrees)
		assert.Equal(t, cfg.CameraTuning.OverviewVerticalFOVDegrees, cfg.RenderTuning.VerticalFOVDegrees)
	})

	t.Run("should draw with at least one goroutine", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.GreaterOrEqual(t, cfg.RenderTuning.Workers, 1)
	})

	t.Run("should default the resolution of the images to 1080 x 1920, vertical", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, 1080, cfg.RenderDefaults.Width)
		assert.Equal(t, 1920, cfg.RenderDefaults.Height)
	})

	t.Run("should default the appearance to the tool's orange trail, red marker and dark background", func(t *testing.T) {
		// when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, "#FFB000", cfg.RenderDefaults.TrailColor)
		assert.Equal(t, 0.005, cfg.RenderDefaults.TrailWidthRatio)
		assert.Equal(t, "#E5252A", cfg.RenderDefaults.MarkerColor)
		assert.Equal(t, 0.012, cfg.RenderDefaults.MarkerRadiusRatio)
		assert.Equal(t, "#20262E", cfg.RenderDefaults.BackgroundColor)
	})
}

func Test_Load_Video(t *testing.T) {
	t.Run("should default the quality of the video to medium, for publishing", func(t *testing.T) {
		// given / when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, config.VideoDefaults{Quality: config.LevelMedium}, cfg.VideoDefaults)
	})

	t.Run("should look for the video encoder as ffmpeg, on the PATH", func(t *testing.T) {
		// given / when
		cfg, err := config.Load()

		// then
		require.NoError(t, err)
		assert.Equal(t, "ffmpeg", cfg.FFmpegBinary)
	})
}
