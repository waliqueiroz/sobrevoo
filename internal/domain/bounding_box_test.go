package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_Route_BoundingBox(t *testing.T) {
	t.Run("should return the zero value for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

		// then
		assert.Equal(t, domain.BoundingBox{}, boundingBox)
	})

	t.Run("should compute a direct bounding box for a route crossing neither the antimeridian nor the equator", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(40.0).WithLongitude(-3.0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(40.5).WithLongitude(-3.5).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(40.2).WithLongitude(-3.2).Build(),
		}

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

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
			builddomain.NewTrackPointBuilder().WithLatitude(1.0).WithLongitude(10.0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(-1.0).WithLongitude(10.5).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(10.2).Build(),
		}

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

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
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.8).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.8).Build(),
		}

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

		// then
		assert.True(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, 179.8, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -179.8, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should compute the same occupied area for a route heading west across the antimeridian", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.8).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.8).Build(),
		}

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

		// then
		assert.True(t, boundingBox.CrossesAntimeridian)
		assert.InDelta(t, 179.8, boundingBox.MinLongitude, 0.0001)
		assert.InDelta(t, -179.8, boundingBox.MaxLongitude, 0.0001)
	})

	t.Run("should detect crossing the antimeridian even when the route also crosses the equator", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0.5).WithLongitude(179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(-0.5).WithLongitude(-179.9).Build(),
		}

		// when
		boundingBox := (domain.Route{Points: points}).BoundingBox()

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

func Test_BoundingBox_Intersects(t *testing.T) {
	t.Run("should report boxes that overlap", func(t *testing.T) {
		// when / then
		assert.True(t, box(0, 2, 0, 2).Intersects(box(1, 3, 1, 3)))
	})

	t.Run("should report a box that contains another", func(t *testing.T) {
		// when / then
		assert.True(t, box(0, 10, 0, 10).Intersects(box(4, 5, 4, 5)))
		assert.True(t, box(4, 5, 4, 5).Intersects(box(0, 10, 0, 10)))
	})

	t.Run("should report boxes that only touch at an edge", func(t *testing.T) {
		// when / then
		assert.True(t, box(0, 1, 0, 1).Intersects(box(1, 2, 1, 2)))
	})

	t.Run("should not report disjoint boxes", func(t *testing.T) {
		// when / then
		assert.False(t, box(0, 1, 0, 1).Intersects(box(2, 3, 0, 1)))
		assert.False(t, box(0, 1, 0, 1).Intersects(box(0, 1, 2, 3)))
	})

	t.Run("should intersect a box crossing the antimeridian with one that does not, when they share longitudes", func(t *testing.T) {
		// given
		crossing := box(0, 1, 170, -170)

		// when / then
		assert.True(t, crossing.Intersects(box(0, 1, 175, 179)))
		assert.True(t, crossing.Intersects(box(0, 1, -179, -175)))
		assert.False(t, crossing.Intersects(box(0, 1, -100, 100)))
	})

	t.Run("should intersect two boxes that both cross the antimeridian", func(t *testing.T) {
		// when / then
		assert.True(t, box(0, 1, 170, -170).Intersects(box(0, 1, 175, -175)))
		assert.True(t, box(0, 1, 170, -175).Intersects(box(0, 1, 178, -160)))
	})
}

