package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
)

func Test_ComputeBoundingBox(t *testing.T) {
	t.Run("should return the zero value for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.Equal(t, domain.BoundingBox{}, boundingBox)
	})

	t.Run("should compute a direct bounding box for a route crossing neither the antimeridian nor the equator", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(40.0).WithLongitude(-3.0).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(40.5).WithLongitude(-3.5).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(40.2).WithLongitude(-3.2).Build(),
		}

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.False(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, 40.0, boundingBox.MinLatitude, 0.0001)
		assert.InDelta(t, 40.5, boundingBox.MaxLatitude, 0.0001)
		assert.InDelta(t, -3.5, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -3.0, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should not confuse crossing the equator with crossing the antimeridian", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(1.0).WithLongitude(10.0).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(-1.0).WithLongitude(10.5).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(10.2).Build(),
		}

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.False(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, -1.0, boundingBox.MinLatitude, 0.0001)
		assert.InDelta(t, 1.0, boundingBox.MaxLatitude, 0.0001)
		assert.InDelta(t, 10.0, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, 10.5, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should compute a narrow occupied area for a route heading east across the antimeridian", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.8).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.8).Build(),
		}

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.True(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, 179.8, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -179.8, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should compute the same occupied area for a route heading west across the antimeridian", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.8).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.8).Build(),
		}

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.True(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, 179.8, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -179.8, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should detect crossing the antimeridian even when the route also crosses the equator", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(179.9).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(-0.5).WithLongitude(-179.9).Build(),
		}

		// when
		boundingBox := domain.ComputeBoundingBox(points)

		// then
		assert.True(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, -0.5, boundingBox.MinLatitude, 0.0001)
		assert.InDelta(t, 0.5, boundingBox.MaxLatitude, 0.0001)
	})
}
