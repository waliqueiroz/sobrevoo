package simplifier_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
	"github.com/waliqueiroz/sobrevoo/internal/domain/build_domain"
	"github.com/waliqueiroz/sobrevoo/internal/infra/outbound/simplifier"
)

// jitteryLine builds points along a straight line from (0,0) to (0,0.01)
// with a small alternating jitter (~5.5m, between the low ~2m and high
// ~15m tolerances, so each level discards a different amount of it), plus
// one point with a large, unmistakable deviation representing a real turn.
func jitteryLine() []domain.TrackPoint {
	points := make([]domain.TrackPoint, 0, 21)
	for i := 0; i <= 20; i++ {
		lon := float64(i) * 0.0005
		lat := 0.0
		switch {
		case i == 10:
			lat = 0.0008 // a real, large turn — must survive any level
		case i%2 == 0:
			lat = 0.00005
		default:
			lat = -0.00005
		}
		points = append(points, build_domain.NewTrackPointBuilder().WithLatitude(lat).WithLongitude(lon).Build())
	}
	return points
}

func Test_Simplifier_Simplify(t *testing.T) {
	t.Run("should return fewer than three points unchanged", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(1).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(2).Build(),
		}

		// when
		result := douglasPeucker.Simplify(points, domain.LevelHigh)

		// then
		assert.Equal(t, points, result)
	})

	t.Run("should keep the first and last point at the low level", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		result := douglasPeucker.Simplify(points, domain.LevelLow)

		// then
		require.NotEmpty(t, result)
		assert.Equal(t, points[0], result[0])
		assert.Equal(t, points[len(points)-1], result[len(result)-1])
	})

	t.Run("should keep the first and last point at the medium level", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		result := douglasPeucker.Simplify(points, domain.LevelMedium)

		// then
		require.NotEmpty(t, result)
		assert.Equal(t, points[0], result[0])
		assert.Equal(t, points[len(points)-1], result[len(result)-1])
	})

	t.Run("should keep the first and last point at the high level", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		result := douglasPeucker.Simplify(points, domain.LevelHigh)

		// then
		require.NotEmpty(t, result)
		assert.Equal(t, points[0], result[0])
		assert.Equal(t, points[len(points)-1], result[len(result)-1])
	})

	t.Run("should preserve a point with a large, real deviation even at the highest level", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		result := douglasPeucker.Simplify(points, domain.LevelHigh)

		// then
		foundRealTurn := false
		for _, p := range result {
			if p.Latitude == 0.0008 {
				foundRealTurn = true
			}
		}
		assert.True(t, foundRealTurn)
	})

	t.Run("should handle a degenerate segment whose endpoints coincide", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := []domain.TrackPoint{
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
			build_domain.NewTrackPointBuilder().WithLatitude(0.001).WithLongitude(0).Build(), // off to the side of a zero-length segment
			build_domain.NewTrackPointBuilder().WithLatitude(0).WithLongitude(0).Build(),
		}

		// when
		result := douglasPeucker.Simplify(points, domain.LevelLow)

		// then
		assert.Equal(t, points[0], result[0])
		assert.Equal(t, points[len(points)-1], result[len(result)-1])
	})

	t.Run("should keep fewer points at the high level than at the low level (SC-006)", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		low := douglasPeucker.Simplify(points, domain.LevelLow)
		high := douglasPeucker.Simplify(points, domain.LevelHigh)

		// then
		assert.Less(t, len(high), len(low))
		assert.Less(t, len(high), len(points))
	})

	t.Run("should keep fewer or as many points at the medium level as at the low level (SC-006)", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		low := douglasPeucker.Simplify(points, domain.LevelLow)
		medium := douglasPeucker.Simplify(points, domain.LevelMedium)

		// then
		assert.LessOrEqual(t, len(medium), len(low))
	})

	t.Run("should keep fewer or as many points at the high level as at the medium level (SC-006)", func(t *testing.T) {
		// given
		douglasPeucker := simplifier.NewDouglasPeucker()
		points := jitteryLine()

		// when
		medium := douglasPeucker.Simplify(points, domain.LevelMedium)
		high := douglasPeucker.Simplify(points, domain.LevelHigh)

		// then
		assert.LessOrEqual(t, len(high), len(medium))
	})
}
