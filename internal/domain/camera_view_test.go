package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

func Test_CameraView_Pose(t *testing.T) {
	t.Run("should place the camera behind the target, along the heading, at the tilt's altitude", func(t *testing.T) {
		// given: heading east (90), tilt 45, distance 100
		view := domain.CameraView{Target: domain.PlanePoint{X: 10, Y: 20}, Heading: 90, TiltDegrees: 45, Distance: 100}

		// when
		pose := view.Pose()

		// then: 70.7 m west of the target, 70.7 m up
		assert.InDelta(t, 10-70.7107, pose.Position.X, 1e-3)
		assert.InDelta(t, 20.0, pose.Position.Y, 1e-3)
		assert.InDelta(t, 70.7107, pose.Altitude, 1e-3)
	})

	t.Run("should put the camera straight above the target when looking straight down", func(t *testing.T) {
		// given
		view := domain.CameraView{Target: domain.PlanePoint{X: 5, Y: 5}, Heading: 30, TiltDegrees: 90, Distance: 100}

		// when
		pose := view.Pose()

		// then
		assert.InDelta(t, 5.0, pose.Position.X, 1e-9)
		assert.InDelta(t, 5.0, pose.Position.Y, 1e-9)
		assert.InDelta(t, 100.0, pose.Altitude, 1e-9)
	})

	t.Run("should place the camera south of the target when heading north", func(t *testing.T) {
		// given
		view := domain.CameraView{Heading: 0, TiltDegrees: 0, Distance: 50}

		// when
		pose := view.Pose()

		// then
		assert.InDelta(t, 0.0, pose.Position.X, 1e-9)
		assert.InDelta(t, -50.0, pose.Position.Y, 1e-9)
		assert.InDelta(t, 0.0, pose.Altitude, 1e-9)
	})
}

func Test_CameraView_Blend(t *testing.T) {
	from := domain.CameraView{Target: domain.PlanePoint{X: 0, Y: 0}, Heading: 350, TiltDegrees: 60, Distance: 10000}
	to := domain.CameraView{Target: domain.PlanePoint{X: 100, Y: 200}, Heading: 10, TiltDegrees: 30, Distance: 100}

	t.Run("should return exactly the first view at 0 and the second at 1", func(t *testing.T) {
		// when
		start, end := from.Blend(to, 0), from.Blend(to, 1)

		// then
		assert.InDelta(t, from.Distance, start.Distance, 1e-9)
		assert.InDelta(t, from.Heading, start.Heading, 1e-9)
		assert.InDelta(t, to.Distance, end.Distance, 1e-9)
		assert.InDelta(t, to.Heading, end.Heading, 1e-9)
		assert.Equal(t, to.Target, end.Target)
		assert.Equal(t, to.TiltDegrees, end.TiltDegrees)
	})

	t.Run("should take the shortest arc for the heading", func(t *testing.T) {
		// when
		middle := from.Blend(to, 0.5)

		// then: 350 → 10 goes through 0, not through 180
		assert.InDelta(t, 0.0, math.Min(middle.Heading, 360-middle.Heading), 1e-9)
	})

	t.Run("should ease gently at both ends", func(t *testing.T) {
		// given
		epsilon := 1e-4

		// when
		first := from.Blend(to, epsilon)
		last := from.Blend(to, 1-epsilon)

		// then: the derivative of a smoothstep is zero at both ends
		assert.InDelta(t, from.TiltDegrees, first.TiltDegrees, 1e-5)
		assert.InDelta(t, to.TiltDegrees, last.TiltDegrees, 1e-5)
	})

	t.Run("should be monotonic between the ends", func(t *testing.T) {
		// when
		previous := from.Blend(to, 0).TiltDegrees
		for i := 1; i <= 100; i++ {
			current := from.Blend(to, float64(i)/100).TiltDegrees

			// then
			assert.LessOrEqual(t, current, previous+1e-12)
			previous = current
		}
	})

	t.Run("should interpolate the distance in log space", func(t *testing.T) {
		// when
		middle := from.Blend(to, 0.5)

		// then: the geometric mean
		assert.InDelta(t, math.Sqrt(from.Distance*to.Distance), middle.Distance, 1e-6)
	})

	t.Run("should clamp u outside [0, 1]", func(t *testing.T) {
		// when
		before, after := from.Blend(to, -1), from.Blend(to, 2)

		// then
		assert.InDelta(t, from.Distance, before.Distance, 1e-9)
		assert.InDelta(t, to.Distance, after.Distance, 1e-9)
	})
}
