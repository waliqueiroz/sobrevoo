package domain_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_SliceTuning_DetailLevel(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().Build()
	offered := domain.LevelRange{Min: 0, Max: 22}

	t.Run("should ask for the smallest level whose resolution fits a screen pixel at the nearest distance", func(t *testing.T) {
		// given: 600 m, FOV 45°, at the equator: footprint = 2 · 2 · 600 · tan(22.5°) / 1080 ≈ 0.92 m

		// when
		detail := tuning.DetailLevel(600, 45, box(-0.1, 0.1, 10, 10.2), offered)

		// then
		assert.Equal(t, 18, detail.Ideal)
	})

	t.Run("should ask for a lower level at a latitude where the ground is finer for the same zoom", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(600, 45, box(59.9, 60.1, 10, 10.2), offered)

		// then
		assert.Equal(t, 17, detail.Ideal)
	})

	t.Run("should use the latitude of the area closest to the equator", func(t *testing.T) {
		// when
		southern := tuning.DetailLevel(600, 45, box(-60.1, -59.9, 10, 10.2), offered)
		straddling := tuning.DetailLevel(600, 45, box(-1, 60, 10, 10.2), offered)
		nearEquator := tuning.DetailLevel(600, 45, box(0, 0, 10, 10), offered)

		// then
		assert.Equal(t, 17, southern.Ideal)
		assert.Equal(t, nearEquator.Ideal, straddling.Ideal)
		assert.Equal(t, 18, straddling.Ideal)
	})

	t.Run("should choose the ideal level when the source offers it", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(600, 45, box(-0.1, 0.1, 10, 10.2), offered)

		// then
		assert.Equal(t, detail.Ideal, detail.Chosen)
		assert.Equal(t, 0, detail.Min)
		assert.Equal(t, 22, detail.Max)
	})

	t.Run("should limit the level to the maximum the source offers", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(600, 45, box(-0.1, 0.1, 10, 10.2), domain.LevelRange{Min: 0, Max: 16})

		// then
		assert.Equal(t, 18, detail.Ideal)
		assert.Equal(t, 16, detail.Chosen)
	})

	t.Run("should limit the level to the minimum the source offers", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(600, 45, box(-0.1, 0.1, 10, 10.2), domain.LevelRange{Min: 20, Max: 22})

		// then
		assert.Equal(t, 20, detail.Chosen)
	})

	t.Run("should never ask for a less detailed level for a closer camera", func(t *testing.T) {
		// given
		area := box(-23.6, -23.5, -46.7, -46.6)
		previous := 1 << 30

		for distance := 200.0; distance <= 8000; distance += 100 {
			// when
			detail := tuning.DetailLevel(distance, 45, area, offered)

			// then
			assert.LessOrEqual(t, detail.Ideal, previous, fmt.Sprintf("distance %.0f m", distance))
			previous = detail.Ideal
		}
	})

	t.Run("should be deterministic", func(t *testing.T) {
		// given
		expected := tuning.DetailLevel(437.5, 45, box(-23.6, -23.5, -46.7, -46.6), offered)

		for i := 0; i < 100; i++ {
			// when
			detail := tuning.DetailLevel(437.5, 45, box(-23.6, -23.5, -46.7, -46.6), offered)

			// then
			assert.Equal(t, expected, detail)
		}
	})
}

func Test_SliceTuning_DetailLevel_Reason(t *testing.T) {
	tuning := builddomain.NewSliceTuningBuilder().Build()
	area := box(-23.6, -23.5, -46.7, -46.6)

	t.Run("should say the level is within the range the source offers", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 22})

		// then
		assert.Equal(t, "within the source's range", detail.Reason)
	})

	t.Run("should say the ideal level is above the maximum the source offers", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 14})

		// then
		assert.Equal(t, "above the source's maximum level", detail.Reason)
	})

	t.Run("should say the ideal level is below the minimum the source offers", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 21, Max: 22})

		// then
		assert.Equal(t, "below the source's minimum level", detail.Reason)
	})

	t.Run("should explain how the ideal level was reached", func(t *testing.T) {
		// when
		detail := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 22})

		// then: 2 · 300 · tan(22.5°) / 1080 = 0.23 m per screen pixel, twice that for a tile pixel
		assert.Equal(t, "nearest camera distance 300.0 m, area closest to the equator at latitude 23.50, tiles of at most 0.46 m/px", detail.Explanation)
	})

	t.Run("should keep the explanation whatever the source offers", func(t *testing.T) {
		// when
		narrow := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 10})
		wide := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 22})

		// then
		assert.Equal(t, wide.Explanation, narrow.Explanation)
		assert.Equal(t, wide.Ideal, narrow.Ideal)
	})

	t.Run("should give a closer camera the finer level, with the same explanation format", func(t *testing.T) {
		// when
		near := tuning.DetailLevel(300, 45, area, domain.LevelRange{Min: 0, Max: 22})
		far := tuning.DetailLevel(3000, 45, area, domain.LevelRange{Min: 0, Max: 22})

		// then
		assert.Greater(t, near.Ideal, far.Ideal)
		assert.Contains(t, far.Explanation, "nearest camera distance 3000.0 m")
	})

	t.Run("should never ask for a less detailed level for a closer camera across a sweep of distances", func(t *testing.T) {
		// given
		offered := domain.LevelRange{Min: 0, Max: 22}
		previous := 1 << 30

		for distance := 200.0; distance <= 8000; distance += 100 {
			// when
			detail := tuning.DetailLevel(distance, 45, area, offered)

			// then
			assert.LessOrEqual(t, detail.Ideal, previous, fmt.Sprintf("distance %.0f m", distance))
			previous = detail.Ideal
		}
	})
}
