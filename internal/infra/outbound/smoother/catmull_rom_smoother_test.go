package smoother_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/builddomain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/smoother"
)

// jitteryPoints builds points along a straight line whose latitude
// alternates above and below it — a discrete stand-in for GPS jitter.
func jitteryPoints() []domain.TrackPoint {
	points := make([]domain.TrackPoint, 0, 10)
	for i := range 10 {
		lat := 0.0001
		if i%2 == 0 {
			lat = -0.0001
		}
		points = append(points, builddomain.NewTrackPointBuilder().WithLatitude(lat).WithLongitude(float64(i)*0.001).Build())
	}
	return points
}

// jitter sums the discrete second difference of latitude across the route:
// a simple, well-known measure of point-to-point zig-zagging. Smoothing
// should reduce it.
func jitter(points []domain.TrackPoint) float64 {
	var total float64
	for i := 1; i < len(points)-1; i++ {
		total += math.Abs(points[i+1].Latitude - 2*points[i].Latitude + points[i-1].Latitude)
	}
	return total
}

func Test_Smoother_Smooth(t *testing.T) {
	t.Run("should return fewer than four points unchanged", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := []domain.TrackPoint{
			builddomain.NewTrackPointBuilder().WithLatitude(1).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(2).Build(),
			builddomain.NewTrackPointBuilder().WithLatitude(3).Build(),
		}

		// when
		result := catmullRom.Smooth(points, domain.LevelHigh)

		// then
		assert.Equal(t, points, result)
	})

	t.Run("should never change the first and last point", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := jitteryPoints()

		// when
		result := catmullRom.Smooth(points, domain.LevelHigh)

		// then
		assert.Equal(t, points[0], result[0])
		assert.Equal(t, points[len(points)-1], result[len(result)-1])
	})

	t.Run("should never change elevation", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := jitteryPoints()
		for i := range points {
			points[i].Elevation = new(float64(i))
		}

		// when
		result := catmullRom.Smooth(points, domain.LevelHigh)

		// then
		require.Len(t, result, len(points))
		for i := range points {
			require.NotNil(t, result[i].Elevation)
			assert.Equal(t, *points[i].Elevation, *result[i].Elevation)
		}
	})

	t.Run("should reduce jitter compared to the original route", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := jitteryPoints()

		// when
		smoothed := catmullRom.Smooth(points, domain.LevelLow)

		// then
		assert.Greater(t, jitter(points), jitter(smoothed))
	})

	t.Run("should reduce jitter further at the medium level than at the low level (SC-007)", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := jitteryPoints()

		// when
		low := catmullRom.Smooth(points, domain.LevelLow)
		medium := catmullRom.Smooth(points, domain.LevelMedium)

		// then
		assert.Greater(t, jitter(low), jitter(medium))
	})

	t.Run("should reduce jitter further at the high level than at the medium level (SC-007)", func(t *testing.T) {
		// given
		catmullRom := smoother.NewCatmullRom()
		points := jitteryPoints()

		// when
		medium := catmullRom.Smooth(points, domain.LevelMedium)
		high := catmullRom.Smooth(points, domain.LevelHigh)

		// then
		assert.Greater(t, jitter(medium), jitter(high))
	})
}
