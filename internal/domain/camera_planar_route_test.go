package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waliqueiroz/sobrevoo/internal/domain"
)

// planarLine returns a route along a straight line on the plane, one point
// every stepMeters, along the direction (dx, dy).
func planarLine(dx, dy, stepMeters float64, count int) domain.PlanarRoute {
	norm := math.Hypot(dx, dy)
	route := domain.PlanarRoute{Points: make([]domain.PlanePoint, count), Distances: make([]float64, count)}
	for i := range route.Points {
		route.Distances[i] = float64(i) * stepMeters
		route.Points[i] = domain.PlanePoint{X: dx / norm * route.Distances[i], Y: dy / norm * route.Distances[i]}
	}
	return route
}

// angleFromAxis is the angle, in degrees, between the camera's optical axis
// and the ray to a point on the ground of the local plane.
func angleFromAxis(view domain.CameraView, point domain.PlanePoint) float64 {
	pose := view.Pose()
	toTarget := [3]float64{view.Target.X - pose.Position.X, view.Target.Y - pose.Position.Y, -pose.Altitude}
	toPoint := [3]float64{point.X - pose.Position.X, point.Y - pose.Position.Y, -pose.Altitude}

	dot := toTarget[0]*toPoint[0] + toTarget[1]*toPoint[1] + toTarget[2]*toPoint[2]
	norms := math.Sqrt(toTarget[0]*toTarget[0]+toTarget[1]*toTarget[1]+toTarget[2]*toTarget[2]) *
		math.Sqrt(toPoint[0]*toPoint[0]+toPoint[1]*toPoint[1]+toPoint[2]*toPoint[2])
	return math.Acos(math.Min(1, dot/norms)) * 180 / math.Pi
}

func Test_PlanarRoute_HeadingAt(t *testing.T) {
	t.Run("should point east along a route going east", func(t *testing.T) {
		// given
		route := planarLine(1, 0, 10, 100)

		// when
		heading, ok := route.HeadingAt(500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 90.0, heading, 1e-9)
	})

	t.Run("should point north along a route going north", func(t *testing.T) {
		// given
		route := planarLine(0, 1, 10, 100)

		// when
		heading, ok := route.HeadingAt(500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 0.0, heading, 1e-9)
	})

	t.Run("should point south-west along a route going south-west", func(t *testing.T) {
		// given
		route := planarLine(-1, -1, 10, 100)

		// when
		heading, ok := route.HeadingAt(500, 100)

		// then
		require.True(t, ok)
		assert.InDelta(t, 225.0, heading, 1e-9)
	})

	t.Run("should report no clear direction where the tangents cancel out in a turn back", func(t *testing.T) {
		// given: out 500 m east, then back the same way
		out := planarLine(1, 0, 10, 51)
		route := domain.PlanarRoute{Points: append([]domain.PlanePoint{}, out.Points...)}
		for i := len(out.Points) - 2; i >= 0; i-- {
			route.Points = append(route.Points, out.Points[i])
		}
		route.Distances = make([]float64, len(route.Points))
		for i := 1; i < len(route.Points); i++ {
			route.Distances[i] = route.Distances[i-1] + 10
		}

		// when
		_, ok := route.HeadingAt(500, 100)

		// then
		assert.False(t, ok)
	})

	t.Run("should report no clear direction for a loop tighter than the window", func(t *testing.T) {
		// given: a 20 m radius circle, many laps, seen through a 500 m window
		var route domain.PlanarRoute
		for i := 0; i < 600; i++ {
			angle := float64(i) * 2 * math.Pi / 50
			route.Points = append(route.Points, domain.PlanePoint{X: 20 * math.Cos(angle), Y: 20 * math.Sin(angle)})
			route.Distances = append(route.Distances, float64(i)*2*math.Pi*20/50)
		}

		// when
		_, ok := route.HeadingAt(route.Distances[300], 500)

		// then
		assert.False(t, ok)
	})

	t.Run("should report no clear direction for a route of fewer than two points or a non-positive sigma", func(t *testing.T) {
		// given
		route := planarLine(1, 0, 10, 10)
		single := domain.PlanarRoute{Points: route.Points[:1], Distances: route.Distances[:1]}

		// when
		_, tooShort := single.HeadingAt(0, 100)
		_, noSigma := route.HeadingAt(50, 0)

		// then
		assert.False(t, tooShort)
		assert.False(t, noSigma)
	})

	t.Run("should report no clear direction when every segment in the window has zero length", func(t *testing.T) {
		// given
		route := domain.PlanarRoute{
			Points:    []domain.PlanePoint{{X: 1, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 1}},
			Distances: []float64{0, 0, 0},
		}

		// when
		_, ok := route.HeadingAt(0, 100)

		// then
		assert.False(t, ok)
	})
}

