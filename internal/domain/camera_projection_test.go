package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func Test_NewLocalPlane(t *testing.T) {
	t.Run("should center a track that crosses the antimeridian near 180 degrees, not near 0", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(20000, 90).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// when
		lat, lon := plane.Unproject(domain.PlanePoint{})

		// then
		assert.InDelta(t, 10.0, lat, 0.01)
		assert.Greater(t, abs(lon), 179.0)
	})

	t.Run("should not collapse the center of a track at a high latitude", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithLine(20000, 45).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// when
		lat, lon := plane.Unproject(domain.PlanePoint{})

		// then
		assert.InDelta(t, 85.0, lat, 0.2)
		assert.InDelta(t, 12.0, lon, 3.0)
	})

	t.Run("should not fail for a single point", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(-23.55).WithLongitude(-46.63).Build()}

		// when
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// then
		lat, lon := plane.Unproject(domain.PlanePoint{})
		assert.InDelta(t, -23.55, lat, 1e-9)
		assert.InDelta(t, -46.63, lon, 1e-9)
	})

	t.Run("should not fail when the points cancel out (antipodes)", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(180).Build(),
		}

		// when
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// then
		lat, lon := plane.Unproject(domain.PlanePoint{})
		assert.InDelta(t, 0.0, lat, 1e-9)
		assert.InDelta(t, 0.0, lon, 1e-9)
	})
}

func Test_LocalPlane_ProjectUnproject(t *testing.T) {
	t.Run("should return to the same coordinate after a round trip at the equator", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(0, 0).WithLine(20000, 45).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		for _, p := range points {
			// when
			lat, lon := plane.Unproject(plane.Project(p.Latitude, p.Longitude))

			// then: less than a millimeter apart, longitude in [-180, 180)
			back := domain.TrackPoint{Latitude: lat, Longitude: lon}
			assert.Less(t, p.DistanceTo(back), 0.001)
			assert.GreaterOrEqual(t, lon, -180.0)
			assert.Less(t, lon, 180.0)
		}
	})

	t.Run("should return to the same coordinate after a round trip at the antimeridian", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(20000, 45).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		for _, p := range points {
			// when
			lat, lon := plane.Unproject(plane.Project(p.Latitude, p.Longitude))

			// then: less than a millimeter apart, longitude in [-180, 180)
			back := domain.TrackPoint{Latitude: lat, Longitude: lon}
			assert.Less(t, p.DistanceTo(back), 0.001)
			assert.GreaterOrEqual(t, lon, -180.0)
			assert.Less(t, lon, 180.0)
		}
	})

	t.Run("should return to the same coordinate after a round trip at latitude 85", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithLine(20000, 45).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		for _, p := range points {
			// when
			lat, lon := plane.Unproject(plane.Project(p.Latitude, p.Longitude))

			// then: less than a millimeter apart, longitude in [-180, 180)
			back := domain.TrackPoint{Latitude: lat, Longitude: lon}
			assert.Less(t, p.DistanceTo(back), 0.001)
			assert.GreaterOrEqual(t, lon, -180.0)
			assert.Less(t, lon, 180.0)
		}
	})

	t.Run("should return to the same coordinate after a round trip at latitude -85", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(-85, -30).WithLine(20000, 45).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		for _, p := range points {
			// when
			lat, lon := plane.Unproject(plane.Project(p.Latitude, p.Longitude))

			// then: less than a millimeter apart, longitude in [-180, 180)
			back := domain.TrackPoint{Latitude: lat, Longitude: lon}
			assert.Less(t, p.DistanceTo(back), 0.001)
			assert.GreaterOrEqual(t, lon, -180.0)
			assert.Less(t, lon, 180.0)
		}
	})
}

func Test_LocalPlane_Project(t *testing.T) {
	t.Run("should preserve distances within 0.1 percent at the equator", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(0, 0).WithLine(20000, 45).WithPointCount(3).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})
		first, last := points[0], points[len(points)-1]

		// when
		a := plane.Project(first.Latitude, first.Longitude)
		b := plane.Project(last.Latitude, last.Longitude)

		// then
		planar := hypot(b.X-a.X, b.Y-a.Y)
		assert.InEpsilon(t, first.DistanceTo(last), planar, 0.001)
	})

	t.Run("should preserve distances within 0.1 percent at the antimeridian", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(10, 179.95).WithLine(20000, 45).WithPointCount(3).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})
		first, last := points[0], points[len(points)-1]

		// when
		a := plane.Project(first.Latitude, first.Longitude)
		b := plane.Project(last.Latitude, last.Longitude)

		// then
		planar := hypot(b.X-a.X, b.Y-a.Y)
		assert.InEpsilon(t, first.DistanceTo(last), planar, 0.001)
	})

	t.Run("should preserve distances within 0.1 percent at latitude 85", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithOrigin(85, 10).WithLine(20000, 45).WithPointCount(3).Build()
		plane := domain.NewLocalPlane(domain.Route{Points: points})
		first, last := points[0], points[len(points)-1]

		// when
		a := plane.Project(first.Latitude, first.Longitude)
		b := plane.Project(last.Latitude, last.Longitude)

		// then
		planar := hypot(b.X-a.X, b.Y-a.Y)
		assert.InEpsilon(t, first.DistanceTo(last), planar, 0.001)
	})

	t.Run("should place a point north of the center at positive Y and east at positive X", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
		}
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// when
		north := plane.Project(0.01, 0)
		east := plane.Project(0, 0.01)

		// then
		assert.Greater(t, north.Y, 0.0)
		assert.InDelta(t, 0.0, north.X, 1e-6)
		assert.Greater(t, east.X, 0.0)
		assert.InDelta(t, 0.0, east.Y, 1e-6)
	})

	t.Run("should project the center to the origin", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(20).Build()}
		plane := domain.NewLocalPlane(domain.Route{Points: points})

		// when
		p := plane.Project(10, 20)

		// then
		assert.Equal(t, domain.PlanePoint{}, p)
	})
}
