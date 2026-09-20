package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

// sphericalLawOfCosinesMeters computes the great-circle distance between two
// points using a formula different from Haversine's, so tests cross-check
// domain.Haversine's implementation instead of restating it.
func sphericalLawOfCosinesMeters(a, b domain.TrackPoint) float64 {
	const earthRadiusMeters = 6371000.0

	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }

	lat1, lat2 := toRad(a.Latitude), toRad(b.Latitude)
	deltaLon := toRad(b.Longitude - a.Longitude)

	cosCentralAngle := math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(deltaLon)
	// Clamp to [-1, 1] to avoid NaN from acos on floating point overshoot.
	cosCentralAngle = math.Max(-1, math.Min(1, cosCentralAngle))

	return earthRadiusMeters * math.Acos(cosCentralAngle)
}

func Test_TrackPoint_DistanceTo(t *testing.T) {
	t.Run("should return zero for the same point", func(t *testing.T) {
		// given
		point := builddomain.NewTrackPointBuilder().WithLatitude(10).WithLongitude(20).Build()

		// when
		distance := point.DistanceTo(point)

		// then
		assert.Equal(t, 0.0, distance)
	})

	t.Run("should match an independent great-circle formula for two nearby points", func(t *testing.T) {
		// given
		a := builddomain.NewTrackPointBuilder().WithLatitude(40.0).WithLongitude(-3.0).Build()
		b := builddomain.NewTrackPointBuilder().WithLatitude(40.5).WithLongitude(-3.5).Build()

		// when
		distance := a.DistanceTo(b)

		// then
		assert.InDelta(t, sphericalLawOfCosinesMeters(a, b), distance, 1.0)
	})

	t.Run("should match an independent great-circle formula across the equator", func(t *testing.T) {
		// given
		a := builddomain.NewTrackPointBuilder().WithLatitude(1.0).WithLongitude(10.0).Build()
		b := builddomain.NewTrackPointBuilder().WithLatitude(-1.0).WithLongitude(10.0).Build()

		// when
		distance := a.DistanceTo(b)

		// then
		assert.InDelta(t, sphericalLawOfCosinesMeters(a, b), distance, 1.0)
	})

	t.Run("should match an independent great-circle formula across the antimeridian", func(t *testing.T) {
		// given
		a := builddomain.NewTrackPointBuilder().WithLatitude(0.0).WithLongitude(179.9).Build()
		b := builddomain.NewTrackPointBuilder().WithLatitude(0.0).WithLongitude(-179.9).Build()

		// when
		distance := a.DistanceTo(b)

		// then
		assert.InDelta(t, sphericalLawOfCosinesMeters(a, b), distance, 1.0)
	})

	t.Run("should report a short distance for a narrow antimeridian crossing", func(t *testing.T) {
		// given: two points only 0.2 degrees apart across the antimeridian
		a := builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build()
		b := builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build()

		// when
		distance := a.DistanceTo(b)

		// then: a naive (non-periodic) longitude subtraction would instead
		// compute a distance close to half the Earth's circumference (~20,000 km)
		assert.Less(t, distance, 30000.0)
	})
}

func Test_Route_Length(t *testing.T) {
	t.Run("should return zero for an empty route", func(t *testing.T) {
		// given
		var points []domain.TrackPoint

		// when
		distance := (domain.Route{Points: points}).Length()

		// then
		assert.Equal(t, 0.0, distance)
	})

	t.Run("should return zero for a single point", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{builddomain.NewTrackPointBuilder().Build()}

		// when
		distance := (domain.Route{Points: points}).Length()

		// then
		assert.Equal(t, 0.0, distance)
	})

	t.Run("should sum the distance of every consecutive segment, including one crossing the antimeridian", func(t *testing.T) {
		// given
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.8).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(179.9).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(-179.9).Build(),
		}
		expectedDistance := points[0].DistanceTo(points[1]) + points[1].DistanceTo(points[2])

		// when
		distance := (domain.Route{Points: points}).Length()

		// then
		assert.InDelta(t, expectedDistance, distance, 0.001)
	})
}