func Test_BoundingBox_TileRange(t *testing.T) {
	t.Run("should cover the whole world with a single tile at level 0", func(t *testing.T) {
		// when
		ranges := box(-50, 50, -100, 100).TileRange(0)

		// then
		assert.Equal(t, []domain.TileRange{{Level: 0, MinX: 0, MaxX: 0, MinY: 0, MaxY: 0}}, ranges)
	})

	t.Run("should find the tile that contains a point", func(t *testing.T) {
		// when
		equator := box(0, 0, 0, 0).TileRange(1)
		saoPaulo := box(-23.55, -23.55, -46.63, -46.63).TileRange(10)

		// then
		assert.Equal(t, []domain.TileRange{{Level: 1, MinX: 1, MaxX: 1, MinY: 1, MaxY: 1}}, equator)
		assert.Equal(t, []domain.TileRange{{Level: 10, MinX: 379, MaxX: 379, MinY: 580, MaxY: 580}}, saoPaulo)
	})

	t.Run("should number rows from the north", func(t *testing.T) {
		// when
		ranges := box(-23.6, -23.5, -46.7, -46.5).TileRange(12)

		// then
		assert.Equal(t, []domain.TileRange{{Level: 12, MinX: 1516, MaxX: 1518, MinY: 2323, MaxY: 2324}}, ranges)
	})

	t.Run("should limit the latitudes to the ones Web Mercator covers", func(t *testing.T) {
		// when
		north := box(89, 90, 0, 0).TileRange(3)
		south := box(-90, -89, 0, 0).TileRange(3)

		// then
		assert.Equal(t, []domain.TileRange{{Level: 3, MinX: 4, MaxX: 4, MinY: 0, MaxY: 0}}, north)
		assert.Equal(t, []domain.TileRange{{Level: 3, MinX: 4, MaxX: 4, MinY: 7, MaxY: 7}}, south)
	})

	t.Run("should give two ranges of columns for a box crossing the antimeridian", func(t *testing.T) {
		// when
		ranges := box(0, 1, 175, -175).TileRange(4)

		// then
		assert.Equal(t, []domain.TileRange{
			{Level: 4, MinX: 15, MaxX: 15, MinY: 7, MaxY: 8},
			{Level: 4, MinX: 0, MaxX: 0, MinY: 7, MaxY: 8},
		}, ranges)
	})

	t.Run("should keep the last column for a longitude of exactly 180", func(t *testing.T) {
		// when
		ranges := box(0, 0, 180, 180).TileRange(2)

		// then
		assert.Equal(t, []domain.TileRange{{Level: 2, MinX: 3, MaxX: 3, MinY: 2, MaxY: 2}}, ranges)
	})

	t.Run("should need four times as many tiles for each level", func(t *testing.T) {
		// given
		count := func(ranges []domain.TileRange) int {
			total := 0
			for _, r := range ranges {
				total += (r.MaxX - r.MinX + 1) * (r.MaxY - r.MinY + 1)
			}
			return total
		}
		area := box(-10, 10, -20, 20)

		// when
		lower := count(area.TileRange(8))
		higher := count(area.TileRange(9))

		// then
		assert.InDelta(t, 4*float64(lower), float64(higher), 0.15*float64(higher))
	})
}

func Test_BoundingBox_Extent(t *testing.T) {
	t.Run("should measure the height by the latitude span and the width by the longitude span at the middle latitude", func(t *testing.T) {
		// when
		width, height := box(-0.5, 0.5, 10, 11).Extent()

		// then
		assert.InDelta(t, 111.32, height, 0.01)
		assert.InDelta(t, 111.32, width, 0.05)
	})

	t.Run("should shrink the width with the latitude", func(t *testing.T) {
		// when
		width, _ := box(59.5, 60.5, 10, 11).Extent()

		// then
		assert.InDelta(t, 111.32*0.5, width, 0.3)
	})

	t.Run("should measure an area that crosses the antimeridian by its real width", func(t *testing.T) {
		// when
		width, _ := box(-0.5, 0.5, 179.5, -179.5).Extent()

		// then
		assert.InDelta(t, 111.32, width, 0.05)
	})
}

