package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_RenderSummary_Add(t *testing.T) {
	t.Run("should count a frame with no hole as drawn only", func(t *testing.T) {
		// given
		var summary domain.RenderSummary

		// when
		summary.Add(domain.FrameStats{})

		// then
		assert.Equal(t, 1, summary.Drawn)
		assert.Equal(t, 0, summary.MapHoleFrames)
		assert.Equal(t, 0, summary.ElevationHoleFrames)
	})

	t.Run("should count a frame with a map hole in the map holes only", func(t *testing.T) {
		// given
		var summary domain.RenderSummary

		// when
		summary.Add(domain.FrameStats{MapHole: true})

		// then
		assert.Equal(t, 1, summary.MapHoleFrames)
		assert.Equal(t, 0, summary.ElevationHoleFrames)
	})

	t.Run("should count a frame with an elevation hole in the elevation holes only", func(t *testing.T) {
		// given
		var summary domain.RenderSummary

		// when
		summary.Add(domain.FrameStats{ElevationHole: true})

		// then
		assert.Equal(t, 0, summary.MapHoleFrames)
		assert.Equal(t, 1, summary.ElevationHoleFrames)
	})

	t.Run("should count a frame with both holes in both", func(t *testing.T) {
		// given
		var summary domain.RenderSummary

		// when
		summary.Add(domain.FrameStats{MapHole: true, ElevationHole: true})
		summary.Add(domain.FrameStats{})

		// then
		assert.Equal(t, 2, summary.Drawn)
		assert.Equal(t, 1, summary.MapHoleFrames)
		assert.Equal(t, 1, summary.ElevationHoleFrames)
	})

	t.Run("should keep the rest of the fields as they were set", func(t *testing.T) {
		// given
		summary := domain.RenderSummary{
			Requested: 60, Kept: 12, Removed: 3, Interrupted: true,
			Resolution: domain.Resolution{Width: 960, Height: 540}, Elapsed: 5 * time.Second,
		}

		// when
		summary.Add(domain.FrameStats{})

		// then
		assert.Equal(t, 60, summary.Requested)
		assert.Equal(t, 12, summary.Kept)
		assert.Equal(t, 3, summary.Removed)
		assert.True(t, summary.Interrupted)
		assert.Equal(t, 5*time.Second, summary.Elapsed)
	})
}
