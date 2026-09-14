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

func Test_BoundingBox_Contains(t *testing.T) {
	t.Run("should report true for a point inside a box that does not cross the antimeridian", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 40.0, MaxLatitude: 50.0, MinLongitude: 10.0, MaxLongitude: 20.0}

		// when
		contains := box.Contains(45.0, 15.0)

		// then
		assert.True(t, contains)
	})

	t.Run("should report false for a point outside a box that does not cross the antimeridian", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 40.0, MaxLatitude: 50.0, MinLongitude: 10.0, MaxLongitude: 20.0}

		// when
		contains := box.Contains(45.0, 25.0)

		// then
		assert.False(t, contains)
	})

	t.Run("should report false for a point whose latitude falls outside the box", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 40.0, MaxLatitude: 50.0, MinLongitude: 10.0, MaxLongitude: 20.0}

		// when
		contains := box.Contains(60.0, 15.0)

		// then
		assert.False(t, contains)
	})

	t.Run("should report true for a point on the far side of a box that crosses the antimeridian", func(t *testing.T) {
		// given: occupied longitude range goes from 170 to 180 and from -180 to -170
		box := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}

		// when
		contains := box.Contains(0, -175.0)

		// then
		assert.True(t, contains)
	})

	t.Run("should report false for a point in the excluded middle of a box that crosses the antimeridian", func(t *testing.T) {
		// given: 0 degrees longitude is on the "inside" arc, not covered by the box
		box := domain.BoundingBox{MinLatitude: -1, MaxLatitude: 1, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}

		// when
		contains := box.Contains(0, 0)

		// then
		assert.False(t, contains)
	})
}

func Test_BoundingBox_AreaDegrees(t *testing.T) {
	t.Run("should compute width times height for a box that does not cross the antimeridian", func(t *testing.T) {
		// given
		box := domain.BoundingBox{MinLatitude: 40.0, MaxLatitude: 50.0, MinLongitude: 10.0, MaxLongitude: 20.0}

		// when
		area := box.AreaDegrees()

		// then
		assert.InDelta(t, 100.0, area, 0.0001)
	})

	t.Run("should unwrap the longitude span for a box that crosses the antimeridian", func(t *testing.T) {
		// given: 10 degrees on each side of the antimeridian (170->180, -180->-170)
		box := domain.BoundingBox{MinLatitude: 0.0, MaxLatitude: 1.0, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}

		// when
		area := box.AreaDegrees()

		// then
		assert.InDelta(t, 20.0, area, 0.0001)
	})

	t.Run("should report a smaller area for a more specific (narrower) box", func(t *testing.T) {
		// given
		wide := domain.BoundingBox{MinLatitude: 0.0, MaxLatitude: 10.0, MinLongitude: 0.0, MaxLongitude: 10.0}
		narrow := domain.BoundingBox{MinLatitude: 0.0, MaxLatitude: 1.0, MinLongitude: 0.0, MaxLongitude: 1.0}

		// when
		wideArea := wide.AreaDegrees()
		narrowArea := narrow.AreaDegrees()

		// then
		assert.Less(t, narrowArea, wideArea)
	})
}