func Test_BoundingBox_ClippedTo(t *testing.T) {
	t.Run("should keep the part of the box that is inside the limits", func(t *testing.T) {
		// when
		clipped := box(-13.661, 0, -40.036, 0).ClippedTo(box(-13.66, -12.56, -40.04, -38.08))

		// then
		assert.Equal(t, box(-13.66, -12.56, -40.036, -38.08).MinLatitude, clipped.MinLatitude)
		assert.Equal(t, -12.56, clipped.MaxLatitude)
		assert.Equal(t, -40.036, clipped.MinLongitude)
		assert.Equal(t, -38.08, clipped.MaxLongitude)
	})

	t.Run("should leave a box that is inside the limits as it is", func(t *testing.T) {
		// given
		inner := box(1, 2, 3, 4)

		// when
		clipped := inner.ClippedTo(box(0, 10, 0, 10))

		// then
		assert.Equal(t, inner, clipped)
	})

	t.Run("should leave the box as it is when the limits do not overlap it", func(t *testing.T) {
		// given
		original := box(1, 2, 3, 4)

		// when
		clipped := original.ClippedTo(box(50, 60, 50, 60))

		// then
		assert.Equal(t, original, clipped)
	})

	t.Run("should leave a box that crosses the antimeridian as it is", func(t *testing.T) {
		// given
		crossing := box(0, 1, 170, -170)

		// when
		clipped := crossing.ClippedTo(box(0, 1, 100, 179))

		// then
		assert.Equal(t, crossing, clipped)
	})
}

func Test_BoundingBox_ContainsBox(t *testing.T) {
	box := domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 20, MaxLongitude: 40}

	t.Run("should contain itself and a box inside it, touching an edge included", func(t *testing.T) {
		// given / when / then
		assert.True(t, box.ContainsBox(box))
		assert.True(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -5, MaxLatitude: 5, MinLongitude: 25, MaxLongitude: 35}))
		assert.True(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -10, MaxLatitude: 0, MinLongitude: 20, MaxLongitude: 30}))
	})

	t.Run("should not contain a bigger box, or one that sticks out of it on a single side", func(t *testing.T) {
		// given / when / then
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -11, MaxLatitude: 10, MinLongitude: 20, MaxLongitude: 40}))
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -10, MaxLatitude: 11, MinLongitude: 20, MaxLongitude: 40}))
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 19, MaxLongitude: 40}))
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 20, MaxLongitude: 41}))
	})

	t.Run("should not contain a box next to it", func(t *testing.T) {
		// given / when / then
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: 10, MaxLatitude: 20, MinLongitude: 20, MaxLongitude: 40}))
		assert.False(t, box.ContainsBox(domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 50, MaxLongitude: 60}))
	})

	t.Run("should contain, across the antimeridian, a box on either side of it or crossing it inside", func(t *testing.T) {
		// given
		crossing := domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}

		// when / then
		assert.True(t, crossing.ContainsBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: 175, MaxLongitude: -175, CrossesAntimeridian: true}))
		assert.True(t, crossing.ContainsBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: 179, MaxLongitude: 179.5}))
		assert.True(t, crossing.ContainsBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: -179.5, MaxLongitude: -178}))
		assert.True(t, crossing.ContainsBox(crossing))
		assert.False(t, crossing.ContainsBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: 160, MaxLongitude: 175}))
		assert.False(t, crossing.ContainsBox(domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: 175, MaxLongitude: -165, CrossesAntimeridian: true}))
	})

	t.Run("should not let a box that crosses the antimeridian be inside one that does not", func(t *testing.T) {
		// given
		crossing := domain.BoundingBox{MinLatitude: 0, MaxLatitude: 5, MinLongitude: 175, MaxLongitude: -175, CrossesAntimeridian: true}

		// when / then
		assert.False(t, box.ContainsBox(crossing))
	})

	t.Run("should have the whole world contain any box, and a box that crosses not contain the world", func(t *testing.T) {
		// given
		world := domain.BoundingBox{MinLatitude: -90, MaxLatitude: 90, MinLongitude: -180, MaxLongitude: 180}
		crossing := domain.BoundingBox{MinLatitude: -10, MaxLatitude: 10, MinLongitude: 170, MaxLongitude: -170, CrossesAntimeridian: true}

		// when / then
		assert.True(t, world.ContainsBox(box))
		assert.True(t, world.ContainsBox(crossing))
		assert.False(t, crossing.ContainsBox(world))
	})
}