func Test_PlanarRoute_Length(t *testing.T) {
	t.Run("should be the distance travelled at the last point", func(t *testing.T) {
		// given
		route := planarLine(1, 1, 10, 11)

		// when / then
		assert.Equal(t, 100.0, route.Length())
	})
}

func Test_PlanarRoute_Span(t *testing.T) {
	t.Run("should be twice the largest distance from the center to a point", func(t *testing.T) {
		// given: the plane's center is the origin
		route := domain.PlanarRoute{Points: []domain.PlanePoint{{X: -300, Y: 0}, {X: 0, Y: 0}, {X: 100, Y: 0}}, Distances: []float64{0, 300, 400}}

		// when / then
		assert.Equal(t, 600.0, route.Span())
	})
}

func Test_PlanarRoute_PointAt(t *testing.T) {
	route := planarLine(1, 0, 100, 11) // 0 to 1000 m east

	t.Run("should interpolate along the segment", func(t *testing.T) {
		// when
		point := route.PointAt(250)

		// then
		assert.InDelta(t, 250.0, point.X, 1e-9)
		assert.InDelta(t, 0.0, point.Y, 1e-9)
	})

	t.Run("should return the first point at distance zero", func(t *testing.T) {
		// when / then
		assert.Equal(t, route.Points[0], route.PointAt(0))
	})

	t.Run("should clamp a distance beyond the end to the last point", func(t *testing.T) {
		// when
		point := route.PointAt(5000)

		// then
		assert.InDelta(t, 1000.0, point.X, 1e-9)
	})
}

func Test_PlanarRoute_ChordHeading(t *testing.T) {
	t.Run("should be the direction from the first to the last point", func(t *testing.T) {
		// given
		east, south, northWest := planarLine(1, 0, 10, 5), planarLine(0, -1, 10, 5), planarLine(-1, 1, 10, 5)

		// when / then
		assert.InDelta(t, 90.0, east.ChordHeading(), 1e-9)
		assert.InDelta(t, 180.0, south.ChordHeading(), 1e-9)
		assert.InDelta(t, 315.0, northWest.ChordHeading(), 1e-9)
	})
}

func Test_PlanarRoute_OverviewView(t *testing.T) {
	tuning := defaultTuning()
	route := planarLine(1, 1, 100, 200) // a 28 km diagonal

	t.Run("should look at the center of the route's bounding box, tilted and pointing along the given heading", func(t *testing.T) {
		// when
		view := route.OverviewView(123, 0, tuning)

		// then
		assert.InDelta(t, route.Points[len(route.Points)-1].X/2, view.Target.X, 1e-6)
		assert.InDelta(t, route.Points[len(route.Points)-1].Y/2, view.Target.Y, 1e-6)
		assert.Equal(t, 60.0, view.TiltDegrees)
		assert.Equal(t, 123.0, view.Heading)
	})

	t.Run("should frame every point of the route within half the vertical field of view", func(t *testing.T) {
		// when
		view := route.OverviewView(45, 0, tuning)

		// then
		for _, p := range route.Points {
			assert.LessOrEqual(t, angleFromAxis(view, p), tuning.OverviewVerticalFOVDegrees/2, "point %v", p)
		}
	})

	t.Run("should never be closer than the minimum distance", func(t *testing.T) {
		// given: a tiny route, framed from very close by the field of view alone
		tiny := planarLine(1, 0, 1, 50)

		// when
		view := tiny.OverviewView(0, 1200, tuning)

		// then
		assert.Equal(t, 1200.0, view.Distance)
	})

	t.Run("should use the framing distance when it exceeds the minimum", func(t *testing.T) {
		// when
		view := route.OverviewView(0, 10, tuning)

		// then: margin × radius / tan(FOV/2), radius half the diagonal
		last := route.Points[len(route.Points)-1]
		radius := math.Hypot(last.X, last.Y) / 2
		assert.InDelta(t, 1.2*radius/math.Tan(22.5*math.Pi/180), view.Distance, 1e-6)
	})
}
