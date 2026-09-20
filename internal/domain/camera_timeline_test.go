package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
)

func defaultTuning() domain.CameraTuning {
	return builddomain.NewCameraTuningBuilder().Build()
}

// timedLine builds a straight route at a constant speed, one point every
// stepSeconds.
func timedLine(speed, stepSeconds float64, count int) []domain.TrackPoint {
	length := speed * stepSeconds * float64(count-1)
	return builddomain.NewSyntheticRouteBuilder().WithLine(length, 90).WithPointCount(count).WithConstantSpeed(speed).Build()
}

func Test_BuildMarkerTimeline(t *testing.T) {
	t.Run("should use the clock when every point has a time, the duration is positive and no timestamps are inconsistent", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 20)

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceClock, timeline.Reference)
		assert.Empty(t, timeline.FallbackReason)
	})

	t.Run("should fall back to distance with reason 'no time data' when points have no time", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(1000, 90).Build()

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceDistance, timeline.Reference)
		assert.Equal(t, "no time data", timeline.FallbackReason)
	})

	t.Run("should fall back to distance when only some points have a time", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 10)
		points[4].Time = nil

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceDistance, timeline.Reference)
		assert.Equal(t, "no time data", timeline.FallbackReason)
	})

	t.Run("should fall back to distance when the total duration is zero", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(1000, 90).WithPointCount(5).Build()
		same := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		for i := range points {
			points[i].Time = &same
		}

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceDistance, timeline.Reference)
		assert.Equal(t, "time data is inconsistent", timeline.FallbackReason)
	})

	t.Run("should fall back to distance when two points share a time but are more than a meter apart", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 10)
		points[5].Time = points[4].Time

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceDistance, timeline.Reference)
		assert.Equal(t, "time data is inconsistent", timeline.FallbackReason)
	})

	t.Run("should keep the clock when two points share a time but are within a meter of each other", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(1000, 90).WithPointCount(3).WithConstantSpeed(5).Build()
		points[1].Longitude = points[0].Longitude
		points[1].Latitude = points[0].Latitude
		points[1].Time = points[0].Time

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceClock, timeline.Reference)
	})

	t.Run("should start at zero, end at the total length and never go backwards", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 30)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when
		previous := -1.0
		for i := 0; i <= 100; i++ {
			distance := timeline.DistanceAt(float64(i) / 100)

			// then
			assert.GreaterOrEqual(t, distance, previous)
			previous = distance
		}

		// then
		assert.Equal(t, 0.0, timeline.DistanceAt(0))
		assert.InDelta(t, (domain.Route{Points: points}).Length(), timeline.DistanceAt(1), 1e-6)
		assert.InDelta(t, (domain.Route{Points: points}).Length(), timeline.Total(), 1e-6)
	})

	t.Run("should advance linearly when the speed is constant", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 30)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())
		total := (domain.Route{Points: points}).Length()

		// when
		quarter, half := timeline.DistanceAt(0.25), timeline.DistanceAt(0.5)

		// then
		assert.InDelta(t, total/4, quarter, 1)
		assert.InDelta(t, total/2, half, 1)
	})

	t.Run("should advance by distance when there is no time data", func(t *testing.T) {
		// given
		points := builddomain.NewSyntheticRouteBuilder().WithLine(3000, 90).Build()
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when
		half := timeline.DistanceAt(0.5)

		// then
		assert.InDelta(t, 1500.0, half, 1)
	})

	t.Run("should clamp fractions outside [0, 1]", func(t *testing.T) {
		// given
		points := timedLine(5, 10, 10)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when / then
		assert.Equal(t, timeline.DistanceAt(0), timeline.DistanceAt(-1))
		assert.Equal(t, timeline.DistanceAt(1), timeline.DistanceAt(2))
	})

	t.Run("should return zero for an empty timeline", func(t *testing.T) {
		// given
		timeline := domain.MarkerTimeline{}

		// when / then
		assert.Equal(t, 0.0, timeline.DistanceAt(0.5))
		assert.Equal(t, 0.0, timeline.Total())
	})

	t.Run("should jump over a segment with no elapsed effective time", func(t *testing.T) {
		// given: two points a few centimeters apart sharing a timestamp
		points := builddomain.NewSyntheticRouteBuilder().WithLine(1000, 90).WithPointCount(4).WithConstantSpeed(5).Build()
		points[2].Time = points[1].Time
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when
		distance := timeline.DistanceAt(0.5)

		// then: no NaN, and still within the route
		assert.False(t, math.IsNaN(distance))
		assert.LessOrEqual(t, distance, timeline.Total())
	})
}

