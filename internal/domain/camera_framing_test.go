package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// angleFromAxis is the angle, in degrees, between the camera's optical axis
// and the ray to a point on the ground of the local plane.
func angleFromAxis(view domain.CameraView, point domain.PlanePoint) float64 {
	pose := domain.ComputeCameraPose(view)
	toTarget := [3]float64{view.Target.X - pose.Position.X, view.Target.Y - pose.Position.Y, -pose.Altitude}
	toPoint := [3]float64{point.X - pose.Position.X, point.Y - pose.Position.Y, -pose.Altitude}

	dot := toTarget[0]*toPoint[0] + toTarget[1]*toPoint[1] + toTarget[2]*toPoint[2]
	norms := math.Sqrt(toTarget[0]*toTarget[0]+toTarget[1]*toTarget[1]+toTarget[2]*toTarget[2]) *
		math.Sqrt(toPoint[0]*toPoint[0]+toPoint[1]*toPoint[1]+toPoint[2]*toPoint[2])
	return math.Acos(math.Min(1, dot/norms)) * 180 / math.Pi
}

func Test_OverviewView(t *testing.T) {
	tuning := defaultTuning()
	route, _ := planarLine(1, 1, 100, 200) // a 28 km diagonal

	t.Run("should look at the center of the route's bounding box, tilted and pointing along the given heading", func(t *testing.T) {
		// when
		view := domain.OverviewView(route, 123, 0, tuning)

		// then
		assert.InDelta(t, route[len(route)-1].X/2, view.Target.X, 1e-6)
		assert.InDelta(t, route[len(route)-1].Y/2, view.Target.Y, 1e-6)
		assert.Equal(t, 60.0, view.TiltDegrees)
		assert.Equal(t, 123.0, view.Heading)
	})

	t.Run("should frame every point of the route within half the vertical field of view", func(t *testing.T) {
		// when
		view := domain.OverviewView(route, 45, 0, tuning)

		// then
		for _, p := range route {
			assert.LessOrEqual(t, angleFromAxis(view, p), tuning.OverviewVerticalFOVDegrees/2, "point %v", p)
		}
	})

	t.Run("should never be closer than the minimum distance", func(t *testing.T) {
		// given: a tiny route, framed from very close by the field of view alone
		tiny, _ := planarLine(1, 0, 1, 50)

		// when
		view := domain.OverviewView(tiny, 0, 1200, tuning)

		// then
		assert.Equal(t, 1200.0, view.Distance)
	})

	t.Run("should use the framing distance when it exceeds the minimum", func(t *testing.T) {
		// when
		view := domain.OverviewView(route, 0, 10, tuning)

		// then: margin × radius / tan(FOV/2), radius half the diagonal
		radius := math.Hypot(route[len(route)-1].X, route[len(route)-1].Y) / 2
		assert.InDelta(t, 1.2*radius/math.Tan(22.5*math.Pi/180), view.Distance, 1e-6)
	})
}

func Test_BlendView(t *testing.T) {
	from := domain.CameraView{Target: domain.PlanePoint{X: 0, Y: 0}, Heading: 350, TiltDegrees: 60, Distance: 10000}
	to := domain.CameraView{Target: domain.PlanePoint{X: 100, Y: 200}, Heading: 10, TiltDegrees: 30, Distance: 100}

	t.Run("should return exactly the first view at 0 and the second at 1", func(t *testing.T) {
		// when
		start, end := domain.BlendView(from, to, 0), domain.BlendView(from, to, 1)

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
		middle := domain.BlendView(from, to, 0.5)

		// then: 350 → 10 goes through 0, not through 180
		assert.InDelta(t, 0.0, math.Min(middle.Heading, 360-middle.Heading), 1e-9)
	})

	t.Run("should ease gently at both ends", func(t *testing.T) {
		// given
		epsilon := 1e-4

		// when
		first := domain.BlendView(from, to, epsilon)
		last := domain.BlendView(from, to, 1-epsilon)

		// then: the derivative of a smoothstep is zero at both ends
		assert.InDelta(t, from.TiltDegrees, first.TiltDegrees, 1e-5)
		assert.InDelta(t, to.TiltDegrees, last.TiltDegrees, 1e-5)
	})

	t.Run("should be monotonic between the ends", func(t *testing.T) {
		// when
		previous := domain.BlendView(from, to, 0).TiltDegrees
		for i := 1; i <= 100; i++ {
			current := domain.BlendView(from, to, float64(i)/100).TiltDegrees

			// then
			assert.LessOrEqual(t, current, previous+1e-12)
			previous = current
		}
	})

	t.Run("should interpolate the distance in log space", func(t *testing.T) {
		// when
		middle := domain.BlendView(from, to, 0.5)

		// then: the geometric mean
		assert.InDelta(t, math.Sqrt(from.Distance*to.Distance), middle.Distance, 1e-6)
	})

	t.Run("should clamp u outside [0, 1]", func(t *testing.T) {
		// when
		before, after := domain.BlendView(from, to, -1), domain.BlendView(from, to, 2)

		// then
		assert.InDelta(t, from.Distance, before.Distance, 1e-9)
		assert.InDelta(t, to.Distance, after.Distance, 1e-9)
	})
}