// routeWithStop builds a route moving at 5 m/s, then standing still for
// stopSeconds, then moving again, with a point every 10 s.
func routeWithStop(stopSeconds int) []domain.TrackPoint {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	var points []domain.TrackPoint
	elapsed, lon := 0, 0.0

	add := func() {
		points = append(points, builddomain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(lon).WithTime(start.Add(time.Duration(elapsed)*time.Second)).Build())
	}
	move := func(seconds int) {
		for s := 0; s < seconds; s += 10 {
			elapsed += 10
			lon += 50 / 111195.0 // 50 m per 10 s at the equator
			add()
		}
	}
	stand := func(seconds int) {
		for s := 0; s < seconds; s += 10 {
			elapsed += 10
			add()
		}
	}

	add()
	move(600)
	stand(stopSeconds)
	move(600)
	return points
}

func Test_BuildMarkerTimeline_LongStops(t *testing.T) {
	stopShare := func(points []domain.TrackPoint, timeline domain.MarkerTimeline) float64 {
		// the share of the video (fraction of the effective time) during which the marker barely moves
		var stopped float64
		const steps = 4000
		for i := 0; i < steps; i++ {
			a, b := timeline.DistanceAt(float64(i)/steps), timeline.DistanceAt(float64(i+1)/steps)
			if b-a < 1e-6 {
				stopped += 1.0 / steps
			}
		}
		return stopped
	}

	t.Run("should compress a ten minute stop to at most 5 percent of the video", func(t *testing.T) {
		// given
		points := routeWithStop(600)

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		assert.LessOrEqual(t, stopShare(points, timeline), 0.05)
		assert.Greater(t, stopShare(points, timeline), 0.0)
	})

	t.Run("should not alter a stop shorter than the long stop threshold", func(t *testing.T) {
		// given: a 20 s stop, not a long one
		points := routeWithStop(20)
		uncompressed := (domain.Route{Points: points}).Length()

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then: the stop keeps its real weight, 20 s out of about 1220 s
		assert.InDelta(t, 20.0/1220.0, stopShare(points, timeline), 0.005)
		assert.InDelta(t, uncompressed, timeline.Total(), 1e-6)
	})

	t.Run("should keep the video/real time ratio constant outside the stops", func(t *testing.T) {
		// given
		points := routeWithStop(600)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when: speed in meters per unit of progress, first and second moving halves
		firstSpeed := (timeline.DistanceAt(0.20) - timeline.DistanceAt(0.10)) / 0.10
		secondSpeed := (timeline.DistanceAt(0.95) - timeline.DistanceAt(0.85)) / 0.10

		// then: within 5 percent
		assert.InEpsilon(t, firstSpeed, secondSpeed, 0.05)
	})

	t.Run("should not make the marker jump while stopped", func(t *testing.T) {
		// given
		points := routeWithStop(600)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when
		maxStep := 0.0
		const steps = 4000
		for i := 0; i < steps; i++ {
			maxStep = math.Max(maxStep, timeline.DistanceAt(float64(i+1)/steps)-timeline.DistanceAt(float64(i)/steps))
		}

		// then: no step larger than a few times the moving average step
		average := timeline.Total() / steps
		assert.Less(t, maxStep, 3*average)
	})

	t.Run("should ignore the GPS jitter inside a long stop, so the marker does not hop", func(t *testing.T) {
		// given: the stop's points wander back and forth by about a meter
		points := routeWithStop(600)
		jitter := 0
		for i := 1; i < len(points); i++ {
			if points[i].Longitude == points[i-1].Longitude { // standing still
				jitter++
				points[i].Latitude += float64(jitter%2*2-1) * 0.00001 // about 1.1 m, alternating sides
			}
		}
		require.Positive(t, jitter)
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// when
		maxStep := 0.0
		const steps = 4000
		for i := 0; i < steps; i++ {
			maxStep = math.Max(maxStep, timeline.DistanceAt(float64(i+1)/steps)-timeline.DistanceAt(float64(i)/steps))
		}

		// then: the jitter adds up to tens of meters, none of which the marker travels
		assert.Less(t, timeline.Total(), (domain.Route{Points: points}).Length()-30)
		assert.Less(t, maxStep, 3*timeline.Total()/steps)
	})

	t.Run("should not compress anything when there is no time data", func(t *testing.T) {
		// given
		points := routeWithStop(600)
		for i := range points {
			points[i].Time = nil
		}

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: points}, defaultTuning())

		// then
		require.Equal(t, domain.TimeReferenceDistance, timeline.Reference)
		assert.InDelta(t, timeline.Total()/2, timeline.DistanceAt(0.5), 1)
	})

	t.Run("should compress each of two long stops to at most 5 percent of the moving time", func(t *testing.T) {
		// given
		first := routeWithStop(600)
		second := routeWithStop(600)
		offset := first[len(first)-1].Time.Sub(*second[0].Time) + 10*time.Second
		lonShift := first[len(first)-1].Longitude
		for _, p := range second {
			shifted := p.Time.Add(offset)
			p.Time = &shifted
			p.Longitude += lonShift
			first = append(first, p)
		}

		// when
		timeline := domain.NewMarkerTimeline(domain.Route{Points: first}, defaultTuning())

		// then
		assert.Equal(t, domain.TimeReferenceClock, timeline.Reference)
		assert.LessOrEqual(t, stopShare(first, timeline), 0.10)
	})
}
